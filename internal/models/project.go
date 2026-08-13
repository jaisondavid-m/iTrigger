package models

import "time"

// ProjectConfig defines a target repository deployment configuration.
type ProjectConfig struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Repository  string    `json:"repository"`  // e.g. "owner/repo" or "repo"
	Branch      string    `json:"branch"`      // e.g. "main" or "master"
	ProjectPath string    `json:"projectPath"` // Server filesystem path e.g. "/my/project"
	Script      string    `json:"script"`      // Shell commands e.g. "git pull origin main\ndocker compose up --build -d"
	Secret      string    `json:"secret"`      // Optional per-project webhook secret override
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DeploymentLog records details and output of a deployment execution.
type DeploymentLog struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"projectId"`
	ProjectName   string    `json:"projectName"`
	Repository    string    `json:"repository"`
	Branch        string    `json:"branch"`
	CommitSHA     string    `json:"commitSha,omitempty"`
	CommitMessage string    `json:"commitMessage,omitempty"`
	TriggeredBy   string    `json:"triggeredBy"` // "webhook:<sender>" or "manual:ui"
	Status        string    `json:"status"`      // "QUEUED", "RUNNING", "SUCCESS", "FAILED"
	Log           string    `json:"log"`
	StartedAt     time.Time `json:"startedAt"`
	CompletedAt   time.Time `json:"completedAt,omitempty"`
	DurationMs    int64     `json:"durationMs"`
}

// CreateProjectRequest payload for creating/updating a project.
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Repository  string `json:"repository"`
	Branch      string `json:"branch"`
	ProjectPath string `json:"projectPath"`
	Script      string `json:"script"`
	Secret      string `json:"secret"`
	Enabled     bool   `json:"enabled"`
}
