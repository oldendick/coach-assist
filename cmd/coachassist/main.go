package main

import (
	"fmt"
	"log"
	"os"

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

	fmt.Print("Warming up Google Workspace Service engine... ")
	driveSvc := drive.NewGWSClient(cfg)
	if err := driveSvc.Probe(); err != nil {
		fmt.Printf("\n\n[!] Connectivity Probe Failed: %v\n", err)
		fmt.Println("    Please ensure you are authenticated by running:")
		fmt.Println("    gws auth login")
		os.Exit(1)
	}
	fmt.Println("Done.")

	RunTUI(cfg, driveSvc, Version)
}
