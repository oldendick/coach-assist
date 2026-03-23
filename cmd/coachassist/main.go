package main

import (
	"fmt"
	"log"

	"github.com/oldendick/coach-assist/internal/config"
	"github.com/oldendick/coach-assist/internal/drive"
)

var Version = "dev"

func main() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	_, exists := cfg.Coaches[cfg.ActiveCoach]
	if !exists {
		log.Fatalf("Active coach profile '%s' not found.", cfg.ActiveCoach)
	}

	fmt.Println("Warming up Google Workspace Service engine...")
	driveSvc := drive.NewGWSClient(cfg)

	RunTUI(cfg, driveSvc, Version)
}
