package ingest

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func createTestExcel(t *testing.T, filename string, sheets map[string][][]interface{}) string {
	f := excelize.NewFile()
	
	// excelize creates "Sheet1" by default. 
	// We'll rename it or delete it later to match the requested sheet names.
	firstSheet := true
	for name, rows := range sheets {
		var sheetName string
		if firstSheet {
			sheetName = "Sheet1"
			f.SetSheetName("Sheet1", name)
			firstSheet = false
		} else {
			f.NewSheet(name)
		}
		sheetName = name
		
		for r, row := range rows {
			cell, _ := excelize.CoordinatesToCellName(1, r+1)
			f.SetSheetRow(sheetName, cell, &row)
		}
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, filename)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to save test excel: %v", err)
	}
	return path
}

func TestParseSchedule(t *testing.T) {
	data := map[string][][]interface{}{
		"Schedule": {
			{"Some random header info"},
			{"Date", "Meet at", "Flying at", "Coach 1", "Group 1", "Coach 2", "Group 2"},
			{"Friday", "10:00 AM", "11:00 AM", "Greg", "Solo Student", "", ""},
			{"", "11:00 AM", "12:00 PM", "Doug", "Team Alpha", "Greg", "Back Seat"},
			{"Saturday", "09:00 AM", "10:00 AM", "Steve", "Another Solo", "", ""},
		},
	}
	path := createTestExcel(t, "schedule.xlsx", data)

	rows, err := ParseSchedule(path)
	if err != nil {
		t.Fatalf("ParseSchedule failed: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("Expected 3 rows, got %d", len(rows))
	}

	// Check forward-fill
	if rows[1].Date != "Friday" {
		t.Errorf("Expected forward-filled Date 'Friday', got '%s'", rows[1].Date)
	}
	if rows[2].Date != "Saturday" {
		t.Errorf("Expected Date 'Saturday', got '%s'", rows[2].Date)
	}

	if rows[1].Coach2 != "Greg" {
		t.Errorf("Expected Coach 2 'Greg', got '%s'", rows[1].Coach2)
	}
}

func TestParseRoster(t *testing.T) {
	// Sheets must be in specific order for ParseGroupAssignments (0) and ParseStudentEmails (1)
	f := excelize.NewFile()
	
	// Sheet 1: Group Assignments
	f.SetSheetName("Sheet1", "Group Assignments")
	f.SetSheetRow("Group Assignments", "A1", &[]interface{}{"Team", "Who makes dives"})
	f.SetSheetRow("Group Assignments", "A2", &[]interface{}{"Team Alpha", "Greg"})
	f.SetSheetRow("Group Assignments", "A3", &[]interface{}{"Solo Student", "Self"})

	// Sheet 2: Student list
	f.NewSheet("Student list")
	f.SetSheetRow("Student list", "A1", &[]interface{}{"Who", "email", "Phone"})
	f.SetSheetRow("Student list", "A2", &[]interface{}{"Solo Student", "solo@test.com", "555-1234"})
	f.SetSheetRow("Student list", "A3", &[]interface{}{"Team Alpha", "team@test.com", ""})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "roster.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to save test excel: %v", err)
	}

	// Test Group Assignments (Sheet 1)
	groups, err := ParseGroupAssignments(path)
	if err != nil {
		t.Fatalf("ParseGroupAssignments failed: %v", err)
	}
	if groups["Team Alpha"] != "Greg" {
		t.Errorf("Expected Team Alpha coach to be Greg, got %s", groups["Team Alpha"])
	}

	// Test Student Emails (Sheet 2)
	emails, err := ParseStudentEmails(path)
	if err != nil {
		t.Fatalf("ParseStudentEmails failed: %v", err)
	}
	if emails["Solo Student"] != "solo@test.com" {
		t.Errorf("Expected Solo Student email to be solo@test.com, got %s", emails["Solo Student"])
	}
}
