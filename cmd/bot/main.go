package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"pocka/internal/bot"
	"pocka/internal/config"
	"pocka/internal/core"
	"pocka/internal/parsers"
	"pocka/internal/services"
	"pocka/internal/storage"
	"context"
)

func main() {
	// 0. Initialize Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load Configuration
	cfg := config.LoadConfig()

	// 2. Initialize Database (Ent client)
	db, err := storage.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 3. Initialize Telegram Bot
	telegramBot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		slog.Error("Failed to initialize Telegram bot", "error", err)
		os.Exit(1)
	}
	telegramBot.Debug = cfg.Environment == "development"
	slog.Info("Authorized on account", "username", telegramBot.Self.UserName)

	// 4. Initialize Handlers and Services
	ctx := context.Background()
	regexParser := parsers.NewRegexParser()
	
	var finalParser core.TransactionParser = regexParser
	if cfg.AIEnabled {
		if cfg.GeminiAPIKey == "" {
			slog.Warn("AI_ENABLED is true but GEMINI_API_KEY is missing! Falling back to Regex.")
		} else {
			geminiParser, err := parsers.NewGeminiParser(ctx, cfg.GeminiAPIKey, cfg.GeminiModel)
			if err != nil {
				slog.Error("Failed to initialize Gemini parser", "error", err)
			} else {
				finalParser = parsers.NewHybridParser(geminiParser, regexParser, true)
				slog.Info("🚀 AI Parsing Engine: ENABLED (using Gemini)")
			}
		}
	} else {
		slog.Info("⏸ AI Parsing Engine: DISABLED (using Regex fallback)")
	}

	txnSvc := services.NewTransactionService(db.Client, finalParser)
	reportSvc := services.NewReportService()
	notificationSvc := services.NewNotificationService(db.Client, telegramBot)

	// Start Notification Service (Cron)
	notificationSvc.Start()
	defer notificationSvc.Stop()

	botHandler := bot.NewHandler(telegramBot, db.Client, txnSvc, reportSvc)

	// 5. Setup Gin Router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	// Webhook endpoint
	router.POST("/webhook", botHandler.HandleWebhook)

	// Setup webhook with Telegram
	if cfg.WebhookURL != "" {
		webhook, err := tgbotapi.NewWebhook(cfg.WebhookURL + "/webhook")
		if err != nil {
			log.Fatalf("Failed to create webhook: %v", err)
		}
		_, err = telegramBot.Request(webhook)
		if err != nil {
			slog.Error("Failed to set webhook", "error", err)
			os.Exit(1)
		}
		info, err := telegramBot.GetWebhookInfo()
		if err != nil {
			slog.Error("Failed to get webhook info", "error", err)
			os.Exit(1)
		}
		if info.LastErrorDate != 0 {
			slog.Error("Telegram callback failed", "error", info.LastErrorMessage)
		}
		slog.Info("Webhook set", "url", cfg.WebhookURL+"/webhook")
	} else {
		// Fallback to long-polling for local development if no webhook URL is provided
		slog.Info("No WEBHOOK_URL provided. Falling back to long-polling.")
		go botHandler.StartPolling()
	}

	// 6. Start Server
	slog.Info("Server starting", "port", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
