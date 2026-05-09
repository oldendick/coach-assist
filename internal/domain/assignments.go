package domain

import "strings"

// CoachMatches checks if a coach field contains the target coach name or any alias.
// Handles slash-separated multi-coach fields like "Greg / Doug".
func CoachMatches(coachField, targetCoach string, aliases []string) bool {
	parts := strings.Split(coachField, "/")
	
	// Create a list of all acceptable names
	names := append([]string{strings.ToLower(targetCoach)}, "")
	names = names[:1]
	for _, alias := range aliases {
		names = append(names, strings.ToLower(alias))
	}

	for _, p := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(p))
		for _, name := range names {
			if strings.Contains(trimmed, name) {
				return true
			}
		}
	}
	return false
}

// isReservedSlot checks if a group name is a parenthesized coach placeholder
// like "(Greg)" meaning the slot is held but no planning is needed.
func isReservedSlot(groupName string) bool {
	trimmed := strings.TrimSpace(groupName)
	return strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")")
}

// BuildAssignments filters the schedule for a specific coach and produces FlightPlan entries.
// It cross-references group assignments from the roster to classify solos vs groups,
// and resolves email addresses via fuzzy matching.
//
// Coach matching supports:
//   - Exact match: Coach1="Greg"
//   - Multi-coach: Coach1="Greg / Doug" (2-2 coaching)
//   - Reserved slots: Group1="(Greg)" — matched but flagged as reserved (no planning needed)
func BuildAssignments(
	coachName string,
	aliases []string,
	schedule []ScheduleRow,
	makingDivesMap map[string]string,
	studentEmails map[string]string,
) []FlightPlan {

	// Track which groups we've already seen to capture only first "Meet At"
	seen := map[string]bool{}
	var plans []FlightPlan

	for _, row := range schedule {
		var matchGroup string

		// Check Coach1 column (exact or slash-separated)
		if CoachMatches(row.Coach1, coachName, aliases) {
			matchGroup = strings.TrimSpace(row.Group1)
		}

		// Check Coach2 column
		if matchGroup == "" && CoachMatches(row.Coach2, coachName, aliases) {
			matchGroup = strings.TrimSpace(row.Group2)
		}

		// Helper to check if a string matches coach name or aliases
		isTargetCoach := func(name string) bool {
			lowerName := strings.ToLower(strings.TrimSpace(name))
			if strings.Contains(lowerName, strings.ToLower(coachName)) {
				return true
			}
			for _, alias := range aliases {
				if strings.Contains(lowerName, strings.ToLower(alias)) {
					return true
				}
			}
			return false
		}

		// If the group is a reserved slot for someone else, skip it
		if matchGroup != "" && isReservedSlot(matchGroup) {
			inner := strings.TrimSpace(strings.Trim(matchGroup, "()"))
			if !isTargetCoach(inner) {
				matchGroup = ""
			}
		}

		// Also check if the Group field itself is a "(CoachName)" reserved placeholder
		if matchGroup == "" {
			for _, gField := range []string{row.Group1, row.Group2} {
				trimmed := strings.TrimSpace(gField)
				if isReservedSlot(trimmed) {
					inner := strings.Trim(trimmed, "()")
					if isTargetCoach(inner) {
						matchGroup = trimmed
						break
					}
				}
			}
		}

		if matchGroup == "" {
			continue
		}

		// Only capture the first occurrence (earliest time slot)
		if seen[matchGroup] {
			continue
		}
		seen[matchGroup] = true

		reserved := isReservedSlot(matchGroup)
		
		// If it's in the makingDivesMap, it's definitely a group.
		// Otherwise check for "way" in name.
		makingCoach, inMap := makingDivesMap[matchGroup]
		isGroup := inMap || containsWay(matchGroup)
		
		// Who is making the dives?
		// Default to us unless specified otherwise in the map.
		whoMakes := coachName
		if inMap {
			whoMakes = makingCoach
		}

		var emails []string
		if !reserved {
			emails = MatchEmails(matchGroup, studentEmails)
		}

		plans = append(plans, FlightPlan{
			SubjectName:      matchGroup,
			IsGroup:          isGroup,
			IsReserved:       reserved,
			ArrivalDay:       row.Date,
			ArrivalTime:      row.MeetAt,
			SubjectEmails:    emails,
			MakingDivesCoach: whoMakes,
		})
	}

	return plans
}

func containsWay(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "way")
}

// SanitizeFileName replaces characters that are illegal or problematic in filenames
// (like forward slashes) with safe alternatives.
func SanitizeFileName(name string) string {
	r := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return r.Replace(name)
}
