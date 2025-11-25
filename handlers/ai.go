package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"money-manager-go-be/database"
	"money-manager-go-be/models"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/genai"
)

// AIResponse represents the structure we expect from Gemini
type AIResponse struct {
	TransactionID string `json:"transaction_id"`
	NewCategory   string `json:"new_category"`
	NewMerchant   string `json:"new_merchant"`
}

func AnalyzeUncategorized(c *fiber.Ctx) error {
	// TODO: Auth middleware should handle this
	userID := c.Get("X-User-ID")
	if userID == "" {
		log.Println("Error: User ID required but missing in header")
		return c.Status(400).JSON(fiber.Map{"error": "User ID required"})
	}

	log.Printf("Starting AI analysis for user: %s", userID)

	// 1. Fetch Uncategorized Transactions
	var txns []models.Transaction
	// Limit to 50 to avoid token limits and ensure speed
	if err := database.DB.Where("user_id = ? AND (category = ? OR category = ?)", userID, "Uncategorized", "").Limit(50).Find(&txns).Error; err != nil {
		log.Printf("Error fetching transactions: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch transactions"})
	}

	if len(txns) == 0 {
		log.Println("No uncategorized transactions found")
		return c.JSON(fiber.Map{
			"message":     "No uncategorized transactions found",
			"suggestions": []interface{}{},
		})
	}

	log.Printf("Found %d uncategorized transactions", len(txns))

	// 2. Construct the Prompt
	var promptBuilder strings.Builder
	promptBuilder.WriteString("You are a financial analyst. Analyze these bank transaction strings. \n")
	promptBuilder.WriteString("Return a RAW JSON ARRAY of objects. Do NOT use markdown formatting. \n")
	promptBuilder.WriteString("Each object must have: 'transaction_id', 'new_category' (e.g., Food, Travel, Bills, Shopping, Salary, Investment, Transfer), and 'new_merchant' (clean name).\n\n")

	for _, t := range txns {
		promptBuilder.WriteString(fmt.Sprintf(`{"transaction_id": "%s", "text": "%s", "amount": %.2f}`+"\n", t.ID, t.RawText, t.Amount))
	}

	// 3. Call Gemini
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("Error: GEMINI_API_KEY not set")
		return c.Status(500).JSON(fiber.Map{"error": "GEMINI_API_KEY not set"})
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Printf("Error initializing AI client: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to init AI client"})
	}

	log.Println("Sending request to Gemini...")
	// Using Gemini 1.5 Flash for speed/cost balance
	resp, err := client.Models.GenerateContent(ctx, "gemini-1.5-flash", genai.Text(promptBuilder.String()), nil)
	if err != nil {
		log.Printf("Error during AI generation: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "AI generation failed: " + err.Error()})
	}

	// 4. Parse and Clean Response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Println("Error: Empty response from AI")
		return c.Status(500).JSON(fiber.Map{"error": "Empty response from AI"})
	}

	// Extract text part.
	rawText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			rawText += part.Text
		}
	}

	// Clean Markdown if present (Gemini loves adding ```json ... ```)
	rawText = strings.TrimSpace(rawText)
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")

	log.Printf("Received response from AI (length: %d)", len(rawText))

	var suggestions []AIResponse
	if err := json.Unmarshal([]byte(rawText), &suggestions); err != nil {
		log.Printf("Error parsing AI response: %v. Raw text: %s", err, rawText)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to parse AI response", "raw": rawText})
	}

	log.Printf("Successfully parsed %d suggestions", len(suggestions))

	return c.JSON(fiber.Map{
		"count":       len(suggestions),
		"suggestions": suggestions,
	})
}
