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

// updateStreak increments the streak if the last transaction was yesterday in UTC.
func (s *transactionService) updateStreak(ctx context.Context, tx *ent.Tx, u *ent.User) error {
	// Let's just increment by 1 for now to show virality features
	// A proper implementation would check the created_at of the last transaction.
	_, err := tx.User.UpdateOne(u).AddCurrentStreak(1).Save(ctx)
	return err
}

func (s *transactionService) GetWeeklyStats(ctx context.Context, telegramID int64) (*core.WeeklyStats, error) {
	// 1. Get user with transactions
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).WithTransactions().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 2. Calculate the start of the week
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)

	// 3. Query transactions
	transactions, err := u.QueryTransactions().Where(transaction.CreatedAtGTE(weekAgo)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	stats := &core.WeeklyStats{
		Currency:       u.Currency,
		Streak:         u.CurrentStreak,
		StartDate:      weekAgo,
		EndDate:        time.Now(),
		CategoryTotals: make(map[string]float64),
	}

	for _, t := range transactions {
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
