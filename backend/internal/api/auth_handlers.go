package api

import (
	"database/sql"
	"kept/internal/auth"
	"kept/internal/models"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RegisterHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req models.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if req.Username == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Username and password are required")
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
		}

		// Insert user
		result, err := db.Exec(
			"INSERT INTO users (username, password_hash) VALUES (?, ?)",
			req.Username, hashedPassword,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusConflict, "Username already exists")
		}

		userID, _ := result.LastInsertId()

		// Generate access and refresh tokens
		accessToken, err := auth.GenerateToken(int(userID), req.Username)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate token")
		}

		days := auth.RefreshDays(req.Remember)
		refreshToken, err := auth.GenerateRefreshToken(int(userID), req.Username, days)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate refresh token")
		}

		user := models.User{
			ID:       int(userID),
			Username: req.Username,
		}

		// Persist refresh token in DB and set cookie
		expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		if err := StoreRefreshToken(db, int(userID), refreshToken, expiresAt, days); err != nil {
			log.Printf("Failed to store refresh token (register): %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to store refresh token")
		}
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Expires:  expiresAt,
			HTTPOnly: true,
			Secure:   auth.CookieSecure,
			SameSite: "Lax",
			Path:     "/api/auth",
		})

		return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
			Token: accessToken,
			User:  user,
		})
	}
}

func LoginHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req models.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Get user
		var user models.User

		// Try to select with email column first, fall back if it doesn't exist
		err := db.QueryRow(
			"SELECT id, username, password_hash, COALESCE(email, '') FROM users WHERE username = ?",
			req.Username,
		).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Email)

		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid username or password")
		}
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Database error")
		}

		// Check password
		if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid username or password")
		}

		// Generate access and refresh tokens
		accessToken, err := auth.GenerateToken(user.ID, user.Username)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate token")
		}

		// Determine TTL days based on remember flag
		days := auth.RefreshDays(req.Remember)
		refreshToken, err := auth.GenerateRefreshToken(user.ID, user.Username, days)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate refresh token")
		}
		// Persist refresh token and set cookie
		expiresAt := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		if err := StoreRefreshToken(db, user.ID, refreshToken, expiresAt, days); err != nil {
			log.Printf("Failed to store refresh token (login): %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to store refresh token")
		}
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Expires:  expiresAt,
			HTTPOnly: true,
			Secure:   auth.CookieSecure,
			SameSite: "Lax",
			Path:     "/api/auth",
		})

		return c.JSON(models.AuthResponse{
			Token: accessToken,
			User:  user,
		})
	}
}

// RefreshTokenHandler generates a new access token from a valid refresh token cookie
func RefreshTokenHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get refresh token from cookie
		refreshToken := c.Cookies("refresh_token")
		if refreshToken == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Refresh token not found")
		}

		// Validate refresh token signature
		claims, err := auth.ValidateRefreshToken(refreshToken)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid or expired refresh token")
		}

		// Check token presence in DB and get its TTL
		dbUserID, ttlDays, err := ValidateRefreshTokenInDB(db, refreshToken)
		if err != nil {
			log.Printf("Refresh token DB validation failed: %v", err)
			return fiber.NewError(fiber.StatusUnauthorized, "Refresh token not valid")
		}
		if dbUserID != claims.UserID {
			return fiber.NewError(fiber.StatusUnauthorized, "Token user mismatch")
		}

		// Generate new access token
		accessToken, err := auth.GenerateToken(claims.UserID, claims.Username)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate access token")
		}

		// Rotate refresh token: create new token with same TTL, store and revoke old
		newRefreshToken, err := auth.GenerateRefreshToken(claims.UserID, claims.Username, ttlDays)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate new refresh token")
		}
		expiresAt := time.Now().Add(time.Duration(ttlDays) * 24 * time.Hour)
		if err := StoreRefreshToken(db, claims.UserID, newRefreshToken, expiresAt, ttlDays); err != nil {
			log.Printf("Failed to store new refresh token (refresh): %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to store new refresh token")
		}
		// Revoke old token
		if err := RevokeRefreshToken(db, refreshToken); err != nil {
			// non-fatal: log but continue
		}

		// Update refresh token cookie
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    newRefreshToken,
			Expires:  expiresAt,
			HTTPOnly: true,
			Secure:   auth.CookieSecure,
			SameSite: "Lax",
			Path:     "/api/auth",
		})

		return c.JSON(fiber.Map{
			"token": accessToken,
		})
	}
}

// LogoutHandler clears the refresh token cookie
func LogoutHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Revoke refresh token in DB if present
		old := c.Cookies("refresh_token")
		if old != "" {
			_ = RevokeRefreshToken(db, old) // best-effort; ignore errors
		}

		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Expires:  time.Now().Add(-1 * time.Hour),
			HTTPOnly: true,
			Secure:   auth.CookieSecure,
			SameSite: "Lax",
			Path:     "/api/auth",
		})

		return c.JSON(fiber.Map{
			"message": "Logged out successfully",
		})
	}
}
