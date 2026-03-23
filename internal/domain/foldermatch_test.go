package domain

import "testing"

func TestFolderMatch_Exact(t *testing.T) {
	if !FolderMatchesAssignment("Daniel Barney", "Daniel Barney") {
		t.Error("Exact match should succeed")
	}
}

func TestFolderMatch_CaseInsensitive(t *testing.T) {
	if !FolderMatchesAssignment("daniel barney", "Daniel Barney") {
		t.Error("Case-insensitive match should succeed")
	}
}

func TestFolderMatch_Containment(t *testing.T) {
	if !FolderMatchesAssignment("Kyle H / Joe N / Dan B 4way - Training Plan", "Kyle H / Joe N / Dan B 4way") {
		t.Error("Folder containing assignment name should match")
	}
}

func TestFolderMatch_TokenOverlap(t *testing.T) {
	// Folder named slightly differently but most tokens overlap
	if !FolderMatchesAssignment("Kyle H Joe N Dan B 4way", "Kyle H / Joe N / Dan B 4way") {
		t.Error("Token overlap should match when ≥60% tokens shared")
	}
}

func TestFolderMatch_PartialGroup(t *testing.T) {
	// Coach named the folder with just 2 of 3 names but added extra
	if !FolderMatchesAssignment("Kyle H Dan B 4way", "Kyle H / Joe N / Dan B 4way") {
		t.Error("Partial token overlap ≥60% should still match")
	}
}

func TestFolderMatch_NoMatch(t *testing.T) {
	if FolderMatchesAssignment("Totally Different Person", "Daniel Barney") {
		t.Error("Unrelated names should not match")
	}
}

func TestFolderMatch_EmptyInput(t *testing.T) {
	if FolderMatchesAssignment("", "Daniel Barney") {
		t.Error("Empty folder name should not match")
	}
}
