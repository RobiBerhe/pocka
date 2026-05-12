package core

import "context"

type ParsedExpense struct {
	Amount   float64
	Category string
	Currency string
}

// ExpenseParser defines how text should be parsed into structured expense data.
type ExpenseParser interface {
	Parse(input string, userCurrency string) (*ParsedExpense, error)
}

// ExpenseService handles the core business logic around expenses.
type ExpenseService interface {
	LogExpense(ctx context.Context, userID int64, text string, userCurrency string) (*ParsedExpense, error)
	GetWeeklyStats(ctx context.Context, userID int64) (string, error)
}
