package state

import (
	"encoding/json"
	"os"

	"github.com/oldendick/coach-assist/internal/domain"
)

// AppState persistently wraps dynamic file metadata securely preserving context across reboots.
type AppState struct {
	LastScheduleSubject  string              `json:"last_schedule_subject"`
	LastScheduleFilename string              `json:"last_schedule_filename"`
	LastRosterSubject    string              `json:"last_roster_subject"`
	LastRosterFilename   string              `json:"last_roster_filename"`
	Assignments          []domain.FlightPlan `json:"assignments"`
}

// LoadState seamlessly recovers cached artifacts. Returns empty struct if none exist.
func LoadState(path string) AppState {
	var s AppState
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

// SaveState aggressively marshals the memory map into physical persistence layers.
func SaveState(path string, s AppState) error {
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path, data, 0644)
}
