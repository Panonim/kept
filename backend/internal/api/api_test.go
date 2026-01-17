package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"kept/internal/api"
	"kept/internal/database"
	"kept/internal/models"

	"github.com/gofiber/fiber/v2"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func setupTestApp(db *sql.DB) *fiber.App {
	app := fiber.New()
	api.SetupRoutes(app, db)
	return app
}

func TestRegisterAndLogin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := setupTestApp(db)

	// Test registration
	registerReq := models.RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	var authResp models.AuthResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &authResp)

	if authResp.Token == "" {
		t.Fatal("Expected token in response")
	}
	if authResp.User.Username != registerReq.Username {
		t.Fatalf("Expected username %s, got %s", registerReq.Username, authResp.User.Username)
	}

	// Test login
	loginReq := models.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var loginResp models.AuthResponse
	bodyBytes, _ = io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &loginResp)

	if loginResp.Token == "" {
		t.Fatal("Expected token in response")
	}
}

func TestCreatePromise(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := setupTestApp(db)

	// Register user first
	registerReq := models.RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	var authResp models.AuthResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &authResp)
	token := authResp.Token

	// Create promise
	promiseReq := models.CreatePromiseRequest{
		Recipient:   "John",
		Description: "Help with project",
	}
	body, _ = json.Marshal(promiseReq)
	req = httptest.NewRequest("POST", "/api/promises/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var promise models.Promise
	bodyBytes, _ = io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &promise)

	if promise.Recipient != promiseReq.Recipient {
		t.Fatalf("Expected recipient %s, got %s", promiseReq.Recipient, promise.Recipient)
	}
	if promise.CurrentState != "active" {
		t.Fatalf("Expected state active, got %s", promise.CurrentState)
	}
}

func TestListPromises(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := setupTestApp(db)

	// Register and get token
	registerReq := models.RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	var authResp models.AuthResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &authResp)
	token := authResp.Token

	// Create two promises
	for i := 0; i < 2; i++ {
		promiseReq := models.CreatePromiseRequest{
			Recipient:   "John",
			Description: "Promise " + string(rune(i+'1')),
		}
		body, _ := json.Marshal(promiseReq)
		req := httptest.NewRequest("POST", "/api/promises/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		app.Test(req)
	}

	// List promises
	req = httptest.NewRequest("GET", "/api/promises/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var promises []models.Promise
	bodyBytes, _ = io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &promises)

	if len(promises) != 2 {
		t.Fatalf("Expected 2 promises, got %d", len(promises))
	}
}

func TestUpdatePromiseState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := setupTestApp(db)

	// Register and get token
	registerReq := models.RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	var authResp models.AuthResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &authResp)
	token := authResp.Token

	// Create promise
	promiseReq := models.CreatePromiseRequest{
		Recipient:   "John",
		Description: "Test promise",
	}
	body, _ = json.Marshal(promiseReq)
	req = httptest.NewRequest("POST", "/api/promises/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ = app.Test(req)

	var promise models.Promise
	bodyBytes, _ = io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &promise)

	// Update state to kept
	updateReq := models.UpdatePromiseStateRequest{
		State:          "kept",
		ReflectionNote: "It went well",
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/api/promises/"+string(rune(promise.ID+'0'))+"/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestMigratePostponedToKept(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Create a user to own the promise
    _, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", "migrator", "x")
    if err != nil {
        t.Fatal(err)
    }
    var userID int
    if err := db.QueryRow("SELECT id FROM users WHERE username = ?", "migrator").Scan(&userID); err != nil {
        t.Fatal(err)
    }

    // Insert a promise with state 'postponed' and an event recording 'postponed'
    res, err := db.Exec("INSERT INTO promises (user_id, recipient, description, due_date, current_state) VALUES (?, ?, ?, ?, 'postponed')", userID, "Alice", "desc", nil)
    if err != nil {
        t.Fatal(err)
    }
    pid64, _ := res.LastInsertId()
    pid := int(pid64)

    if _, err := db.Exec("INSERT INTO promise_events (promise_id, state, reflection_note) VALUES (?, 'postponed', ?)", pid, "postponed before migration"); err != nil {
        t.Fatal(err)
    }

    // Run migration
    if err := api.MigratePostponedToKept(db); err != nil {
        t.Fatal(err)
    }

    // Verify promise state has been updated
    var state string
    if err := db.QueryRow("SELECT current_state FROM promises WHERE id = ?", pid).Scan(&state); err != nil {
        t.Fatal(err)
    }
    if state != "kept" {
        t.Fatalf("Expected promise state 'kept', got '%s'", state)
    }

    // Verify events were normalized to 'kept'
    var evState string
    if err := db.QueryRow("SELECT state FROM promise_events WHERE promise_id = ? LIMIT 1", pid).Scan(&evState); err != nil {
        t.Fatal(err)
    }
    if evState != "kept" {
        t.Fatalf("Expected event state 'kept', got '%s'", evState)
    }
}

func TestAutoKeepOverduePromises(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Create a user to own the promise
    _, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", "autoer", "x")
    if err != nil {
        t.Fatal(err)
    }
    var userID int
    if err := db.QueryRow("SELECT id FROM users WHERE username = ?", "autoer").Scan(&userID); err != nil {
        t.Fatal(err)
    }

    // Insert an active promise with a due_date in the past
    res, err := db.Exec("INSERT INTO promises (user_id, recipient, description, due_date, current_state) VALUES (?, ?, ?, datetime('now', '-1 day'), 'active')", userID, "Bob", "do something")
    if err != nil {
        t.Fatal(err)
    }
    pid64, _ := res.LastInsertId()
    pid := int(pid64)

    // Run the auto-keep function
    if err := api.AutoKeepOverduePromises(db); err != nil {
        t.Fatal(err)
    }

    // Verify promise state has been updated
    var state string
    if err := db.QueryRow("SELECT current_state FROM promises WHERE id = ?", pid).Scan(&state); err != nil {
        t.Fatal(err)
    }
    if state != "kept" {
        t.Fatalf("Expected promise state 'kept', got '%s'", state)
    }

    // Verify an auto-kept event was inserted
    var note string
    if err := db.QueryRow("SELECT reflection_note FROM promise_events WHERE promise_id = ? AND state = 'kept' ORDER BY created_at DESC LIMIT 1", pid).Scan(&note); err != nil {
        t.Fatal(err)
    }
    if note != "Auto-kept: due date passed" {
        t.Fatalf("Expected auto-kept reflection note, got '%s'", note)
    }
}
