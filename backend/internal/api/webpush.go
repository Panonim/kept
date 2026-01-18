package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gofiber/fiber/v2"
)

// PushPayload represents the notification payload sent to clients
type PushPayload struct {
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Icon  string                 `json:"icon,omitempty"`
	Badge string                 `json:"badge,omitempty"`
	Tag   string                 `json:"tag,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// GetVapidOptions returns configured VAPID options from environment
func GetVapidOptions() *webpush.Options {
	return &webpush.Options{
		Subscriber:      os.Getenv("VAPID_SUBJECT"),
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		TTL:             30,
	}
}

// IsWebPushConfigured checks if VAPID keys are configured
func IsWebPushConfigured() bool {
	publicKey := os.Getenv("VAPID_PUBLIC_KEY")
	privateKey := os.Getenv("VAPID_PRIVATE_KEY")
	subject := os.Getenv("VAPID_SUBJECT")

	return publicKey != "" && privateKey != "" && subject != ""
}

// SendPushToUser sends a push notification to all subscriptions for a user
func SendPushToUser(db *sql.DB, userID int, payload PushPayload) error {
	if !IsWebPushConfigured() {
		log.Println("Web push not configured - skipping notification")
		return nil
	}

	// Get all push subscriptions for the user
	rows, err := db.Query(
		"SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %w", err)
	}
	defer rows.Close()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	log.Printf("Payload to send: %s", string(payloadJSON))

	options := GetVapidOptions()
	successCount := 0
	failCount := 0
	subscriptionCount := 0

	for rows.Next() {
		subscriptionCount++
		var endpoint, p256dh, auth string
		if err := rows.Scan(&endpoint, &p256dh, &auth); err != nil {
			log.Printf("Error scanning subscription: %v", err)
			failCount++
			continue
		}

		log.Printf("Sending push to endpoint: %s", endpoint[:50]+"...")

		subscription := &webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				P256dh: p256dh,
				Auth:   auth,
			},
		}

		resp, err := webpush.SendNotification(payloadJSON, subscription, options)
		if err != nil {
			log.Printf("Failed to send push to %s: %v", endpoint, err)
			failCount++

			// If subscription is expired/invalid (410 Gone or 404), remove it
			if resp != nil && (resp.StatusCode == 410 || resp.StatusCode == 404) {
				_, _ = db.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint)
				log.Printf("Removed expired subscription: %s", endpoint)
			}
			continue
		}

		if resp != nil {
			log.Printf("Push response status: %d", resp.StatusCode)
			// Log response body for debugging (especially for errors like 403)
			if resp.StatusCode >= 400 {
				body, _ := io.ReadAll(resp.Body)
				log.Printf("Push service error response: %s", string(body))
			}
			resp.Body.Close()

			// If 403 Forbidden, the VAPID keys don't match - delete the subscription
			// so the client will re-subscribe with current keys
			if resp.StatusCode == 403 {
				_, _ = db.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint)
				log.Printf("Deleted mismatched subscription (403 Forbidden): %s", endpoint)
				failCount++
				continue
			}
		}

		successCount++
		log.Printf("Push sent successfully to user %d", userID)
	}

	log.Printf("Push notification summary for user %d: subscriptions=%d, success=%d, failed=%d", userID, subscriptionCount, successCount, failCount)

	if subscriptionCount == 0 {
		return fmt.Errorf("no push subscriptions found for user %d", userID)
	}

	if failCount > 0 && successCount == 0 {
		return fmt.Errorf("failed to send any push notifications (attempted %d)", failCount)
	}

	return nil
}

// VapidPublicKeyHandler returns the VAPID public key for client subscription
func VapidPublicKeyHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		publicKey := os.Getenv("VAPID_PUBLIC_KEY")
		if publicKey == "" {
			return fiber.NewError(fiber.StatusServiceUnavailable, "Push notifications not configured")
		}
		return c.JSON(fiber.Map{
			"publicKey": publicKey,
		})
	}
}

// SendTestPushHandler sends an actual push notification for testing
func SendTestPushHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		if !IsWebPushConfigured() {
			return fiber.NewError(fiber.StatusServiceUnavailable, "Push notifications not configured. Set VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, and VAPID_SUBJECT environment variables.")
		}

		// Get most recent promise for context
		row := db.QueryRow("SELECT id, recipient, description FROM promises WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1", userID)
		var promiseID int
		var recipient, description string
		err := row.Scan(&promiseID, &recipient, &description)

		var payload PushPayload
		if err == sql.ErrNoRows {
			payload = PushPayload{
				Title: "Kept — Test Notification",
				Body:  "This is a test notification",
				Icon:  "/Static/logos/Kept%20Mascot%20Colored.svg",
				Badge: "/Static/logos/Kept%20Mascot%20Colored.svg",
				Tag:   fmt.Sprintf("kept-test-%d", time.Now().Unix()),
			}
		} else if err != nil {
			return err
		} else {
			payload = PushPayload{
				Title: "Kept — Test Notification",
				Body:  "This is a test notification",
				Icon:  "/Static/logos/Kept%20Mascot%20Colored.svg",
				Badge: "/Static/logos/Kept%20Mascot%20Colored.svg",
				Tag:   fmt.Sprintf("kept-test-%d", time.Now().Unix()),
				Data:  map[string]interface{}{"promise_id": promiseID},
			}
		}

		if err := SendPushToUser(db, userID, payload); err != nil {
			log.Printf("Test push failed for user %d: %v", userID, err)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to send test notification: "+err.Error())
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Test notification sent",
		})
	}
}
