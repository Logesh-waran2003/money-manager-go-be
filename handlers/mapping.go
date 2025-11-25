package handlers

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"money-manager-go-be/database"
	"money-manager-go-be/models"
)

// RemapRequest represents the payload for remapping a transaction
type RemapRequest struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	NewMerchant   string    `json:"new_merchant"`
	NewCategory   string    `json:"new_category"`
	CreateRule    bool      `json:"create_rule"`
}

// RemapTransaction updates a transaction and optionally creates a rule
func RemapTransaction(c *fiber.Ctx) error {
	// TODO: Auth middleware
	userIDStr := c.Get("X-User-ID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		userID = uuid.Nil
	}
	if userID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID required in X-User-ID header"})
	}

	var req RemapRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// 1. Update the specific Transaction
	var transaction models.Transaction
	if err := database.DB.Where("id = ? AND user_id = ?", req.TransactionID, userID).First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Transaction not found"})
	}

	transaction.Merchant = req.NewMerchant
	transaction.Category = req.NewCategory
	transaction.IsManual = true

	if err := database.DB.Save(&transaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update transaction"})
	}

	// 2. If create_rule is true
	if req.CreateRule {
		// Extract keyword: For simplicity, we might use the merchant name or a part of raw text.
		// The prompt says "Extract a keyword... (simple logic or take from payload)".
		// Let's assume the user wants to match future transactions like this one.
		// A simple heuristic: if the raw text contains the new merchant name, use that as pattern.
		// Otherwise, maybe just use the raw text? Or better, let's just use the NewMerchant as the pattern if it's found in RawText,
		// otherwise we might need a more sophisticated extraction or explicit pattern in payload.
		// For this MVP, let's assume the NewMerchant IS the keyword we want to match.

		pattern := req.NewMerchant
		// Verify pattern exists in raw text to be safe, otherwise rule might be too broad or wrong?
		// Actually, sometimes the merchant name isn't in the text exactly.
		// Let's try to find a common substring or just use the whole RawText? No, that's too specific.
		// Let's stick to: Use NewMerchant as the pattern.

		rule := models.CategoryRule{
			UserID:         userID,
			Pattern:        pattern,
			TargetCategory: req.NewCategory,
			TargetMerchant: req.NewMerchant,
		}

		if err := database.DB.Create(&rule).Error; err != nil {
			log.Printf("Failed to create rule: %v", err)
			// Don't fail the request, just log it? Or maybe return warning.
		} else {
			// (Bonus) Trigger background re-scan
			go func(uid uuid.UUID, r models.CategoryRule) {
				// Find all "Uncategorized" transactions for this user that match the new rule
				var txs []models.Transaction
				database.DB.Where("user_id = ? AND category = ?", uid, "Uncategorized").Find(&txs)

				for _, tx := range txs {
					if strings.Contains(strings.ToLower(tx.RawText), strings.ToLower(r.Pattern)) {
						tx.Category = r.TargetCategory
						tx.Merchant = r.TargetMerchant
						database.DB.Save(&tx)
					}
				}
			}(userID, rule)
		}
	}

	return c.JSON(fiber.Map{"message": "Transaction updated successfully"})
}
