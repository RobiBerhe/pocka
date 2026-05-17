package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"pocka/ent"
	"pocka/ent/budget"
	"pocka/ent/transaction"
	"pocka/ent/user"
	"pocka/internal/core"
)

type Handler struct {
	bot       *tgbotapi.BotAPI
	db        *ent.Client
	txnSvc    core.TransactionService
	reportSvc core.ReportService
	i18n      core.I18nService
}

func NewHandler(bot *tgbotapi.BotAPI, db *ent.Client, txnSvc core.TransactionService, reportSvc core.ReportService, i18n core.I18nService) *Handler {
	return &Handler{
		bot:       bot,
		db:        db,
		txnSvc:    txnSvc,
		reportSvc: reportSvc,
		i18n:      i18n,
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
			refStr := strings.TrimPrefix(args, "ref_")
			refID, err := strconv.ParseInt(refStr, 10, 64)
			if err == nil && refID != u.TelegramID {
				if !u.OnboardingCompleted && u.ReferrerID == 0 {
					_, err = h.db.User.UpdateOne(u).SetReferrerID(refID).Save(ctx)
					if err != nil {
						slog.Error("Failed to set referrer", "user_id", u.TelegramID, "error", err)
					}
				}
			}
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
		responseText = h.i18n.Translate(u.Language, "help_text")
	case "share":
		botUsername := "PockaaBot"
		if h.bot.Self.UserName != "" {
			botUsername = h.bot.Self.UserName
		}
		link := fmt.Sprintf("https://t.me/%s?start=ref_%d", botUsername, u.TelegramID)
		responseText = h.i18n.Translate(u.Language, "share_text", link)
	case "referrals":
		count, err := h.db.User.Query().Where(user.ReferrerIDEQ(u.TelegramID)).Count(ctx)
		if err != nil {
			slog.Error("Error getting referrals", "user_id", u.TelegramID, "error", err)
			responseText = h.i18n.Translate(u.Language, "error_not_understood")
		} else {
			responseText = h.i18n.Translate(u.Language, "referral_stats", count)
		}
	case "stats", "weekly":
		stats, err := h.txnSvc.GetWeeklyStats(ctx, u.TelegramID)
		if err != nil {
			slog.Error("Error getting stats", "user_id", u.TelegramID, "error", err)
			responseText = h.i18n.Translate(u.Language, "error_not_understood") // Fallback
		} else {
			responseText = h.formatStatsText(stats, u.Language, "stats_header_weekly")
		}
		
		msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
		msg.ParseMode = tgbotapi.ModeMarkdown
		
		// Add "Generate Card" button if there are transactions
		if stats != nil && stats.TotalExpense > 0 {
			btnText := h.i18n.Translate(u.Language, "generate_card_btn")
			btn := tgbotapi.NewInlineKeyboardButtonData(btnText, "generate_card_weekly")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
		}
		
		h.bot.Send(msg)
		return
	case "monthly":
		stats, err := h.txnSvc.GetMonthlyStats(ctx, u.TelegramID)
		if err != nil {
			slog.Error("Error getting stats", "user_id", u.TelegramID, "error", err)
			responseText = h.i18n.Translate(u.Language, "error_not_understood") // Fallback
		} else {
			responseText = h.formatStatsText(stats, u.Language, "stats_header_monthly")
		}
		
		msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
		msg.ParseMode = tgbotapi.ModeMarkdown
		
		// Add "Generate Card" button if there are transactions
		if stats != nil && stats.TotalExpense > 0 {
			btnText := h.i18n.Translate(u.Language, "generate_card_btn")
			btn := tgbotapi.NewInlineKeyboardButtonData(btnText, "generate_card_monthly")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
		}
		
		h.bot.Send(msg)
		return
	case "export":
		csvBytes, err := h.txnSvc.ExportDataCSV(ctx, u.TelegramID)
		if err != nil {
			slog.Error("Error exporting CSV", "user_id", u.TelegramID, "error", err)
			responseText = h.i18n.Translate(u.Language, "error_not_understood")
			msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
			h.bot.Send(msg)
			return
		}
		file := tgbotapi.FileBytes{Name: "transactions.csv", Bytes: csvBytes}
		doc := tgbotapi.NewDocument(message.Chat.ID, file)
		doc.Caption = "Here is your full transaction history! 📁"
		h.bot.Send(doc)
		return
	case "budget":
		parts := strings.Fields(args)
		if len(parts) == 0 {
			responseText = "To set a budget, use: `/budget amount` or `/budget category amount`."
		} else {
			amountStr := parts[0]
			category := ""
			if len(parts) > 1 {
				category = parts[0]
				amountStr = parts[1]
			}
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				responseText = "❌ Invalid amount."
			} else {
				b, _ := h.db.Budget.Query().Where(
					budget.HasUserWith(user.IDEQ(u.ID)),
					budget.CategoryEQ(category),
				).Only(ctx)
				if b != nil {
					h.db.Budget.UpdateOne(b).SetAmount(amount).Save(ctx)
				} else {
					h.db.Budget.Create().SetUser(u).SetAmount(amount).SetCategory(category).Save(ctx)
				}
				if category == "" {
					responseText = fmt.Sprintf("✅ Total monthly budget set to %.2f %s", amount, u.Currency)
				} else {
					responseText = fmt.Sprintf("✅ Budget for '%s' set to %.2f %s", category, amount, u.Currency)
				}
			}
		}
	case "search":
		if args == "" {
			responseText = "Please provide a search term. Example: `/search coffee`"
		} else {
			txs, err := h.db.Transaction.Query().
				Where(
					transaction.HasUserWith(user.IDEQ(u.ID)),
					transaction.Or(
						transaction.CategoryContainsFold(args),
						transaction.DescriptionContainsFold(args),
						transaction.MerchantContainsFold(args),
					),
				).Order(ent.Desc(transaction.FieldCreatedAt)).Limit(10).All(ctx)
			
			if err != nil || len(txs) == 0 {
				responseText = "No transactions found matching your search."
			} else {
				var response strings.Builder
				response.WriteString(fmt.Sprintf("🔍 *Search Results for '%s':*\n\n", args))
				total := 0.0
				for _, t := range txs {
					date := t.CreatedAt.Format("Jan 02")
					response.WriteString(fmt.Sprintf("• %s: *%.2f %s* — %s\n", date, t.Amount, t.Currency, t.Category))
					if t.Type == transaction.TypeEXPENSE {
						total += t.Amount
					}
				}
				response.WriteString(fmt.Sprintf("\n*Total Spent (from top 10):* %.2f %s", total, u.Currency))
				responseText = response.String()
			}
		}
	default:
		responseText = h.i18n.Translate(u.Language, "unknown_command")
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	h.bot.Send(msg)
}

