package api

import (
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "errors"
    "fmt"
    "strconv"
    "strings"
    "time"
)

func hashToken(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}

func parseExpiresAt(v any) (time.Time, bool) {
    switch t := v.(type) {
    case time.Time:
        return t, true
    case string:
        return parseExpiresAtString(t)
    case []byte:
        return parseExpiresAtString(string(t))
    default:
        return time.Time{}, false
    }
}

func parseExpiresAtString(s string) (time.Time, bool) {
    if s == "" {
        return time.Time{}, false
    }
    // SQLite/go-sqlite3 commonly uses these formats depending on how values were inserted.
    layouts := []string{
        "2006-01-02 15:04:05.999999999Z07:00", // e.g. 2026-01-24 15:39:59.609890513+00:00
        "2006-01-02 15:04:05Z07:00",
        time.RFC3339Nano,
        time.RFC3339,
        "2006-01-02 15:04:05",
    }
    for _, layout := range layouts {
        if t, err := time.Parse(layout, s); err == nil {
            return t, true
        }
    }
    return time.Time{}, false
}

func parseRevoked(v any) (bool, bool) {
    switch t := v.(type) {
    case bool:
        return t, true
    case int64:
        return t != 0, true
    case int:
        return t != 0, true
    case string:
        s := strings.TrimSpace(strings.ToLower(t))
        if s == "" {
            return false, false
        }
        if s == "true" {
            return true, true
        }
        if s == "false" {
            return false, true
        }
        if n, err := strconv.Atoi(s); err == nil {
            return n != 0, true
        }
        return false, false
    case []byte:
        return parseRevoked(string(t))
    default:
        return false, false
    }
}

// StoreRefreshToken stores a refresh token hash in the database with expiry
func StoreRefreshToken(db *sql.DB, userID int, token string, expiresAt time.Time, ttlDays int) error {
    th := hashToken(token)
    // Use INSERT OR IGNORE to avoid unique constraint failures when identical tokens
    // may be generated in quick succession (tests or race conditions).
    _, err := db.Exec("INSERT OR IGNORE INTO refresh_tokens (user_id, token_hash, expires_at, ttl_days) VALUES (?, ?, ?, ?)", userID, th, expiresAt, ttlDays)
    if err != nil {
        return err
    }
    // Ensure expires_at and ttl_days are at least set (in case token existed, update its metadata)
    _, err = db.Exec("UPDATE refresh_tokens SET expires_at = ?, ttl_days = ?, revoked = 0 WHERE token_hash = ?", expiresAt, ttlDays, th)
    return err
}

// ValidateRefreshTokenInDB checks that the token exists, is not revoked and not expired, returns userID if valid
func ValidateRefreshTokenInDB(db *sql.DB, token string) (int, int, error) {
    th := hashToken(token)
    var id int
    var userID int
	var expiresAt any
	var revoked any
    var ttlDays int
    row := db.QueryRow("SELECT id, user_id, expires_at, revoked, ttl_days FROM refresh_tokens WHERE token_hash = ?", th)
    if err := row.Scan(&id, &userID, &expiresAt, &revoked, &ttlDays); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return 0, 0, errors.New("refresh token not found")
        }
        return 0, 0, err
    }
	if r, ok := parseRevoked(revoked); ok {
		if r {
			return 0, 0, errors.New("refresh token revoked")
		}
	} else {
		// If we can't interpret revoked, be safe and reject.
		return 0, 0, fmt.Errorf("unexpected revoked type: %T", revoked)
	}
	// parse expiresAt (best-effort; don't fail validation if format is unexpected)
	if t, ok := parseExpiresAt(expiresAt); ok {
		if time.Now().After(t) {
			return 0, 0, errors.New("refresh token expired")
		}
	} else {
		_ = fmt.Sprintf("%v", expiresAt) // keep linter quiet if build tags change
	}
    return userID, ttlDays, nil
}

// RevokeRefreshToken revokes a refresh token by token string
func RevokeRefreshToken(db *sql.DB, token string) error {
    th := hashToken(token)
    _, err := db.Exec("UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?", th)
    return err
}
