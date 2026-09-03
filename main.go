package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/chai2010/webp"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	xwebp "golang.org/x/image/webp"
)

const (
	apiURL    = ""
	botToken  = ""
	chatID    = ""
	imagePath = "icekarim.webp"
	fontPath  = "Hack-Bold.ttf"
)

type apiItem struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

type apiResponse struct {
	Gold     []apiItem `json:"gold"`
	Currency []apiItem `json:"currency"`
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	baseImg, err := loadBaseImage(imagePath)
	if err != nil {
		log.Fatalf("load base image: %v", err)
	}
	fontFace, err := loadFont(fontPath)
	if err != nil {
		log.Fatalf("load font: %v", err)
	}

	run(baseImg, fontFace)

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		run(baseImg, fontFace)
	}
}

var (
	lastUSD  float64
	lastGold float64
)

func run(baseImg image.Image, fontFace *truetype.Font) {
	log.Println("starting price update cycle")

	loc, err := time.LoadLocation("Asia/Tehran")
	if err == nil {
		now := time.Now().In(loc)
		hour := now.Hour()
		weekday := now.Weekday()

		if weekday == time.Friday {
			log.Println("skipping: market closed on Fridays")
			return
		}
		if hour < 11 || hour >= 19 {
			log.Printf("skipping: market closed at %02d:00", hour)
			return
		}
	}

	usd, gold, err := fetchPrices()
	if err != nil {
		log.Printf("fetch prices: %v", err)
		return
	}

	if usd == lastUSD && gold == lastGold {
		log.Println("skipping: prices unchanged")
		return
	}

	lastUSD = usd
	lastGold = gold

	usdText := fmt.Sprintf("%gT", usd)
	goldText := fmt.Sprintf("%.3fM", gold)

	log.Printf("USD: %s  |  Gold 18K: %s", usdText, goldText)

	buf, err := renderSticker(baseImg, fontFace, usdText, goldText)
	if err != nil {
		log.Printf("render sticker: %v", err)
		return
	}

	if err := sendSticker(buf); err != nil {
		log.Printf("send sticker: %v", err)
		return
	}

	log.Println("sticker sent successfully")
}

func fetchPrices() (usd, gold float64, err error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", "curl/8.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("GET api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var data apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, fmt.Errorf("decode json: %w", err)
	}

	var foundUSD, foundGold bool

	for _, item := range data.Currency {
		if item.Symbol == "USD" {
			usd = item.Price / 1_000
			foundUSD = true
			break
		}
	}

	for _, item := range data.Gold {
		if item.Symbol == "IR_GOLD_18K" {
			gold = item.Price / 1_000_000
			foundGold = true
			break
		}
	}

	if !foundUSD {
		return 0, 0, fmt.Errorf("symbol USD not found in response")
	}
	if !foundGold {
		return 0, 0, fmt.Errorf("symbol IR_GOLD_18K not found in response")
	}

	return usd, gold, nil
}

func loadBaseImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return xwebp.Decode(f)
}

func loadFont(path string) (*truetype.Font, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return freetype.ParseFont(b)
}

func renderSticker(baseImg image.Image, fontFace *truetype.Font, usdText, goldText string) (*bytes.Buffer, error) {
	bounds := baseImg.Bounds()
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, baseImg, bounds.Min, draw.Src)

	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetFont(fontFace)
	ctx.SetFontSize(32)
	ctx.SetClip(canvas.Bounds())
	ctx.SetDst(canvas)
	ctx.SetHinting(font.HintingFull)

	ctx.SetSrc(image.NewUniform(color.RGBA{R: 0, G: 0, B: 0, A: 255}))
	if _, err := ctx.DrawString(usdText, freetype.Pt(147, 62)); err != nil {
		return nil, fmt.Errorf("draw usd shadow: %w", err)
	}
	if _, err := ctx.DrawString(goldText, freetype.Pt(147, 132)); err != nil {
		return nil, fmt.Errorf("draw gold shadow: %w", err)
	}

	ctx.SetSrc(image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}))
	if _, err := ctx.DrawString(usdText, freetype.Pt(145, 60)); err != nil {
		return nil, fmt.Errorf("draw usd text: %w", err)
	}
	if _, err := ctx.DrawString(goldText, freetype.Pt(145, 130)); err != nil {
		return nil, fmt.Errorf("draw gold text: %w", err)
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, canvas, &webp.Options{Lossless: false, Quality: 30}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}

	return &buf, nil
}

func sendSticker(stickerBuf *bytes.Buffer) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("chat_id", chatID); err != nil {
		return fmt.Errorf("write chat_id: %w", err)
	}

	part, err := writer.CreateFormFile("sticker", "sticker.webp")
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(stickerBuf.Bytes()); err != nil {
		return fmt.Errorf("write sticker data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendSticker", botToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, writer.FormDataContentType(), body)
	if err != nil {
		return fmt.Errorf("POST sendSticker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result json.RawMessage
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("telegram returned %d: %s", resp.StatusCode, result)
	}

	return nil
}
