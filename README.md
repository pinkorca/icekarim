# icekarim

[![GitHub](https://img.shields.io/github/license/pinkorca/icekarim)](https://github.com/pinkorca/icekarim)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)

High-performance Telegram bot for rendering live Iranian gold and USD prices as WebP stickers.

## Key Features

- **Market-Aware Scheduling**: Skips cycles outside Iranian market hours (11:00–19:00 IRST, Sat–Thu)
- **Price Change Detection**: Only renders and uploads when prices actually change
- **Zero Disk I/O**: Encodes the final WebP sticker entirely in memory using `bytes.Buffer`
- **Strict Resource Budget**: Single goroutine, strict HTTP timeouts, static assets loaded once at startup
- **Minimal Footprint**: ~10–15 MB RAM at runtime, effectively 0% CPU between cycles

## Requirements

- `icekarim.webp` — base sticker image
- `Hack-Bold.ttf` — font file
- A Telegram bot token and target channel

## Configuration

Open `main.go` and fill in the three constants at the top:

```go
const (
    apiURL   = ""  // BrsApi endpoint, e.g. https://Api.BrsApi.ir/Market/Gold_Currency.php?key=YOUR_KEY
    botToken = ""  // Telegram bot token from @BotFather
    chatID   = ""  // Target channel or chat, e.g. @yourChannel
)
```

## Installation

```bash
git clone https://github.com/pinkorca/icekarim.git
cd icekarim
go build -o icekarim .
```

## Deployment

Copy the binary and assets to your server, then create a systemd service:

```ini
[Unit]
Description=Icekarim Price Sticker Bot
After=network.target

[Service]
Type=simple
WorkingDirectory=/root/XD/icekarim
ExecStart=/root/XD/icekarim/icekarim
Restart=on-failure
RestartSec=10
User=root

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now icekarim
```

## License

[GNU General Public License v3.0](LICENSE)
