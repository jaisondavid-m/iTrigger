package deployer

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"iTrigger/internal/models"
	"iTrigger/internal/store"
)

type Runner struct {
	projectStore    *store.ProjectStore
	deploymentStore *store.DeploymentStore
}

func NewRunner(ps *store.ProjectStore, ds *store.DeploymentStore) *Runner {
	return &Runner{
		projectStore:    ps,
		deploymentStore: ds,
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

	// 2. Prepare command runner
	script := strings.TrimSpace(project.Script)
	if script == "" {
		logBuf.WriteString("\n[WARNING] No deployment script defined for this project. Skipping execution.\n")
		depLog.Status = "SKIPPED"
		depLog.CompletedAt = time.Now()
		depLog.DurationMs = time.Since(startTime).Milliseconds()
		depLog.Log = logBuf.String()
		r.deploymentStore.Update(depLog)
		return
	}

	logBuf.WriteString(fmt.Sprintf("Executing deployment script in %s...\n\n", cleanPath))

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", script)
	} else {
		cmd = exec.Command("sh", "-c", script)
	}

	cmd.Dir = cleanPath

	// Inherit environment variables
	cmd.Env = os.Environ()

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err = cmd.Run()
	logBuf.Write(outputBuf.Bytes())

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
