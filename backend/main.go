package main

import (
	"log"
	"os"
	"strings"
	"time"

	"kept/internal/api"
	"kept/internal/database"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Initialize database
	db, err := database.Initialize("./data/kept.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Run migrations only if explicitly enabled (opt-in for safety)
	runMigrations := os.Getenv("RUN_MIGRATIONS") == "true"
	if runMigrations {
		log.Println("Running database migrations...")
		if err := api.MigratePostponedToKept(db); err != nil {
			log.Printf("Migration error: %v", err)
		}
		if err := api.MigrateAddReminderFrequency(db); err != nil {
			log.Printf("Migration error (reminder freq): %v", err)
		}
	} else {
		log.Println("Migrations skipped (set RUN_MIGRATIONS=true to enable)")
	}

	// Run background workers only if enabled (default: true for backward compatibility)
	enableWorkers := os.Getenv("ENABLE_WORKERS")
	if enableWorkers == "" {
		enableWorkers = "true" // Default to enabled
	}

	if enableWorkers == "true" {
		log.Println("Starting background workers...")
		// Run auto-keep once at startup and start background worker to repeatedly
		// auto-keep overdue promises (active → kept when due_date has passed).
		if err := api.AutoKeepOverduePromises(db); err != nil {
			log.Printf("Auto-keep error at startup: %v", err)
		}
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				if err := api.AutoKeepOverduePromises(db); err != nil {
					log.Printf("Auto-keep worker error: %v", err)
				}
				if err := api.ProcessRecurringReminders(db); err != nil {
					log.Printf("Recurring reminder worker error: %v", err)
				}
			}
		}()
	} else {
		log.Println("Background workers disabled (set ENABLE_WORKERS=true to enable)")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())

	// CORS configuration: restrict to specific origins for security
	allowedOriginsRaw := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := strings.TrimSpace(allowedOriginsRaw)
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:80,http://localhost:5173" // Default for local dev
		log.Println("WARNING: Using default ALLOWED_ORIGINS. Set ALLOWED_ORIGINS env var for production.")
	} else {
		// Normalize comma-separated list (trim whitespace around entries)
		if allowedOrigins != "*" {
			parts := strings.Split(allowedOrigins, ",")
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			allowedOrigins = strings.Join(parts, ",")
		}
	}

	log.Printf("CORS allowed origins: %s", allowedOrigins)

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true, // Required for cookies
	}))

	// Setup routes
	api.SetupRoutes(app, db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
