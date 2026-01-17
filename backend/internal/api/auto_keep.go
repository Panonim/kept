package api

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
)

func idsToCSV(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

// AutoKeepOverduePromises finds promises in state 'active' whose due_date has
// passed and converts them to 'kept', appending an event noting the automatic
// transition.
func AutoKeepOverduePromises(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id FROM promises WHERE current_state = 'active' AND due_date IS NOT NULL AND due_date <= CURRENT_TIMESTAMP")
	if err != nil {
		return err
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return tx.Commit()
	}

	// Update promises to 'kept' using parameterized queries
	updateStmt, err := tx.Prepare("UPDATE promises SET current_state = 'kept', updated_at = CURRENT_TIMESTAMP WHERE id = ?")
	if err != nil {
		return err
	}
	defer updateStmt.Close()
	
	updatedCount := 0
	for _, id := range ids {
		if _, err := updateStmt.Exec(id); err != nil {
			return err
		}
		updatedCount++
	}

	// Insert events for each auto-kept promise
	eventStmt, err := tx.Prepare("INSERT INTO promise_events (promise_id, state, reflection_note) VALUES (?, 'kept', ?)")
	if err != nil {
		return err
	}
	defer eventStmt.Close()

	for _, id := range ids {
		if _, err := eventStmt.Exec(id, "Auto-kept: due date passed"); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Auto-kept %d overdue promises", updatedCount)
	return nil
}
