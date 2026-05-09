package drive

import (
	"testing"
)

// TestMockCreateDraft demonstrates how to use the MockWorkspaceService
// to test logic that interacts with Google APIs without needing a real account.
func TestMockCreateDraft(t *testing.T) {
	var capturedTo string
	called := false

	mock := &MockWorkspaceService{
		CreateDraftFunc: func(from, to, cc, subject, body string) error {
			called = true
			capturedTo = to
			return nil
		},
	}

	// This simulates a call from the TUI or other business logic
	err := mock.CreateDraft("coach@rhythmskydiving.com", "student@example.com", "team@example.com", "Subject", "Body")
	
	if err != nil {
		t.Fatalf("Expected no error from mock, got %v", err)
	}
	if !called {
		t.Error("Expected Mock CreateDraft to be called, but it wasn't")
	}
	if capturedTo != "student@example.com" {
		t.Errorf("Expected recipient 'student@example.com', got '%s'", capturedTo)
	}
}

func TestMockCreateFolder(t *testing.T) {
	mock := &MockWorkspaceService{
		CreateFolderFunc: func(parentID, name string) (string, error) {
			return "new-folder-123", nil
		},
	}

	id, err := mock.CreateFolder("parent-id", "Weekend Folder")
	if err != nil {
		t.Fatalf("Mock failed: %v", err)
	}
	if id != "new-folder-123" {
		t.Errorf("Expected new-folder-123, got %s", id)
	}
}
