package auth

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte
var refreshSecret []byte
var accessTokenMinutes = 15
var refreshTokenDays = 7
var rememberRefreshDays = 30
var CookieSecure = true

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required and must not be empty")
	}
	if len(secret) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters long")
	}
	jwtSecret = []byte(secret)
	
	// Check if cookie security should be disabled (useful for local HTTP dev)
	if os.Getenv("COOKIE_SECURE") == "false" {
		CookieSecure = false
	}
	
	// Refresh tokens use a separate secret for better security
	refreshSecretEnv := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecretEnv == "" {
		refreshSecretEnv = secret + "-refresh" // Derive from main secret if not provided
	}
	refreshSecret = []byte(refreshSecretEnv)

	// Load expiry configuration from environment (optional overrides)
	if v := os.Getenv("ACCESS_TOKEN_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			accessTokenMinutes = n
		}
	}
	if v := os.Getenv("REFRESH_TOKEN_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			refreshTokenDays = n
		}
	}
	if v := os.Getenv("REMEMBER_REFRESH_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rememberRefreshDays = n
		}
	}
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	TokenType string `json:"token_type,omitempty"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// GenerateToken creates a short-lived access token (15 minutes)
func GenerateToken(userID int, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(accessTokenMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateRefreshToken creates a long-lived refresh token (7 days)
// GenerateRefreshToken creates a refresh token that expires after the given number of days
func GenerateRefreshToken(userID int, username string, days int) (string, error) {
	if days <= 0 {
		days = refreshTokenDays
	}
	claims := Claims{
		UserID:   userID,
		Username: username,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(days) * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(refreshSecret)
}

func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.TokenType != "access" {
			return nil, errors.New("invalid token type")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken validates a refresh token
func ValidateRefreshToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return refreshSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.TokenType != "refresh" {
			return nil, errors.New("invalid token type")
		}
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// RefreshDays returns configured refresh token TTL in days depending on remember flag
func RefreshDays(remember bool) int {
	if remember {
		return rememberRefreshDays
	}
	return refreshTokenDays
}
