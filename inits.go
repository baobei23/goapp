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
	"log/slog"

	"github.com/baobei23/goapp/internal/pkg/health"
	"github.com/baobei23/goapp/internal/pkg/jwt"
	"github.com/baobei23/goapp/internal/pkg/postgres"
	"github.com/baobei23/goapp/internal/usernotes"
	"github.com/baobei23/goapp/internal/users"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var now = time.Now()

func setupTelemetry(cfgs *configs.Configs) *sdktrace.TracerProvider {
	if !cfgs.EnableTracing && !cfgs.EnableMetrics {
		return nil
	}

	if cfgs.EnableMetrics {
		go func() {
			port := 9090
			mux := http.NewServeMux()
			mux.Handle("/-/metrics", promhttp.Handler())
			slog.Info(fmt.Sprintf("[otel/http] starting prometheus metrics on :%d/-/metrics", port))
			_ = http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
		}()
	}

	var tp *sdktrace.TracerProvider
	if cfgs.EnableTracing {
		res, _ := resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(cfgs.AppName),
				semconv.ServiceVersion(cfgs.AppVersion),
				semconv.DeploymentEnvironment(cfgs.Environment.String()),
			),
		)

		exporter, _ := otlptracegrpc.New(context.Background())

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.5)),
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(exporter),
		)
		otel.SetTracerProvider(tp)
	}

	return tp
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
		defer slog.InfoContext(ctx, fmt.Sprintf("[http/healthresponder] :%d shutdown complete", port))
		slog.InfoContext(ctx, fmt.Sprintf("[http/healthresponder] listening on :%d", port))
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
