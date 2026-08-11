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
)

const githubSignaturePrefix = "sha256="

type Handler struct {
	secret []byte
	logger *log.Logger
}

type pushPayload struct {
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
	Ref        string `json:"ref"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
}

type pullRequestPayload struct {
	Action     string `json:"action"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"pull_request"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

func New(secret string) *Handler {
	return &Handler{
		secret: []byte(secret),
		logger: log.Default(),
	}
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

	switch eventType {
	case "push":
		var payload pushPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid push payload")
			return
		}

		repositoryName := firstNonEmpty(payload.Repository.FullName, payload.Repository.Name)
		branch := branchFromRef(payload.Ref)
		h.logger.Printf(
			"GitHub webhook received delivery=%s event=%s repository=%s ref=%s branch=%s sender=%s sha=%s message=%q",
			deliveryID,
			eventType,
			repositoryName,
			payload.Ref,
			branch,
			payload.Sender.Login,
			payload.HeadCommit.ID,
			payload.HeadCommit.Message,
		)
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	case "pull_request":
		var payload pullRequestPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid pull_request payload")
			return
		}

		repositoryName := firstNonEmpty(payload.Repository.FullName, payload.Repository.Name)
		h.logger.Printf(
			"GitHub webhook received delivery=%s event=%s repository=%s action=%s pull_request_number=%d title=%q sender=%s",
			deliveryID,
			eventType,
			repositoryName,
			payload.Action,
			payload.PullRequest.Number,
			payload.PullRequest.Title,
			payload.Sender.Login,
		)
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	case "ping":
		h.logger.Printf("GitHub webhook received delivery=%s event=%s", deliveryID, eventType)
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	default:
		h.logger.Printf("GitHub webhook received delivery=%s event=%s", deliveryID, eventType)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
	}
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
