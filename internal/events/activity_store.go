package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Activity struct {
	ID         string
	EventType  string
	UserID     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
	CreatedAt  time.Time
}

type ActivityStore struct {
	db *pgxpool.Pool
}

func (as *ActivityStore) GetUserActivity(ctx context.Context, userID string, limit int) ([]Activity, error) {

	query := `
		SELECT id, event_type, user_id, entity_type, entity_id, metadata, created_at
		FROM activity_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := as.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.ID, &a.EventType, &a.UserID, &a.EntityType, &a.EntityID, &a.Metadata, &a.CreatedAt); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

func NewActivityStore(db *pgxpool.Pool) *ActivityStore {
	return &ActivityStore{db: db}
}
