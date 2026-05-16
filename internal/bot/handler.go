package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"pocka/ent"
	"pocka/ent/user"
	"pocka/internal/core"
)

type Handler struct {
	bot       *tgbotapi.BotAPI
	db        *ent.Client
	txnSvc    core.TransactionService
	reportSvc core.ReportService
}

func NewHandler(bot *tgbotapi.BotAPI, db *ent.Client, txnSvc core.TransactionService, reportSvc core.ReportService) *Handler {
	return &Handler{
		bot:       bot,
		db:        db,
		txnSvc:    txnSvc,
		reportSvc: reportSvc,
	}
}

// HandleWebhook processes incoming updates from the Telegram webhook.
func (h *Handler) HandleWebhook(c *gin.Context) {
	var update tgbotapi.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.processUpdate(update)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// StartPolling is a fallback for local development.
func (h *Handler) StartPolling() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for update := range updates {
		h.processUpdate(update)
	}
}

func (h *Handler) processUpdate(update tgbotapi.Update) {
	// Log everything for debugging purposes as requested
	updateJSON, _ := json.Marshal(update)
	slog.Info("Incoming Telegram Update", "update", string(updateJSON))

	ctx := context.Background()

	// Handle Callbacks
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil { // ignore any non-Message updates
		return
	}

	// Ensure user exists
	u, err := h.getOrCreateUser(ctx, update.Message.From)
	if err != nil {
		slog.Error("Error getting/creating user", "telegram_id", update.Message.From.ID, "error", err)
		return
	}

	// Handle Onboarding if not completed
	if !u.OnboardingCompleted {
		h.handleOnboarding(ctx, update.Message, u)
		return
	}

	// Handle Commands
	if update.Message.IsCommand() {
		h.handleCommand(ctx, update.Message, u)
		return
	}

	// Handle Text
	h.handleText(ctx, update.Message, u)
}

