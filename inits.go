package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/naughtygopher/errors"

	"github.com/baobei23/goapp/cmd/server/grpc"
	xhttp "github.com/baobei23/goapp/cmd/server/http"
	"github.com/baobei23/goapp/internal/api"
	"github.com/baobei23/goapp/internal/configs"
	"github.com/baobei23/goapp/internal/pkg/apm"
	"github.com/baobei23/goapp/internal/pkg/health"
	"github.com/baobei23/goapp/internal/pkg/jwt"
	"github.com/baobei23/goapp/internal/pkg/logger"
	"github.com/baobei23/goapp/internal/pkg/postgres"
	"github.com/baobei23/goapp/internal/usernotes"
	"github.com/baobei23/goapp/internal/users"
)

var now = time.Now()

func startAPM(ctx context.Context, cfg *configs.Configs) *apm.APM {
	if !cfg.EnableTracing && !cfg.EnableMetrics {
		return nil
	}
	ap, err := apm.New(ctx, &apm.Options{
		Debug:                cfg.Environment == configs.EnvLocal,
		Environment:          cfg.Environment.String(),
		ServiceName:          cfg.AppName,
		ServiceVersion:       cfg.AppVersion,
		PrometheusScrapePort: 9090,
		TracesSampleRate:     50.00,
		UseStdOut:            cfg.Environment == configs.EnvLocal,
		EnableTracing:        cfg.EnableTracing,
		EnableMetrics:        cfg.EnableMetrics,
	})
	if err != nil {
		panic(errors.Wrap(err, "failed to start APM"))
	}
	return ap
}

func startServers(svr api.Server, cfgs *configs.Configs, tm *jwt.TokenManager, fatalErr chan<- error) (*xhttp.HTTP, *grpc.GRPC) {
	hcfg, _ := cfgs.HTTP()
	hserver, err := xhttp.NewService(hcfg, svr, tm)
	if err != nil {
		fatalErr <- errors.Wrap(err, "failed to initialize HTTP server")
	}

	go func() {
		defer func() {
			rec := recover()
			if rec != nil {
				fatalErr <- errors.New(fmt.Sprintf("%+v", rec))
			}
		}()
		err = hserver.Start()
		if err != nil {
			fatalErr <- errors.Wrap(err, "failed to start HTTP server")
		}
	}()

	return hserver, nil
}

func healthResponseHandler(ps *health.ProbeResponder, cfg *configs.Configs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{
			"env":        cfg.Environment.String(),
			"version":    cfg.AppVersion,
			"commit":     "<git commit hash>",
			"status":     "all systems up and running",
			"startedAt":  now.String(),
			"releasedOn": now.String(),
		}

		for key, value := range ps.HealthResponse() {
			payload[key] = value
		}
		b, _ := json.Marshal(payload)
		w.Header().Add("Content-Type", "application/json")
		_, _ = w.Write(b)
	}
}

func startHealthResponder(ctx context.Context, ps *health.ProbeResponder, cfgs *configs.Configs, fatalErr chan<- error) (*http.Server, error) {
	port := uint32(2000)
	srv := health.Server(
		ps, "", uint16(port),
		health.Handler{
			Method:  http.MethodGet,
			Path:    "/-/health",
			Handler: healthResponseHandler(ps, cfgs),
		},
	)

	go func() {
		defer logger.Info(ctx, fmt.Sprintf("[http/healthresponder] :%d shutdown complete", port))
		logger.Info(ctx, fmt.Sprintf("[http/healthresponder] listening on :%d", port))
		fatalErr <- srv.ListenAndServe()
	}()

	return srv, nil
}

func start(
	ctx context.Context,
	probestatus *health.ProbeResponder,
	cfgs *configs.Configs,
	fatalErr chan<- error,
) (hserver *xhttp.HTTP, gserver *grpc.GRPC) {
	_ = ctx
	pqdriver, err := postgres.NewPool(cfgs.Postgres())
	if err != nil {
		panic(errors.Wrap(err))
	}

	health.Start(time.Minute, probestatus, &health.Probe{
		ID:               "postgres",
		AffectedStatuses: []health.Statuskey{health.StatusLive, health.StatusReady},
		Checker: health.CheckerFunc(func(ctx context.Context) error {
			err := pqdriver.Ping(ctx)
			if err != nil {
				return errors.Wrap(err, "postgres ping failed")
			}
			return nil
		}),
	})

	userPGstore := users.NewPostgresStore(pqdriver, cfgs.UserPostgresTable())
	userSvc := users.NewService(userPGstore)

	notePGstore := usernotes.NewPostgresStore(pqdriver, "user_notes")
	noteSvc := usernotes.NewService(notePGstore)

	svrAPIs := api.NewServer(userSvc, noteSvc)

	tm := cfgs.JWT()
	hserver, gserver = startServers(svrAPIs, cfgs, tm, fatalErr)
	return
}
