package api

import (
	"database/sql"
	"kept/internal/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func CreateReminderHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)
		promiseID, err := strconv.Atoi(c.Params("promiseId"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid promise ID")
		}

		var req models.CreateReminderRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
		}

		// Get promise and check ownership
		var dueDate sql.NullTime
		var promiseUserID int
		err = db.QueryRow(
			"SELECT user_id, due_date FROM promises WHERE id = ?",
			promiseID,
		).Scan(&promiseUserID, &dueDate)

		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusNotFound, "Promise not found")
		}
		if err != nil {
			return err
		}
		if promiseUserID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Not authorized")
		}
		if !dueDate.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Promise has no due date")
		}

		// Calculate remind_at time
		remindAt := dueDate.Time.Add(time.Duration(-req.OffsetMinutes) * time.Minute)

		// Insert reminder
		result, err := db.Exec(
			`INSERT INTO reminders (promise_id, user_id, remind_at, offset_minutes) 
			VALUES (?, ?, ?, ?)`,
			promiseID, userID, remindAt, req.OffsetMinutes,
		)
		if err != nil {
			return err
		}

		reminderID, _ := result.LastInsertId()

		reminder := models.Reminder{
			ID:            int(reminderID),
			PromiseID:     promiseID,
			UserID:        userID,
			RemindAt:      remindAt,
			OffsetMinutes: req.OffsetMinutes,
			IsSent:        false,
		}

		return c.Status(fiber.StatusCreated).JSON(reminder)
	}
}

func ListRemindersHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)

		rows, err := db.Query(
			`SELECT id, promise_id, user_id, remind_at, offset_minutes, is_sent, created_at 
			FROM reminders WHERE user_id = ? ORDER BY remind_at ASC`,
			userID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		reminders := []models.Reminder{}
		for rows.Next() {
			var r models.Reminder
			err := rows.Scan(
				&r.ID, &r.PromiseID, &r.UserID, &r.RemindAt,
				&r.OffsetMinutes, &r.IsSent, &r.CreatedAt,
			)
			if err != nil {
				return err
			}
			reminders = append(reminders, r)
		}

		return c.JSON(reminders)
	}
}

func DeleteReminderHandler(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(int)
		reminderID, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid reminder ID")
		}

		result, err := db.Exec(
			"DELETE FROM reminders WHERE id = ? AND user_id = ?",
			reminderID, userID,
		)
		if err != nil {
			return err
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Reminder not found")
		}

		return c.JSON(fiber.Map{"success": true})
	}
}
