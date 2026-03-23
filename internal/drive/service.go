package drive

// DriveItem represents a file or folder in Google Drive.
type DriveItem struct {
	Name string
	ID   string
}

// WorkspaceService defines the exact abstract behaviors interacting with external Google dependencies.
type WorkspaceService interface {
	// DownloadLatestAttachment searches Gmail and extracts the attachment to destFilename.
	// Returns the original attachment filename from the email and any error.
	DownloadLatestAttachment(subjectQuery, destFilename string, log func(string)) (originalFilename string, err error)

	// ListFolderContents returns all non-trashed children in a Drive folder.
	ListFolderContents(parentFolderID string) ([]DriveItem, error)

	// ExportFile exports a Google Workspace file (e.g. Google Sheet) as a specific MIME type and saves it to destPath.
	ExportFile(id, mimeType, destPath string) error

	// DownloadFile downloads a raw binary file from Google Drive and saves it to destPath.
	DownloadFile(id, destPath string) error

	// CreateFolder creates a new folder in Google Drive.
	CreateFolder(parentID, name string) (string, error)

	// CopyFile creates a copy of an existing file in the specified parent folder with a new name.
	CopyFile(fileID, parentID, newName string) (string, error)

	// CreatePermission adds a new permission to a file or folder.
	CreatePermission(fileID, role, pType string) error

	// SearchMessages searches Gmail for messages matching a query.
	SearchMessages(query string) ([]MessageSummary, error)

	// GetMessageAttachments returns metadata for all attachments in a message.
	GetMessageAttachments(messageID string) ([]AttachmentInfo, error)

	// DownloadAttachment downloads a specific attachment from a message.
	DownloadAttachment(messageID, attachmentID, destFilename string) error

	// SearchFiles finds files or folders matching a specific Google Drive query.
	SearchFiles(query string) ([]DriveItem, error)

	// UpdateSheetValues updates multiple ranges in a Google Sheet in a single batch.
	UpdateSheetValues(spreadsheetID string, updates []SheetUpdate) error

	// GetSheetValues retrieves values from a specific range in a Google Sheet.
	GetSheetValues(spreadsheetID, rangeStr string) ([][]interface{}, error)

	// GetSpreadsheetMetadata retrieves structural metadata (like merges) for a spreadsheet.
	GetSpreadsheetMetadata(spreadsheetID string) (*SpreadsheetMetadata, error)

	// CreateDraft creates a new draft email in Gmail.
	CreateDraft(from, to, cc, subject, body string) error
}

// MessageSummary represents basic metadata for a Gmail message.
type MessageSummary struct {
	ID      string
	Subject string
	Date    string
	Snippet string
}

// AttachmentInfo represents metadata for a Gmail attachment.
type AttachmentInfo struct {
	ID       string
	Filename string
}

// SpreadsheetMetadata represents the structural metadata of a Google Sheet.
type SpreadsheetMetadata struct {
	Sheets []struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
		Merges []SheetMerge `json:"merges"`
	} `json:"sheets"`
}

// SheetMerge represents a merged cell range in 0-indexed coordinates.
type SheetMerge struct {
	StartRowIndex    int `json:"startRowIndex"`
	EndRowIndex      int `json:"endRowIndex"`
	StartColumnIndex int `json:"startColumnIndex"`
	EndColumnIndex   int `json:"endColumnIndex"`
}

// SheetUpdate represents a single range and value to update in a Google Sheet.
type SheetUpdate struct {
	Range  string
	Values [][]interface{}
}
