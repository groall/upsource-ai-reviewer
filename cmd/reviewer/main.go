package main

import (
	"context"
	"flag"
	"log"

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

	// Run the reviewer
	log.Println("Starting AI Reviewer...")
	err = reviewer.Run()
	if err != nil {
		log.Fatalf("Failed to run reviewer: %v", err)
	}
	log.Println("AI Reviewer finished successfully.")
}
