package events

import "time"

const (
	EventTypeNoteCreated = "note.created"
	EventTypeNoteUpdated = "note.updated"
	EventTypeNoteDeleted = "note.deleted"
)

type Event struct {
	EventType string         `json:"event_type"`
	UserID    string         `json:"user_id"`
	EntityID  string         `json:"entity_id"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}
