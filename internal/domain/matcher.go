package domain

import "strings"

// MatchEmails resolves a group name (e.g. "Kyle H / Dan B 4way") to a list of
// email addresses using a multi-tier fuzzy heuristic against the student directory.
//
// Matching tiers (evaluated per slash-separated chunk):
//  1. Direct exact match          — "Daniel Barney" == "Daniel Barney"
//  2. First name + last initial   — "kyle h" matches "Kyle Henderson"
//  3. Prefix match                — "dan b" matches "Daniel Barney"
//  4. First name only fallback    — "kelsea" matches "Kelsea Kiene"
func MatchEmails(groupName string, studentEmails map[string]string) []string {
	// Strip group-type suffixes before matching
	cleaned := strings.ToLower(groupName)
	for _, suffix := range []string{"4way", "3way", "2way"} {
		cleaned = strings.ReplaceAll(cleaned, suffix, "")
	}
	cleaned = strings.TrimSpace(cleaned)

	// Tier 1: Direct exact match on the full group name
	for name, email := range studentEmails {
		if strings.EqualFold(name, groupName) {
			return []string{email}
		}
	}

	// Split on "/" for multi-person groups
	chunks := strings.Split(cleaned, "/")
	for i := range chunks {
		chunks[i] = strings.TrimSpace(chunks[i])
	}

	seen := map[string]bool{}
	var matched []string

	for name, email := range studentEmails {
		nameParts := strings.Fields(strings.ToLower(name))
		if len(nameParts) == 0 {
			continue
		}

		for _, chunk := range chunks {
			if chunk == "" {
				continue
			}
			chunkParts := strings.Fields(chunk)

			// Tier 2: First name + last initial (e.g., "kyle h")
			if len(nameParts) >= 2 && len(chunkParts) == 2 && len(chunkParts[1]) == 1 {
				candidate := nameParts[0] + " " + string(nameParts[1][0])
				if candidate == chunk {
					if !seen[email] {
						seen[email] = true
						matched = append(matched, email)
					}
					continue
				}
			}

			// Tier 3: Prefix match (e.g., "dan b" matches "daniel barney")
			if len(chunkParts) >= 2 && len(nameParts) >= 2 {
				if strings.HasPrefix(nameParts[0], chunkParts[0]) &&
					strings.HasPrefix(nameParts[1], chunkParts[1]) {
					if !seen[email] {
						seen[email] = true
						matched = append(matched, email)
					}
					continue
				}
			}

			// Tier 3b: Initials match (e.g., "jp" matches "John Paul")
			if len(chunk) == 2 && len(chunkParts) == 1 && len(nameParts) >= 2 {
				initials := string(nameParts[0][0]) + string(nameParts[1][0])
				if initials == chunk {
					if !seen[email] {
						seen[email] = true
						matched = append(matched, email)
					}
					continue
				}
			}

			// Tier 4: First name only fallback
			if len(chunkParts) == 1 && chunkParts[0] == nameParts[0] {
				if !seen[email] {
					seen[email] = true
					matched = append(matched, email)
				}
			}
		}
	}

	return matched
}
