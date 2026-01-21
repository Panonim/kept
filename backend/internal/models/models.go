package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Promise struct {
	ID                int        `json:"id"`
	UserID            int        `json:"user_id"`
	Recipient         string     `json:"recipient"`
	Description       string     `json:"description"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	CurrentState      string     `json:"current_state"`
	ReminderFrequency string     `json:"reminder_frequency,omitempty"`
	LastRemindedAt    *time.Time `json:"last_reminded_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Events            []Event    `json:"events,omitempty"`
}

type Event struct {
	ID             int       `json:"id"`
	PromiseID      int       `json:"promise_id"`
	State          string    `json:"state"`
	ReflectionNote string    `json:"reflection_note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type Reminder struct {
	ID            int       `json:"id"`
	PromiseID     int       `json:"promise_id"`
	UserID        int       `json:"user_id"`
	RemindAt      time.Time `json:"remind_at"`
	OffsetMinutes int       `json:"offset_minutes"`
	IsSent        bool      `json:"is_sent"`
	CreatedAt     time.Time `json:"created_at"`
}

type PushSubscription struct {
	ID       int    `json:"id"`
	UserID   int    `json:"user_id"`
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

type CreatePromiseRequest struct {
	Recipient         string     `json:"recipient"`
	Description       string     `json:"description"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	ReminderFrequency string     `json:"reminder_frequency,omitempty"`
}

type UpdatePromiseStateRequest struct {
	State          string     `json:"state"`
	ReflectionNote string     `json:"reflection_note,omitempty"`
	NewDueDate     *time.Time `json:"new_due_date,omitempty"`
}

type UpdatePromiseRequest struct {
	ReminderFrequency *int       `json:"reminder_frequency,omitempty"`
	DueDate           *time.Time `json:"due_date,omitempty"`
}

type CreateReminderRequest struct {
	OffsetMinutes int `json:"offset_minutes"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Remember bool   `json:"remember,omitempty"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
