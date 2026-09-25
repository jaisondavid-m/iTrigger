package deployer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
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

	// Sensitive strings to redact from logs
	var secretsToRedact []string

	// 1. Verify and prepare project path
	cleanPath := strings.TrimSpace(project.ProjectPath)
	if cleanPath == "" {
		cleanPath = "."
	}

	if err := os.MkdirAll(cleanPath, 0755); err != nil {
		logBuf.WriteString(fmt.Sprintf("\n[ERROR] Could not create or access project directory: %s (%v)\n", cleanPath, err))
		depLog.Status = "FAILED"
		depLog.CompletedAt = time.Now()
		depLog.DurationMs = time.Since(startTime).Milliseconds()
		depLog.Log = logBuf.String()
		r.deploymentStore.Update(depLog)
		return
	}

	// Automatically run git safe.directory for the specific project path & wildcard
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", cleanPath).Run()
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()

	// 2. Prepare Git credentials and environment
	env := os.Environ()
	env = append(env,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=safe.directory",
		"GIT_CONFIG_VALUE_0=*",
	)

	var tempKeyFile string
	authType := strings.ToLower(strings.TrimSpace(project.AuthType))
	if authType == "" && project.IsPrivate {
		authType = "token"
	}

	// Resolve token if token or global auth is used
	token := strings.TrimSpace(project.AuthToken)
	if token == "" && (authType == "global" || (project.IsPrivate && authType == "token")) {
		token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}

	if token != "" {
		secretsToRedact = append(secretsToRedact, token)
		authHeaderVal := fmt.Sprintf("AUTHORIZATION: basic %s", base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token)))
		env = append(env,
			"GITHUB_TOKEN="+token,
			"GIT_CONFIG_COUNT=3",
			"GIT_CONFIG_KEY_1=http.extraheader",
			"GIT_CONFIG_VALUE_1="+authHeaderVal,
			"GIT_CONFIG_KEY_2=credential.helper",
			"GIT_CONFIG_VALUE_2=",
		)
	}

	// Resolve SSH key if SSH auth is used
	if authType == "ssh_key" && strings.TrimSpace(project.SSHPrivateKey) != "" {
		privKey := strings.TrimSpace(project.SSHPrivateKey)
		secretsToRedact = append(secretsToRedact, privKey)

		keysDir := filepath.Join("data", "keys")
		_ = os.MkdirAll(keysDir, 0700)
		tempKeyFile = filepath.Join(keysDir, fmt.Sprintf("id_%s.key", depLog.ID))
		if err := os.WriteFile(tempKeyFile, []byte(privKey+"\n"), 0600); err == nil {
			defer func() {
				_ = os.Remove(tempKeyFile)
			}()
			sshCmd := fmt.Sprintf("ssh -i %q -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o BatchMode=yes", tempKeyFile)
			env = append(env, "GIT_SSH_COMMAND="+sshCmd)
		} else {
			logBuf.WriteString(fmt.Sprintf("\n[WARNING] Failed to write temporary SSH key file: %v\n", err))
		}
	}

	// 3. Auto-clone if target directory is empty
	entries, readErr := os.ReadDir(cleanPath)
	if readErr == nil && len(entries) == 0 && strings.TrimSpace(project.Repository) != "" {
		repoURL := strings.TrimSpace(project.Repository)
		if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "git@") {
			if authType == "ssh_key" {
				repoURL = fmt.Sprintf("git@github.com:%s.git", strings.TrimSuffix(repoURL, ".git"))
			} else {
				repoURL = fmt.Sprintf("https://github.com/%s.git", strings.TrimSuffix(repoURL, ".git"))
			}
		}

		branchName := strings.TrimSpace(project.Branch)
		if branchName == "" {
			branchName = "main"
		}

		logBuf.WriteString(fmt.Sprintf("Directory is empty. Performing initial clone from %s (branch: %s)...\n", repoURL, branchName))
		cloneCmd := exec.Command("git", "clone", "--branch", branchName, repoURL, ".")
		cloneCmd.Dir = cleanPath
		cloneCmd.Env = env
		var cloneBuf bytes.Buffer
		cloneCmd.Stdout = &cloneBuf
		cloneCmd.Stderr = &cloneBuf
		if cloneErr := cloneCmd.Run(); cloneErr != nil {
			logBuf.WriteString(fmt.Sprintf("[ERROR] Initial git clone failed: %v\n%s\n", cloneErr, cloneBuf.String()))
		} else {
			logBuf.WriteString("--> Repository cloned successfully.\n\n")
		}
	}

	// 4. Resolve deployment script (Check target repository for .itrigger, .itrigger.sh, itrigger.sh, or fallback to UI script)
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

	err := cmd.Run()
	outputStr := outputBuf.String()

	// Redact sensitive secrets from log output
	for _, secret := range secretsToRedact {
		if secret != "" && len(secret) > 3 {
			outputStr = strings.ReplaceAll(outputStr, secret, "[REDACTED_SECRET]")
		}
	}
	logBuf.WriteString(outputStr)

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
