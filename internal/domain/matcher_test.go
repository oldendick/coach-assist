package domain

import (
	"testing"
)

// Reusable student directory fixture matching the real roster data patterns
var testStudents = map[string]string{
	"Daniel Barney":   "daniel@example.com",
	"Kyle Henderson":  "kyle@example.com",
	"Joe Nguyen":      "joe@example.com",
	"Kelsea Kiene":    "kelsea@example.com",
	"Daniel Winter":   "danw@example.com",
	"John Paul Smith": "jp@example.com",
}

func TestMatchEmails_DirectMatch(t *testing.T) {
	result := MatchEmails("Daniel Barney", testStudents)
	if len(result) != 1 || result[0] != "daniel@example.com" {
		t.Errorf("Direct match failed: got %v", result)
	}
}

func TestMatchEmails_DirectMatchCaseInsensitive(t *testing.T) {
	result := MatchEmails("daniel barney", testStudents)
	if len(result) != 1 || result[0] != "daniel@example.com" {
		t.Errorf("Case-insensitive direct match failed: got %v", result)
	}
}

func TestMatchEmails_FirstPlusLastInitial(t *testing.T) {
	result := MatchEmails("Kyle H", testStudents)
	if len(result) != 1 || result[0] != "kyle@example.com" {
		t.Errorf("First + last initial match failed: got %v", result)
	}
}

func TestMatchEmails_PrefixMatch(t *testing.T) {
	result := MatchEmails("Dan B", testStudents)

	// Should match Daniel Barney but NOT Daniel Winter
	if len(result) != 1 || result[0] != "daniel@example.com" {
		t.Errorf("Prefix match failed: got %v", result)
	}
}

func TestMatchEmails_PrefixMatchDifferentLastName(t *testing.T) {
	result := MatchEmails("Dan Wi", testStudents)
	if len(result) != 1 || result[0] != "danw@example.com" {
		t.Errorf("Prefix match (Dan Wi -> Daniel Winter) failed: got %v", result)
	}
}

func TestMatchEmails_InitialsMatch(t *testing.T) {
	result := MatchEmails("jp", testStudents)
	if len(result) != 1 || result[0] != "jp@example.com" {
		t.Errorf("Initials match failed: got %v", result)
	}
}

func TestMatchEmails_FirstNameFallback(t *testing.T) {
	result := MatchEmails("kelsea", testStudents)
	if len(result) != 1 || result[0] != "kelsea@example.com" {
		t.Errorf("First name fallback failed: got %v", result)
	}
}

func TestMatchEmails_MultiPersonGroup(t *testing.T) {
	result := MatchEmails("Kyle H / Dan B 4way", testStudents)
	if len(result) != 2 {
		t.Fatalf("Multi-person group should match 2 emails, got %d: %v", len(result), result)
	}
	emails := map[string]bool{}
	for _, e := range result {
		emails[e] = true
	}
	if !emails["kyle@example.com"] || !emails["daniel@example.com"] {
		t.Errorf("Multi-person group match wrong: got %v", result)
	}
}

func TestMatchEmails_ThreePersonGroup(t *testing.T) {
	result := MatchEmails("Kyle H / Dan B / Kelsea 3way", testStudents)
	if len(result) != 3 {
		t.Fatalf("3-way group should match 3 emails, got %d: %v", len(result), result)
	}
}

func TestMatchEmails_NoMatch(t *testing.T) {
	result := MatchEmails("Nobody Real", testStudents)
	if len(result) != 0 {
		t.Errorf("Expected no matches, got %v", result)
	}
}

func TestMatchEmails_EmptyInput(t *testing.T) {
	result := MatchEmails("", testStudents)
	if len(result) != 0 {
		t.Errorf("Expected no matches for empty input, got %v", result)
	}
}

func TestMatchEmails_NoDuplicates(t *testing.T) {
	// "Daniel" as first name could match both Daniel Barney and Daniel Winter
	result := MatchEmails("Daniel / Daniel", testStudents)
	seen := map[string]int{}
	for _, e := range result {
		seen[e]++
	}
	for email, count := range seen {
		if count > 1 {
			t.Errorf("Duplicate email found: %s appeared %d times", email, count)
		}
	}
}
