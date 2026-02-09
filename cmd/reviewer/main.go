package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/groall/upsource-ai-reviewer/internal/review"
	"github.com/groall/upsource-ai-reviewer/pkg/config"
)

func main() {
	var configFile string // main configuration file
	flag.StringVar(&configFile, "config", "config.yaml", "path to config file")
	flag.Parse()

	var appConfig *config.Config
	var err error
	// Load configuration from YAML
	if appConfig, err = config.LoadConfig(configFile); err != nil {
		log.Fatalf("Unable to load config from %s: %v", configFile, err)
	}

	// Validate configuration
	if err = config.ValidateConfig(appConfig); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	ctx := context.Background()
	reviewer, err := review.New(ctx, appConfig)
	if err != nil {
		log.Fatalf("Failed to create reviewer: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Get polling interval from config
	interval := time.Duration(appConfig.Polling.IntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting AI Reviewer service (polling every %v)...", interval)

	// Run immediately on startup
	if err := reviewer.Run(); err != nil {
		log.Printf("Error during review: %v", err)
	}

	// Run the reviewer in a loop
	for {
		select {
		case <-ticker.C:
			log.Println("Checking for new reviews...")
			if err := reviewer.Run(); err != nil {
				log.Printf("Error during review: %v", err)
			}
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			return
		}
	}
}