func (h *Handler) getOrCreateUser(ctx context.Context, tgUser *tgbotapi.User) (*ent.User, error) {
	u, err := h.db.User.Query().Where(user.TelegramIDEQ(tgUser.ID)).Only(ctx)
	if err == nil {
		return u, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	// Create user
	return h.db.User.Create().
		SetTelegramID(tgUser.ID).
		SetNillableUsername(&tgUser.UserName).
		Save(ctx)
}

func (h *Handler) handleCommand(ctx context.Context, message *tgbotapi.Message, u *ent.User) {
	command := message.Command()
	args := message.CommandArguments()

	var responseText string

	switch command {
	case "start":
		if strings.HasPrefix(args, "ref_") {
			slog.Info("User used referral code", "user_id", u.TelegramID, "ref_code", args)
		}
		
		// Reset onboarding if requested via /start or first time
		_, err := h.db.User.UpdateOne(u).
			SetOnboardingCompleted(false).
			SetOnboardingState(string(core.OnboardingStateAwaitingName)).
			Save(ctx)
		if err != nil {
			slog.Error("Failed to reset onboarding", "user_id", u.TelegramID, "error", err)
		}

		h.handleOnboarding(ctx, message, u)
		return
	case "help":
		responseText = "📖 *How to use Pocka:*\n\n" +
			"1. **Log Expense**: `amount description` or `description amount` (e.g., `100 lunch` or `taxi 50`).\n" +
			"2. **Log Income**: Use keywords like `salary`, `sold`, or `bonus` (e.g., `salary 15000`).\n" +
			"3. **Stats**: Send /stats for a weekly breakdown of your income, expenses, and net balance.\n\n" +
			"It's that simple! No forms, no complex apps."
	case "stats":
		stats, err := h.txnSvc.GetWeeklyStats(ctx, u.TelegramID)
		if err != nil {
			slog.Error("Error getting stats", "user_id", u.TelegramID, "error", err)
			responseText = "❌ Sorry, I couldn't fetch your stats right now."
		} else {
			responseText = formatStatsText(stats)
		}
		
		msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
		msg.ParseMode = tgbotapi.ModeMarkdown
		
		// Add "Generate Card" button if there are transactions
		if stats != nil && stats.TotalExpense > 0 {
			btn := tgbotapi.NewInlineKeyboardButtonData("Generate Shareable Card 🖼️", "generate_card")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
		}
		
		h.bot.Send(msg)
		return
	default:
		responseText = "I don't know that command."
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	h.bot.Send(msg)
}

func (h *Handler) handleOnboarding(ctx context.Context, message *tgbotapi.Message, u *ent.User) {
	state := core.OnboardingState(u.OnboardingState)

	switch state {
	case core.OnboardingStateAwaitingName:
		if message.IsCommand() {
			msg := tgbotapi.NewMessage(message.Chat.ID, "👋 *Welcome to Pocka!*\n\nI'm your personal money assistant. To get started, what should I call you?\n\n_(You can type your name below)_")
			msg.ParseMode = tgbotapi.ModeMarkdown
			h.bot.Send(msg)
			return
		}

		name := strings.TrimSpace(message.Text)
		if name == "" {
			name = message.From.FirstName
		}

		h.db.User.UpdateOne(u).
			SetFullName(name).
			SetOnboardingState(string(core.OnboardingStateAwaitingLanguage)).
			SaveX(ctx)

		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Nice to meet you, *%s*! 😊\n\nWhich language do you prefer?", name))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("English 🇬🇧"),
				tgbotapi.NewKeyboardButton("Amharic 🇪🇹"),
			),
		)
		h.bot.Send(msg)

	case core.OnboardingStateAwaitingLanguage:
		lang := "en"
		if strings.Contains(message.Text, "Amharic") {
			lang = "am"
		}

		h.db.User.UpdateOne(u).
			SetLanguage(lang).
			SetOnboardingState(string(core.OnboardingStateAwaitingCurrency)).
			SaveX(ctx)

		msg := tgbotapi.NewMessage(message.Chat.ID, "Got it! And what currency do you use most?")
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("ETB"),
				tgbotapi.NewKeyboardButton("USD"),
			),
		)
		h.bot.Send(msg)

	case core.OnboardingStateAwaitingCurrency:
		currency := strings.ToUpper(strings.TrimSpace(message.Text))
		if currency == "" {
			currency = "ETB"
		}

		h.db.User.UpdateOne(u).
			SetCurrency(currency).
			SetOnboardingState(string(core.OnboardingStateAwaitingContact)).
			SaveX(ctx)

		msg := tgbotapi.NewMessage(message.Chat.ID, "Almost there! 🚀\n\nCan you share your phone number? This helps secure your account and link your data if you switch devices.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		
		// Share Contact Button
		btn := tgbotapi.NewKeyboardButtonContact("Share Phone Number 📱")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(btn),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Skip for now ➡️")),
		)
		h.bot.Send(msg)

	case core.OnboardingStateAwaitingContact:
		var phone string
		if message.Contact != nil {
			phone = message.Contact.PhoneNumber
		} else if !strings.Contains(message.Text, "Skip") {
			phone = message.Text
		}

		update := h.db.User.UpdateOne(u).
			SetOnboardingState(string(core.OnboardingStateCompleted)).
			SetOnboardingCompleted(true)
		
		if phone != "" {
			update.SetPhoneNumber(phone)
		}
		
		update.SaveX(ctx)

		msg := tgbotapi.NewMessage(message.Chat.ID, "✨ *Onboarding Complete!*\n\nYou're all set to track your money. Just send me messages like:\n\n• `coffee 50`\n• `salary 15000`\n• `taxi 100`\n\nUse /stats anytime to see your summary. Let's grow! 🚀")
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		h.bot.Send(msg)
	}
}

