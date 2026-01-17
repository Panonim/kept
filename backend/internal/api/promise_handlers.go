package api

import (
	"database/sql"
	"kept/internal/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func CreatePromiseHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		var req models.CreatePromiseRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		if req.Recipient == "" || req.Description == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Recipient and description are required")
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Insert promise
		result, err := tx.Exec(
			`INSERT INTO promises (user_id, recipient, description, due_date, current_state, reminder_frequency, last_reminded_at) 
			VALUES (?, ?, ?, ?, 'active', ?, ?)`,
			userID, req.Recipient, req.Description, req.DueDate, req.ReminderFrequency, nil,
		)
		if err != nil {
			return err
		}

		promiseID, _ := result.LastInsertId()

		// Create initial event
		_, err = tx.Exec(
			"INSERT INTO promise_events (promise_id, state) VALUES (?, 'active')",
			promiseID,
		)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		// Get the created promise
		var promise models.Promise
		err = db.QueryRow(
			`SELECT id, user_id, recipient, description, due_date, current_state, reminder_frequency, last_reminded_at, created_at, updated_at 
			FROM promises WHERE id = ?`,
			promiseID,
		).Scan(
			&promise.ID, &promise.UserID, &promise.Recipient, &promise.Description,
			&promise.DueDate, &promise.CurrentState, &promise.ReminderFrequency, &promise.LastRemindedAt, &promise.CreatedAt, &promise.UpdatedAt,
		)
		if err != nil {
			return err
		}

		return c.Status(fiber.StatusCreated).JSON(promise)
	}
}

func ListPromisesHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		state := c.Query("state")
		
		query := `SELECT id, user_id, recipient, description, due_date, current_state, reminder_frequency, last_reminded_at, created_at, updated_at 
				FROM promises WHERE user_id = ?`
		args := []interface{}{userID}

		if state != "" {
			query += " AND current_state = ?"
			args = append(args, state)
		}

		query += " ORDER BY (current_state = 'kept') DESC, created_at DESC"

		rows, err := db.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		promises := []models.Promise{}
		for rows.Next() {
			var p models.Promise
			err := rows.Scan(
				&p.ID, &p.UserID, &p.Recipient, &p.Description,
				&p.DueDate, &p.CurrentState, &p.ReminderFrequency, &p.LastRemindedAt, &p.CreatedAt, &p.UpdatedAt,
			)
			if err != nil {
				return err
			}
			promises = append(promises, p)
		}

		return c.JSON(promises)
	}
}

func GetPromiseHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)
		promiseID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid promise ID")
		}

		var promise models.Promise
		err = db.QueryRow(
			`SELECT id, user_id, recipient, description, due_date, current_state, reminder_frequency, last_reminded_at, created_at, updated_at 
			FROM promises WHERE id = ? AND user_id = ?`,
			promiseID, userID,
		).Scan(
			&promise.ID, &promise.UserID, &promise.Recipient, &promise.Description,
			&promise.DueDate, &promise.CurrentState, &promise.ReminderFrequency, &promise.LastRemindedAt, &promise.CreatedAt, &promise.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusNotFound, "Promise not found")
		}
		if err != nil {
			return err
		}

		// Get events
		rows, err := db.Query(
			`SELECT id, promise_id, state, reflection_note, created_at 
			FROM promise_events WHERE promise_id = ? ORDER BY created_at ASC`,
			promiseID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		events := []models.Event{}
		for rows.Next() {
			var e models.Event
			var note sql.NullString
			err := rows.Scan(&e.ID, &e.PromiseID, &e.State, &note, &e.CreatedAt)
			if err != nil {
				return err
			}
			if note.Valid {
				e.ReflectionNote = note.String
			}
			events = append(events, e)
		}

		promise.Events = events
		return c.JSON(promise)
	}
}

func UpdatePromiseStateHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)
		promiseID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid promise ID")
		}

		var req models.UpdatePromiseStateRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		validStates := map[string]bool{"active": true, "kept": true, "broken": true, "postponed": true}
		if !validStates[req.State] {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid state")
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Check ownership
		var currentUserID int
		err = tx.QueryRow("SELECT user_id FROM promises WHERE id = ?", promiseID).Scan(&currentUserID)
		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusNotFound, "Promise not found")
		}
		if err != nil {
			return err
		}
		if currentUserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized")
		}

		// If the client requested "postponed", permanently convert to "kept" in storage
		storedState := req.State
		if req.State == "postponed" {
			storedState = "kept"
		}

		// Update promise state (we do not apply postponed new due dates — postponed is converted to kept)
		_, err = tx.Exec(
			"UPDATE promises SET current_state = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			storedState, promiseID,
		)
		if err != nil {
			return err
		}

		// Create event (store the converted state)
		_, err = tx.Exec(
			"INSERT INTO promise_events (promise_id, state, reflection_note) VALUES (?, ?, ?)",
			promiseID, storedState, req.ReflectionNote,
		)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

func UpdatePromiseHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)
		promiseID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid promise ID")
		}

		var req models.UpdatePromiseRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Check ownership
		var currentUserID int
		err = db.QueryRow("SELECT user_id FROM promises WHERE id = ?", promiseID).Scan(&currentUserID)
		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusNotFound, "Promise not found")
		}
		if err != nil {
			return err
		}
		if currentUserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized")
		}

		// Convert reminder frequency to string for storage (null remains null)
		var reminderFreqStr *string
		if req.ReminderFrequency != nil {
			freqStr := strconv.Itoa(*req.ReminderFrequency)
			reminderFreqStr = &freqStr
		}

		// Update the promise fields
		_, err = db.Exec(
			"UPDATE promises SET reminder_frequency = ?, due_date = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			reminderFreqStr, req.DueDate, promiseID,
		)
		if err != nil {
			return err
		}

		// Get the updated promise
		var promise models.Promise
		err = db.QueryRow(
			`SELECT id, user_id, recipient, description, due_date, current_state, reminder_frequency, last_reminded_at, created_at, updated_at 
			FROM promises WHERE id = ?`,
			promiseID,
		).Scan(
			&promise.ID, &promise.UserID, &promise.Recipient, &promise.Description,
			&promise.DueDate, &promise.CurrentState, &promise.ReminderFrequency, &promise.LastRemindedAt, &promise.CreatedAt, &promise.UpdatedAt,
		)
		if err != nil {
			return err
		}

		return c.JSON(promise)
	}
}

func DeletePromiseHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)
		promiseID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid promise ID")
		}

		result, err := db.Exec("DELETE FROM promises WHERE id = ? AND user_id = ?", promiseID, userID)
		if err != nil {
			return err
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Promise not found")
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

func GetTimelineHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		query := `
			SELECT 
				p.id, p.user_id, p.recipient, p.description, p.due_date, 
				p.current_state, p.reminder_frequency, p.last_reminded_at, p.created_at, p.updated_at,
				pe.id as event_id, pe.state, pe.reflection_note, pe.created_at as event_created_at
			FROM promises p
			LEFT JOIN promise_events pe ON p.id = pe.promise_id
			WHERE p.user_id = ?
			ORDER BY pe.created_at DESC, p.created_at DESC
		`

		rows, err := db.Query(query, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		promiseMap := make(map[int]*models.Promise)
		var orderedPromises []int

		for rows.Next() {
			var p models.Promise
			var e models.Event
			var eventID sql.NullInt64
			var eventState sql.NullString
			var eventNote sql.NullString
			var eventCreatedAt sql.NullTime

			err := rows.Scan(
				&p.ID, &p.UserID, &p.Recipient, &p.Description, &p.DueDate,
				&p.CurrentState, &p.ReminderFrequency, &p.LastRemindedAt, &p.CreatedAt, &p.UpdatedAt,
				&eventID, &eventState, &eventNote, &eventCreatedAt,
			)
			if err != nil {
				return err
			}

			if _, exists := promiseMap[p.ID]; !exists {
				p.Events = []models.Event{}
				promiseMap[p.ID] = &p
				orderedPromises = append(orderedPromises, p.ID)
			}

			if eventID.Valid {
				e.ID = int(eventID.Int64)
				e.PromiseID = p.ID
				e.State = eventState.String
				if eventNote.Valid {
					e.ReflectionNote = eventNote.String
				}
				e.CreatedAt = eventCreatedAt.Time
				promiseMap[p.ID].Events = append(promiseMap[p.ID].Events, e)
			}
		}

		timeline := []models.Promise{}
		for _, id := range orderedPromises {
			timeline = append(timeline, *promiseMap[id])
		}

		return c.JSON(timeline)
	}
}
