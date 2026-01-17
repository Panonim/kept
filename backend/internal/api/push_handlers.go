package api

import (
	"database/sql"
	"fmt"
	"kept/internal/models"

	"github.com/gofiber/fiber/v2"
)

func SubscribePushHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		var sub models.PushSubscription
		if err := c.BodyParser(&sub); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Missing subscription fields")
		}

		// Upsert subscription
		_, err := db.Exec(
			`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) 
			VALUES (?, ?, ?, ?)
			ON CONFLICT(user_id, endpoint) DO UPDATE SET
			p256dh = excluded.p256dh,
			auth = excluded.auth`,
			userID, sub.Endpoint, sub.P256dh, sub.Auth,
		)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

func UnsubscribePushHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		var body struct {
			Endpoint string `json:"endpoint"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		_, err := db.Exec(
			"DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?",
			userID, body.Endpoint,
		)
		if err != nil {
			return err
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

// TestPushHandler returns a sample push payload about the user's most recent promise.
func TestPushHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		row := db.QueryRow("SELECT id, recipient, description FROM promises WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1", userID)
		var id int
		var recipient, description string
		err := row.Scan(&id, &recipient, &description)
		if err == sql.ErrNoRows {
			return c.JSON(fiber.Map{
				"title": "Kept — Test Reminder",
				"body":  "You have no promises yet. Create one to receive periodic reminders.",
				"options": fiber.Map{"tag": "kept-test"},
			})
		} else if err != nil {
			return err
		}

		title := "Reminder: Check your promise"
		body := fmt.Sprintf("Remember your promise to %s: %s", recipient, description)

		return c.JSON(fiber.Map{
			"title": title,
			"body":  body,
			"options": fiber.Map{
				"tag": "kept-test",
				"data": fiber.Map{"promise_id": id},
			},
		})
	}
}
