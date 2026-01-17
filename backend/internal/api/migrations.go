package api

import (
	"database/sql"
	"fmt"
)

// columnExists checks if a column exists on a given table (SQLite PRAGMA table_info)
func columnExists(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var cid int
	var name string
	var ctype string
	var notnull int
	var dflt sql.NullString
	var pk int

	for rows.Next() {
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, nil
}

// MigrateAddReminderFrequency ensures the promises table has reminder_frequency
// and last_reminded_at columns (idempotent).
func MigrateAddReminderFrequency(db *sql.DB) error {
	exists, err := columnExists(db, "promises", "reminder_frequency")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := db.Exec("ALTER TABLE promises ADD COLUMN reminder_frequency TEXT"); err != nil {
			return err
		}
	}

	exists, err = columnExists(db, "promises", "last_reminded_at")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := db.Exec("ALTER TABLE promises ADD COLUMN last_reminded_at DATETIME"); err != nil {
			return err
		}
	}
	return nil
}

// MigratePostponedToKept converts any promises in state 'postponed' to 'kept'
// and normalizes events from 'postponed' to 'kept'. It's idempotent.
func MigratePostponedToKept(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE promises SET current_state = 'kept', updated_at = CURRENT_TIMESTAMP WHERE current_state = 'postponed'"); err != nil {
		return err
	}

	if _, err := tx.Exec("UPDATE promise_events SET state = 'kept' WHERE state = 'postponed'"); err != nil {
		return err
	}

	return tx.Commit()
}