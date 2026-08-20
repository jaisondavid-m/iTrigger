package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"iTrigger/internal/deployer"
	"iTrigger/internal/models"
	"iTrigger/internal/store"
)

const githubSignaturePrefix = "sha256="

type Handler struct {
	secret       []byte
	logger       *log.Logger
	store        *Store
	projectStore *store.ProjectStore
	runner       *deployer.Runner
}

func New(secret string, store *Store) *Handler {
	if store == nil {
		store = NewStore()
	}
	return &Handler{
		secret: []byte(secret),
		logger: log.Default(),
		store:  store,
	}
}

func (h *Handler) SetDeployer(ps *store.ProjectStore, runner *deployer.Runner) {
	h.projectStore = ps
	h.runner = runner
}

func (h *Handler) Store() *Store {
	return h.store
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	deliveryID := r.Header.Get("X-GitHub-Delivery")
	eventType := r.Header.Get("X-GitHub-Event")
	signature := r.Header.Get("X-Hub-Signature-256")

	if deliveryID == "" || eventType == "" || signature == "" {
		h.writeError(w, http.StatusBadRequest, "missing required GitHub webhook headers")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "unable to read request body")
		return
	}

	summary := extractSummary(deliveryID, eventType, body)

	// Determine matching projects for signature verification
	var matchingProjects []models.ProjectConfig
	if h.projectStore != nil {
		if eventType == "push" {
			var payload models.PushPayload
			if err := json.Unmarshal(body, &payload); err == nil {
				branch := branchFromRef(payload.Ref)
				matchingProjects = h.projectStore.FindByRepoAndBranch(summary.RepositoryName, branch)
			}
		} else {
			// For non-push events (like ping, pull_request, etc.), find all projects matching the repository name
			allProjects := h.projectStore.GetAll()
			repoClean := strings.ToLower(strings.TrimSpace(summary.RepositoryName))
			for _, p := range allProjects {
				if !p.Enabled {
					continue
				}
				pRepo := strings.ToLower(strings.TrimSpace(p.Repository))
				repoMatch := pRepo == repoClean || strings.HasSuffix(repoClean, "/"+pRepo) || strings.HasSuffix(pRepo, "/"+repoClean)
				if repoMatch {
					matchingProjects = append(matchingProjects, p)
				}
			}
		}
	}

	// Verify signature using matching projects' secrets or system fallback
	isSignatureVerified := false
	if len(matchingProjects) > 0 {
		for _, proj := range matchingProjects {
			secretToUse := h.secret
			if proj.Secret != "" {
				secretToUse = []byte(proj.Secret)
			}
			if verifySignature(secretToUse, body, signature) {
				isSignatureVerified = true
				break
			}
		}
	} else {
		// Fallback to system secret if no projects match yet (e.g. initial setup ping before adding project)
		if len(h.secret) > 0 && verifySignature(h.secret, body, signature) {
			isSignatureVerified = true
		}
	}

	if !isSignatureVerified {
		h.writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	switch eventType {
	case "push":
		var payload models.PushPayload
		if err := json.Unmarshal(body, &payload); err == nil {
			branch := branchFromRef(payload.Ref)
			h.logger.Printf(
				"GitHub webhook received delivery=%s event=%s repository=%s ref=%s branch=%s sender=%s sha=%s message=%q",
				deliveryID,
				eventType,
				summary.RepositoryName,
				payload.Ref,
				branch,
				summary.Sender,
				payload.HeadCommit.ID,
				payload.HeadCommit.Message,
			)

			// Trigger automated deployments for matching verified projects
			if h.runner != nil {
				for _, proj := range matchingProjects {
					secretToUse := h.secret
					if proj.Secret != "" {
						secretToUse = []byte(proj.Secret)
					}
					if verifySignature(secretToUse, body, signature) {
						h.logger.Printf("Auto-deploying matching project id=%s name=%s repo=%s branch=%s", proj.ID, proj.Name, proj.Repository, proj.Branch)
						h.runner.TriggerDeployment(proj, "webhook:"+summary.Sender, payload.HeadCommit.ID, payload.HeadCommit.Message)
					}
				}
			}
		}
	case "pull_request":
		var payload models.PullRequestPayload
		if err := json.Unmarshal(body, &payload); err == nil {
			h.logger.Printf(
				"GitHub webhook received delivery=%s event=%s repository=%s action=%s pull_request_number=%d title=%q sender=%s",
				deliveryID,
				eventType,
				summary.RepositoryName,
				summary.Action,
				summary.PRNumber,
				summary.PRTitle,
				summary.Sender,
			)
		}
	default:
		h.logger.Printf("GitHub webhook received delivery=%s event=%s", deliveryID, eventType)
	}

	if h.store != nil {
		h.store.Add(summary)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (h *Handler) GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	events := h.store.GetAll()
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"count":  len(events),
		"events": events,
	})
}

func (h *Handler) ClearEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.store.Clear()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func extractSummary(deliveryID, eventType string, body []byte) models.WebhookEventSummary {
	summary := models.WebhookEventSummary{
		DeliveryID: deliveryID,
		EventType:  eventType,
		ReceivedAt: time.Now().Format(time.RFC3339),
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err == nil {
		// Repository Name
		if repoRaw, ok := raw["repository"].(map[string]any); ok {
			if fullName, ok := repoRaw["full_name"].(string); ok && fullName != "" {
				summary.RepositoryName = fullName
			} else if name, ok := repoRaw["name"].(string); ok {
				summary.RepositoryName = name
			}
		}

		// Action
		if action, ok := raw["action"].(string); ok {
			summary.Action = action
		}

		// Sender Login
		if senderRaw, ok := raw["sender"].(map[string]any); ok {
			if login, ok := senderRaw["login"].(string); ok {
				summary.Sender = login
			}
		}

		// Pull Request details
		if prRaw, ok := raw["pull_request"].(map[string]any); ok {
			if num, ok := prRaw["number"].(float64); ok {
				summary.PRNumber = int(num)
			}
			if title, ok := prRaw["title"].(string); ok {
				summary.PRTitle = title
			}
		}
	}

	return summary
}

func verifySignature(secret, body []byte, signature string) bool {
	if len(secret) == 0 || !strings.HasPrefix(signature, githubSignaturePrefix) {
		return false
	}

	providedSignature, err := hex.DecodeString(strings.TrimPrefix(signature, githubSignaturePrefix))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(body); err != nil {
		return false
	}

	expectedSignature := mac.Sum(nil)
	if len(expectedSignature) != len(providedSignature) {
		return false
	}

	return subtle.ConstantTimeCompare(expectedSignature, providedSignature) == 1
}

func branchFromRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, statusCode int, message string) {
	if h != nil && h.logger != nil {
		h.logger.Printf("GitHub webhook request rejected status=%d error=%s", statusCode, message)
	}
	http.Error(w, fmt.Sprintf("%s\n", message), statusCode)
}
