package grpc

import (
	"context"

	"github.com/baobei23/goapp/internal/api"
)

type GRPC struct {
	apis api.Server
}

func (gr *GRPC) Shutdown(ctx context.Context) error {
	_ = ctx
	return nil
}

func New(apis api.Server) *GRPC {
	return &GRPC{
		apis: apis,
	}
}
