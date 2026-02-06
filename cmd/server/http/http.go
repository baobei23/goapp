package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/baobei23/goapp/internal/api"
	"github.com/baobei23/goapp/internal/pkg/apm"
	"github.com/baobei23/goapp/internal/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Config holds all the configuration required to start the HTTP server
type Config struct {
	Host string
	Port uint16

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DialTimeout  time.Duration

	TemplatesBasePath string
	EnableAccessLog   bool
	EnableTracing     bool
}

type HTTP struct {
	server *http.Server
	router *gin.Engine
}

// Start starts the HTTP server
func (h *HTTP) Start() error {
	return h.server.ListenAndServe()
}

func (h *HTTP) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// NewService returns an instance of HTTP with all its dependencies set
func NewService(cfg *Config, apis api.Server, tm *jwt.TokenManager) (*HTTP, error) {
	home, err := loadHomeTemplate(cfg.TemplatesBasePath)
	if err != nil {
		return nil, err
	}

	handlers := &Handlers{
		apis: apis,
		home: home,
		tm:   tm,
	}

	if !cfg.EnableAccessLog {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	if cfg.EnableAccessLog {
		router.Use(gin.Logger())
	}

	if cfg.EnableTracing {
		// Use the global TracerProvider that was already set up in startAPM
		tp := apm.Global().GetTracerProvider()
		router.Use(otelgin.Middleware("goapp", otelgin.WithTracerProvider(tp)))
	}

	handlers.registerRoutes(router)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &HTTP{
		server: srv,
		router: router,
	}, nil
}
