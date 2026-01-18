package api

import (
	"database/sql"
	"fmt"
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
			if err := sendRecurringReminder(db, p); err != nil {
				log.Printf("Failed to send reminder for promise %d: %v", p.ID, err)
			}
		}
	}
	return nil
}

// ProcessScheduledReminders checks for one-time reminders that are due
// and sends push notifications for them.
func ProcessScheduledReminders(db *sql.DB) error {
	query := `
		SELECT r.id, r.promise_id, r.user_id, r.remind_at, p.recipient, p.description
		FROM reminders r
		JOIN promises p ON r.promise_id = p.id
		WHERE r.is_sent = FALSE AND r.remind_at <= CURRENT_TIMESTAMP
	`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var reminderID, promiseID, userID int
		var remindAt time.Time
		var recipient, description string

		err := rows.Scan(&reminderID, &promiseID, &userID, &remindAt, &recipient, &description)
		if err != nil {
			log.Printf("Error scanning reminder: %v", err)
			continue
		}

		payload := PushPayload{
			Title: fmt.Sprintf("Reminder about your promise to: %s", recipient),
			Body:  description,
			Icon:  "/Static/logos/Kept Mascot Colored.svg",
			Badge: "/Static/logos/Kept Mascot Colored.svg",
			Tag:   fmt.Sprintf("kept-reminder-%d", reminderID),
			Data:  map[string]interface{}{"promise_id": promiseID, "reminder_id": reminderID},
		}

		if err := SendPushToUser(db, userID, payload); err != nil {
			log.Printf("Failed to send scheduled reminder %d: %v", reminderID, err)
			continue
		}

		// Mark reminder as sent
		_, err = db.Exec("UPDATE reminders SET is_sent = TRUE WHERE id = ?", reminderID)
		if err != nil {
			log.Printf("Failed to mark reminder %d as sent: %v", reminderID, err)
		} else {
			log.Printf("Sent scheduled reminder %d for promise %d to user %d", reminderID, promiseID, userID)
		}
	}
	return nil
}

func shouldRemind(p models.Promise) bool {
	lastReminded := p.CreatedAt
	if p.LastRemindedAt != nil {
		lastReminded = *p.LastRemindedAt
	}

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

func sendRecurringReminder(db *sql.DB, p models.Promise) error {
	payload := PushPayload{
		Title: fmt.Sprintf("Reminder about your promise to: %s", p.Recipient),
		Body:  p.Description,
		Icon:  "/Static/logos/Kept Mascot Colored.svg",
		Badge: "/Static/logos/Kept Mascot Colored.svg",
		Tag:   fmt.Sprintf("kept-recurring-%d", p.ID),
		Data:  map[string]interface{}{"promise_id": p.ID},
	}

	if err := SendPushToUser(db, p.UserID, payload); err != nil {
		return err
	}

	// Update last_reminded_at
	_, err := db.Exec("UPDATE promises SET last_reminded_at = CURRENT_TIMESTAMP WHERE id = ?", p.ID)
	if err != nil {
		return fmt.Errorf("failed to update last_reminded_at: %w", err)
	}

	log.Printf("Sent recurring reminder for promise %d to user %d", p.ID, p.UserID)
	return nil
}
