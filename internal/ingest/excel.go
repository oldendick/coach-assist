package ingest

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/oldendick/coach-assist/internal/domain"
)

// ParseSchedule reads the tunnel schedule spreadsheet and returns structured rows.
// Mirrors the Python logic: skip 2 header rows, forward-fill the Date column.
func ParseSchedule(path string) ([]domain.ScheduleRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed opening schedule: %w", err)
	}
	defer f.Close()

	// Use the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	// Dynamically find the header row by scanning for the one containing both "Date" and "Coach 1"
	headerIdx := -1
	for i, row := range rows {
		hasDate := false
		hasCoach := false
		for _, cell := range row {
			trimmed := strings.TrimSpace(cell)
			if trimmed == "Date" {
				hasDate = true
			}
			if trimmed == "Coach 1" {
				hasCoach = true
			}
		}
		if hasDate && hasCoach {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, fmt.Errorf("could not find header row containing 'Date' in schedule")
	}

	// Build column index map from the discovered header row
	headerRow := rows[headerIdx]
	colIndex := map[string]int{}
	for i, cell := range headerRow {
		key := strings.ToLower(strings.TrimSpace(cell))
		if key != "" {
			colIndex[key] = i
		}
	}

	// Validate required columns exist
	required := []string{"date", "meet at", "flying at", "coach 1", "group 1"}
	for _, col := range required {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column '%s' (case-insensitive) in schedule header: %v", col, headerRow)
		}
	}

	getCell := func(row []string, colName string) string {
		idx, ok := colIndex[strings.ToLower(colName)]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	var result []domain.ScheduleRow
	lastDate := ""

	// Data rows start after the header
	for _, row := range rows[headerIdx+1:] {
		// Forward-fill: if Date is blank, inherit from previous row
		dateVal := getCell(row, "Date")

		// Skip repeated header rows (Excel merged-cell artifacts)
		if dateVal == "Date" {
			continue
		}
		if dateVal != "" {
			lastDate = formatDate(dateVal)
		}

		meetAt := formatTime(getCell(row, "Meet at"))
		flyingAt := formatTime(getCell(row, "Flying at"))
		coach1 := getCell(row, "Coach 1")
		group1 := getCell(row, "Group 1")
		coach2 := getCell(row, "Coach 2")
		group2 := getCell(row, "Group 2")

		// Skip entirely empty rows
		if coach1 == "" && coach2 == "" && group1 == "" && group2 == "" {
			continue
		}

		result = append(result, domain.ScheduleRow{
			Date:     lastDate,
			MeetAt:   meetAt,
			FlyingAt: flyingAt,
			Coach1:   coach1,
			Group1:   group1,
			Coach2:   coach2,
			Group2:   group2,
		})
	}

	return result, nil
}
// ReadRawExcel reads all rows from a specific sheet of an Excel file.
// If sheetName is empty, the first sheet is used.
func ReadRawExcel(path, sheetName string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if sheetName == "" {
		sheetName = f.GetSheetName(0)
	}
	return f.GetRows(sheetName)
}
// Returns a map of team/group name -> coach name responsible for making dives.
func ParseGroupAssignments(path string) (map[string]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed opening roster: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Who makes dives")
	if err != nil {
		return nil, fmt.Errorf("sheet 'Who makes dives' not found: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("roster sheet has too few rows")
	}

	// Build column map from header row
	colIndex := map[string]int{}
	for i, cell := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(cell))
		if key != "" {
			colIndex[key] = i
		}
	}

	teamIdx, hasTeam := colIndex["team"]
	coachIdx, hasCoach := colIndex["who makes dives"]
	if !hasTeam || !hasCoach {
		return nil, fmt.Errorf("roster missing 'Team' or 'Who makes dives' columns: %v", rows[0])
	}

	result := map[string]string{}
	for _, row := range rows[1:] {
		if coachIdx >= len(row) || teamIdx >= len(row) {
			continue
		}
		coach := strings.TrimSpace(row[coachIdx])
		team := strings.TrimSpace(row[teamIdx])
		if coach != "" && team != "" {
			result[team] = coach
		}
	}

	return result, nil
}

// ParseStudentEmails reads the "Student list" sheet from the roster workbook.
// Returns a map of full name -> email.
func ParseStudentEmails(path string) (map[string]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed opening roster: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Student list")
	if err != nil {
		return nil, fmt.Errorf("sheet 'Student list' not found: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("student list sheet has too few rows")
	}

	colIndex := map[string]int{}
	for i, cell := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(cell))
		if key != "" {
			colIndex[key] = i
		}
	}

	whoIdx, hasWho := colIndex["who"]
	emailIdx, hasEmail := colIndex["email"]
	if !hasWho || !hasEmail {
		return nil, fmt.Errorf("student list missing 'Who' or 'email' columns: %v", rows[0])
	}

	result := map[string]string{}
	for _, row := range rows[1:] {
		if whoIdx >= len(row) || emailIdx >= len(row) {
			continue
		}
		name := strings.TrimSpace(row[whoIdx])
		email := strings.TrimSpace(row[emailIdx])
		if name != "" && email != "" {
			result[name] = email
		}
	}

	return result, nil
}

// formatDate attempts to parse Excel date values into clean day names.
func formatDate(raw string) string {
	// excelize returns dates as strings. Try common formats.
	for _, layout := range []string{
		"01-02-06",        // Excel short date
		"1/2/06",          // US short
		"2006-01-02",      // ISO
		time.DateOnly,     // Go standard
		"January 2, 2006", // Long form
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Weekday().String()
		}
	}
	// If it already looks like a day name, return as-is
	return raw
}

// formatTime cleans up time strings from Excel into readable AM/PM format.
func formatTime(raw string) string {
	if raw == "" {
		return ""
	}
	// Try parsing as various time formats
	for _, layout := range []string{
		"3:04 PM",
		"15:04:05",
		"15:04",
		"3:04:05 PM",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("3:04 PM")
		}
	}
	return raw
}
