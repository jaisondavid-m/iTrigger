package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"iTrigger/internal/db"
	"iTrigger/internal/models"
	"iTrigger/internal/store"
)

func TestProjectStorePrivateRepo(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()
	defer os.Remove(dbPath)

	ps := store.NewProjectStore(database)

	req := models.CreateProjectRequest{
		Name:          "Private Project",
		Repository:    "owner/private-repo",
		Branch:        "main",
		ProjectPath:   "/var/www/private",
		Script:        "git pull origin main",
		Secret:        "my-secret",
		Enabled:       true,
		IsPrivate:     true,
		AuthType:      "token",
		AuthToken:     "ghp_test1234567890",
		SSHPrivateKey: "",
		SSHPublicKey:  "",
	}

	saved, err := ps.Save(req, "")
	if err != nil {
		t.Fatalf("Save project failed: %v", err)
	}

	if !saved.IsPrivate {
		t.Errorf("expected IsPrivate to be true")
	}
	if saved.AuthType != "token" {
		t.Errorf("expected AuthType 'token', got %q", saved.AuthType)
	}
	if saved.AuthToken != "ghp_test1234567890" {
		t.Errorf("expected AuthToken 'ghp_test1234567890', got %q", saved.AuthToken)
	}

	fetched, ok := ps.Get(saved.ID)
	if !ok {
		t.Fatalf("Get project failed")
	}
	if !fetched.IsPrivate || fetched.AuthToken != "ghp_test1234567890" {
		t.Errorf("fetched project does not match saved credentials: %+v", fetched)
	}

	// Test update with masked value preserves original token
	updateReq := models.CreateProjectRequest{
		Name:          "Updated Private Project",
		Repository:    "owner/private-repo",
		Branch:        "main",
		ProjectPath:   "/var/www/private",
		Script:        "git pull origin main",
		Secret:        "my-secret",
		Enabled:       true,
		IsPrivate:     true,
		AuthType:      "token",
		AuthToken:     "••••••••", // Masked value from UI
		SSHPrivateKey: "",
		SSHPublicKey:  "",
	}

	updated, err := ps.Save(updateReq, saved.ID)
	if err != nil {
		t.Fatalf("Update project failed: %v", err)
	}

	if updated.AuthToken != "ghp_test1234567890" {
		t.Errorf("expected preserved AuthToken 'ghp_test1234567890', got %q", updated.AuthToken)
	}
}
