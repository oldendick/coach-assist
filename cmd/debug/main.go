package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oldendick/coach-assist/internal/config"
	"github.com/oldendick/coach-assist/internal/ingest"
)

func main() {
	cfg, _ := config.LoadConfig("config.json")
	coachName := cfg.Coaches[cfg.ActiveCoach].Name
	fmt.Printf("Active Coach Name: '%s'\n\n", coachName)

	schedPath := filepath.Join("artifacts", "latest-schedule.xlsx")
	if _, err := os.Stat(schedPath); err != nil {
		fmt.Printf("No schedule file at %s\n", schedPath)
		return
	}

	rows, err := ingest.ParseSchedule(schedPath)
	if err != nil {
		fmt.Printf("ParseSchedule error: %v\n", err)
		return
	}

	fmt.Printf("Parsed %d schedule rows\n\n", len(rows))
	for i, r := range rows {
		if i > 10 {
			fmt.Println("... (truncated)")
			break
		}
		fmt.Printf("Row %d: Date=%q MeetAt=%q Coach1=%q Group1=%q Coach2=%q Group2=%q\n",
			i, r.Date, r.MeetAt, r.Coach1, r.Group1, r.Coach2, r.Group2)
	}

	fmt.Printf("\n--- Rows matching coach '%s' ---\n", coachName)
	count := 0
	for _, r := range rows {
		if r.Coach1 == coachName || r.Coach2 == coachName {
			count++
			fmt.Printf("  MATCH: Group1=%q (Coach1=%q) | Group2=%q (Coach2=%q)\n", r.Group1, r.Coach1, r.Group2, r.Coach2)
		}
	}
	fmt.Printf("Total matches: %d\n", count)
}
