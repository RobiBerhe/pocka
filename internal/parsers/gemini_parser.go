package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"pocka/internal/core"
)

type geminiParser struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiParser(ctx context.Context, apiKey string, modelName string) (core.TransactionParser, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	model := client.GenerativeModel(modelName)
	
	// Set generation config for structured output
	model.ResponseMIMEType = "application/json"
	
	return &geminiParser{
		client: client,
		model:  model,
	}, nil
}

type geminiTransaction struct {
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
}

func (p *geminiParser) Parse(ctx context.Context, input string, userCurrency string) ([]*core.ParsedTransaction, error) {
	now := time.Now()
	
	systemPrompt := fmt.Sprintf(`You are Pocka, an AI Financial Assistant.
Extract financial transactions from the user's message.
- Today's date: %s
- User's default currency: %s
- Languages: Support English, Amharic, and others.

Output ONLY a JSON array of transactions.
Schema: [{"amount": float, "type": "INCOME"|"EXPENSE", "category": string, "description": string}]

Categories: Food, Transport, Rent, Salary, Entertainment, Health, Utilities, Shopping, Misc.`, 
	now.Format("Monday, Jan 02, 2006"), userCurrency)

	p.model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	resp, err := p.model.GenerateContent(ctx, genai.Text(input))
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from gemini")
	}

	var rawJSON string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			rawJSON += string(text)
		}
	}

	slog.Debug("Gemini Raw Response", "raw", rawJSON)

	var gTxns []geminiTransaction
	if err := json.Unmarshal([]byte(rawJSON), &gTxns); err != nil {
		slog.Error("Failed to unmarshal Gemini JSON", "raw", rawJSON, "error", err)
		return nil, fmt.Errorf("invalid json from gemini: %w", err)
	}

	if len(gTxns) == 0 {
		return nil, fmt.Errorf("no transactions found in input")
	}

	var result []*core.ParsedTransaction
	for _, gt := range gTxns {
		txType := core.TransactionTypeExpense
		if gt.Type == "INCOME" {
			txType = core.TransactionTypeIncome
		}

		result = append(result, &core.ParsedTransaction{
			Amount:      gt.Amount,
			Type:        txType,
			Category:    gt.Category,
			Description: gt.Description,
			Currency:    userCurrency,
			RawInput:    input,
			Date:        now, // AI could potentially extract dates too in the future
			Metadata:    map[string]interface{}{"parser": "gemini"},
		})
	}

	return result, nil
}
