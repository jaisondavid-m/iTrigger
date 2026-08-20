package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserInfo struct {
	Username         string            `json:"username"`
	IsAdmin          bool              `json:"isAdmin"`
	CanCreateProject bool              `json:"canCreateProject"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	Permissions      map[string]string `json:"permissions"` // projectId -> permission
}

type UserHandler struct {
	db           *sql.DB
	sessionStore *SessionStore
}

func NewUserHandler(db *sql.DB, store *SessionStore) *UserHandler {
	return &UserHandler{
		db:           db,
		sessionStore: store,
	}
}

// GET /api/users
func (uh *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	// Query user permissions first
	rowsPerm, err := uh.db.Query("SELECT username, project_id, permission FROM user_project_permissions")
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rowsPerm.Close()

	userPerms := make(map[string]map[string]string)
	for rowsPerm.Next() {
		var username, projectID, perm string
		if err := rowsPerm.Scan(&username, &projectID, &perm); err == nil {
			if _, exists := userPerms[username]; !exists {
				userPerms[username] = make(map[string]string)
			}
			userPerms[username][projectID] = perm
		}
	}

	// Query all users
	rows, err := uh.db.Query("SELECT username, is_admin, can_create_project, created_at, updated_at FROM users ORDER BY username ASC")
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		var isAdmin, canCreateProject int
		if err := rows.Scan(&u.Username, &isAdmin, &canCreateProject, &u.CreatedAt, &u.UpdatedAt); err == nil {
			u.IsAdmin = (isAdmin == 1)
			u.CanCreateProject = (canCreateProject == 1)
			if perms, ok := userPerms[u.Username]; ok {
				u.Permissions = perms
			} else {
				u.Permissions = make(map[string]string)
			}
			users = append(users, u)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

// POST /api/users
func (uh *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username         string            `json:"username"`
		Password         string            `json:"password"`
		IsAdmin          bool              `json:"isAdmin"`
		CanCreateProject bool              `json:"canCreateProject"`
		Permissions      map[string]string `json:"permissions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password

	if username == "" || password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	var exists int
	err := uh.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "hashing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tx, err := uh.db.Begin()
	if err != nil {
		http.Error(w, "failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	isAdminVal := 0
	if req.IsAdmin {
		isAdminVal = 1
	}

	canCreateVal := 0
	if req.CanCreateProject {
		canCreateVal = 1
	}

	now := time.Now()
	_, err = tx.Exec(`
		INSERT INTO users (username, password_hash, is_admin, can_create_project, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, username, string(hashed), isAdminVal, canCreateVal, now, now)
	if err != nil {
		http.Error(w, "failed to insert user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Only insert permissions if not admin
	if !req.IsAdmin {
		for projectID, perm := range req.Permissions {
			if perm == "read" || perm == "write" {
				_, err = tx.Exec(`
					INSERT INTO user_project_permissions (username, project_id, permission)
					VALUES (?, ?, ?)
				`, username, projectID, perm)
				if err != nil {
					http.Error(w, "failed to save permission: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

// PUT /api/users/{username}
func (uh *UserHandler) Update(w http.ResponseWriter, r *http.Request, username string) {
	var req struct {
		Password         string            `json:"password"` // optional
		IsAdmin          bool              `json:"isAdmin"`
		CanCreateProject bool              `json:"canCreateProject"`
		Permissions      map[string]string `json:"permissions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := uh.db.Begin()
	if err != nil {
		http.Error(w, "failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	isAdminVal := 0
	if req.IsAdmin {
		isAdminVal = 1
	}

	canCreateVal := 0
	if req.CanCreateProject {
		canCreateVal = 1
	}

	now := time.Now()

	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "hashing error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec(`
			UPDATE users
			SET password_hash = ?, is_admin = ?, can_create_project = ?, updated_at = ?
			WHERE username = ?
		`, string(hashed), isAdminVal, canCreateVal, now, username)
		if err != nil {
			http.Error(w, "failed to update user: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err = tx.Exec(`
			UPDATE users
			SET is_admin = ?, can_create_project = ?, updated_at = ?
			WHERE username = ?
		`, isAdminVal, canCreateVal, now, username)
		if err != nil {
			http.Error(w, "failed to update user: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Recreate permissions: delete first
	_, err = tx.Exec("DELETE FROM user_project_permissions WHERE username = ?", username)
	if err != nil {
		http.Error(w, "failed to clean permissions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert new permissions if not admin
	if !req.IsAdmin {
		for projectID, perm := range req.Permissions {
			if perm == "read" || perm == "write" {
				_, err = tx.Exec(`
					INSERT INTO user_project_permissions (username, project_id, permission)
					VALUES (?, ?, ?)
				`, username, projectID, perm)
				if err != nil {
					http.Error(w, "failed to save permission: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

// DELETE /api/users/{username}
func (uh *UserHandler) Delete(w http.ResponseWriter, r *http.Request, username string) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	caller, ok := uh.sessionStore.Get(cookie.Value)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if caller == username {
		http.Error(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	if username == "itrigger" {
		http.Error(w, "cannot delete the default administrator account", http.StatusBadRequest)
		return
	}

	tx, err := uh.db.Begin()
	if err != nil {
		http.Error(w, "failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_project_permissions WHERE username = ?", username)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}
