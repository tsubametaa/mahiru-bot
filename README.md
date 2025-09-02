# Mahiru Bot

Bot Discord yang memantau channel YouTube Pak RT Unyul dan mengirim notifikasi ke Discord ketika ada live stream atau video baru.

## Fitur

- Memantau live streams aktif
- Mengirim notifikasi untuk video baru
- Menggunakan YouTube Data API v3
- Deploy ke Fly.io

## Setup

1. Clone repo ini
2. Buat file `.env` dengan:
   - DISCORD_TOKEN
   - CHANNEL_ID
   - YOUTUBE_API_KEY
   - YOUTUBE_CHANNEL_ID
3. Jalankan `go run main.go`

## Deploy ke Fly.io

Lihat Dockerfile dan fly.toml untuk detail deploy.
