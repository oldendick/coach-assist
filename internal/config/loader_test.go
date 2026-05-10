package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Create a dummy config
	initialCfg := &AppConfig{
		Version:     1,
		ActiveCoach: "test-coach",
		Coaches: map[string]CoachProfile{
			"test-coach": {
				Name:          "Test Coach",
				GoogleAccount: "test@example.com",
				Signature:     "Initial Signature",
				EmailTemplates: map[string]map[string]EmailTemplate{
					"General": {
						"Welcome": {
							Subject:   "Hello",
							Body:      "Body text",
							Type:      "initial",
							SortOrder: 1,
						},
					},
				},
			},
		},
		Drive: DriveConfig{
			WorkshopParentFolderID: "parent",
			TeamsFolderID:          "teams",
		},
	}
	initialCfg.Drive.Templates.IndividualSkillsWorksheetID = "id1"
	initialCfg.Drive.Templates.TeamTrainingPlanID = "id2"

	err = SaveConfig(configPath, initialCfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load it back
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.Coaches["test-coach"].Signature != "Initial Signature" {
		t.Errorf("Expected signature 'Initial Signature', got '%s'", loadedCfg.Coaches["test-coach"].Signature)
	}

	// Modify and save (simulating the TUI logic)
	coach := loadedCfg.Coaches["test-coach"]
	coach.Signature = "Updated Signature"
	
	tmpls := coach.EmailTemplates["General"]
	tmpl := tmpls["Welcome"]
	tmpl.Body = "Updated Body"
	tmpls["Welcome"] = tmpl
	coach.EmailTemplates["General"] = tmpls
	
	loadedCfg.Coaches["test-coach"] = coach

	err = SaveConfig(configPath, loadedCfg)
	if err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}

	// Verify final state
	finalCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load final config: %v", err)
	}

	if finalCfg.Coaches["test-coach"].Signature != "Updated Signature" {
		t.Errorf("Expected updated signature, got '%s'", finalCfg.Coaches["test-coach"].Signature)
	}
	if finalCfg.Coaches["test-coach"].EmailTemplates["General"]["Welcome"].Body != "Updated Body" {
		t.Errorf("Expected updated body, got '%s'", finalCfg.Coaches["test-coach"].EmailTemplates["General"]["Welcome"].Body)
	}
}

func TestTemplateValidation(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    EmailTemplate
		wantErr bool
	}{
		{"valid", EmailTemplate{Subject: "S", Body: "B"}, false},
		{"empty subject", EmailTemplate{Subject: "", Body: "B"}, true},
		{"empty body", EmailTemplate{Subject: "S", Body: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tmpl.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("EmailTemplate.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
