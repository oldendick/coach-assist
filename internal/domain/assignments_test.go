package domain

import "testing"

func TestBuildAssignments_FiltersCorrectCoach(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Friday", MeetAt: "10:15 AM", Coach1: "Greg", Group1: "Daniel Barney", Coach2: "Doug", Group2: "Kyle H / Joe N 4way"},
		{Date: "Friday", MeetAt: "11:00 AM", Coach1: "Doug", Group1: "Kelsea Kiene", Coach2: "Greg", Group2: "Test Student"},
	}
	students := map[string]string{
		"Daniel Barney": "dan@test.com",
		"Test Student":  "test@test.com",
	}

	plans := BuildAssignments("Greg", schedule, nil, students)

	if len(plans) != 2 {
		t.Fatalf("Expected 2 assignments for Greg, got %d", len(plans))
	}
	if plans[0].SubjectName != "Daniel Barney" {
		t.Errorf("First assignment should be Daniel Barney, got %s", plans[0].SubjectName)
	}
	if plans[1].SubjectName != "Test Student" {
		t.Errorf("Second assignment should be Test Student, got %s", plans[1].SubjectName)
	}
}

func TestBuildAssignments_DeduplicatesGroups(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Friday", MeetAt: "10:15 AM", Coach1: "Greg", Group1: "Daniel Barney"},
		{Date: "Friday", MeetAt: "11:00 AM", Coach1: "Greg", Group1: "Daniel Barney"},
		{Date: "Saturday", MeetAt: "09:00 AM", Coach1: "Greg", Group1: "Daniel Barney"},
	}

	plans := BuildAssignments("Greg", schedule, nil, nil)
	if len(plans) != 1 {
		t.Errorf("Expected 1 deduplicated assignment, got %d", len(plans))
	}
	if plans[0].ArrivalTime != "10:15 AM" {
		t.Errorf("Should keep first occurrence time, got %s", plans[0].ArrivalTime)
	}
}

func TestBuildAssignments_ClassifiesGroups(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Friday", MeetAt: "10:00 AM", Coach1: "Greg", Group1: "Kyle H / Dan B 4way"},
		{Date: "Friday", MeetAt: "11:00 AM", Coach1: "Greg", Group1: "Kelsea Kiene"},
	}
	makingDivesMap := map[string]string{"Kyle H / Dan B 4way": "Greg"}

	plans := BuildAssignments("Greg", schedule, makingDivesMap, nil)

	if len(plans) != 2 {
		t.Fatalf("Expected 2 plans, got %d", len(plans))
	}
	if !plans[0].IsGroup {
		t.Error("4way should be classified as group")
	}
	if plans[1].IsGroup {
		t.Error("Solo should not be classified as group")
	}
}

func TestBuildAssignments_Coach2Column(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Saturday", MeetAt: "09:00 AM", Coach1: "Doug", Group1: "Someone", Coach2: "Greg", Group2: "Back Seat"},
	}

	plans := BuildAssignments("Greg", schedule, nil, nil)
	if len(plans) != 1 || plans[0].SubjectName != "Back Seat" {
		t.Errorf("Should pick up Greg's assignment from Coach2 column, got %v", plans)
	}
}

func TestBuildAssignments_MultiCoachSlash(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Friday", MeetAt: "10:00 AM", Coach1: "Greg / Doug", Group1: "Team Alpha 4way"},
	}

	plans := BuildAssignments("Greg", schedule, nil, nil)
	if len(plans) != 1 {
		t.Fatalf("Expected 1 assignment from multi-coach field, got %d", len(plans))
	}
	if plans[0].SubjectName != "Team Alpha 4way" {
		t.Errorf("Expected Team Alpha 4way, got %s", plans[0].SubjectName)
	}

	// Doug should also match
	plans2 := BuildAssignments("Doug", schedule, nil, nil)
	if len(plans2) != 1 {
		t.Fatalf("Doug should also match multi-coach field, got %d", len(plans2))
	}
}

func TestBuildAssignments_ReservedSlot(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Friday", MeetAt: "10:00 AM", Coach1: "Doug", Group1: "(Greg)", Coach2: "Steve", Group2: "Real Student"},
	}

	plans := BuildAssignments("Greg", schedule, nil, nil)
	if len(plans) != 1 {
		t.Fatalf("Expected 1 reserved assignment, got %d", len(plans))
	}
	if !plans[0].IsReserved {
		t.Error("(Greg) should be flagged as reserved")
	}
	if plans[0].SubjectName != "(Greg)" {
		t.Errorf("SubjectName should be (Greg), got %s", plans[0].SubjectName)
	}
}

func TestBuildAssignments_ReservedSlotNotMatchOther(t *testing.T) {
	schedule := []ScheduleRow{
		{Date: "Friday", MeetAt: "10:00 AM", Coach1: "Doug", Group1: "(Greg)"},
	}

	// Doug shouldn't pick up (Greg) as his assignment
	plans := BuildAssignments("Doug", schedule, nil, nil)
	found := false
	for _, p := range plans {
		if p.SubjectName == "(Greg)" {
			found = true
		}
	}
	if found {
		t.Error("Doug should NOT pick up (Greg) as his assignment")
	}
}
