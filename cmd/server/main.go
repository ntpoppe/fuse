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

	_ "modernc.org/sqlite"

	"github.com/ntpoppe/fuse/internal/api"
	"github.com/ntpoppe/fuse/internal/config"
	connectionmanager "github.com/ntpoppe/fuse/internal/connection_manager"
	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/storage"
)

func main() {
	config := config.NewConfig()
	parseFlags(config)

	stateDB, err := sql.Open("sqlite", "fuse.db")
	if err != nil {
		log.Fatalf("failed to open local state store file database: %v", err)
	}
	defer stateDB.Close()

	store := storage.NewStore(stateDB)
	if err := store.InitializeSchema(); err != nil {
		log.Fatalf("failed to verify schema migrations on local state database: %v", err)
	}

	reg := registry.NewRegistry()
	cm := connectionmanager.NewConnectionManager(reg)
	exec := executor.NewExecutor(reg)

	initCtx, initCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer initCancel()

	savedPools, err := store.GetAllConnections(initCtx)
	if err != nil {
		log.Printf("warning: failed to restore historical database targets configuration: %v", err)
	}

	for _, p := range savedPools {
		log.Printf("registering connection pool for target %q", p.ID)
		if err := cm.RegisterConnection(p.ID, p.Driver, p.Host); err != nil {
			log.Printf("failed to register connection pool for target %q: %v", p.ID, err)
		}
	}

	router := api.NewRouter(cm, store, exec)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown, missed deadline: %v\n", err)
	}

	fmt.Println("server exiting")
}

func parseFlags(config *config.Config) {
	flag.IntVar(&config.Port, "port", 5000, "port to listen on")
	flag.StringVar(&config.Env, "env", "dev", "environment to run on")

	flag.Parse()

	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("port: ", config.Port)
	fmt.Println("env: ", config.Env)
}
