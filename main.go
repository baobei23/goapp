package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/baobei23/goapp/docs"
	"github.com/naughtygopher/errors"

	"github.com/baobei23/goapp/internal/configs"
	"github.com/baobei23/goapp/internal/pkg/health"
	"github.com/baobei23/goapp/internal/pkg/logger"
	"github.com/baobei23/goapp/internal/pkg/sysignals"
)

//	@title			GoApp API
//	@description	API for GoApp
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	https://github.com/baobei23
//	@contact.email	reginaldsaja98@gmail.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@BasePath					/
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Bearer token

var exitErr error

// recoverer is used for panic recovery of the application (note: this is not for the HTTP/gRPC servers).
// So that even if the main function panics we can produce required logs for troubleshooting

func recoverer() {
	exitCode := 0
	var exitInfo any
	rec := recover()
	err, _ := rec.(error)
	if err != nil {
		exitCode = 1
		exitInfo = err
	} else if rec != nil {
		exitCode = 2
		exitInfo = rec
	} else if exitErr != nil {
		exitCode = 3
		exitInfo = exitErr
	}

	// exiting after receiving a quit signal can be considered a *clean/successful* exit
	if errors.Is(exitErr, sysignals.ErrSigQuit) {
		exitCode = 0
	}

	ctx := context.Background()
	// logging this because we have info logs saying "listening to" various port numbers
	// based on the server type (gRPC, HTTP etc.). But it's unclear *from the logs*
	// if the server is up and running, if it exits for any reason
	if exitCode == 0 {
		logger.Info(ctx, fmt.Sprintf("shutdown complete: %+v", exitInfo))
	} else {
		logger.Error(ctx, fmt.Sprintf("shutdown complete (exit: %d): %+v", exitCode, exitInfo))
	}

	os.Exit(exitCode)
}

func main() {
	defer recoverer()
	var (
		ctx                 = context.Background()
		fatalErr            = make(chan error, 1)
		shutdownGraceperiod = time.Minute
		probeInterval       = time.Second * 3
		probestatus         = health.New()
	)

	cfgs, err := configs.New()
	if err != nil {
		panic(errors.Wrap(err))
	}

	logger.UpdateDefaultLogger(logger.New(
		cfgs.AppName, cfgs.AppVersion, 0,
		map[string]string{
			"env": cfgs.Environment.String(),
		}),
	)

	// This needs to remain after log initialisation and before server initialisation.
	ap := startAPM(ctx, cfgs)

	var healthResponder *http.Server

	healthResponder, err = startHealthResponder(ctx, probestatus, cfgs, fatalErr)
	if err != nil {
		panic(err)
	}

	hserver, gserver := start(ctx, probestatus, cfgs, fatalErr)

	defer shutdown(
		shutdownGraceperiod,
		probeInterval,
		probestatus,
		healthResponder,
		hserver,
		gserver,
		ap,
	)
	exitErr = <-fatalErr
}
