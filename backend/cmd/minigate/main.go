package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"minigate/internal/admin"
	"minigate/internal/balancer"
	"minigate/internal/circuit"
	"minigate/internal/config"
	"minigate/internal/health"
	"minigate/internal/logger"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/proxy"
	"minigate/internal/router"

	_ "minigate/internal/middleware"
)

func main() {
	path := os.Getenv("GATEWAY_CONFIG")
	if path == "" {
		path = "/app/config/gateway.yaml"
	}
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}
	log := logger.Init(level)

	table := router.NewTable()
	lb := balancer.NewRegistry()
	metric := metrics.New()
	breaks := circuit.NewRegistry()
	src := &config.FileSource{Path: path}

	rl := config.NewReloader(src, func(cfg *model.GatewayConfig) error {
		table.Swap(cfg)
		lb.Rebuild(cfg.Upstreams)
		breaks.Reset(cfg.Upstreams)
		if cfg.LogLevel != "" && cfg.LogLevel != level {
			level = cfg.LogLevel
			logger.Init(level)
		}
		return nil
	})

	cfg, err := rl.LoadAndApply()
	if err != nil {
		log.Error("load config", slog.String("err", err.Error()))
		os.Exit(1)
	}

	stop := make(chan struct{})
	if err := rl.WatchFile(path, stop); err != nil {
		log.Warn("watch config failed", slog.String("err", err.Error()))
	}
	health.New(lb).Start(stop)

	engine := &proxy.Engine{Table: table, LB: lb, Metric: metric, Circuit: breaks}
	adminSrv := &admin.Server{Table: table, LB: lb, Metric: metric, Reloader: rl, Circuit: breaks}

	gw := &http.Server{Addr: cfg.Listen, Handler: engine, ReadHeaderTimeout: 5 * time.Second}
	ad := &http.Server{Addr: cfg.AdminListen, Handler: adminSrv.Handler(), ReadHeaderTimeout: 5 * time.Second}

	go func() {
		log.Info("gateway listen", slog.String("addr", cfg.Listen))
		if err := gw.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("gateway exit", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()
	go func() {
		log.Info("admin listen", slog.String("addr", cfg.AdminListen))
		if err := ad.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("admin exit", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = gw.Shutdown(ctx)
	_ = ad.Shutdown(ctx)
	log.Info("minigate stopped")
}
