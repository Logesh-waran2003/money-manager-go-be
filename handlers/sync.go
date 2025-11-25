package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"money-manager-go-be/database"
	"money-manager-go-be/models"
)

// SMSLog represents the incoming payload for sync
type SMSLog struct {
	MobileID string    `json:"mobile_id"`
	Text     string    `json:"text"`
	Amount   float64   `json:"amount"`
	Date     time.Time `json:"date"`
}

// SyncResponse represents the response for the sync endpoint
type SyncResponse struct {
	Synced     int `json:"synced"`
	Duplicates int `json:"duplicates"`
}

// BatchSync handles the batch synchronization of SMS logs
func BatchSync(c *fiber.Ctx) error {
	// TODO: Get UserID from context/auth middleware.
	// For now, we'll assume a specific user ID or extract it if available.
	// In a real app, this would be c.Locals("userID").
	// We will use a hardcoded ID for demonstration if not present, or expect it in headers.
	// Let's assume we get it from a header "X-User-ID" for this core implementation.
	userIDStr := c.Get("X-User-ID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		// Fallback for testing/dev without auth
		// In production, return unauthorized
		// return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid User ID"})
		// Creating a dummy UUID for now if header is missing to prevent crash
		userID = uuid.Nil
	}

	// If userID is Nil, we might want to return error, but let's proceed for now or return error.
	if userID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID required in X-User-ID header"})
	}

	var logs []SMSLog
	if err := c.BodyParser(&logs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(logs) == 0 {
		return c.JSON(SyncResponse{Synced: 0, Duplicates: 0})
	}

	// 1. Deduplication: Get existing MobileIDs for this user
	var existingMobileIDs []string
	// Extract mobile IDs from payload to filter query
	payloadMobileIDs := make([]string, len(logs))
	for i, log := range logs {
		payloadMobileIDs[i] = log.MobileID
	}

	database.DB.Model(&models.Transaction{}).
		Where("user_id = ? AND mobile_id IN ?", userID, payloadMobileIDs).
		Pluck("mobile_id", &existingMobileIDs)

	existingMap := make(map[string]bool)
	for _, id := range existingMobileIDs {
		existingMap[id] = true
	}

	// 2. Fetch Category Rules
	var rules []models.CategoryRule
	database.DB.Where("user_id = ?", userID).Find(&rules)

	var newTransactions []models.Transaction
	duplicates := 0

	for _, log := range logs {
		if existingMap[log.MobileID] {
			duplicates++
			continue
		}

		// 3. Rule Engine
		category := "Uncategorized"
		merchant := ""

		// Simple normalization for matching
		lowerText := strings.ToLower(log.Text)

		for _, rule := range rules {
			if strings.Contains(lowerText, strings.ToLower(rule.Pattern)) {
				category = rule.TargetCategory
				merchant = rule.TargetMerchant
				break // Stop at first match
			}
		}

		// If merchant is empty, maybe try to extract it?
		// For now, if no rule matches, merchant remains empty or we could set it to "Unknown"
		if merchant == "" {
			merchant = "Unknown"
		}

		newTransactions = append(newTransactions, models.Transaction{
			UserID:          userID,
			MobileID:        log.MobileID,
			RawText:         log.Text,
			Amount:          log.Amount,
			Merchant:        merchant,
			Category:        category,
			IsManual:        false,
			TransactionDate: log.Date,
		})
	}

	// 4. Batch Insert
	if len(newTransactions) > 0 {
		// CreateInBatches is efficient for large datasets
		result := database.DB.CreateInBatches(newTransactions, 100)
		if result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to sync transactions"})
		}
	}

	return c.JSON(SyncResponse{
		Synced:     len(newTransactions),
		Duplicates: duplicates,
	})
}
