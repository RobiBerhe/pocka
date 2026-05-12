package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatabaseURL  string `envconfig:"DATABASE_URL" required:"true"`
	TelegramToken string `envconfig:"TELEGRAM_TOKEN" required:"true"`
	Port         string `envconfig:"PORT" default:"8080"`
	WebhookURL   string `envconfig:"WEBHOOK_URL"`
	Environment  string `envconfig:"ENVIRONMENT" default:"development"`
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error reading it. Relying on environment variables.")
	}

	var cfg Config
	err = envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("Failed to process env var: %v", err)
	}

	return &cfg
}
