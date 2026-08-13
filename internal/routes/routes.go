package routes

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

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

	// Initialize stores using SQLite DB
	webhookStore := webhook.NewStore()
	projectStore := store.NewProjectStore(database)
	deploymentStore := store.NewDeploymentStore(database)

	runner := deployer.NewRunner(projectStore, deploymentStore)

	webhookHandler := webhook.New(secret, webhookStore)
	webhookHandler.SetDeployer(projectStore, runner)

	// Health check endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)

	// Webhook API endpoints
	mux.Handle("/api/webhooks/github", webhookHandler)
	mux.HandleFunc("/api/webhooks", webhookHandler.GetEventsHandler)
	mux.HandleFunc("/api/webhooks/clear", webhookHandler.ClearEventsHandler)

	// Backup endpoint
	mux.HandleFunc("/api/backups", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// Project Management API endpoints
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			projects := projectStore.GetAll()
			writeJSON(w, http.StatusOK, map[string]any{
				"status":   "success",
				"projects": projects,
			})
		case http.MethodPost:
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
			writeJSON(w, http.StatusCreated, map[string]any{
				"status":  "success",
				"project": proj,
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
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
			writeJSON(w, http.StatusOK, map[string]any{
				"status":  "success",
				"project": proj,
			})
		case http.MethodPut, http.MethodPost:
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
	})

	// Deployment Logs API endpoints
	mux.HandleFunc("/api/deployments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		deployments := deploymentStore.GetAll()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"count":       len(deployments),
			"deployments": deployments,
		})
	})

	mux.HandleFunc("/api/deployments/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		subPath := strings.TrimPrefix(r.URL.Path, "/api/deployments/")
		parts := strings.Split(strings.Trim(subPath, "/"), "/")

		// Handle /api/deployments/{id}/logs or /api/deployments/{id}/log
		if len(parts) == 2 && (parts[1] == "logs" || parts[1] == "log") {
			depID := parts[0]
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
		depID := parts[0]
		depLog, ok := deploymentStore.Get(depID)
		if !ok {
			http.Error(w, "deployment log not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "success",
			"deployment": depLog,
		})
	})

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
