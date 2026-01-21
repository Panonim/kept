package api

import (
	"database/sql"
	"fmt"
	"kept/internal/models"
	"regexp"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

var (
	lastEmailTestTime time.Time
	emailTestMutex    sync.Mutex
)

// TestEmailHandler sends a test email to verify SMTP configuration
// Rate limited to once per 10 minutes per server
func TestEmailHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		// Rate limiting check
		emailTestMutex.Lock()
		timeSinceLastTest := time.Since(lastEmailTestTime)
		if timeSinceLastTest < 10*time.Minute {
			emailTestMutex.Unlock()
			remaining := 10*time.Minute - timeSinceLastTest
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Email test rate limited",
				"retry_after_seconds": int(remaining.Seconds()),
				"message": fmt.Sprintf("Please wait %s before testing again", formatDuration(remaining)),
			})
		}
		lastEmailTestTime = time.Now()
		emailTestMutex.Unlock()

		// Check if SMTP is configured
		config, err := GetSMTPConfig()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "SMTP not configured",
				"message": "Please configure SMTP environment variables (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM)",
				"details": err.Error(),
			})
		}

		// Get user email from database
		var userEmail sql.NullString
		var username string
		err = db.QueryRow("SELECT email, username FROM users WHERE id = ?", userID).Scan(&userEmail, &username)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user data")
		}

		if !userEmail.Valid || userEmail.String == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No email address",
				"message": "Your account does not have an email address set. Please update your profile.",
			})
		}

		// Validate email format
		if !isValidEmail(userEmail.String) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid email address",
				"message": "The email address on your account is not valid. Please update your profile with a valid email address.",
			})
		}

		// Create a test promise for the email
		now := time.Now()
		tomorrow := now.Add(24 * time.Hour)
		testPromise := models.Promise{
			ID:          0, // Test promise
			Recipient:   "yourself",
			Description: "This is a test reminder to verify your email notifications are working correctly. If you received this, your Kept email system is configured properly!",
			DueDate:     &tomorrow,
			CurrentState: "active",
		}

		// Send test email
		err = SendReminderEmail(db, testPromise, userEmail.String)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to send test email",
				"message": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": fmt.Sprintf("Test email sent successfully to %s", userEmail.String),
			"smtp_config": fiber.Map{
				"host": config.Host,
				"port": config.Port,
				"from": config.From,
				"tls":  config.UseTLS,
			},
			"note": "Check your inbox (and spam folder) for the test email",
		})
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) - (minutes * 60)
	if seconds > 0 {
		return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

// isValidEmail checks if an email address is valid (simple regex)
func isValidEmail(email string) bool {
    // Basic RFC 5322 regex for demonstration (not exhaustive)
    // Accepts most common valid emails, rejects obvious invalid ones
    re := `^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$`
    matched, _ := regexp.MatchString(re, email)
    return matched
}
