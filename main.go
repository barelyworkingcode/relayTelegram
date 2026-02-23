package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type Config struct {
	BotToken      string
	AllowedUserID int64
	EveURL        string
}

func loadConfig() (Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	uidStr := os.Getenv("TELEGRAM_ALLOWED_USER_ID")
	if uidStr == "" {
		return Config{}, fmt.Errorf("TELEGRAM_ALLOWED_USER_ID is required")
	}
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("TELEGRAM_ALLOWED_USER_ID must be a number: %w", err)
	}

	eveURL := os.Getenv("EVE_URL")
	if eveURL == "" {
		eveURL = "http://localhost:3000"
	}

	return Config{
		BotToken:      token,
		AllowedUserID: uid,
		EveURL:        eveURL,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	mappings, err := LoadMappings()
	if err != nil {
		log.Fatalf("Failed to load mappings: %v", err)
	}

	eve := NewEveClient(cfg.EveURL)

	bot, err := NewBot(cfg, eve, mappings)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		log.Println("Shutting down...")
		bot.Stop()
	}()

	log.Printf("Starting Telegram bot (allowed user: %d, Eve: %s)", cfg.AllowedUserID, cfg.EveURL)
	bot.Start()
}
