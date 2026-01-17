package api

import (
	"database/sql"
	"kept/internal/models"
	"log"
	"time"
)

// ProcessRecurringReminders checks for active promises with recurring reminders
// and triggers push notifications if enough time has passed since the last reminder.
func ProcessRecurringReminders(db *sql.DB) error {
	query := `
		SELECT id, user_id, recipient, description, reminder_frequency, last_reminded_at, created_at 
		FROM promises 
		WHERE current_state = 'active' AND reminder_frequency != '' AND reminder_frequency IS NOT NULL
	`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Promise
		// partially scan relevant fields
		err := rows.Scan(&p.ID, &p.UserID, &p.Recipient, &p.Description, &p.ReminderFrequency, &p.LastRemindedAt, &p.CreatedAt)
		if err != nil {
			log.Printf("Error scanning promise for reminder: %v", err)
			continue
		}

		if shouldRemind(p) {
			if err := sendReminder(db, p); err != nil {
				log.Printf("Failed to send reminder for promise %d: %v", p.ID, err)
			}
		}
	}
	return nil
}

func shouldRemind(p models.Promise) bool {
	lastReminded := p.CreatedAt
	if p.LastRemindedAt != nil {
		lastReminded = *p.LastRemindedAt
	}

    // Don't remind if last reminder was very recent (to avoid duplicate runs within seconds)
    // But since we update last_reminded_at immediately, this should be fine.

	var duration time.Duration
	switch p.ReminderFrequency {
	case "daily":
		duration = 24 * time.Hour
	case "weekly":
		duration = 7 * 24 * time.Hour
	case "monthly":
		duration = 30 * 24 * time.Hour
	default:
		return false // Unknown frequency
	}

	return time.Since(lastReminded) >= duration
}

func sendReminder(db *sql.DB, p models.Promise) error {
	// 1. Send Notification Logic
	// In a real implementation with webpush-go, we would fetch subscriptions for p.UserID and send.
	// For now, we log it.
	log.Printf("Use webpush-go to SEND PUSH NOTIFICATION for promise %d to user %d: Remember your promise to %s: %s", 
        p.ID, p.UserID, p.Recipient, p.Description)

	// 2. Update last_reminded_at
	_, err := db.Exec("UPDATE promises SET last_reminded_at = CURRENT_TIMESTAMP WHERE id = ?", p.ID)
	return err
}
