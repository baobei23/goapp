package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/baobei23/goapp/cmd/server/grpc"
	xhttp "github.com/baobei23/goapp/cmd/server/http"
	"github.com/baobei23/goapp/internal/api"
	"github.com/baobei23/goapp/internal/configs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/baobei23/goapp/internal/pkg/aerospike"
	"github.com/baobei23/goapp/internal/pkg/jwt"
	"github.com/baobei23/goapp/internal/pkg/postgres"
	"github.com/baobei23/goapp/internal/usernotes"
	"github.com/baobei23/goapp/internal/users"

	as "github.com/aerospike/aerospike-client-go/v8"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

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
		fatalErr <- fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	go func() {
		defer func() {
			rec := recover()
			if rec != nil {
				fatalErr <- fmt.Errorf("%+v", rec)
			}
		}()
		err = hserver.Start()
		if err != nil {
			fatalErr <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	return hserver, nil
}

func startHealthServer(ctx context.Context, db *pgxpool.Pool, isReady *atomic.Bool, fatalErr chan<- error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/-/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/-/ready", func(w http.ResponseWriter, r *http.Request) {
		if !isReady.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		// Aerospike only backs refresh tokens. If it's down, login/register/notes
		// still work — only /auth/refresh degrades. Don't pull the whole pod.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{Addr: ":2000", Handler: mux}
	go func() {
		slog.InfoContext(ctx, "[http/health] listening on :2000")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatalErr <- err
		}
	}()
	return srv
}

func start(
	ctx context.Context,
	isReady *atomic.Bool,
	cfgs *configs.Configs,
	fatalErr chan<- error,
) (hserver *xhttp.HTTP, gserver *grpc.GRPC, healthServer *http.Server, asClient *as.Client) {
	pqdriver, err := postgres.NewPool(cfgs.Postgres())
	if err != nil {
		panic(fmt.Errorf("failed to connect to postgres: %w", err))
	}

	asClient, err = aerospike.NewClient(cfgs.Aerospike())
	if err != nil {
		// Only misconfiguration reaches here; an unreachable cluster still
		// returns a client that reconnects in the background.
		panic(fmt.Errorf("failed to create aerospike client: %w", err))
	}

	healthServer = startHealthServer(ctx, pqdriver, isReady, fatalErr)

	userPGstore := users.NewPostgresStore(pqdriver)

	// Refresh tokens live in Aerospike, not Postgres: key-value with a native
	// record TTL, so expired tokens drop themselves with no cleanup job. User
	// CRUD stays on Postgres. To roll back, pass userPGstore for both args --
	// pgstore still implements the refresh-token methods.
	//
	// NOTE: Existing tokens in the `refresh_tokens` table are invisible after
	// deploy. All users logout within minutes. The table becomes orphaned: no
	// writes, no deletes. Acceptable for a boilerplate; production deploys need
	// dual-read (Aerospike first, Postgres fallback) during one refresh period.
	asTokenStore := users.NewAerospikeStore(asClient, cfgs.Aerospike().Namespace, "refresh_tokens")

	userSvc := users.NewService(userPGstore, asTokenStore)

	notePGstore := usernotes.NewPostgresStore(pqdriver)
	noteSvc := usernotes.NewService(notePGstore)

	svrAPIs := api.NewServer(userSvc, noteSvc)

	tm := cfgs.JWT()
	hserver, gserver = startServers(svrAPIs, cfgs, tm, fatalErr)
	return hserver, gserver, healthServer, asClient
}
