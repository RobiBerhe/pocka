package parsers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"pocka/internal/core"
)

var (
	ErrInvalidFormat = errors.New("invalid format. Example: 100 lunch")
	ErrInvalidAmount = errors.New("invalid amount. Amount must be a number")
)

type RegexParser struct {
	categoryMapping map[string][]string
}

func NewRegexParser() *RegexParser {
	return &RegexParser{
		categoryMapping: map[string][]string{
			"Food":      {"lunch", "breakfast", "dinner", "food", "coffee", "restaurant", "grocery", "water"},
			"Transport": {"taxi", "bus", "transport", "ride", "uber", "bolt", "train", "flight", "fuel"},
			"Airtime":   {"data", "airtime", "wifi", "internet", "card"},
			"Shopping":  {"clothes", "shoes", "shopping", "gift"},
			"Health":    {"medicine", "pharmacy", "hospital", "clinic"},
			"Housing":   {"rent", "electricity", "water", "maintenance"},
		},
	}
}

// Parse extracts the amount and category from the input string.
func (p *RegexParser) Parse(input string, userCurrency string) (*core.ParsedExpense, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrInvalidFormat
	}

	// Split by space. We expect at least an amount.
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, ErrInvalidFormat
	}

	// Parse Amount
	amountStr := parts[0]
	// Remove commas if any (e.g., 1,000)
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, ErrInvalidAmount
	}

	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	// Determine Category
	category := "Misc"
	if len(parts) > 1 {
		description := strings.ToLower(strings.Join(parts[1:], " "))
		category = p.inferCategory(description)
	}

	return &core.ParsedExpense{
		Amount:   amount,
		Category: category,
		Currency: userCurrency,
	}, nil
}

func (p *RegexParser) inferCategory(description string) string {
	for cat, keywords := range p.categoryMapping {
		for _, kw := range keywords {
			// Basic substring check (we can improve this later with word boundary regex)
			if strings.Contains(description, kw) {
				return cat
			}
		}
	}
	return "Misc"
}
