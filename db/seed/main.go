package main

import (
	"fmt"

	"github.com/baobei23/goapp/internal/configs"
	"github.com/baobei23/goapp/internal/pkg/postgres"
	"github.com/naughtygopher/errors"
)

func main() {
	cfgs, err := configs.New()
	if err != nil {
		panic(errors.Wrap(err, "failed to load configurations"))
	}

	pgConfig := cfgs.Postgres()
	pgConfig.EnableTracing = false

	pqdriver, err := postgres.NewPool(pgConfig)
	if err != nil {
		panic(errors.Wrap(err, "failed to connect to postgres"))
	}
	defer pqdriver.Close()
	fmt.Println("Successfully connected to database!")
}
