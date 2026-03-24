package config

import (
	"encoding/json"
	"os"
)

// AppConfig represents the root structure of config.json
type AppConfig struct {
	ActiveCoach    string               `json:"active_coach"`
	GWSPath        string               `json:"gws_path"`
	Coaches        map[string]CoachProfile `json:"coaches"`
	Workshop       WorkshopConfig       `json:"workshop"`
	Drive          DriveConfig          `json:"google_drive"`
	GmailDiscovery GmailDiscoveryConfig `json:"gmail_discovery"`
}

type GmailDiscoveryConfig struct {
	SenderName    string `json:"sender_name"`
	NewerThanDays int    `json:"newer_than_days"`
}

type EmailTemplate struct {
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	IncludeCC bool   `json:"include_cc"`
	Type      string `json:"type"`
	SortOrder int    `json:"sort_order"`
}

type CoachProfile struct {
	Name           string                                  `json:"name"`
	Signature      string                                  `json:"signature"`
	GoogleAccount  string                                  `json:"google_account"`
	GmailAccount   string                                  `json:"gmail_account"`
	DraftedFrom    string                                  `json:"drafted_emails_from"`
	EmailTemplates map[string]map[string]EmailTemplate `json:"email_templates"`
}

type WorkshopConfig struct {
	CCEmails []string `json:"cc_emails"`
}

type DriveConfig struct {
	WorkshopParentFolderID string `json:"workshop_parent_folder_id"`
	TeamsFolderID          string `json:"teams_folder_id"`
	Templates              struct {
		IndividualSkillsWorksheetID string `json:"individual_skills_worksheet_id"`
		TeamTrainingPlanID          string `json:"team_training_plan_id"`
	} `json:"templates"`
}

// LoadConfig reads and unmarshals the configuration payload from disk
func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig gracefully serializes the runtime configuration block entirely back to JSON.
func SaveConfig(path string, cfg *AppConfig) error {
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0644)
}
