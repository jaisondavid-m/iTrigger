package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"iTrigger/internal/models"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

// InitDB initializes SQLite connection, creates tables/indexes, and runs migrations.
func InitDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = filepath.Join("data", "itrigger.db")
	}

	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Enable WAL mode & foreign keys for optimal performance and safety
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		log.Printf("[DB] Pragma warning: %v", err)
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	// Initialize default user if users table is empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	if count == 0 {
		hashed, err := bcrypt.GenerateFromPassword([]byte("itrigger"), bcrypt.DefaultCost)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to hash default password: %w", err)
		}
		now := time.Now()
		_, err = db.Exec(`
			INSERT INTO users (username, password_hash, created_at, updated_at)
			VALUES (?, ?, ?, ?)
		`, "itrigger", string(hashed), now, now)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to insert default user: %w", err)
		}
		log.Println("[DB] Successfully initialized default user 'itrigger' with password 'itrigger'")
	}

	if err := migrateLegacyJSON(db, dataDir); err != nil {
		log.Printf("[DB] Migration warning: %v", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		repository TEXT NOT NULL,
		branch TEXT NOT NULL DEFAULT 'main',
		project_path TEXT NOT NULL DEFAULT '.',
		script TEXT NOT NULL DEFAULT '',
		secret TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		project_name TEXT NOT NULL,
		repository TEXT NOT NULL,
		branch TEXT NOT NULL,
		commit_sha TEXT NOT NULL DEFAULT '',
		commit_message TEXT NOT NULL DEFAULT '',
		triggered_by TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'QUEUED',
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		log_path TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS webhooks (
		delivery_id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,
		repository_name TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL DEFAULT '',
		pr_number INTEGER NOT NULL DEFAULT 0,
		pr_title TEXT NOT NULL DEFAULT '',
		sender TEXT NOT NULL DEFAULT '',
		received_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_deployments_project_id ON deployments(project_id);
	CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
	CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at);
	CREATE INDEX IF NOT EXISTS idx_webhooks_received_at ON webhooks(received_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func migrateLegacyJSON(db *sql.DB, dataDir string) error {
	projectsFile := filepath.Join(dataDir, "projects.json")
	deploymentsFile := filepath.Join(dataDir, "deployments.json")

	// 1. Migrate projects.json
	if data, err := os.ReadFile(projectsFile); err == nil && len(data) > 0 {
		var list []models.ProjectConfig
		if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
			tx, err := db.Begin()
			if err == nil {
				stmt, _ := tx.Prepare(`
					INSERT OR REPLACE INTO projects (id, name, repository, branch, project_path, script, secret, enabled, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`)
				if stmt != nil {
					for _, p := range list {
						enabledInt := 0
						if p.Enabled {
							enabledInt = 1
						}
						_, _ = stmt.Exec(p.ID, p.Name, p.Repository, p.Branch, p.ProjectPath, p.Script, p.Secret, enabledInt, p.CreatedAt, p.UpdatedAt)
					}
					stmt.Close()
				}
				_ = tx.Commit()
				log.Printf("[DB] Successfully migrated %d projects from %s to SQLite", len(list), projectsFile)
			}
		}
		_ = os.Rename(projectsFile, projectsFile+".migrated")
	}

	// 2. Migrate deployments.json
	if data, err := os.ReadFile(deploymentsFile); err == nil && len(data) > 0 {
		var list []models.DeploymentLog
		if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
			tx, err := db.Begin()
			if err == nil {
				stmt, _ := tx.Prepare(`
					INSERT OR REPLACE INTO deployments (id, project_id, project_name, repository, branch, commit_sha, commit_message, triggered_by, status, started_at, completed_at, duration_ms, log_path, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`)
				if stmt != nil {
					for _, d := range list {
						depDir := filepath.Join(dataDir, "deployments", d.ID)
						_ = os.MkdirAll(depDir, 0755)

						// Save console log
						logFile := filepath.Join(depDir, "console.log")
						_ = os.WriteFile(logFile, []byte(d.Log), 0644)

						// Save metadata.json
						metaFile := filepath.Join(depDir, "metadata.json")
						metaBytes, _ := json.MarshalIndent(d, "", "  ")
						_ = os.WriteFile(metaFile, metaBytes, 0644)

						createdAt := d.StartedAt
						if createdAt.IsZero() {
							createdAt = time.Now()
						}

						var completedAt *time.Time
						if !d.CompletedAt.IsZero() {
							completedAt = &d.CompletedAt
						}

						_, _ = stmt.Exec(d.ID, d.ProjectID, d.ProjectName, d.Repository, d.Branch, d.CommitSHA, d.CommitMessage, d.TriggeredBy, d.Status, d.StartedAt, completedAt, d.DurationMs, logFile, createdAt)
					}
					stmt.Close()
				}
				_ = tx.Commit()
				log.Printf("[DB] Successfully migrated %d deployment logs from %s to SQLite & file storage", len(list), deploymentsFile)
			}
		}
		_ = os.Rename(deploymentsFile, deploymentsFile+".migrated")
	}

	return nil
}
