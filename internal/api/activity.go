package api

import (
	"context"

	"github.com/baobei23/goapp/internal/events"
)

func (a *API) GetUserActivity(ctx context.Context, userID string, limit int) ([]events.Activity, error) {
	if a.astore == nil {
		return nil, nil // graceful: no Kafka = no activity log
	}
	return a.astore.GetUserActivity(ctx, userID, limit)
}
