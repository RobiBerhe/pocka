package parsers

import (
	"context"
	"testing"
	"pocka/internal/core"
)

func TestRegexParser(t *testing.T) {
	parser := NewRegexParser()
	ctx := context.Background()

	tests := []struct {
		name         string
		input        string
		expectedAmt  float64
		expectedCat  string
		expectedType core.TransactionType
		expectErr    bool
	}{
		{"Basic int", "100 lunch", 100, "Food", core.TransactionTypeExpense, false},
		{"Decimal amount", "10.50 taxi", 10.50, "Transport", core.TransactionTypeExpense, false},
		{"Reverse order", "coffee 40", 40, "Food", core.TransactionTypeExpense, false},
		{"Income detection", "salary 12000", 12000, "Income", core.TransactionTypeIncome, false},
		{"Amharic support", "ቡና 50", 50, "Food", core.TransactionTypeExpense, false},
		{"Amharic income", "ሽያጭ 5000", 5000, "Income", core.TransactionTypeIncome, false},
		{"Comma format", "1,000 rent", 1000, "Housing", core.TransactionTypeExpense, false},
		{"Invalid amount", "abc lunch", 0, "", "", true},
		{"Empty string", "", 0, "", "", true},
		{"Unmapped category", "20 haircut", 20, "Misc", core.TransactionTypeExpense, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := parser.Parse(ctx, tc.input, "ETB")
			
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("did not expect error, got: %v", err)
			}
			
			if parsed.Amount != tc.expectedAmt {
				t.Errorf("expected amount %.2f, got %.2f", tc.expectedAmt, parsed.Amount)
			}
			
			if parsed.Category != tc.expectedCat {
				t.Errorf("expected category %s, got %s", tc.expectedCat, parsed.Category)
			}

			if parsed.Type != tc.expectedType {
				t.Errorf("expected type %s, got %s", tc.expectedType, parsed.Type)
			}
		})
	}
}
