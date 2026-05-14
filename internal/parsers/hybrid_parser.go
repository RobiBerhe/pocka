package parsers

import (
	"context"
	"log/slog"

	"pocka/internal/core"
)

type hybridParser struct {
	gemini core.TransactionParser
	regex  core.TransactionParser
	enabled bool
}

func NewHybridParser(gemini core.TransactionParser, regex core.TransactionParser, enabled bool) core.TransactionParser {
	return &hybridParser{
		gemini: gemini,
		regex:  regex,
		enabled: enabled,
	}
}

func (p *hybridParser) Parse(ctx context.Context, input string, userCurrency string) ([]*core.ParsedTransaction, error) {
	// Try Gemini if enabled
	if p.enabled && p.gemini != nil {
		slog.Debug("Attempting to parse with Gemini AI", "input", input)
		txns, err := p.gemini.Parse(ctx, input, userCurrency)
		if err == nil && len(txns) > 0 {
			slog.Info("Successfully parsed using Gemini", "input", input, "count", len(txns))
			return txns, nil
		}
		slog.Warn("Gemini parsing failed or returned no results, falling back to regex", "input", input, "error", err)
	} else {
		slog.Debug("AI Parsing is disabled or not initialized, using regex", "input", input)
	}

	// Fallback to Regex
	return p.regex.Parse(ctx, input, userCurrency)
}
