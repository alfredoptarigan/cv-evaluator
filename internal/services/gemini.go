package services

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type GeminiService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateText(ctx context.Context, prompt string, temperature float32) (string, error)
	GenerateTextWithRetry(ctx context.Context, prompt string, temperature float32, maxRetries int) (string, error)
}

type geminiService struct {
	client     *genai.Client
	modelName  string
	embedModel string
}

func NewGeminiService(apiKey string) (GeminiService, error) {
	ctx := context.Background()

	fmt.Println("🔑 Gemini API key:", apiKey)

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &geminiService{
		client:     client,
		modelName:  "gemini-2.5-flash",
		embedModel: "text-embedding-004",
	}, nil
}

// GenerateEmbedding implements GeminiService.
func (g *geminiService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Truncate text if too long (max ~10000 tokens for embedding)
	if len(text) > 40000 {
		text = text[:40000]
	}

	result, err := g.client.Models.EmbedContent(ctx, g.embedModel, genai.Text(text), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if result == nil || len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}

	return result.Embeddings[0].Values, nil
}

// GenerateText implements GeminiService.
func (g *geminiService) GenerateText(ctx context.Context, prompt string, temperature float32) (string, error) {
	// Create generation config
	config := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: 4096,
	}

	// Generate response
	resp, err := g.client.Models.GenerateContent(ctx, g.modelName, genai.Text(prompt), config)
	if err != nil {
		fmt.Printf("❌ Gemini API error: %v\n", err)
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	if resp == nil {
		fmt.Println("❌ Gemini API returned nil response")
		return "", fmt.Errorf("no response generated (nil response)")
	}

	// Log response for debugging
	fmt.Printf("📊 Gemini response received\n")

	// Get text from response
	text := resp.Text()
	if text == "" {
		fmt.Println("❌ No text content in response")

		// Try to extract any content from candidates if available
		if resp.Candidates != nil && len(resp.Candidates) > 0 {
			var textParts []string
			for i, candidate := range resp.Candidates {
				fmt.Printf("📄 Candidate %d: %+v\n", i, candidate)
				if candidate.Content != nil {
					textParts = append(textParts, fmt.Sprintf("%v", candidate.Content))
				}
			}

			if len(textParts) > 0 {
				fmt.Println("⚠️ Using fallback string representation of response parts")
				return strings.Join(textParts, "\n"), nil
			}
		}

		return "", fmt.Errorf("no text content in response")
	}

	return text, nil
}

// GenerateTextWithRetry implements GeminiService.
func (g *geminiService) GenerateTextWithRetry(ctx context.Context, prompt string, temperature float32, maxRetries int) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := g.GenerateText(ctx, prompt, temperature)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		// Log retry attempt
		if attempt < maxRetries {
			fmt.Printf("⚠️ Attempt %d failed: %v. Retrying...\n", attempt, err)
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
