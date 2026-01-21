package api

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

type UpdateEmailRequest struct {
	Email *string `json:"email"`
}

// UpdateUserEmailHandler updates the user's email address
func UpdateUserEmailHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		var req UpdateEmailRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Validate email format if provided
		if req.Email != nil && *req.Email != "" {
			email := *req.Email
			// Basic email validation
			if len(email) < 3 || len(email) > 254 {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
			}
		}

		// Update email in database
		var emailValue interface{}
		if req.Email == nil || *req.Email == "" {
			emailValue = nil
		} else {
			emailValue = *req.Email
		}

		_, err := db.Exec("UPDATE users SET email = ? WHERE id = ?", emailValue, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update email")
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Email updated successfully",
		})
	}
}

// GetUserProfileHandler returns the current user's profile information
func GetUserProfileHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		var username string
		var email sql.NullString
		var createdAt string

		err := db.QueryRow(
			"SELECT username, email, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&username, &email, &createdAt)

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user profile")
		}

		profile := fiber.Map{
			"id":         userID,
			"username":   username,
			"created_at": createdAt,
		}

		if email.Valid {
			profile["email"] = email.String
		} else {
			profile["email"] = nil
		}

		return c.JSON(profile)
	}
}
