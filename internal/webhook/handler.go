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

	"iTrigger/internal/models"
)

const githubSignaturePrefix = "sha256="

type Handler struct {
	secret []byte
	logger *log.Logger
	store  *Store
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

func (h *Handler) Store() *Store {
	return h.store
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h == nil || len(h.secret) == 0 {
		h.writeError(w, http.StatusInternalServerError, "webhook secret is not configured")
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

	if !verifySignature(h.secret, body, signature) {
		h.writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	summary := extractSummary(deliveryID, eventType, body)

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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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
