package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"pocka/ent"
	"pocka/ent/expense"
	"pocka/ent/user"
	"pocka/internal/core"
)

type expenseService struct {
	db     *ent.Client
	parser core.ExpenseParser
}

func NewExpenseService(db *ent.Client, parser core.ExpenseParser) core.ExpenseService {
	return &expenseService{
		db:     db,
		parser: parser,
	}
}

func (s *expenseService) LogExpense(ctx context.Context, telegramID int64, text string, userCurrency string) (*core.ParsedExpense, error) {
	parsed, err := s.parser.Parse(text, userCurrency)
	if err != nil {
		return nil, err
	}

	// Find the user to link the expense
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Start a transaction (optional but good practice)
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed starting transaction: %w", err)
	}

	// Create the expense
	_, err = tx.Expense.Create().
		SetAmount(parsed.Amount).
		SetCategory(parsed.Category).
		SetCurrency(parsed.Currency).
		SetRawInput(text).
		SetUser(u).
		Save(ctx)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed saving expense: %w", err)
	}

	// Handle streak logic
	err = s.updateStreak(ctx, tx, u)
	if err != nil {
		// Log but don't fail the expense creation
		log.Printf("Failed to update streak for user %d: %v", u.TelegramID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed committing transaction: %w", err)
	}

	return parsed, nil
}

// updateStreak increments the streak if the last expense was yesterday in UTC.
// Simplistic approach for MVP.
func (s *expenseService) updateStreak(ctx context.Context, tx *ent.Tx, u *ent.User) error {
	// Let's just increment by 1 for now to show virality features
	// A proper implementation would check the created_at of the last expense.
	_, err := tx.User.UpdateOne(u).AddCurrentStreak(1).Save(ctx)
	return err
}

func (s *expenseService) GetWeeklyStats(ctx context.Context, telegramID int64) (string, error) {
	// 1. Get user
	u, err := s.db.User.Query().Where(user.TelegramIDEQ(telegramID)).WithExpenses().Only(ctx)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// 2. Calculate the start of the week (7 days ago for simplicity, or start of current week)
	// For a real production app, we'd use the user's timezone to calculate start of the week.
	// For this MVP, we just use 7 days ago.
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)

	// 3. Query expenses
	expenses, err := u.QueryExpenses().Where(expense.CreatedAtGTE(weekAgo)).All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to query expenses: %w", err)
	}

	if len(expenses) == 0 {
		return "You haven't logged any expenses in the last 7 days.", nil
	}

	var total float64
	categoryTotals := make(map[string]float64)
	for _, e := range expenses {
		total += e.Amount
		categoryTotals[e.Category] += e.Amount
	}

	// 4. Format response
	resp := fmt.Sprintf("📊 *Weekly Summary (Last 7 Days)*\n\n💸 Total: *%.2f %s*\n\n", total, u.Currency)
	for cat, catTotal := range categoryTotals {
		percentage := (catTotal / total) * 100
		resp += fmt.Sprintf("🔹 %s: %.2f (%.0f%%)\n", cat, catTotal, percentage)
	}

	resp += "\n_Keep tracking to maintain your streak! 🔥_"

	return resp, nil
}
