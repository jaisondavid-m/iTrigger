package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"iTrigger/internal/models"
)

type ProjectStore struct {
	mu       sync.RWMutex
	filePath string
	projects map[string]models.ProjectConfig
}

func NewProjectStore(filePath string) (*ProjectStore, error) {
	if filePath == "" {
		filePath = filepath.Join("data", "projects.json")
	}

	ps := &ProjectStore{
		filePath: filePath,
		projects: make(map[string]models.ProjectConfig),
	}

	if err := ps.load(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	return ps, nil
}

func (ps *ProjectStore) load() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(ps.filePath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(ps.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Save empty file
			return ps.saveLocked()
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var list []models.ProjectConfig
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	for _, p := range list {
		ps.projects[p.ID] = p
	}

	return nil
}

func (ps *ProjectStore) saveLocked() error {
	list := make([]models.ProjectConfig, 0, len(ps.projects))
	for _, p := range ps.projects {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ps.filePath, data, 0644)
}

func (ps *ProjectStore) GetAll() []models.ProjectConfig {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make([]models.ProjectConfig, 0, len(ps.projects))
	for _, p := range ps.projects {
		result = append(result, p)
	}
	return result
}

func (ps *ProjectStore) Get(id string) (models.ProjectConfig, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	p, ok := ps.projects[id]
	return p, ok
}

func (ps *ProjectStore) Save(req models.CreateProjectRequest, existingID string) (models.ProjectConfig, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	var p models.ProjectConfig

	if existingID != "" {
		existing, ok := ps.projects[existingID]
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

	ps.projects[p.ID] = p

	if err := ps.saveLocked(); err != nil {
		return models.ProjectConfig{}, err
	}

	return p, nil
}

func (ps *ProjectStore) Delete(id string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.projects[id]; !ok {
		return false
	}

	delete(ps.projects, id)
	_ = ps.saveLocked()
	return true
}

func (ps *ProjectStore) FindByRepoAndBranch(repo, branch string) []models.ProjectConfig {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	repoClean := strings.ToLower(strings.TrimSpace(repo))
	branchClean := strings.TrimPrefix(strings.TrimSpace(branch), "refs/heads/")

	var matches []models.ProjectConfig
	for _, p := range ps.projects {
		if !p.Enabled {
			continue
		}

		pRepo := strings.ToLower(strings.TrimSpace(p.Repository))
		pBranch := strings.TrimPrefix(strings.TrimSpace(p.Branch), "refs/heads/")

		// Match repo name either exact match or trailing repo name (e.g. "owner/repo" or "repo")
		repoMatch := pRepo == repoClean || strings.HasSuffix(repoClean, "/"+pRepo) || strings.HasSuffix(pRepo, "/"+repoClean)
		branchMatch := pBranch == "" || pBranch == branchClean

		if repoMatch && branchMatch {
			matches = append(matches, p)
		}
	}
	return matches
}
