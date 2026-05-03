package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ntpoppe/fuse/internal/api"
	"github.com/ntpoppe/fuse/internal/config"
)

func main() {
	config := config.NewConfig()
	parseFlags(config)

	router := api.NewRouter()

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

	// listen for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// blocks until signal is received
	<-quit
	fmt.Println("shutting down server...")

	// create context with timeout, grace period for existing requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// shutdown the server under timeout context
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
	}

	fmt.Println("port: ", config.Port)
	fmt.Println("env: ", config.Env)
}
