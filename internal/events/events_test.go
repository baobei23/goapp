package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventRoundTrip(t *testing.T) {
	in := Event{
		EventType: EventTypeNoteCreated,
		UserID:    "u1",
		EntityID:  "n1",
		Timestamp: time.Unix(0, 0).UTC(),
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out Event
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.EventType != in.EventType || out.UserID != in.UserID ||
		out.EntityID != in.EntityID || !out.Timestamp.Equal(in.Timestamp) {
		t.Fatalf("round trip mismatch: got %+v want %+v", out, in)
	}
}

func TestEntityType(t *testing.T) {
	if got := entityType(EventTypeNoteCreated); got != "note" {
		t.Errorf("note.created -> %q, want note", got)
	}
	if got := entityType("something.else"); got != "unknown" {
		t.Errorf("unknown event -> %q, want unknown", got)
	}
}
