package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// AppConfig represents the root structure of config.json
type AppConfig struct {
	Version        int                  `json:"version"`
	ActiveCoach    string               `json:"active_coach"`
	GWSPath        string               `json:"gws_path"`
	Coaches        map[string]CoachProfile `json:"coaches"`
	Workshop       WorkshopConfig       `json:"workshop"`
	Drive          DriveConfig          `json:"google_drive"`
	GmailDiscovery GmailDiscoveryConfig `json:"gmail_discovery"`
}

func (c *AppConfig) Validate() error {
	if c.ActiveCoach == "" {
		return errors.New("active_coach is not set in config.json")
	}
	if _, ok := c.Coaches[c.ActiveCoach]; !ok {
		return fmt.Errorf("active_coach '%s' not found in 'coaches' section", c.ActiveCoach)
	}

	for name, coach := range c.Coaches {
		if err := coach.Validate(); err != nil {
			return fmt.Errorf("coach '%s': %w", name, err)
		}
	}

	if c.Drive.WorkshopParentFolderID == "" {
		return errors.New("google_drive.workshop_parent_folder_id is required")
	}
	if c.Drive.TeamsFolderID == "" {
		return errors.New("google_drive.teams_folder_id is required")
	}
	if c.Drive.Templates.IndividualSkillsWorksheetID == "" {
		return errors.New("google_drive.templates.individual_skills_worksheet_id is required")
	}
	if c.Drive.Templates.TeamTrainingPlanID == "" {
		return errors.New("google_drive.templates.team_training_plan_id is required")
	}

	for _, email := range c.Workshop.CCEmails {
		if email != "" && !strings.Contains(email, "@") {
			return fmt.Errorf("invalid workshop CC email address: %s", email)
		}
	}

	return nil
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

func (t EmailTemplate) Validate() error {
	if t.Subject == "" {
		return errors.New("subject is empty")
	}
	if t.Body == "" {
		return errors.New("body is empty")
	}
	return nil
}

type CoachProfile struct {
	Name           string                                  `json:"name"`
	Signature      string                                  `json:"signature"`
	GoogleAccount  string                                  `json:"google_account"`
	GmailAccount   string                                  `json:"gmail_account"`
	DraftedFrom    string                                  `json:"drafted_emails_from"`
	EmailTemplates map[string]map[string]EmailTemplate `json:"email_templates"`
}

func (p CoachProfile) Validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.GoogleAccount == "" {
		return errors.New("google_account is required")
	}
	if !strings.Contains(p.GoogleAccount, "@") {
		return fmt.Errorf("invalid google_account: %s", p.GoogleAccount)
	}
	if p.GmailAccount != "" && !strings.Contains(p.GmailAccount, "@") {
		return fmt.Errorf("invalid gmail_account: %s", p.GmailAccount)
	}

	for cat, tmpls := range p.EmailTemplates {
		for name, tmpl := range tmpls {
			if err := tmpl.Validate(); err != nil {
				return fmt.Errorf("template '%s' in category '%s': %w", name, cat, err)
			}
		}
	}
	return nil
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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validaton error: %w", err)
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