func (h *Handler) handleText(ctx context.Context, message *tgbotapi.Message, u *ent.User) {
	parsedList, err := h.txnSvc.LogTransaction(ctx, u.TelegramID, message.Text, u.Currency)
	if err != nil {
		slog.Warn("Failed to log transaction",
			"user_id", u.TelegramID,
			"input", message.Text,
			"error", err,
		)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ I couldn't understand that. Try something like `100 lunch` or `salary 5000`.")
		msg.ParseMode = tgbotapi.ModeMarkdown
		h.bot.Send(msg)
		return
	}

	var response strings.Builder
	response.WriteString("✅ *Saved:*\n")

	for _, parsed := range parsedList {
		// Use AI emoji if available, fallback to default icons
		icon := parsed.Emoji
		if icon == "" {
			icon = "💸"
			if parsed.Type == core.TransactionTypeIncome {
				icon = "💰"
			}
		}

		// Include merchant if available
		desc := parsed.Category
		if parsed.Merchant != "" {
			desc = fmt.Sprintf("%s (%s)", parsed.Category, parsed.Merchant)
		}

		response.WriteString(fmt.Sprintf("• *%.2f %s* — %s %s\n", parsed.Amount, parsed.Currency, desc, icon))
	}

	// Add flavor for income if present
	hasIncome := false
	for _, p := range parsedList {
		if p.Type == core.TransactionTypeIncome {
			hasIncome = true
			break
		}
	}
	if hasIncome {
		response.WriteString("\n_Nice! Keep it coming!_ 🚀")
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	h.bot.Send(msg)
}

func (h *Handler) handleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	if query.Data == "generate_card" {
		// 1. Get stats
		stats, err := h.txnSvc.GetWeeklyStats(ctx, query.From.ID)
		if err != nil {
			slog.Error("Error getting stats for card", "user_id", query.From.ID, "error", err)
			return
		}

		// 2. Ack the callback to remove loading state
		h.bot.Send(tgbotapi.NewCallback(query.ID, "Generating your card... 🪄"))

		// 3. Generate image
		imgBytes, err := h.reportSvc.GenerateWeeklyCard(ctx, stats)
		if err != nil {
			slog.Error("Error generating card", "user_id", query.From.ID, "error", err)
			return
		}

		// 4. Send photo
		file := tgbotapi.FileBytes{
			Name:  "pocka_summary.png",
			Bytes: imgBytes,
		}
		photo := tgbotapi.NewPhoto(query.Message.Chat.ID, file)
		photo.Caption = "Here is your weekly summary card! 📊 Share it with your friends to show off your financial discipline. 🚀"
		h.bot.Send(photo)
	}
}

func formatStatsText(stats *core.WeeklyStats) string {
	if stats == nil || (stats.TotalIncome == 0 && stats.TotalExpense == 0) {
		return "You haven't logged any transactions in the last 7 days. Start by sending something like 'coffee 40'!"
	}

	resp := "📊 *Weekly Summary (Last 7 Days)*\n\n"
	resp += fmt.Sprintf("💰 Income: *%.2f %s*\n", stats.TotalIncome, stats.Currency)
	resp += fmt.Sprintf("💸 Expense: *%.2f %s*\n", stats.TotalExpense, stats.Currency)
	resp += fmt.Sprintf("⚖️ Net: *%.2f %s*\n\n", stats.NetBalance, stats.Currency)

	if stats.TotalExpense > 0 {
		resp += "*Top Expenses:*\n"
		
		// Sort categories
		type catVal struct {
			Name   string
			Amount float64
		}
		var cats []catVal
		for name, val := range stats.CategoryTotals {
			cats = append(cats, catVal{name, val})
		}
		sort.Slice(cats, func(i, j int) bool {
			return cats[i].Amount > cats[j].Amount
		})

		for _, cat := range cats {
			percentage := (cat.Amount / stats.TotalExpense) * 100
			resp += fmt.Sprintf("🔹 %s: %.2f (%.0f%%)\n", cat.Name, cat.Amount, percentage)
		}
	}

	resp += fmt.Sprintf("\nStreak: %d days 🔥", stats.Streak)
	resp += "\n\n_Keep tracking to maintain your streak!_"

	return resp
}
