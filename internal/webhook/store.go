package webhook

import (
	"sync"

	"iTrigger/internal/models"
)

const maxStoredEvents = 100

// Store provides thread-safe in-memory storage for webhook event summaries.
type Store struct {
	mu     sync.RWMutex
	events []models.WebhookEventSummary
}

// NewStore initializes a new Store.
func NewStore() *Store {
	return &Store{
		events: make([]models.WebhookEventSummary, 0, maxStoredEvents),
	}
}

// Add appends a new webhook event summary to the store, maintaining max capacity.
func (s *Store) Add(event models.WebhookEventSummary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prepend new event (latest first)
	s.events = append([]models.WebhookEventSummary{event}, s.events...)
	if len(s.events) > maxStoredEvents {
		s.events = s.events[:maxStoredEvents]
	}
}

// GetAll returns a copy of all stored webhook event summaries.
func (s *Store) GetAll() []models.WebhookEventSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.WebhookEventSummary, len(s.events))
	copy(result, s.events)
	return result
}

// Clear removes all stored webhook event summaries.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = make([]models.WebhookEventSummary, 0, maxStoredEvents)
}
