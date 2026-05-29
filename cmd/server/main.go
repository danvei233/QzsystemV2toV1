package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	adminapp "xiaoheiproxy/internal/app/admin"
	gatewayapp "xiaoheiproxy/internal/app/gateway"
	"xiaoheiproxy/internal/domain"
	cacheinfra "xiaoheiproxy/internal/infrastructure/cache"
	"xiaoheiproxy/internal/infrastructure/config"
	httpadapter "xiaoheiproxy/internal/infrastructure/http"
	"xiaoheiproxy/internal/infrastructure/repository"
	"xiaoheiproxy/internal/infrastructure/upstream"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logRepo, metaRepo, err := repository.OpenGatewayRepositories(cfg.Database.DSN, domain.LogRetentionPolicy{
		RetentionDays: cfg.Logs.Request.RetentionDays,
		MaxSizeBytes:  cfg.Logs.Request.MaxSizeMB * 1024 * 1024,
	})
	if err != nil {
		log.Fatalf("open log store: %v", err)
	}
	upstreamClient := upstream.NewClient(cfg.Gateway.UpstreamBaseURL, cfg.Gateway.UpstreamAPIKey, cfg.Gateway.Timeout)
	memCache := cacheinfra.NewMemoryCache()
	gatewaySvc := gatewayapp.NewService(upstreamClient, logRepo, metaRepo, memCache, cfg.Gateway.CacheTTL, cfg.Gateway.MaxBodyBytes, cfg.Gateway.RedactFields)
	adminSvc := adminapp.NewService(cfg.Admin.Username, cfg.Admin.Password, cfg.Admin.SessionTTL, logRepo)
	router := httpadapter.NewRouter(httpadapter.RouterDeps{
		Config:     cfg,
		ConfigPath: *configPath,
		Gateway:    gatewaySvc,
		Admin:      adminSvc,
		Version:    version,
		Commit:     commit,
		BuildTime:  buildTime,
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("xiaoheiproxy listening on %s", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
