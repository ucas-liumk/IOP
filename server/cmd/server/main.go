package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/leo/iop/server/internal/app"
	"github.com/leo/iop/server/internal/config"
	iface "github.com/leo/iop/server/internal/interface"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	warnings, err := cfg.Validate()
	for _, w := range warnings {
		log.Printf("[config warning] %s", w)
	}
	if err != nil {
		log.Fatalf("config validation: %v", err)
	}

	a, cleanup, err := app.Build(ctx, cfg)
	if err != nil {
		log.Fatalf("app build: %v", err)
	}
	defer cleanup()

	if err := iface.Run(ctx, cfg.Server.Addr, a.Engine(), a.Logger); err != nil {
		log.Fatalf("server: %v", err)
	}
}
