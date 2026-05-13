package core

import (
	"context"
	"time"
)

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "INCOME"
	TransactionTypeExpense TransactionType = "EXPENSE"
)

type ParsedTransaction struct {
	Amount      float64
	Type        TransactionType
	Category    string
	Description string
	Currency    string
	RawInput    string
	Date        time.Time
	// Metadata allows for future AI-driven insights (e.g., merchant, location, confidence score)
	Metadata map[string]interface{}
}

// TransactionParser defines how text should be parsed into structured transaction data.
// This interface is designed to be implemented by both simple Regex-based parsers
// and advanced LLM-based parsers in the future.
type TransactionParser interface {
	Parse(ctx context.Context, input string, userCurrency string) (*ParsedTransaction, error)
}

// TransactionService handles the core business logic around transactions.
type TransactionService interface {
	LogTransaction(ctx context.Context, userID int64, text string, userCurrency string) (*ParsedTransaction, error)
	GetWeeklyStats(ctx context.Context, userID int64) (string, error)
}
