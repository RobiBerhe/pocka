package parsers

import (
	"testing"
)

func TestRegexParser(t *testing.T) {
	parser := NewRegexParser()

	tests := []struct {
		name         string
		input        string
		expectedAmt  float64
		expectedCat  string
		expectErr    bool
	}{
		{"Basic int", "100 lunch", 100, "Food", false},
		{"Decimal amount", "10.50 taxi", 10.50, "Transport", false},
		{"Amount only", "50", 50, "Misc", false},
		{"Comma format", "1,000 rent", 1000, "Housing", false},
		{"Invalid amount", "abc lunch", 0, "", true},
		{"Empty string", "", 0, "", true},
		{"Unmapped category", "20 haircut", 20, "Misc", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := parser.Parse(tc.input, "ETB")
			
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
		})
	}
}
