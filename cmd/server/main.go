package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ntpoppe/fuse/internal/api"
	"github.com/ntpoppe/fuse/internal/config"
	"github.com/ntpoppe/fuse/internal/demo"
	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/storage"
)

const (
	serverReadTimeout  = 5 * time.Second
	serverWriteTimeout = 10 * time.Second
	serverIdleTimeout  = 120 * time.Second
	shutdownTimeout    = 30 * time.Second
	restoreTimeout     = 5 * time.Second
)

func main() {
	cfg := config.NewConfig()
	parseFlags(cfg)

	stateDB, err := sql.Open(driver.DriverSQLite, cfg.StateDBPath)
	if err != nil {
		log.Fatalf("open state database: %v", err)
	}
	defer stateDB.Close()

	store := storage.NewStore(stateDB)
	if err := store.InitializeSchema(); err != nil {
		log.Fatalf("initialize state database: %v", err)
	}

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)
	exec := executor.NewExecutor(reg, cfg.MaxQueryRows)
	fedExec := executor.NewFederatedExecutor(reg, cfg.MaxQueryRows)

	initCtx, initCancel := context.WithTimeout(context.Background(), restoreTimeout)
	defer initCancel()

	saved, err := store.GetAllConnections(initCtx)
	if err != nil {
		log.Printf("warning: restore saved connections: %v", err)
	}

	if cfg.DemoMode {
		if err := demo.SeedConnections(initCtx, cm, store, cfg); err != nil {
			log.Fatalf("seed demo connections: %v", err)
		}
	} else {
		for _, conn := range saved {
			log.Printf("restoring connection %q (driver=%q)", conn.ID, conn.Driver)
			if err := cm.RegisterConnection(conn.ID, conn.Driver, conn.Host); err != nil {
				log.Printf("failed to restore connection %q: %v", conn.ID, err)
			}
		}
	}

	router := api.NewRouter(cm, store, exec, fedExec, cfg.DemoMode)
	handler := api.WithCORS(cfg.CORSOrigins, router)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      handler,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	go func() {
		fmt.Printf("starting server on %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	fmt.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v\n", err)
	}

	fmt.Println("server exiting")
}

func parseFlags(cfg *config.Config) {
	config.ApplyEnv(cfg)

	var corsOriginsFlag string

	flag.StringVar(&cfg.Host, "host", cfg.Host, "host address to listen on")
	flag.IntVar(&cfg.Port, "port", config.DefaultPort, "port to listen on")
	flag.StringVar(&cfg.StateDBPath, "state-db", cfg.StateDBPath, "path to local state database")
	flag.IntVar(&cfg.MaxQueryRows, "max-query-rows", cfg.MaxQueryRows, "maximum rows returned per query")
	flag.BoolVar(&cfg.DemoMode, "demo", cfg.DemoMode, "enable demo mode (fixed connections, no connection CRUD)")
	flag.StringVar(&cfg.DemoSQLitePath, "demo-sqlite-path", cfg.DemoSQLitePath, "sqlite database path for demo shop connection")
	flag.StringVar(&cfg.DemoMySQLDSN, "demo-mysql-dsn", cfg.DemoMySQLDSN, "mysql DSN for demo warehouse connection")
	flag.StringVar(&corsOriginsFlag, "cors-origins", "", "comma-separated allowed CORS origins (overrides FUSE_CORS_ORIGINS)")

	flag.Parse()

	if corsOriginsFlag != "" {
		cfg.CORSOrigins = config.SplitCSV(corsOriginsFlag)
	}

	config.ApplyDemoDefaults(cfg)

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}
}
