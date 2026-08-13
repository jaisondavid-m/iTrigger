package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"iTrigger/internal/models"
)

const maxDeploymentLogs = 100

type DeploymentStore struct {
	mu       sync.RWMutex
	filePath string
	logs     map[string]models.DeploymentLog
}

func NewDeploymentStore(filePath string) (*DeploymentStore, error) {
	if filePath == "" {
		filePath = filepath.Join("data", "deployments.json")
	}

	ds := &DeploymentStore{
		filePath: filePath,
		logs:     make(map[string]models.DeploymentLog),
	}

	if err := ds.load(); err != nil {
		return nil, fmt.Errorf("failed to load deployment logs: %w", err)
	}

	return ds, nil
}

func (ds *DeploymentStore) load() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(ds.filePath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(ds.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ds.saveLocked()
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var list []models.DeploymentLog
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	for _, l := range list {
		ds.logs[l.ID] = l
	}

	return nil
}

func (ds *DeploymentStore) saveLocked() error {
	list := make([]models.DeploymentLog, 0, len(ds.logs))
	for _, l := range ds.logs {
		list = append(list, l)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].StartedAt.After(list[j].StartedAt)
	})

	if len(list) > maxDeploymentLogs {
		list = list[:maxDeploymentLogs]
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ds.filePath, data, 0644)
}

func (ds *DeploymentStore) Add(log models.DeploymentLog) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.logs[log.ID] = log
	_ = ds.saveLocked()
}

func (ds *DeploymentStore) Update(log models.DeploymentLog) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.logs[log.ID] = log
	_ = ds.saveLocked()
}

func (ds *DeploymentStore) Get(id string) (models.DeploymentLog, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	log, ok := ds.logs[id]
	return log, ok
}

func (ds *DeploymentStore) GetAll() []models.DeploymentLog {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	list := make([]models.DeploymentLog, 0, len(ds.logs))
	for _, l := range ds.logs {
		list = append(list, l)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].StartedAt.After(list[j].StartedAt)
	})

	return list
}

func (ds *DeploymentStore) GetByProject(projectID string) []models.DeploymentLog {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	list := make([]models.DeploymentLog, 0)
	for _, l := range ds.logs {
		if l.ProjectID == projectID {
			list = append(list, l)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].StartedAt.After(list[j].StartedAt)
	})

	return list
}
