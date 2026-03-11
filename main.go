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
	LLMURL        string
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

	llmURL := os.Getenv("RELAY_LLM_URL")
	if llmURL == "" {
		llmURL = "http://localhost:3001"
	}

	return Config{
		BotToken:      token,
		AllowedUserID: uid,
		LLMURL:        llmURL,
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

	llm := NewLLMClient(cfg.LLMURL)

	bot, err := NewBot(cfg, llm, mappings)
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

	log.Printf("Starting Telegram bot (allowed user: %d, relayLLM: %s)", cfg.AllowedUserID, cfg.LLMURL)
	bot.Start()
}
