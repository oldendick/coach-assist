package domain

import (
	"strings"
)

// PopulateTemplate replaces all known placeholders in a template string with actual flight plan data.
func PopulateTemplate(template string, plan FlightPlan, folderLink string) string {
	firstName := strings.Split(plan.SubjectName, " ")[0]
	
	out := template
	out = strings.ReplaceAll(out, "{folder_link}", folderLink)
	out = strings.ReplaceAll(out, "{groupname}", plan.SubjectName)
	out = strings.ReplaceAll(out, "{name}", plan.SubjectName)
	out = strings.ReplaceAll(out, "{firstname}", firstName)
	out = strings.ReplaceAll(out, "{initial_meet_time}", plan.ArrivalTime)
	
	return out
}
