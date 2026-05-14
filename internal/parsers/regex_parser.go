package parsers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"pocka/internal/core"
)

var (
	ErrInvalidFormat = errors.New("invalid format. Example: 100 lunch or coffee 40")
	ErrInvalidAmount = errors.New("invalid amount. Amount must be a number")
)

type RegexParser struct {
	categoryMapping map[string][]string
	incomeKeywords  []string
	numberRegex    *regexp.Regexp
}

func NewRegexParser() *RegexParser {
	return &RegexParser{
		categoryMapping: map[string][]string{
			"Food":      {"lunch", "breakfast", "dinner", "food", "coffee", "restaurant", "grocery", "water", "macchiato", "ቡና", "ምሳ", "ቁርስ", "እራት"},
			"Transport": {"taxi", "bus", "transport", "ride", "uber", "bolt", "train", "flight", "fuel", "ታክሲ", "ባቡር"},
			"Airtime":   {"data", "airtime", "wifi", "internet", "card", "ካርድ"},
			"Shopping":  {"clothes", "shoes", "shopping", "gift", "ልብስ", "ጫማ"},
			"Health":    {"medicine", "pharmacy", "hospital", "clinic", "መድሃኒት"},
			"Housing":   {"rent", "electricity", "water", "maintenance", "ቤት"},
			"Income":    {"salary", "sold", "got", "received", "bonus", "profit", "ደመወዝ", "ትርፍ", "ሽያጭ"},
		},
		incomeKeywords: []string{"salary", "sold", "got", "received", "bonus", "profit", "ደመወዝ", "ትርፍ", "ሽያጭ"},
		// Regex to find numbers including commas and decimals
		numberRegex: regexp.MustCompile(`^(\d{1,3}(,\d{3})*(\.\d+)?|\d+(\.\d+)?)$`),
	}
}

// Parse extracts the amount, category, and type from the input string.
func (p *RegexParser) Parse(ctx context.Context, input string, userCurrency string) ([]*core.ParsedTransaction, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrInvalidFormat
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, ErrInvalidFormat
	}

	var amount float64
	var description string
	var foundAmount bool

	// Strategy: Check if the first or last part is a number.
	// This supports "100 coffee" and "coffee 100".

	// 1. Check first part
	firstPart := strings.ReplaceAll(parts[0], ",", "")
	if val, err := strconv.ParseFloat(firstPart, 64); err == nil {
		amount = val
		description = strings.Join(parts[1:], " ")
		foundAmount = true
	} else if len(parts) > 1 {
		// 2. Check last part
		lastIdx := len(parts) - 1
		lastPart := strings.ReplaceAll(parts[lastIdx], ",", "")
		if val, err := strconv.ParseFloat(lastPart, 64); err == nil {
			amount = val
			description = strings.Join(parts[:lastIdx], " ")
			foundAmount = true
		}
	}

	if !foundAmount {
		return nil, ErrInvalidAmount
	}

	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	description = strings.ToLower(strings.TrimSpace(description))
	
	// Determine Type (Income vs Expense)
	txType := core.TransactionTypeExpense
	for _, kw := range p.incomeKeywords {
		if strings.Contains(description, kw) {
			txType = core.TransactionTypeIncome
			break
		}
	}

	// Determine Category
	category := "Misc"
	if txType == core.TransactionTypeIncome {
		category = "Income"
	}
	
	if description != "" {
		inferred := p.inferCategory(description)
		if inferred != "Misc" {
			category = inferred
		}
	}

	return []*core.ParsedTransaction{
		{
			Amount:      amount,
			Type:        txType,
			Category:    category,
			Description: description,
			Currency:    userCurrency,
			RawInput:    input,
			Metadata:    make(map[string]interface{}),
		},
	}, nil
}

func (p *RegexParser) inferCategory(description string) string {
	for cat, keywords := range p.categoryMapping {
		for _, kw := range keywords {
			// Basic substring check
			if strings.Contains(description, kw) {
				return cat
			}
		}
	}
	return "Misc"
}
