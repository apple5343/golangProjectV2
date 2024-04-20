package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/apple5343/golangProjectV2/internal/app"
	"github.com/apple5343/golangProjectV2/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.InitConfig("config/local.yaml")
	if err != nil {
		return err
	}
	app, err := app.New(cfg.StoragePath, cfg)
	if err != nil {
		return err
	}
	go app.GRPCServer.Run()
	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: app,
	}
	fmt.Printf("Config: %+v\n", cfg)
	if err := httpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
