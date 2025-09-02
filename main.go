package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Gagal memuat file .env")
	}

	token := os.Getenv("DISCORD_TOKEN")
	discordChannelID := os.Getenv("CHANNEL_ID")
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	youtubeChannelID := os.Getenv("YOUTUBE_CHANNEL_ID")

	// Session bot
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Gagal membuat Discord session:", err)
	}

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot sudah online sebagai", s.State.User.Username)
	})

	err = dg.Open()
	if err != nil {
		log.Fatal("Tidak bisa membuka koneksi:", err)
	}
	defer dg.Close()

	// Setup YouTube API
	ctx := context.Background()
	yts, err := youtube.NewService(ctx, option.WithAPIKey(youtubeAPIKey))
	if err != nil {
		log.Fatal("Gagal membuat service YouTube:", err)
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastVideoID := ""

	checkForNewContent := func() {
		fmt.Println("=== Checking for new content ===")
		
		// 1) Search for live streams first
		fmt.Println("Mencari live streams aktif...")
		liveCall := yts.Search.List([]string{"id", "snippet"}).
			ChannelId(youtubeChannelID).
			Type("video").
			EventType("live").
			MaxResults(10)

		liveResp, err := liveCall.Do()
		if err != nil {
			fmt.Println("Error searching live streams:", err)
		} else if len(liveResp.Items) > 0 {
			fmt.Printf("Ditemukan %d live stream aktif\n", len(liveResp.Items))
			for _, item := range liveResp.Items {
				videoID := item.Id.VideoId
				if videoID == "" {
					continue
				}

				title := html.UnescapeString(item.Snippet.Title)
				url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
				
				fmt.Printf("Live stream found: %s (ID: %s)\n", title, videoID)

				if videoID != lastVideoID {
					// Confirm via Videos API
					videoResp, err := yts.Videos.List([]string{"snippet", "liveStreamingDetails"}).Id(videoID).Do()
					if err != nil {
						fmt.Println("Error fetching video details:", err)
						continue
					}

					isActuallyLive := false
					if len(videoResp.Items) > 0 {
						v := videoResp.Items[0]
						if v.LiveStreamingDetails != nil {
							fmt.Printf("LiveStreamingDetails - Start: %s, End: %s\n", 
								v.LiveStreamingDetails.ActualStartTime, 
								v.LiveStreamingDetails.ActualEndTime)
							
							if v.LiveStreamingDetails.ActualStartTime != "" && v.LiveStreamingDetails.ActualEndTime == "" {
								isActuallyLive = true
							}
						}
					}

					if isActuallyLive {
						lastVideoID = videoID
						message := fmt.Sprintf("Halo <@&1411326750956458055> Pak RT Unyul lagi Stream **%s** nih, Gas nonton!\n%s", title, url)
						
						_, err = dg.ChannelMessageSend(discordChannelID, message)
						if err != nil {
							fmt.Println("Gagal kirim pesan stream:", err)
						} else {
							fmt.Println("✅ Pesan stream terkirim:", title)
						}
						return
					}
				}
			}
		} else {
			fmt.Println("Tidak ada live stream aktif ditemukan")
		}

		// 2) Fallback: check recent uploads
		fmt.Println("Fallback: Checking recent uploads...")
		call := yts.Search.List([]string{"id", "snippet"}).
			ChannelId(youtubeChannelID).
			Order("date").
			MaxResults(5)

		resp, err := call.Do()
		if err != nil {
			fmt.Println("Error fetching recent videos:", err)
			return
		}

		fmt.Printf("Ditemukan %d video terbaru\n", len(resp.Items))
		for _, item := range resp.Items {
			videoID := item.Id.VideoId
			if videoID == "" || videoID == lastVideoID {
				continue
			}

			title := html.UnescapeString(item.Snippet.Title)
			url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
			fmt.Printf("Checking video: %s (ID: %s)\n", title, videoID)

			// Check if this is actually a live stream
			videoResp, err := yts.Videos.List([]string{"snippet", "liveStreamingDetails"}).Id(videoID).Do()
			if err != nil {
				fmt.Println("Error fetching video details:", err)
				continue
			}

			isActuallyLive := false
			liveStatus := "none"
			if len(videoResp.Items) > 0 {
				v := videoResp.Items[0]
				liveStatus = v.Snippet.LiveBroadcastContent
				if v.LiveStreamingDetails != nil && v.LiveStreamingDetails.ActualStartTime != "" && v.LiveStreamingDetails.ActualEndTime == "" {
					isActuallyLive = true
				}
				fmt.Printf("LiveBroadcastContent: %s, isActuallyLive: %v\n", liveStatus, isActuallyLive)
			}

			if isActuallyLive || liveStatus == "live" {
				lastVideoID = videoID
				message := fmt.Sprintf("Halo <@&1411326750956458055> Pak RT Unyul lagi Stream **%s** nih, Gas nonton!\n%s", title, url)

				_, err = dg.ChannelMessageSend(discordChannelID, message)
				if err != nil {
					fmt.Println("Gagal kirim pesan stream (fallback):", err)
				} else {
					fmt.Println("Pesan stream terkirim (fallback):", title)
				}
				return
			} else if liveStatus == "none" && lastVideoID == "" {
				lastVideoID = videoID
				message := fmt.Sprintf("Halo <@&1411326750956458055> Pak RT Unyul upload video baru nih! **%s** nih, Gas nonton!\n%s", title, url)
				
				_, err = dg.ChannelMessageSend(discordChannelID, message)
				if err != nil {
					fmt.Println("Gagal kirim pesan upload:", err)
				} else {
					fmt.Println("Pesan upload terkirim:", title)
				}
				return
			}
		}
		fmt.Println("=== End check ===")
	}

	go func() {
		// Run immediately then on interval
		checkForNewContent()
		for range ticker.C {
			checkForNewContent()
		}
	}()

	fmt.Println("Bot berjalan... tekan CTRL+C untuk berhenti")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
}
