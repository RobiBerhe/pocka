package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pocka/ent"
	"pocka/ent/transaction"
	"pocka/ent/user"
	"pocka/internal/core"
)

type transactionService struct {
	db     *ent.Client
	parser core.TransactionParser
}

func NewTransactionService(db *ent.Client, parser core.TransactionParser) core.TransactionService {
	return &transactionService{
		db:     db,
		parser: parser,
	}
}

func (s *transactionService) LogTransaction(ctx context.Context, telegramID int64, text string, userCurrency string) ([]*core.ParsedTransaction, []string, error) {
	parsedList, err := s.parser.Parse(ctx, text, userCurrency)
	if err != nil {
		return nil, nil, err
	}

	// Find the user to link the transaction
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found: %w", err)
	}

	// Start a database transaction to ensure all or nothing
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed starting database transaction: %w", err)
	}

	for _, parsed := range parsedList {
		// Create the transaction
		_, err = tx.Transaction.Create().
			SetAmount(parsed.Amount).
			SetType(transaction.Type(parsed.Type)).
			SetCategory(parsed.Category).
			SetDescription(parsed.Description).
			SetMerchant(parsed.Merchant).
			SetEmoji(parsed.Emoji).
			SetCurrency(parsed.Currency).
			SetRawInput(text).
			SetUser(u).
			Save(ctx)

		if err != nil {
			tx.Rollback()
			return nil, nil, fmt.Errorf("failed saving transaction: %w", err)
		}
	}

	// Handle streak logic (once per message is fine)
	err = s.updateStreak(ctx, tx, u)
	if err != nil {
		slog.Error("Failed to update streak", "user_id", u.TelegramID, "error", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed committing transactions: %w", err)
	}

	// Calculate budget alerts
	var alerts []string
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	nowLocal := time.Now().In(loc)
	startOfMonth := time.Date(nowLocal.Year(), nowLocal.Month(), 1, 0, 0, 0, 0, loc).UTC()

	monthTxs, err := s.db.Transaction.Query().
		Where(
			transaction.HasUserWith(user.IDEQ(u.ID)),
			transaction.CreatedAtGTE(startOfMonth),
			transaction.TypeEQ(transaction.TypeEXPENSE),
		).All(ctx)

	if err == nil {
		budgets, _ := u.QueryBudgets().All(ctx)
		slog.Info("Budget check", "user_id", telegramID, "budgets_count", len(budgets), "month_txs_count", len(monthTxs))
		for _, b := range budgets {
			spent := 0.0
			name := b.Category
			if name == "" {
				name = "Total Monthly Budget"
				for _, t := range monthTxs {
					spent += t.Amount
				}
			} else {
				for _, t := range monthTxs {
					if strings.EqualFold(t.Category, b.Category) ||
						strings.EqualFold(t.Description, b.Category) ||
						strings.Contains(strings.ToLower(t.Description), strings.ToLower(b.Category)) ||
						strings.EqualFold(t.Merchant, b.Category) {
						spent += t.Amount
					}
				}
			}
			
			slog.Info("Budget evaluation", "category", name, "spent", spent, "budget", b.Amount)
			
			if spent > 0 && b.Amount > 0 {
				ratio := spent / b.Amount
				if ratio >= 1.0 {
					alerts = append(alerts, fmt.Sprintf("🚨 Alert: You have exceeded your budget for %s (%.0f/%.0f)!", name, spent, b.Amount))
				} else if ratio >= 0.8 {
					alerts = append(alerts, fmt.Sprintf("⚠️ Warning: You have reached 80%% of your budget for %s (%.0f/%.0f).", name, spent, b.Amount))
				} else if ratio >= 0.5 {
					alerts = append(alerts, fmt.Sprintf("ℹ️ Notice: You have used 50%% of your budget for %s (%.0f/%.0f).", name, spent, b.Amount))
				}
			}
		}
	} else {
		slog.Error("Failed to query month transactions for budget check", "user_id", telegramID, "error", err)
	}

	return parsedList, alerts, nil
}

