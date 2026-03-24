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

	fmt.Print("Warming up Google Workspace Service engine... ")
	driveSvc := drive.NewGWSClient(cfg)
	if err := driveSvc.Probe(); err != nil {
		fmt.Printf("\n[!] Connectivity Probe Failed: %v\n", err)

		// Interactive login attempt
		if loginErr := driveSvc.Login(); loginErr != nil {
			log.Fatalf("Authentication failed: %v", loginErr)
		}

		// Retry probe after login
		fmt.Print("Re-verifying connectivity... ")
		if err := driveSvc.Probe(); err != nil {
			log.Fatalf("\nConnectivity probe failed after login: %v", err)
		}
	}
	fmt.Println("Done.")

	RunTUI(cfg, driveSvc, Version)
}
