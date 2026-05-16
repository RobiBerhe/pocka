package services

import (
	"context"
	"fmt"
	"log/slog"
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

func (s *transactionService) LogTransaction(ctx context.Context, telegramID int64, text string, userCurrency string) ([]*core.ParsedTransaction, error) {
	parsedList, err := s.parser.Parse(ctx, text, userCurrency)
	if err != nil {
		return nil, err
	}

	// Find the user to link the transaction
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Start a database transaction to ensure all or nothing
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed starting database transaction: %w", err)
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
			return nil, fmt.Errorf("failed saving transaction: %w", err)
		}
	}

	// Handle streak logic (once per message is fine)
	err = s.updateStreak(ctx, tx, u)
	if err != nil {
		slog.Error("Failed to update streak", "user_id", u.TelegramID, "error", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed committing transactions: %w", err)
	}

	return parsedList, nil
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

func (s *transactionService) GetWeeklyStats(ctx context.Context, telegramID int64) (*core.WeeklyStats, error) {
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
			transaction.CreatedAtGTE(startOfWeek),
		).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	stats := &core.WeeklyStats{
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
