package domain

// ScheduleRow represents a single parsed row from the tunnel schedule spreadsheet.
type ScheduleRow struct {
	Date   string // E.g., "Friday" or "2026-03-06"
	MeetAt   string // E.g., "10:15 AM"
	FlyingAt string // E.g., "10:30 AM"
	Coach1 string
	Group1 string
	Coach2 string
	Group2 string
}
