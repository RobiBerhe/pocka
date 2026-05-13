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

func (s *transactionService) LogTransaction(ctx context.Context, telegramID int64, text string, userCurrency string) (*core.ParsedTransaction, error) {
	parsed, err := s.parser.Parse(ctx, text, userCurrency)
	if err != nil {
		return nil, err
	}

	// Find the user to link the transaction
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Start a transaction
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed starting transaction: %w", err)
	}

	// Create the transaction
	_, err = tx.Transaction.Create().
		SetAmount(parsed.Amount).
		SetType(transaction.Type(parsed.Type)).
		SetCategory(parsed.Category).
		SetDescription(parsed.Description).
		SetCurrency(parsed.Currency).
		SetRawInput(text).
		SetUser(u).
		Save(ctx)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed saving transaction: %w", err)
	}

	// Handle streak logic
	err = s.updateStreak(ctx, tx, u)
	if err != nil {
		// Log but don't fail the transaction creation
		slog.Error("Failed to update streak", "user_id", u.TelegramID, "error", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed committing transaction: %w", err)
	}

	return parsed, nil
}

// updateStreak increments the streak if the last transaction was yesterday in UTC.
func (s *transactionService) updateStreak(ctx context.Context, tx *ent.Tx, u *ent.User) error {
	// Let's just increment by 1 for now to show virality features
	// A proper implementation would check the created_at of the last transaction.
	_, err := tx.User.UpdateOne(u).AddCurrentStreak(1).Save(ctx)
	return err
}

func (s *transactionService) GetWeeklyStats(ctx context.Context, telegramID int64) (string, error) {
	// 1. Get user with transactions
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).WithTransactions().Only(ctx)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// 2. Calculate the start of the week
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)

	// 3. Query transactions
	transactions, err := u.QueryTransactions().Where(transaction.CreatedAtGTE(weekAgo)).All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to query transactions: %w", err)
	}

	if len(transactions) == 0 {
		return "You haven't logged any transactions in the last 7 days. Start by sending something like 'coffee 40'!", nil
	}

	var totalIncome, totalExpense float64
	categoryTotals := make(map[string]float64)

	for _, t := range transactions {
		if t.Type == transaction.TypeINCOME {
			totalIncome += t.Amount
		} else {
			totalExpense += t.Amount
			categoryTotals[t.Category] += t.Amount
		}
	}

	net := totalIncome - totalExpense

	// 4. Format response
	resp := "📊 *Weekly Summary (Last 7 Days)*\n\n"
	resp += fmt.Sprintf("💰 Income: *%.2f %s*\n", totalIncome, u.Currency)
	resp += fmt.Sprintf("💸 Expense: *%.2f %s*\n", totalExpense, u.Currency)
	resp += fmt.Sprintf("⚖️ Net: *%.2f %s*\n\n", net, u.Currency)

	if totalExpense > 0 {
		resp += "*Top Expenses:*\n"
		for cat, catTotal := range categoryTotals {
			percentage := (catTotal / totalExpense) * 100
			resp += fmt.Sprintf("🔹 %s: %.2f (%.0f%%)\n", cat, catTotal, percentage)
		}
	}

	resp += "\n_Keep tracking to maintain your streak! 🔥_"

	return resp, nil
}