// updateStreak handles the authentic habit-forming streak logic.
func (s *transactionService) updateStreak(ctx context.Context, tx *ent.Tx, u *ent.User) error {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		slog.Warn("Invalid user timezone, falling back to UTC", "timezone", u.Timezone)
		loc = time.UTC
	}

	nowLocal := time.Now().In(loc)
	todayDate := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc)

	// If the user has never logged before
	if u.LastLogDate.IsZero() {
		return tx.User.UpdateOne(u).
			SetCurrentStreak(1).
			SetLastLogDate(todayDate).
			Exec(ctx)
	}

	lastLogLocal := u.LastLogDate.In(loc)
	lastLogDate := time.Date(lastLogLocal.Year(), lastLogLocal.Month(), lastLogLocal.Day(), 0, 0, 0, 0, loc)

	// Case 1: Already logged today -> Streak stays the same
	if todayDate.Equal(lastLogDate) {
		return nil
	}

	// Case 2: Logged yesterday -> Increment streak
	yesterday := todayDate.AddDate(0, 0, -1)
	if lastLogDate.Equal(yesterday) {
		return tx.User.UpdateOne(u).
			AddCurrentStreak(1).
			SetLastLogDate(todayDate).
			Exec(ctx)
	}

	// Case 3: Missed at least one day -> Reset streak to 1
	return tx.User.UpdateOne(u).
		SetCurrentStreak(1).
		SetLastLogDate(todayDate).
		Exec(ctx)
}

func (s *transactionService) GetWeeklyStats(ctx context.Context, telegramID int64) (*core.SummaryStats, error) {
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// Calculate the 7-day window based on user's local time
	nowLocal := time.Now().In(loc)
	startOfWeek := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -6)

	// Query transactions for this user within the local week window
	txs, err := s.db.Transaction.Query().
		Where(
			transaction.HasUserWith(user.IDEQ(u.ID)),
			transaction.CreatedAtGTE(startOfWeek.UTC()),
		).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	stats := &core.SummaryStats{
		Currency:       u.Currency,
		CategoryTotals: make(map[string]float64),
		Streak:         u.CurrentStreak,
		StartDate:      startOfWeek,
		EndDate:        nowLocal,
	}

	for _, t := range txs {
		if t.Type == transaction.TypeINCOME {
			stats.TotalIncome += t.Amount
		} else {
			stats.TotalExpense += t.Amount
			stats.CategoryTotals[t.Category] += t.Amount
		}
	}

	stats.NetBalance = stats.TotalIncome - stats.TotalExpense

	return stats, nil
}

func (s *transactionService) GetMonthlyStats(ctx context.Context, telegramID int64) (*core.SummaryStats, error) {
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}

	nowLocal := time.Now().In(loc)
	startOfMonth := time.Date(nowLocal.Year(), nowLocal.Month(), 1, 0, 0, 0, 0, loc)

	return s.GetCustomStats(ctx, telegramID, startOfMonth, nowLocal)
}

func (s *transactionService) GetCustomStats(ctx context.Context, telegramID int64, start, end time.Time) (*core.SummaryStats, error) {
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	txs, err := s.db.Transaction.Query().
		Where(
			transaction.HasUserWith(user.IDEQ(u.ID)),
			transaction.CreatedAtGTE(start.UTC()),
			transaction.CreatedAtLTE(end.UTC()),
		).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	stats := &core.SummaryStats{
		Currency:       u.Currency,
		CategoryTotals: make(map[string]float64),
		Streak:         u.CurrentStreak,
		StartDate:      start,
		EndDate:        end,
	}

	for _, t := range txs {
		if t.Type == transaction.TypeINCOME {
			stats.TotalIncome += t.Amount
		} else {
			stats.TotalExpense += t.Amount
			stats.CategoryTotals[t.Category] += t.Amount
		}
	}

	stats.NetBalance = stats.TotalIncome - stats.TotalExpense

	return stats, nil
}

func (s *transactionService) ExportDataCSV(ctx context.Context, telegramID int64) ([]byte, error) {
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	txs, err := s.db.Transaction.Query().
		Where(transaction.HasUserWith(user.IDEQ(u.ID))).
		Order(ent.Desc(transaction.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	var buf []byte
	buf = append(buf, "Date,Type,Amount,Currency,Category,Merchant,Description\n"...)

	for _, t := range txs {
		dateStr := t.CreatedAt.Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("%s,%s,%.2f,%s,%q,%q,%q\n",
			dateStr,
			t.Type,
			t.Amount,
			t.Currency,
			t.Category,
			t.Merchant,
			t.Description,
		)
		buf = append(buf, line...)
	}

	return buf, nil
}
