package routes

import (
	"database/sql"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"iTrigger/internal/auth"
	"iTrigger/internal/backup"
	"iTrigger/internal/db"
	"iTrigger/internal/deployer"
	"iTrigger/internal/models"
	"iTrigger/internal/store"
	"iTrigger/internal/webhook"
)

// Register declares and maps all application HTTP routes onto the provided ServeMux.
func Register(mux *http.ServeMux, secret string, webFS fs.FS) {
	// Initialize SQLite Database
	database, err := db.InitDB("data/itrigger.db")
	if err != nil {
		log.Fatalf("failed to initialize SQLite database: %v", err)
	}

	// Initialize stores using persistent SQLite DB
	webhookStore := webhook.NewStore(database)
	projectStore := store.NewProjectStore(database)
	deploymentStore := store.NewDeploymentStore(database)

	runner := deployer.NewRunner(projectStore, deploymentStore)

	webhookHandler := webhook.New(secret, webhookStore)
	webhookHandler.SetDeployer(projectStore, runner)

	// Initialize Auth Module
	sessionStore := auth.NewSessionStore()
	authHandler := auth.NewAuthHandler(database, sessionStore)

	// Authentication API endpoints
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/auth/status", authHandler.Status)
	mux.HandleFunc("/api/auth/settings", authHandler.UpdateSettings)

	// User Management API endpoints (Admin only)
	userHandler := auth.NewUserHandler(database, sessionStore)
	mux.HandleFunc("/api/users", RequireAdmin(sessionStore, database, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.List(w, r)
		case http.MethodPost:
			userHandler.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/users/", RequireAdmin(sessionStore, database, func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimPrefix(r.URL.Path, "/api/users/")
		username = strings.Trim(username, "/")
		if username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut, http.MethodPost:
			userHandler.Update(w, r, username)
		case http.MethodDelete:
			userHandler.Delete(w, r, username)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Health check endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)

	// Webhook API endpoints
	mux.Handle("/api/webhooks/github", webhookHandler)
	mux.HandleFunc("/api/webhooks", RequireAdmin(sessionStore, database, webhookHandler.GetEventsHandler))
	mux.HandleFunc("/api/webhooks/clear", RequireAdmin(sessionStore, database, webhookHandler.ClearEventsHandler))

	// Backup endpoint
	mux.HandleFunc("/api/backups", RequireAdmin(sessionStore, database, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		backupPath, err := backup.CreateBackup("data")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"status": "success",
			"path":   backupPath,
		})
	}))

	// Server Filesystem Directory Browser API
	mux.HandleFunc("/api/fs/browse", RequireAdmin(sessionStore, database, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
		if targetPath == "" {
			var err error
			targetPath, err = os.Getwd()
			if err != nil {
				targetPath = "."
			}
		}

		// Resolve absolute clean path
		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			absPath = targetPath
		}
		absPath = filepath.Clean(absPath)

		info, err := os.Stat(absPath)
		if err != nil || !info.IsDir() {
			parentDir := filepath.Dir(absPath)
			if pInfo, pErr := os.Stat(parentDir); pErr == nil && pInfo.IsDir() {
				absPath = parentDir
			} else {
				absPath, _ = os.Getwd()
			}
		}

		entries, err := os.ReadDir(absPath)
		if err != nil {
			http.Error(w, "failed to read directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		type FolderEntry struct {
			Name             string    `json:"name"`
			Path             string    `json:"path"`
			IsDir            bool      `json:"isDir"`
			IsRepo           bool      `json:"isRepo"`
			HasTriggerScript bool      `json:"hasTriggerScript"`
			ModTime          time.Time `json:"modTime"`
		}

		type FileEntry struct {
			Name    string    `json:"name"`
			Path    string    `json:"path"`
			Size    int64     `json:"size"`
			ModTime time.Time `json:"modTime"`
		}

		folders := make([]FolderEntry, 0)
		files := make([]FileEntry, 0)

		currentIsRepo := false
		currentHasTriggerScript := false

		for _, entry := range entries {
			eName := entry.Name()
			ePath := filepath.Join(absPath, eName)

			if eName == ".git" {
				currentIsRepo = true
			}
			if eName == ".itrigger" || eName == ".itrigger.sh" || eName == "itrigger.sh" {
				currentHasTriggerScript = true
			}

			eInfo, err := entry.Info()
			var modTime time.Time
			var size int64
			if err == nil {
				modTime = eInfo.ModTime()
				size = eInfo.Size()
			}

			if entry.IsDir() {
				subGit := false
				subScript := false
				if _, err := os.Stat(filepath.Join(ePath, ".git")); err == nil {
					subGit = true
				}
				if _, err := os.Stat(filepath.Join(ePath, ".itrigger")); err == nil {
					subScript = true
				} else if _, err := os.Stat(filepath.Join(ePath, ".itrigger.sh")); err == nil {
					subScript = true
				} else if _, err := os.Stat(filepath.Join(ePath, "itrigger.sh")); err == nil {
					subScript = true
				}

				folders = append(folders, FolderEntry{
					Name:             eName,
					Path:             filepath.ToSlash(ePath),
					IsDir:            true,
					IsRepo:           subGit,
					HasTriggerScript: subScript,
					ModTime:          modTime,
				})
			} else {
				files = append(files, FileEntry{
					Name:    eName,
					Path:    filepath.ToSlash(ePath),
					Size:    size,
					ModTime: modTime,
				})
			}
		}

		sort.Slice(folders, func(i, j int) bool {
			return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
		})
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})

		parentPath := filepath.Dir(absPath)
		if parentPath == absPath {
			parentPath = ""
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":           "success",
			"currentPath":      filepath.ToSlash(absPath),
			"parentPath":       filepath.ToSlash(parentPath),
			"isRepo":           currentIsRepo,
			"hasTriggerScript": currentHasTriggerScript,
			"folders":          folders,
			"files":            files,
		})
	}))

	// Project Management API endpoints
	mux.HandleFunc("/api/projects", RequireAuth(sessionStore, func(w http.ResponseWriter, r *http.Request) {
		username, _ := getUsername(r, sessionStore)
		var isAdmin int
		_ = database.QueryRow("SELECT is_admin FROM users WHERE username = ?", username).Scan(&isAdmin)

		switch r.Method {
		case http.MethodGet:
			projects := projectStore.GetAll()
			var filtered []models.ProjectConfig

			if isAdmin == 1 {
				for _, p := range projects {
					p.UserPermission = "write"
					filtered = append(filtered, p)
				}
			} else {
				rowsPerm, err := database.Query("SELECT project_id, permission FROM user_project_permissions WHERE username = ?", username)
				if err != nil {
					http.Error(w, "database error", http.StatusInternalServerError)
					return
				}
				defer rowsPerm.Close()

				userPerms := make(map[string]string)
				for rowsPerm.Next() {
					var pID, perm string
					if err := rowsPerm.Scan(&pID, &perm); err == nil {
						userPerms[pID] = perm
					}
				}

				for _, p := range projects {
					if perm, exists := userPerms[p.ID]; exists {
						p.UserPermission = perm
						filtered = append(filtered, p)
					}
				}
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"status":   "success",
				"projects": filtered,
			})
		case http.MethodPost:
			var canCreate int
			err = database.QueryRow("SELECT can_create_project FROM users WHERE username = ?", username).Scan(&canCreate)
			if err != nil || (isAdmin != 1 && canCreate != 1) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			var req models.CreateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			proj, err := projectStore.Save(req, "")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Non-admin creators get delete permission by default
			if isAdmin != 1 {
				_, _ = database.Exec("INSERT OR REPLACE INTO user_project_permissions (username, project_id, permission) VALUES (?, ?, 'delete')", username, proj.ID)
			}

			writeJSON(w, http.StatusCreated, map[string]any{
				"status":  "success",
				"project": proj,
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/projects/", RequireAuth(sessionStore, func(w http.ResponseWriter, r *http.Request) {
		username, _ := getUsername(r, sessionStore)
		var isAdmin int
		_ = database.QueryRow("SELECT is_admin FROM users WHERE username = ?", username).Scan(&isAdmin)

		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "project ID required", http.StatusBadRequest)
			return
		}

		projectID := parts[0]

		// Handle POST /api/projects/{id}/deploy
		if len(parts) == 2 && parts[1] == "deploy" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			proj, ok := projectStore.Get(projectID)
			if !ok {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}

			// Validate write/delete permission
			if isAdmin != 1 {
				var perm string
				err := database.QueryRow("SELECT permission FROM user_project_permissions WHERE username = ? AND project_id = ?", username, projectID).Scan(&perm)
				if err != nil || (perm != "write" && perm != "delete") {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			depLog := runner.TriggerDeployment(proj, "manual:ui", "HEAD", "Manual trigger from UI")
			writeJSON(w, http.StatusOK, map[string]any{
				"status":     "triggered",
				"deployment": depLog,
			})
			return
		}

		// Handle single project endpoints /api/projects/{id}
		switch r.Method {
		case http.MethodGet:
			proj, ok := projectStore.Get(projectID)
			if !ok {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}

			// Validate read/write permission
			var userPerm string
			if isAdmin == 1 {
				userPerm = "write"
			} else {
				err := database.QueryRow("SELECT permission FROM user_project_permissions WHERE username = ? AND project_id = ?", username, projectID).Scan(&userPerm)
				if err != nil {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
			proj.UserPermission = userPerm

			writeJSON(w, http.StatusOK, map[string]any{
				"status":  "success",
				"project": proj,
			})
		case http.MethodPut, http.MethodPost:
			// Validate write/delete permission
			if isAdmin != 1 {
				var perm string
				err := database.QueryRow("SELECT permission FROM user_project_permissions WHERE username = ? AND project_id = ?", username, projectID).Scan(&perm)
				if err != nil || (perm != "write" && perm != "delete") {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			var req models.CreateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			proj, err := projectStore.Save(req, projectID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status":  "success",
				"project": proj,
			})
		case http.MethodDelete:
			// Validate delete permission
			if isAdmin != 1 {
				var perm string
				err := database.QueryRow("SELECT permission FROM user_project_permissions WHERE username = ? AND project_id = ?", username, projectID).Scan(&perm)
				if err != nil || perm != "delete" {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			if !projectStore.Delete(projectID) {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"status": "deleted",
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Deployment Logs API endpoints
	mux.HandleFunc("/api/deployments", RequireAuth(sessionStore, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		username, _ := getUsername(r, sessionStore)
		var isAdmin int
		_ = database.QueryRow("SELECT is_admin FROM users WHERE username = ?", username).Scan(&isAdmin)

		allDeployments := deploymentStore.GetAll()
		var filtered []models.DeploymentLog

		if isAdmin == 1 {
			filtered = allDeployments
		} else {
			// Query projects this user has read or write permissions for
			rowsPerm, err := database.Query("SELECT project_id FROM user_project_permissions WHERE username = ?", username)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			defer rowsPerm.Close()

			userProjects := make(map[string]bool)
			for rowsPerm.Next() {
				var pID string
				if err := rowsPerm.Scan(&pID); err == nil {
					userProjects[pID] = true
				}
			}

			for _, d := range allDeployments {
				if userProjects[d.ProjectID] {
					filtered = append(filtered, d)
				}
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"count":       len(filtered),
			"deployments": filtered,
		})
	}))

	mux.HandleFunc("/api/deployments/", RequireAuth(sessionStore, func(w http.ResponseWriter, r *http.Request) {
		subPath := strings.TrimPrefix(r.URL.Path, "/api/deployments/")
		parts := strings.Split(strings.Trim(subPath, "/"), "/")

		// Validate project access
		depID := parts[0]
		depLog, ok := deploymentStore.Get(depID)
		if !ok {
			http.Error(w, "deployment log not found", http.StatusNotFound)
			return
		}

		username, _ := getUsername(r, sessionStore)
		var isAdmin int
		_ = database.QueryRow("SELECT is_admin FROM users WHERE username = ?", username).Scan(&isAdmin)
		if isAdmin != 1 {
			var perm string
			err := database.QueryRow("SELECT permission FROM user_project_permissions WHERE username = ? AND project_id = ?", username, depLog.ProjectID).Scan(&perm)
			if err != nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		// Handle POST /api/deployments/{id}/stop
		if len(parts) == 2 && parts[1] == "stop" && r.Method == http.MethodPost {
			// Require write/delete permission to stop
			if isAdmin != 1 {
				var perm string
				err := database.QueryRow("SELECT permission FROM user_project_permissions WHERE username = ? AND project_id = ?", username, depLog.ProjectID).Scan(&perm)
				if err != nil || (perm != "write" && perm != "delete") {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			if err := runner.StopDeployment(depID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{
				"status": "stopped",
			})
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle /api/deployments/{id}/logs or /api/deployments/{id}/log
		if len(parts) == 2 && (parts[1] == "logs" || parts[1] == "log") {
			reader, err := deploymentStore.GetLogStreamReader(depID)
			if err != nil {
				http.Error(w, "log file not found", http.StatusNotFound)
				return
			}
			defer reader.Close()

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = io.Copy(w, reader)
			return
		}

		// Handle /api/deployments/{id}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "success",
			"deployment": depLog,
		})
	}))

	// Static web interface
	if webFS != nil {
		subFS, err := fs.Sub(webFS, "web")
		if err != nil {
			log.Fatalf("failed to sub embed filesystem: %v", err)
		}
		fileServer := http.FileServer(http.FS(subFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := strings.ToLower(r.URL.Path)
			switch {
			case strings.HasSuffix(path, ".css"):
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			case strings.HasSuffix(path, ".js"):
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			case strings.HasSuffix(path, ".html") || path == "/" || path == "":
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			case strings.HasSuffix(path, ".json"):
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
			case strings.HasSuffix(path, ".svg"):
				w.Header().Set("Content-Type", "image/svg+xml")
			}
			fileServer.ServeHTTP(w, r)
		})
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(models.HealthResponse{Status: "ok"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func RequireAuth(sessionStore *auth.SessionStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, ok := sessionStore.Get(cookie.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func getUsername(r *http.Request, sessionStore *auth.SessionStore) (string, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", false
	}
	return sessionStore.Get(cookie.Value)
}

func RequireAdmin(sessionStore *auth.SessionStore, db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		username, ok := sessionStore.Get(cookie.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var isAdmin int
		err = db.QueryRow("SELECT is_admin FROM users WHERE username = ?", username).Scan(&isAdmin)
		if err != nil || isAdmin != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
