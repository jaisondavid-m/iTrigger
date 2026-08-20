package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	schema := `
	CREATE TABLE users (
		username TEXT PRIMARY KEY,
		password_hash TEXT NOT NULL,
		is_admin INTEGER NOT NULL DEFAULT 0,
		can_create_project INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE user_project_permissions (
		username TEXT NOT NULL,
		project_id TEXT NOT NULL,
		permission TEXT NOT NULL,
		PRIMARY KEY (username, project_id),
		FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to initialize schema: %v", err)
	}
	return db
}

func TestUserManagementCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewSessionStore()
	handler := NewUserHandler(db, store)

	// Create caller admin session
	token := store.Create("admin_user")
	_, _ = db.Exec(`INSERT INTO users (username, password_hash, is_admin, can_create_project, created_at, updated_at) VALUES ('admin_user', 'hash', 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	// 1. Test Create User
	createUserJSON := `{
		"username": "developer1",
		"password": "password123",
		"isAdmin": false,
		"canCreateProject": true,
		"permissions": {
			"proj_1": "write",
			"proj_2": "read"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader([]byte(createUserJSON)))
	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify database insertion
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'developer1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user, found %d", count)
	}

	var permCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM user_project_permissions WHERE username = 'developer1'").Scan(&permCount)
	if permCount != 2 {
		t.Errorf("expected 2 project permissions, found %d", permCount)
	}

	// 2. Test List Users
	reqList := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	recList := httptest.NewRecorder()
	handler.List(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recList.Code)
	}

	var users []UserInfo
	if err := json.NewDecoder(recList.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	if len(users) != 2 { // admin_user and developer1
		t.Errorf("expected 2 users in list, got %d", len(users))
	}

	var devUser UserInfo
	for _, u := range users {
		if u.Username == "developer1" {
			devUser = u
			break
		}
	}

	if devUser.Username != "developer1" {
		t.Errorf("developer1 user not found in listing")
	}
	if devUser.IsAdmin {
		t.Errorf("developer1 should not be admin")
	}
	if !devUser.CanCreateProject {
		t.Errorf("developer1 should have canCreateProject enabled")
	}
	if devUser.Permissions["proj_1"] != "write" || devUser.Permissions["proj_2"] != "read" {
		t.Errorf("incorrect permissions for developer1: %v", devUser.Permissions)
	}

	// 3. Test Update User
	updateUserJSON := `{
		"isAdmin": false,
		"canCreateProject": false,
		"permissions": {
			"proj_1": "read"
		}
	}`

	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/users/developer1", bytes.NewReader([]byte(updateUserJSON)))
	recUpdate := httptest.NewRecorder()
	handler.Update(recUpdate, reqUpdate, "developer1")

	if recUpdate.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recUpdate.Code)
	}

	// Verify updates
	var canCreateVal int
	_ = db.QueryRow("SELECT can_create_project FROM users WHERE username = 'developer1'").Scan(&canCreateVal)
	if canCreateVal != 0 {
		t.Errorf("expected can_create_project to be updated to 0, got %d", canCreateVal)
	}

	var updatedPerm string
	_ = db.QueryRow("SELECT permission FROM user_project_permissions WHERE username = 'developer1' AND project_id = 'proj_1'").Scan(&updatedPerm)
	if updatedPerm != "read" {
		t.Errorf("expected permission to be updated to read, got %s", updatedPerm)
	}

	// Check that proj_2 permission was removed
	var proj2Count int
	_ = db.QueryRow("SELECT COUNT(*) FROM user_project_permissions WHERE username = 'developer1' AND project_id = 'proj_2'").Scan(&proj2Count)
	if proj2Count != 0 {
		t.Errorf("expected proj_2 permission to be deleted, found %d", proj2Count)
	}

	// 4. Test Delete User
	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/users/developer1", nil)
	reqDelete.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	recDelete := httptest.NewRecorder()
	handler.Delete(recDelete, reqDelete, "developer1")

	if recDelete.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recDelete.Code)
	}

	// Verify deletion
	_ = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'developer1'").Scan(&count)
	if count != 0 {
		t.Errorf("expected user developer1 to be deleted")
	}

	// Verify cascaded deletion of permissions
	_ = db.QueryRow("SELECT COUNT(*) FROM user_project_permissions WHERE username = 'developer1'").Scan(&permCount)
	if permCount != 0 {
		t.Errorf("expected user permissions to be deleted, got %d", permCount)
	}
}
