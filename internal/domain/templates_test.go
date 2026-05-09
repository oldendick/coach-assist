package domain

import "testing"

func TestPopulateTemplate(t *testing.T) {
	plan := FlightPlan{
		SubjectName: "Daniel Barney",
		ArrivalTime: "10:15 AM",
	}
	folderLink := "https://drive.google.com/test"
	
	template := "Hi {firstname}, your {groupname} plan is at {folder_link}. Meet at {initial_meet_time}."
	expected := "Hi Daniel, your Daniel Barney plan is at https://drive.google.com/test. Meet at 10:15 AM."
	
	result := PopulateTemplate(template, plan, folderLink)
	if result != expected {
		t.Errorf("Template mismatch.\nExpected: %s\nGot:      %s", expected, result)
	}
}
