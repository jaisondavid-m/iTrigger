package webhook

import (
	"database/sql"
	"sync"

	"iTrigger/internal/models"

	_ "modernc.org/sqlite"
)

const maxStoredEvents = 100

// Store provides thread-safe persistent SQLite storage for webhook event summaries.
type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

// NewStore initializes a new Store with SQLite database backend.
// If no database is provided, it falls back to an in-memory SQLite instance for testing.
func NewStore(dbs ...*sql.DB) *Store {
	var db *sql.DB
	if len(dbs) > 0 && dbs[0] != nil {
		db = dbs[0]
	} else {
		memDB, err := sql.Open("sqlite", ":memory:")
		if err == nil {
			_, _ = memDB.Exec(`
				CREATE TABLE IF NOT EXISTS webhooks (
					delivery_id TEXT PRIMARY KEY,
					event_type TEXT NOT NULL,
					repository_name TEXT NOT NULL DEFAULT '',
					action TEXT NOT NULL DEFAULT '',
					pr_number INTEGER NOT NULL DEFAULT 0,
					pr_title TEXT NOT NULL DEFAULT '',
					sender TEXT NOT NULL DEFAULT '',
					received_at TEXT NOT NULL
				);
			`)
			db = memDB
		}
	}

	return &Store{
		db: db,
	}
}

// Add appends a new webhook event summary to SQLite, maintaining max capacity.
func (s *Store) Add(event models.WebhookEventSummary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return
	}

	_, _ = s.db.Exec(`
		INSERT INTO webhooks (delivery_id, event_type, repository_name, action, pr_number, pr_title, sender, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(delivery_id) DO UPDATE SET
			event_type=excluded.event_type,
			repository_name=excluded.repository_name,
			action=excluded.action,
			pr_number=excluded.pr_number,
			pr_title=excluded.pr_title,
			sender=excluded.sender,
			received_at=excluded.received_at
	`, event.DeliveryID, event.EventType, event.RepositoryName, event.Action, event.PRNumber, event.PRTitle, event.Sender, event.ReceivedAt)

	// Clean up old events beyond maxStoredEvents limit
	_, _ = s.db.Exec(`
		DELETE FROM webhooks WHERE delivery_id NOT IN (
			SELECT delivery_id FROM webhooks ORDER BY rowid DESC LIMIT ?
		)
	`, maxStoredEvents)
}

// GetAll returns all stored webhook event summaries ordered newest first.
func (s *Store) GetAll() []models.WebhookEventSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		return []models.WebhookEventSummary{}
	}

	rows, err := s.db.Query(`
		SELECT delivery_id, event_type, repository_name, action, pr_number, pr_title, sender, received_at
		FROM webhooks
		ORDER BY rowid DESC
	`)
	if err != nil {
		return []models.WebhookEventSummary{}
	}
	defer rows.Close()

	var events []models.WebhookEventSummary
	for rows.Next() {
		var ev models.WebhookEventSummary
		if err := rows.Scan(&ev.DeliveryID, &ev.EventType, &ev.RepositoryName, &ev.Action, &ev.PRNumber, &ev.PRTitle, &ev.Sender, &ev.ReceivedAt); err == nil {
			events = append(events, ev)
		}
	}
	return events
}

// Clear removes all stored webhook event summaries.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return
	}

	_, _ = s.db.Exec(`DELETE FROM webhooks`)
}