func (h *Handler) handleOnboarding(ctx context.Context, message *tgbotapi.Message, u *ent.User) {
	state := core.OnboardingState(u.OnboardingState)
	lang := u.Language
	if lang == "" {
		lang = "en"
	}

	switch state {
	case core.OnboardingStateAwaitingName:
		if message.IsCommand() {
			msg := tgbotapi.NewMessage(message.Chat.ID, h.i18n.Translate(lang, "welcome_name"))
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

		msg := tgbotapi.NewMessage(message.Chat.ID, h.i18n.Translate(lang, "nice_to_meet", name))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("English 🇬🇧"),
				tgbotapi.NewKeyboardButton("Amharic 🇪🇹"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Tigrigna 🇪🇷/🇪🇹"),
				tgbotapi.NewKeyboardButton("Afaan Oromo 🇪🇹"),
			),
		)
		h.bot.Send(msg)

	case core.OnboardingStateAwaitingLanguage:
		lang = "en"
		text := message.Text
		if strings.Contains(text, "Amharic") || strings.Contains(text, "አማርኛ") || strings.Contains(text, "🇪🇹") && strings.Contains(text, "Amharic") {
			lang = "am"
		} else if strings.Contains(text, "Tigrigna") || strings.Contains(text, "ትግርኛ") {
			lang = "ti"
		} else if strings.Contains(text, "Oromo") || strings.Contains(text, "Afaan") {
			lang = "om"
		}

		h.db.User.UpdateOne(u).
			SetLanguage(lang).
			SetOnboardingState(string(core.OnboardingStateAwaitingCurrency)).
			SaveX(ctx)

		msg := tgbotapi.NewMessage(message.Chat.ID, h.i18n.Translate(lang, "currency_prompt"))
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

		msg := tgbotapi.NewMessage(message.Chat.ID, h.i18n.Translate(lang, "contact_prompt"))
		msg.ParseMode = tgbotapi.ModeMarkdown
		
		btnShare := tgbotapi.NewKeyboardButtonContact(h.i18n.Translate(lang, "share_contact_btn"))
		btnSkip := tgbotapi.NewKeyboardButton(h.i18n.Translate(lang, "skip_btn"))
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(btnShare),
			tgbotapi.NewKeyboardButtonRow(btnSkip),
		)
		h.bot.Send(msg)

	case core.OnboardingStateAwaitingContact:
		var phone string
		if message.Contact != nil {
			phone = message.Contact.PhoneNumber
		} else if !strings.Contains(message.Text, "Skip") && !strings.Contains(message.Text, "➡️") && !strings.Contains(message.Text, "ይለፍ") && !strings.Contains(message.Text, "dhiisi") && !strings.Contains(message.Text, "ሕለፍ") {
			phone = message.Text
		}

		update := h.db.User.UpdateOne(u).
			SetOnboardingState(string(core.OnboardingStateCompleted)).
			SetOnboardingCompleted(true)
		
		if phone != "" {
			update.SetPhoneNumber(phone)
		}
		
		update.SaveX(ctx)

		msg := tgbotapi.NewMessage(message.Chat.ID, h.i18n.Translate(lang, "onboarding_complete"))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		h.bot.Send(msg)
	}
}

