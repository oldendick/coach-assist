package drive

// MockWorkspaceService is a testable implementation of WorkspaceService.
// It allows overriding specific methods with custom functions for unit testing.
type MockWorkspaceService struct {
	ProbeFunc                    func() error
	LoginFunc                    func() error
	DownloadLatestAttachmentFunc func(subjectQuery, destFilename string, log func(string)) (string, error)
	ListFolderContentsFunc       func(parentFolderID string) ([]DriveItem, error)
	ExportFileFunc               func(id, mimeType, destPath string) error
	DownloadFileFunc             func(id, destPath string) error
	CreateFolderFunc             func(parentID, name string) (string, error)
	CopyFileFunc                 func(fileID, parentID, newName string) (string, error)
	CreatePermissionFunc         func(fileID, role, pType string) error
	SearchMessagesFunc           func(query string) ([]MessageSummary, error)
	GetMessageAttachmentsFunc    func(messageID string) ([]AttachmentInfo, error)
	DownloadAttachmentFunc       func(messageID, attachmentID, destFilename string) error
	SearchFilesFunc              func(query string) ([]DriveItem, error)
	UpdateSheetValuesFunc        func(spreadsheetID string, updates []SheetUpdate) error
	GetSheetValuesFunc           func(spreadsheetID, rangeStr string) ([][]interface{}, error)
	GetSpreadsheetMetadataFunc   func(spreadsheetID string) (*SpreadsheetMetadata, error)
	CreateDraftFunc              func(from, to, cc, subject, body string) error
	GetFileInfoFunc              func(id string) (DriveItem, error)
}

func (m *MockWorkspaceService) Probe() error {
	if m.ProbeFunc != nil {
		return m.ProbeFunc()
	}
	return nil
}

func (m *MockWorkspaceService) Login() error {
	if m.LoginFunc != nil {
		return m.LoginFunc()
	}
	return nil
}

func (m *MockWorkspaceService) DownloadLatestAttachment(subjectQuery, destFilename string, log func(string)) (string, error) {
	if m.DownloadLatestAttachmentFunc != nil {
		return m.DownloadLatestAttachmentFunc(subjectQuery, destFilename, log)
	}
	return "", nil
}

func (m *MockWorkspaceService) ListFolderContents(parentFolderID string) ([]DriveItem, error) {
	if m.ListFolderContentsFunc != nil {
		return m.ListFolderContentsFunc(parentFolderID)
	}
	return nil, nil
}

func (m *MockWorkspaceService) ExportFile(id, mimeType, destPath string) error {
	if m.ExportFileFunc != nil {
		return m.ExportFileFunc(id, mimeType, destPath)
	}
	return nil
}

func (m *MockWorkspaceService) DownloadFile(id, destPath string) error {
	if m.DownloadFileFunc != nil {
		return m.DownloadFileFunc(id, destPath)
	}
	return nil
}

func (m *MockWorkspaceService) CreateFolder(parentID, name string) (string, error) {
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(parentID, name)
	}
	return "mock-folder-id", nil
}

func (m *MockWorkspaceService) CopyFile(fileID, parentID, newName string) (string, error) {
	if m.CopyFileFunc != nil {
		return m.CopyFileFunc(fileID, parentID, newName)
	}
	return "mock-file-id", nil
}

func (m *MockWorkspaceService) CreatePermission(fileID, role, pType string) error {
	if m.CreatePermissionFunc != nil {
		return m.CreatePermissionFunc(fileID, role, pType)
	}
	return nil
}

func (m *MockWorkspaceService) SearchMessages(query string) ([]MessageSummary, error) {
	if m.SearchMessagesFunc != nil {
		return m.SearchMessagesFunc(query)
	}
	return nil, nil
}

func (m *MockWorkspaceService) GetMessageAttachments(messageID string) ([]AttachmentInfo, error) {
	if m.GetMessageAttachmentsFunc != nil {
		return m.GetMessageAttachmentsFunc(messageID)
	}
	return nil, nil
}

func (m *MockWorkspaceService) DownloadAttachment(messageID, attachmentID, destFilename string) error {
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(messageID, attachmentID, destFilename)
	}
	return nil
}

func (m *MockWorkspaceService) SearchFiles(query string) ([]DriveItem, error) {
	if m.SearchFilesFunc != nil {
		return m.SearchFilesFunc(query)
	}
	return nil, nil
}

func (m *MockWorkspaceService) UpdateSheetValues(spreadsheetID string, updates []SheetUpdate) error {
	if m.UpdateSheetValuesFunc != nil {
		return m.UpdateSheetValuesFunc(spreadsheetID, updates)
	}
	return nil
}

func (m *MockWorkspaceService) GetSheetValues(spreadsheetID, rangeStr string) ([][]interface{}, error) {
	if m.GetSheetValuesFunc != nil {
		return m.GetSheetValuesFunc(spreadsheetID, rangeStr)
	}
	return nil, nil
}

func (m *MockWorkspaceService) GetSpreadsheetMetadata(spreadsheetID string) (*SpreadsheetMetadata, error) {
	if m.GetSpreadsheetMetadataFunc != nil {
		return m.GetSpreadsheetMetadataFunc(spreadsheetID)
	}
	return &SpreadsheetMetadata{}, nil
}

func (m *MockWorkspaceService) CreateDraft(from, to, cc, subject, body string) error {
	if m.CreateDraftFunc != nil {
		return m.CreateDraftFunc(from, to, cc, subject, body)
	}
	return nil
}

func (m *MockWorkspaceService) GetFileInfo(id string) (DriveItem, error) {
	if m.GetFileInfoFunc != nil {
		return m.GetFileInfoFunc(id)
	}
	return DriveItem{ID: id, Name: "Mock File"}, nil
}
