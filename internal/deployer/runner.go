package deployer

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"iTrigger/internal/models"
	"iTrigger/internal/store"
)

type Runner struct {
	projectStore    *store.ProjectStore
	deploymentStore *store.DeploymentStore
	mu              sync.Mutex
	activeCmds      map[string]*exec.Cmd
}

func NewRunner(ps *store.ProjectStore, ds *store.DeploymentStore) *Runner {
	// Auto-configure git safe.directory '*' globally inside container on runner startup
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()

	return &Runner{
		projectStore:    ps,
		deploymentStore: ds,
		activeCmds:      make(map[string]*exec.Cmd),
	}
}

func generateDeploymentID(commitSHA string) string {
	now := time.Now()
	dateStr := now.Format("20060102")

	if len(commitSHA) >= 6 {
		return fmt.Sprintf("%s-%s", dateStr, commitSHA[:6])
	}

	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", dateStr, hex.EncodeToString(b))
}

func (r *Runner) TriggerDeployment(project models.ProjectConfig, triggeredBy, commitSHA, commitMsg string) *models.DeploymentLog {
	depID := generateDeploymentID(commitSHA)
	startTime := time.Now()

	depLog := models.DeploymentLog{
		ID:            depID,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		Repository:    project.Repository,
		Branch:        project.Branch,
		CommitSHA:     commitSHA,
		CommitMessage: commitMsg,
		TriggeredBy:   triggeredBy,
		Status:        "RUNNING",
		StartedAt:     startTime,
		Log:           fmt.Sprintf("Starting deployment for project %q (%s)\nPath: %s\nTriggered by: %s\n----------------------------------------\n", project.Name, project.Repository, project.ProjectPath, triggeredBy),
	}

	r.deploymentStore.Add(depLog)

	// Execute asynchronously in background goroutine
	go r.execute(depLog, project)

	return &depLog
}

func (r *Runner) execute(depLog models.DeploymentLog, project models.ProjectConfig) {
	startTime := time.Now()
	var logBuf bytes.Buffer
	logBuf.WriteString(depLog.Log)

	// 1. Verify project path exists
	cleanPath := strings.TrimSpace(project.ProjectPath)
	if cleanPath == "" {
		cleanPath = "."
	}

	// Automatically run git safe.directory for the specific project path & wildcard
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", cleanPath).Run()
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()

	fi, err := os.Stat(cleanPath)
	if err != nil || !fi.IsDir() {
		logBuf.WriteString(fmt.Sprintf("\n[ERROR] Project directory does not exist or is invalid: %s\n", cleanPath))
		depLog.Status = "FAILED"
		depLog.CompletedAt = time.Now()
		depLog.DurationMs = time.Since(startTime).Milliseconds()
		depLog.Log = logBuf.String()
		r.deploymentStore.Update(depLog)
		return
	}

	// 2. Resolve deployment script (Check target repository for .itrigger, .itrigger.sh, itrigger.sh, or fallback to UI script)
	scriptSource := "UI script"
	script := strings.TrimSpace(project.Script)

	repoFiles := []string{".itrigger", ".itrigger.sh", "itrigger.sh", filepath.Join(".itrigger", "deploy.sh")}
	for _, fname := range repoFiles {
		targetFile := filepath.Join(cleanPath, fname)
		if info, err := os.Stat(targetFile); err == nil && !info.IsDir() {
			if content, err := os.ReadFile(targetFile); err == nil && len(bytes.TrimSpace(content)) > 0 {
				script = string(content)
				scriptSource = fmt.Sprintf("file: %s", fname)
				break
			}
		}
	}

	if script == "" {
		logBuf.WriteString("\n[WARNING] No deployment script found in repository (.itrigger / .itrigger.sh / itrigger.sh) or project UI config. Skipping execution.\n")
		depLog.Status = "SKIPPED"
		depLog.CompletedAt = time.Now()
		depLog.DurationMs = time.Since(startTime).Milliseconds()
		depLog.Log = logBuf.String()
		r.deploymentStore.Update(depLog)
		return
	}

	if scriptSource != "UI script" {
		logBuf.WriteString(fmt.Sprintf("--> Detected repository deployment configuration %s\n", scriptSource))
	}
	logBuf.WriteString(fmt.Sprintf("Executing deployment script (%s) in %s...\n\n", scriptSource, cleanPath))

	// Run git safe.directory specifically for cleanPath directory before executing user script
	safeCmd := exec.Command("git", "config", "--global", "--add", "safe.directory", cleanPath)
	safeCmd.Dir = cleanPath
	_ = safeCmd.Run()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", script)
	} else {
		cmd = exec.Command("sh", "-c", script)
	}

	cmd.Dir = cleanPath

	// Inherit environment variables & bypass Git safe directory restriction automatically
	env := os.Environ()
	env = append(env,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=safe.directory",
		"GIT_CONFIG_VALUE_0=*",
	)
	cmd.Env = env

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	// Register active command before starting
	r.mu.Lock()
	r.activeCmds[depLog.ID] = cmd
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.activeCmds, depLog.ID)
		r.mu.Unlock()
	}()

	err = cmd.Run()
	logBuf.Write(outputBuf.Bytes())

	// If manual stop set it to STOPPED, do not overwrite it here
	if current, ok := r.deploymentStore.Get(depLog.ID); ok && current.Status == "STOPPED" {
		return
	}

	depLog.CompletedAt = time.Now()
	depLog.DurationMs = time.Since(startTime).Milliseconds()

	if err != nil {
		logBuf.WriteString(fmt.Sprintf("\n----------------------------------------\n[FAILED] Execution completed with error: %v\n", err))
		depLog.Status = "FAILED"
		log.Printf("Deployment %s for project %s failed: %v", depLog.ID, project.Name, err)
	} else {
		logBuf.WriteString("\n----------------------------------------\n[SUCCESS] Deployment script executed successfully.\n")
		depLog.Status = "SUCCESS"
		log.Printf("Deployment %s for project %s succeeded in %dms", depLog.ID, project.Name, depLog.DurationMs)
	}

	depLog.Log = logBuf.String()
	r.deploymentStore.Update(depLog)
}

func (r *Runner) StopDeployment(depID string) error {
	r.mu.Lock()
	cmd, ok := r.activeCmds[depID]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("deployment is not running or already completed")
	}

	if cmd.Process != nil {
		// Update status first to ensure the goroutine doesn't overwrite it
		depLog, ok := r.deploymentStore.Get(depID)
		if ok {
			depLog.Status = "STOPPED"
			depLog.CompletedAt = time.Now()
			depLog.Log += "\n----------------------------------------\n[STOPPED] Deployment stopped manually by administrator.\n"
			r.deploymentStore.Update(depLog)
		}

		err := cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to stop deployment process: %w", err)
		}
		return nil
	}

	return fmt.Errorf("process not started yet")
}
