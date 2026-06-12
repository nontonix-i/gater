package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/gater/config"
	"github.com/user/gater/internal/api"
	"github.com/user/gater/internal/database"
	"github.com/user/gater/internal/keepalive"
	"github.com/user/gater/internal/provider"
	"github.com/user/gater/internal/provider/abyss"
	"github.com/user/gater/internal/provider/anonmp4"
	"github.com/user/gater/internal/provider/doodstream"
	"github.com/user/gater/internal/provider/gofile"
	"github.com/user/gater/internal/provider/lulustream"
	"github.com/user/gater/internal/provider/rapidgator"
	"github.com/user/gater/internal/provider/rpmshare"
	"github.com/user/gater/internal/provider/seekstreaming"
	"github.com/user/gater/internal/provider/streamtape"
	"github.com/user/gater/internal/provider/turboviplay"
	"github.com/user/gater/internal/provider/vidoza"
	"github.com/user/gater/internal/provider/vikingfiles"
	"github.com/user/gater/internal/task"
)

//go:embed web/dist
var webFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.New(cfg.Database.DSN)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	reg := provider.NewRegistry()
	registerProviders(reg)

	tm := task.NewManager(db, reg, cfg.Upload.TempDir)

	subFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		slog.Error("failed to get web FS", "error", err)
		os.Exit(1)
	}
	router := api.NewRouter(db, tm, reg, cfg.Auth.Enabled, subFS)

	ks := keepalive.New(db, cfg.Keepalive)
	go ks.Start(context.Background())

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("starting server", "address", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down...")
		ks.Stop()
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func registerProviders(reg *provider.Registry) {
	providers := []struct {
		name string
		p    provider.Provider
	}{
		{"abyss", abyss.New()},
		{"anonmp4", anonmp4.New()},
		{"doodstream", doodstream.New()},
		{"gofile", gofile.New()},
		{"lulustream", lulustream.New()},
		{"rapidgator", rapidgator.New()},
		{"rpmshare", rpmshare.New()},
		{"seekstreaming", seekstreaming.New()},
		{"streamtape", streamtape.New()},
		{"turboviplay", turboviplay.New()},
		{"vidoza", vidoza.New()},
		{"vikingfiles", vikingfiles.New()},
	}

	for _, p := range providers {
		reg.Register(p.p)
		slog.Info("registered provider", "name", p.name, "type", p.p.Type())
	}
}
