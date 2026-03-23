package domain

import "strings"

// FolderMatchesAssignment checks whether a Drive folder name is a fuzzy match
// for a given assignment's SubjectName using multi-tier heuristics.
//
// Matching tiers:
//  1. Case-insensitive exact match
//  2. One contains the other (substring)
//  3. Token overlap ≥ 60% of assignment tokens found in folder name
func FolderMatchesAssignment(folderName string, assignmentName string) bool {
	fLower := strings.ToLower(strings.TrimSpace(folderName))
	aLower := strings.ToLower(strings.TrimSpace(assignmentName))

	if fLower == "" || aLower == "" {
		return false
	}

	// Tier 1: exact match
	if fLower == aLower {
		return true
	}

	// Tier 2: containment
	if strings.Contains(fLower, aLower) || strings.Contains(aLower, fLower) {
		return true
	}

	// Tier 3: token overlap
	aTokens := tokenize(aLower)
	fTokens := tokenize(fLower)

	if len(aTokens) == 0 {
		return false
	}

	fSet := map[string]bool{}
	for _, t := range fTokens {
		fSet[t] = true
	}

	hits := 0
	for _, t := range aTokens {
		if fSet[t] {
			hits++
			continue
		}
		// Prefix match for abbreviated tokens (e.g., "h" matches "henderson")
		for ft := range fSet {
			if len(t) >= 2 && strings.HasPrefix(ft, t) {
				hits++
				break
			}
			if len(ft) >= 2 && strings.HasPrefix(t, ft) {
				hits++
				break
			}
		}
	}

	ratio := float64(hits) / float64(len(aTokens))
	return ratio >= 0.6
}

// tokenize splits a name string on common separators and strips noise words.
func tokenize(s string) []string {
	// Normalize separators
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "-", " ")

	fields := strings.Fields(s)
	var tokens []string
	for _, f := range fields {
		f = strings.Trim(f, "()[]")
		if f == "" {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}
