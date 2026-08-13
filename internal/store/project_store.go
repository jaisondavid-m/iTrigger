package store

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"iTrigger/internal/models"
)

type ProjectStore struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewProjectStore(db *sql.DB) *ProjectStore {
	return &ProjectStore{
		db: db,
	}
}

func (ps *ProjectStore) GetAll() []models.ProjectConfig {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	rows, err := ps.db.Query(`
		SELECT id, name, repository, branch, project_path, script, secret, enabled, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`)
	if err != nil {
		return []models.ProjectConfig{}
	}
	defer rows.Close()

	var projects []models.ProjectConfig
	for rows.Next() {
		var p models.ProjectConfig
		var enabledInt int
		if err := rows.Scan(&p.ID, &p.Name, &p.Repository, &p.Branch, &p.ProjectPath, &p.Script, &p.Secret, &enabledInt, &p.CreatedAt, &p.UpdatedAt); err == nil {
			p.Enabled = (enabledInt == 1)
			projects = append(projects, p)
		}
	}
	return projects
}

func (ps *ProjectStore) Get(id string) (models.ProjectConfig, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.getLocked(id)
}

func (ps *ProjectStore) Save(req models.CreateProjectRequest, existingID string) (models.ProjectConfig, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	var p models.ProjectConfig

	if existingID != "" {
		existing, ok := ps.getLocked(existingID)
		if !ok {
			return models.ProjectConfig{}, fmt.Errorf("project not found: %s", existingID)
		}
		p = existing
		p.UpdatedAt = now
	} else {
		p = models.ProjectConfig{
			ID:        fmt.Sprintf("proj_%d", time.Now().UnixNano()),
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	p.Name = strings.TrimSpace(req.Name)
	p.Repository = strings.TrimSpace(req.Repository)
	p.Branch = strings.TrimSpace(req.Branch)
	p.ProjectPath = strings.TrimSpace(req.ProjectPath)
	p.Script = req.Script
	p.Secret = strings.TrimSpace(req.Secret)
	p.Enabled = req.Enabled

	if p.Branch == "" {
		p.Branch = "main"
	}

	enabledInt := 0
	if p.Enabled {
		enabledInt = 1
	}

	_, err := ps.db.Exec(`
		INSERT OR REPLACE INTO projects (id, name, repository, branch, project_path, script, secret, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Repository, p.Branch, p.ProjectPath, p.Script, p.Secret, enabledInt, p.CreatedAt, p.UpdatedAt)

	if err != nil {
		return models.ProjectConfig{}, fmt.Errorf("failed to save project to db: %w", err)
	}

	return p, nil
}

func (ps *ProjectStore) getLocked(id string) (models.ProjectConfig, bool) {
	var p models.ProjectConfig
	var enabledInt int
	err := ps.db.QueryRow(`
		SELECT id, name, repository, branch, project_path, script, secret, enabled, created_at, updated_at
		FROM projects WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Repository, &p.Branch, &p.ProjectPath, &p.Script, &p.Secret, &enabledInt, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return models.ProjectConfig{}, false
	}

	p.Enabled = (enabledInt == 1)
	return p, true
}

func (ps *ProjectStore) Delete(id string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	res, err := ps.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return false
	}
	rowsAffected, _ := res.RowsAffected()
	return rowsAffected > 0
}

func (ps *ProjectStore) FindByRepoAndBranch(repo, branch string) []models.ProjectConfig {
	allProjects := ps.GetAll()

	repoClean := strings.ToLower(strings.TrimSpace(repo))
	branchClean := strings.TrimPrefix(strings.TrimSpace(branch), "refs/heads/")

	var matches []models.ProjectConfig
	for _, p := range allProjects {
		if !p.Enabled {
			continue
		}

		pRepo := strings.ToLower(strings.TrimSpace(p.Repository))
		pBranch := strings.TrimPrefix(strings.TrimSpace(p.Branch), "refs/heads/")

		repoMatch := pRepo == repoClean || strings.HasSuffix(repoClean, "/"+pRepo) || strings.HasSuffix(pRepo, "/"+repoClean)
		branchMatch := pBranch == "" || pBranch == branchClean

		if repoMatch && branchMatch {
			matches = append(matches, p)
		}
	}
	return matches
}