func (h *Handler) handleText(ctx context.Context, message *tgbotapi.Message, u *ent.User) {
	parsedList, alerts, err := h.txnSvc.LogTransaction(ctx, u.TelegramID, message.Text, u.Currency)
	if err != nil {
		slog.Warn("Failed to log transaction",
			"user_id", u.TelegramID,
			"input", message.Text,
			"error", err,
		)
		msg := tgbotapi.NewMessage(message.Chat.ID, h.i18n.Translate(u.Language, "error_not_understood"))
		msg.ParseMode = tgbotapi.ModeMarkdown
		h.bot.Send(msg)
		return
	}

	var response strings.Builder
	response.WriteString(h.i18n.Translate(u.Language, "saved_header"))

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
		response.WriteString(h.i18n.Translate(u.Language, "nice_income"))
	}

	for _, alert := range alerts {
		response.WriteString("\n\n" + alert)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response.String())
	msg.ParseMode = tgbotapi.ModeMarkdown
	h.bot.Send(msg)
}

func (h *Handler) handleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	u, err := h.db.User.Query().Where(user.TelegramIDEQ(query.From.ID)).Only(ctx)
	lang := "en"
	if err == nil && u != nil && u.Language != "" {
		lang = u.Language
	}

	if strings.HasPrefix(query.Data, "generate_card") {
		var stats *core.SummaryStats
		var err error
		title := "Pocka Wrap"
		
		if query.Data == "generate_card_monthly" {
			stats, err = h.txnSvc.GetMonthlyStats(ctx, query.From.ID)
			title = "Pocka Monthly Wrap"
		} else {
			stats, err = h.txnSvc.GetWeeklyStats(ctx, query.From.ID)
			title = "Pocka Weekly Wrap"
		}

		if err != nil {
			slog.Error("Error getting stats for card", "user_id", query.From.ID, "error", err)
			return
		}

		// 2. Ack the callback to remove loading state
		h.bot.Send(tgbotapi.NewCallback(query.ID, h.i18n.Translate(lang, "generating_card")))

		// 3. Generate image
		imgBytes, err := h.reportSvc.GenerateSummaryCard(ctx, title, stats)
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
		photo.Caption = h.i18n.Translate(lang, "card_caption")
		h.bot.Send(photo)
	}
}

func (h *Handler) formatStatsText(stats *core.SummaryStats, langCode string, headerKey string) string {
	if stats == nil || (stats.TotalIncome == 0 && stats.TotalExpense == 0) {
		return h.i18n.Translate(langCode, "stats_empty")
	}

	resp := h.i18n.Translate(langCode, headerKey)
	resp += h.i18n.Translate(langCode, "stats_income", stats.TotalIncome, stats.Currency)
	resp += h.i18n.Translate(langCode, "stats_expense", stats.TotalExpense, stats.Currency)
	resp += h.i18n.Translate(langCode, "stats_net", stats.NetBalance, stats.Currency)

	if stats.TotalExpense > 0 {
		resp += h.i18n.Translate(langCode, "stats_top_expenses")
		
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

	resp += h.i18n.Translate(langCode, "stats_streak", stats.Streak)
	resp += h.i18n.Translate(langCode, "stats_keep_tracking")

	return resp
}
