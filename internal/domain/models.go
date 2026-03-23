package domain

// FlightPlan represents a pending or completed coaching engagement for a weekend
type FlightPlan struct {
	SubjectName   string   // E.g., "Daniel Barney" or "Kyle H / Joe N / Dan B 4way"
	IsGroup       bool
	IsReserved    bool     // True if "(Coach)" style reserved placeholder
	ArrivalDay    string   // E.g., "Friday"
	ArrivalTime   string   // E.g., "10:15 AM"
	SubjectEmails []string // Dynamically mapped via heuristic substring searches
	MakingDivesCoach string // The coach responsible for creating the training plan for this group

	
	// State populated during Google Drive Automation
	DriveFolderID string 
	DriveFileID   string 
	ShareableLink string 
	
	// Workflow State Tracking
	IsWorkspaceCreated bool
	HasTrainingPlan    bool
	IsDiscoveryDrafted bool
	IsFinalPlanDrafted bool
}
