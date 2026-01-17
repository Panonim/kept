package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func Initialize(dbPath string) (*sql.DB, error) {
	// Create data directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Detect whether DB file already exists. If it does, require an
	// encryption key to be provided via `DB_ENCRYPTION_KEY` so the
	// application doesn't open an existing (possibly encrypted) DB
	// without a key.
	if _, err := os.Stat(dbPath); err == nil {
		if os.Getenv("DB_ENCRYPTION_KEY") == "" {
			return nil, fmt.Errorf("existing database detected at %s: DB_ENCRYPTION_KEY must be set to open it", dbPath)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// If an encryption key is provided via environment, apply it
	// immediately after opening. This enables use with SQLCipher
	// (requires the image/build to be linked against SQLCipher).
	if key := os.Getenv("DB_ENCRYPTION_KEY"); key != "" {
		// Escape single quotes in the key for the PRAGMA statement
		esc := strings.ReplaceAll(key, "'", "''")
		if _, err := db.Exec(fmt.Sprintf("PRAGMA key = '%s';", esc)); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set database encryption key: %w", err)
		}
		// Optional: set compatibility mode for newer SQLCipher versions
		_, _ = db.Exec("PRAGMA cipher_compatibility = 4;")
		// Quick accessibility check: ensure we can read the schema. If the key is wrong
		// or the file is not a valid SQLite database, this will return an error.
		var count int
		row := db.QueryRow("SELECT count(*) FROM sqlite_master;")
		if err := row.Scan(&count); err != nil {
			db.Close()
			return nil, fmt.Errorf("database inaccessible with provided encryption key: %w", err)
		}
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS promises (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		recipient TEXT NOT NULL,
		description TEXT NOT NULL,
		due_date DATETIME,
		reminder_frequency TEXT,
		last_reminded_at DATETIME,
		current_state TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS promise_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		promise_id INTEGER NOT NULL,
		state TEXT NOT NULL,
		reflection_note TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (promise_id) REFERENCES promises(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS reminders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		promise_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		remind_at DATETIME NOT NULL,
		offset_minutes INTEGER NOT NULL,
		is_sent BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (promise_id) REFERENCES promises(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS push_subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		endpoint TEXT NOT NULL,
		p256dh TEXT NOT NULL,
		auth TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, endpoint),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Server-side refresh token store for rotating refresh tokens
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		ttl_days INTEGER NOT NULL DEFAULT 7,
		revoked BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_promises_user_id ON promises(user_id);
	CREATE INDEX IF NOT EXISTS idx_promise_events_promise_id ON promise_events(promise_id);
	CREATE INDEX IF NOT EXISTS idx_reminders_user_id ON reminders(user_id);
	CREATE INDEX IF NOT EXISTS idx_reminders_remind_at ON reminders(remind_at);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
	`

	_, err := db.Exec(schema)
	return err
}
