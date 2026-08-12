package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iTrigger/internal/models"
)

func TestStore(t *testing.T) {
	store := NewStore()
	if len(store.GetAll()) != 0 {
		t.Fatalf("expected empty store, got %d", len(store.GetAll()))
	}

	event := models.WebhookEventSummary{
		DeliveryID:     "del-123",
		EventType:      "pull_request",
		RepositoryName: "jaisondavid-m/iTrigger",
		Action:         "opened",
		PRNumber:       1,
		PRTitle:        "Test PR",
		Sender:         "octocat",
	}

	store.Add(event)
	events := store.GetAll()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].DeliveryID != "del-123" {
		t.Errorf("expected del-123, got %s", events[0].DeliveryID)
	}

	store.Clear()
	if len(store.GetAll()) != 0 {
		t.Fatalf("expected empty store after clear, got %d", len(store.GetAll()))
	}
}

func TestWebhookHandlerIntegration(t *testing.T) {
	secret := "testsecret"
	store := NewStore()
	h := New(secret, store)

	payloadJSON := `{
		"action": "opened",
		"repository": {
			"full_name": "jaisondavid-m/iTrigger"
		},
		"pull_request": {
			"number": 10,
			"title": "Fix webhook handling"
		},
		"sender": {
			"login": "jaisondavid-m"
		}
	}`

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadJSON))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/github", strings.NewReader(payloadJSON))
	req.Header.Set("X-GitHub-Delivery", "github-del-999")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signature)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Test Get Events
	reqGet := httptest.NewRequest(http.MethodGet, "/api/webhooks", nil)
	recGet := httptest.NewRecorder()

	h.GetEventsHandler(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recGet.Code)
	}

	var resp struct {
		Status string                       `json:"status"`
		Count  int                          `json:"count"`
		Events []models.WebhookEventSummary `json:"events"`
	}
	if err := json.NewDecoder(recGet.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}

	if resp.Count != 1 {
		t.Fatalf("expected 1 event, got %d", resp.Count)
	}

	evt := resp.Events[0]
	if evt.DeliveryID != "github-del-999" {
		t.Errorf("expected deliveryID github-del-999, got %s", evt.DeliveryID)
	}
	if evt.EventType != "pull_request" {
		t.Errorf("expected eventType pull_request, got %s", evt.EventType)
	}
	if evt.RepositoryName != "jaisondavid-m/iTrigger" {
		t.Errorf("expected repository jaisondavid-m/iTrigger, got %s", evt.RepositoryName)
	}
	if evt.Action != "opened" {
		t.Errorf("expected action opened, got %s", evt.Action)
	}
	if evt.PRNumber != 10 {
		t.Errorf("expected PRNumber 10, got %d", evt.PRNumber)
	}
	if evt.PRTitle != "Fix webhook handling" {
		t.Errorf("expected PRTitle 'Fix webhook handling', got %s", evt.PRTitle)
	}
	if evt.Sender != "jaisondavid-m" {
		t.Errorf("expected sender jaisondavid-m, got %s", evt.Sender)
	}
}
