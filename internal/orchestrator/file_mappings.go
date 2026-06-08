package orchestrator

import "bauer/internal/gdocs"

// buildFileMappings creates file mappings from the extraction result
// This maps which files should receive which suggestions based on the document metadata and suggestion locations
func buildFileMappings(result *gdocs.ProcessingResult) map[string]*FileMapping {
	fileMappings := make(map[string]*FileMapping)

	// If document has metadata with a suggested URL, use that as the primary mapping
	if result.Metadata != nil && result.Metadata.SuggestedUrl != "" {
		fileMappings[result.Metadata.SuggestedUrl] = &FileMapping{
			SuggestedFile:   result.Metadata.SuggestedUrl,
			SourceReference: result.DocumentTitle,
			SuggestionCount: len(result.ActionableSuggestions),
			SuggestionIDs:   getSuggestionIDs(result.ActionableSuggestions),
		}
		return fileMappings
	}

	// Otherwise, create one mapping per location group
	// Each section/location might correspond to a different file
	for _, group := range result.GroupedSuggestions {
		location := group.Location

		// Use parent heading as the file/section identifier
		var fileKey string
		if location.ParentHeading != "" {
			fileKey = location.ParentHeading
		} else if location.Section != "" {
			fileKey = location.Section
		} else {
			fileKey = "general"
		}

		// Create or update mapping for this file/section
		if _, exists := fileMappings[fileKey]; !exists {
			fileMappings[fileKey] = &FileMapping{
				SuggestedFile:   fileKey,
				SourceReference: result.DocumentTitle,
				SuggestionCount: 0,
				SuggestionIDs:   []string{},
			}
		}

		// Add suggestion IDs
		for _, suggestion := range group.Suggestions {
			fileMappings[fileKey].SuggestionIDs = append(fileMappings[fileKey].SuggestionIDs, suggestion.ID)
		}
		fileMappings[fileKey].SuggestionCount = len(fileMappings[fileKey].SuggestionIDs)
	}

	return fileMappings
}

// getSuggestionIDs extracts all suggestion IDs from a list of suggestions
func getSuggestionIDs(suggestions []gdocs.ActionableSuggestion) []string {
	ids := make([]string, len(suggestions))
	for i, suggestion := range suggestions {
		ids[i] = suggestion.ID
	}
	return ids
}

// buildParseResultSummary creates a summary with statistics by file and type
func buildParseResultSummary(suggestions []SimplifiedActionableSuggestion, fileMappings map[string]*FileMapping) ParseResultSummary {
	summary := ParseResultSummary{
		TotalSuggestions: len(suggestions),
		TotalFiles:       len(fileMappings),
		ByFile:           make(map[string]SuggestionSummaryByFile),
		ByType: SuggestionSummaryByType{
			Insert:  0,
			Delete:  0,
			Replace: 0,
		},
	}

	// Count by file and type
	for _, suggestion := range suggestions {
		// Count by type
		switch suggestion.Change.Type {
		case "insert":
			summary.ByType.Insert++
		case "delete":
			summary.ByType.Delete++
		case "replace":
			summary.ByType.Replace++
		case "link_add":
			summary.ByType.LinkAdd++
		case "link_change":
			summary.ByType.LinkChange++
		case "link_remove":
			summary.ByType.LinkRemove++
		}

		// Count by file
		if _, exists := summary.ByFile[suggestion.File]; !exists {
			summary.ByFile[suggestion.File] = SuggestionSummaryByFile{}
		}

		fileStats := summary.ByFile[suggestion.File]
		switch suggestion.Change.Type {
		case "insert":
			fileStats.Insertions++
		case "delete":
			fileStats.Deletions++
		case "replace":
			fileStats.Replacements++
		case "link_add":
			fileStats.LinkAdds++
		case "link_change":
			fileStats.LinkChanges++
		case "link_remove":
			fileStats.LinkRemoves++
		}
		summary.ByFile[suggestion.File] = fileStats
	}

	return summary
}

// buildSimplifiedSuggestions creates simplified suggestions with file references
func buildSimplifiedSuggestions(originalSuggestions []gdocs.ActionableSuggestion, fileMappings map[string]*FileMapping) []SimplifiedActionableSuggestion {
	simplified := make([]SimplifiedActionableSuggestion, 0, len(originalSuggestions))

	// Create a quick lookup map from suggestion ID to file
	suggestionToFile := make(map[string]string)
	for file, mapping := range fileMappings {
		for _, suggestionID := range mapping.SuggestionIDs {
			suggestionToFile[suggestionID] = file
		}
	}

	// Convert to simplified format
	for _, suggestion := range originalSuggestions {
		// Find which file this suggestion maps to
		targetFile := suggestionToFile[suggestion.ID]
		if targetFile == "" {
			// Deterministic fallback strategy to avoid nondeterministic map iteration:
			// 1. Prefer "general" if it exists in mappings
			if _, hasGeneral := fileMappings["general"]; hasGeneral {
				targetFile = "general"
			} else if len(fileMappings) == 1 {
				// 2. If exactly one file mapping exists, use it
				for file := range fileMappings {
					targetFile = file
					break
				}
			} else {
				// 3. Default to "general" as a catch-all
				targetFile = "general"
			}
		}

		simplified = append(simplified, SimplifiedActionableSuggestion{
			ID:           suggestion.ID,
			File:         targetFile,
			Anchor:       suggestion.Anchor,
			Change:       suggestion.Change,
			Verification: suggestion.Verification,
		})
	}

	return simplified
}
