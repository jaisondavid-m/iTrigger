package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"iTrigger/internal/models"
)

type DeploymentStore struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewDeploymentStore(db *sql.DB) *DeploymentStore {
	return &DeploymentStore{
		db: db,
	}
}

func (ds *DeploymentStore) Add(logEntry models.DeploymentLog) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.saveToDBAndDisk(logEntry)
}

func (ds *DeploymentStore) Update(logEntry models.DeploymentLog) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.saveToDBAndDisk(logEntry)
}

func (ds *DeploymentStore) saveToDBAndDisk(logEntry models.DeploymentLog) {
	depDir := filepath.Join("data", "deployments", logEntry.ID)
	_ = os.MkdirAll(depDir, 0755)

	// Create artifacts directory inside deployment directory
	_ = os.MkdirAll(filepath.Join(depDir, "artifacts"), 0755)

	logFilePath := filepath.Join(depDir, "console.log")
	_ = os.WriteFile(logFilePath, []byte(logEntry.Log), 0644)

	metaFilePath := filepath.Join(depDir, "metadata.json")
	metaBytes, _ := json.MarshalIndent(logEntry, "", "  ")
	_ = os.WriteFile(metaFilePath, metaBytes, 0644)

	createdAt := logEntry.StartedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	var completedAt *time.Time
	if !logEntry.CompletedAt.IsZero() {
		completedAt = &logEntry.CompletedAt
	}

	_, _ = ds.db.Exec(`
		INSERT INTO deployments (id, project_id, project_name, repository, branch, commit_sha, commit_message, triggered_by, status, started_at, completed_at, duration_ms, log_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status,
			completed_at=excluded.completed_at,
			duration_ms=excluded.duration_ms,
			log_path=excluded.log_path
	`, logEntry.ID, logEntry.ProjectID, logEntry.ProjectName, logEntry.Repository, logEntry.Branch, logEntry.CommitSHA, logEntry.CommitMessage, logEntry.TriggeredBy, logEntry.Status, logEntry.StartedAt, completedAt, logEntry.DurationMs, logFilePath, createdAt)
}

func (ds *DeploymentStore) Get(id string) (models.DeploymentLog, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var d models.DeploymentLog
	var startedAt NullTime
	var completedAt NullTime
	var logPath string

	err := ds.db.QueryRow(`
		SELECT id, project_id, project_name, repository, branch, commit_sha, commit_message, triggered_by, status, started_at, completed_at, duration_ms, log_path
		FROM deployments WHERE id = ?
	`, id).Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.Repository, &d.Branch, &d.CommitSHA, &d.CommitMessage, &d.TriggeredBy, &d.Status, &startedAt, &completedAt, &d.DurationMs, &logPath)

	if err != nil {
		return models.DeploymentLog{}, false
	}

	if startedAt.Valid {
		d.StartedAt = startedAt.Time
	}

	if completedAt.Valid {
		d.CompletedAt = completedAt.Time
	}

	if logPath == "" {
		logPath = filepath.Join("data", "deployments", id, "console.log")
	}

	if logData, err := os.ReadFile(logPath); err == nil {
		d.Log = string(logData)
	}

	return d, true
}

func (ds *DeploymentStore) GetAll() []models.DeploymentLog {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	rows, err := ds.db.Query(`
		SELECT id, project_id, project_name, repository, branch, commit_sha, commit_message, triggered_by, status, started_at, completed_at, duration_ms, log_path
		FROM deployments
		ORDER BY created_at DESC
	`)
	if err != nil {
		return []models.DeploymentLog{}
	}
	defer rows.Close()

	var list []models.DeploymentLog
	for rows.Next() {
		var d models.DeploymentLog
		var startedAt NullTime
		var completedAt NullTime
		var logPath string
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.Repository, &d.Branch, &d.CommitSHA, &d.CommitMessage, &d.TriggeredBy, &d.Status, &startedAt, &completedAt, &d.DurationMs, &logPath); err == nil {
			if startedAt.Valid {
				d.StartedAt = startedAt.Time
			}
			if completedAt.Valid {
				d.CompletedAt = completedAt.Time
			}
			list = append(list, d)
		}
	}
	return list
}

func (ds *DeploymentStore) GetByProject(projectID string) []models.DeploymentLog {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	rows, err := ds.db.Query(`
		SELECT id, project_id, project_name, repository, branch, commit_sha, commit_message, triggered_by, status, started_at, completed_at, duration_ms, log_path
		FROM deployments
		WHERE project_id = ?
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return []models.DeploymentLog{}
	}
	defer rows.Close()

	var list []models.DeploymentLog
	for rows.Next() {
		var d models.DeploymentLog
		var startedAt NullTime
		var completedAt NullTime
		var logPath string
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.ProjectName, &d.Repository, &d.Branch, &d.CommitSHA, &d.CommitMessage, &d.TriggeredBy, &d.Status, &startedAt, &completedAt, &d.DurationMs, &logPath); err == nil {
			if startedAt.Valid {
				d.StartedAt = startedAt.Time
			}
			if completedAt.Valid {
				d.CompletedAt = completedAt.Time
			}
			list = append(list, d)
		}
	}
	return list
}

func (ds *DeploymentStore) GetLogStreamReader(id string) (io.ReadCloser, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	logPath := filepath.Join("data", "deployments", id, "console.log")
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("deployment log file not found: %w", err)
	}
	return file, nil
}

type NullTime struct {
	Time  time.Time
	Valid bool
}

func (nt *NullTime) Scan(value any) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = v, true
		return nil
	case string:
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, v); err == nil {
				nt.Time, nt.Valid = t, true
				return nil
			}
		}
	}
	nt.Time, nt.Valid = time.Time{}, false
	return nil
}
