package api

import (
	"database/sql"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, db *sql.DB) {
	api := app.Group("/api")

	// Check if registration is disabled
	disableRegistration := strings.ToLower(os.Getenv("DISABLE_REGISTRATION")) == "true"

	// Configuration endpoint (public)
	api.Get("/config", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"disableRegistration": disableRegistration,
		})
	})

	// Auth routes
	auth := api.Group("/auth")
	if !disableRegistration {
		auth.Post("/register", RegisterHandler(db))
	}
	auth.Post("/login", LoginHandler(db))
	auth.Post("/refresh", RefreshTokenHandler(db))
	auth.Post("/logout", LogoutHandler(db))

	// VAPID public key endpoint (public - must be before protected routes for proper routing)
	api.Get("/push/vapid-public-key", VapidPublicKeyHandler())

	// Protected routes
	protected := api.Group("/", AuthMiddleware())

	// Promise routes
	promises := protected.Group("/promises")
	promises.Post("/", CreatePromiseHandler(db))
	promises.Get("/", ListPromisesHandler(db))
	promises.Get("/:id", GetPromiseHandler(db))
	promises.Put("/:id/state", UpdatePromiseStateHandler(db))
	promises.Put("/:id", UpdatePromiseHandler(db))
	promises.Delete("/:id", DeletePromiseHandler(db))

	// Timeline route
	protected.Get("/timeline", GetTimelineHandler(db))

	// Reminder routes
	reminders := protected.Group("/reminders")
	reminders.Post("/promise/:promiseId", CreateReminderHandler(db))
	reminders.Get("/", ListRemindersHandler(db))
	reminders.Delete("/:id", DeleteReminderHandler(db))

	// Push subscription routes
	push := protected.Group("/push")
	push.Post("/subscribe", SubscribePushHandler(db))
	push.Delete("/unsubscribe", UnsubscribePushHandler(db))

	// User profile routes
	user := protected.Group("/user")
	user.Get("/profile", GetUserProfileHandler(db))
	user.Put("/email", UpdateUserEmailHandler(db))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
