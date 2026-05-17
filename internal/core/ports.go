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

type OnboardingState string

const (
	OnboardingStateAwaitingName     OnboardingState = "AWAITING_NAME"
	OnboardingStateAwaitingLanguage OnboardingState = "AWAITING_LANGUAGE"
	OnboardingStateAwaitingCurrency OnboardingState = "AWAITING_CURRENCY"
	OnboardingStateAwaitingContact  OnboardingState = "AWAITING_CONTACT"
	OnboardingStateCompleted        OnboardingState = "COMPLETED"
)

type ParsedTransaction struct {
	Amount      float64
	Type        TransactionType
	Category    string
	Description string
	Merchant    string
	Emoji       string
	Currency    string
	RawInput    string
	Date        time.Time
	// Metadata allows for future AI-driven insights (e.g., merchant, location, confidence score)
	Metadata map[string]interface{}
}

type SummaryStats struct {
	TotalIncome    float64
	TotalExpense   float64
	NetBalance     float64
	Currency       string
	CategoryTotals map[string]float64
	Streak         int
	StartDate      time.Time
	EndDate        time.Time
}

// TransactionParser defines how text should be parsed into structured transaction data.
type TransactionParser interface {
	Parse(ctx context.Context, input string, userCurrency string) ([]*ParsedTransaction, error)
}

// TransactionService handles the core business logic around transactions.
type TransactionService interface {
	LogTransaction(ctx context.Context, userID int64, text string, userCurrency string) ([]*ParsedTransaction, []string, error)
	GetWeeklyStats(ctx context.Context, userID int64) (*SummaryStats, error)
	GetMonthlyStats(ctx context.Context, userID int64) (*SummaryStats, error)
	GetCustomStats(ctx context.Context, userID int64, start, end time.Time) (*SummaryStats, error)
	ExportDataCSV(ctx context.Context, userID int64) ([]byte, error)
}

// ReportService generates visual reports and summary cards.
type ReportService interface {
	GenerateSummaryCard(ctx context.Context, title string, stats *SummaryStats) ([]byte, error)
}

// NotificationService handles proactive user engagement.
type NotificationService interface {
	Start()
	Stop()
}

// I18nService handles multi-language translations.
type I18nService interface {
	Translate(langCode, key string, args ...interface{}) string
}
