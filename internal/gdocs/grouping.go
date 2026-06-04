package gdocs

import (
	"sort"
	"strings"
)

// GroupActionableSuggestions groups related atomic suggestions into logical units.
// Suggestions are first grouped by their location (section, heading, table), then by
// their ID within each location. Suggestions with the same ID must be contiguous in position.
// Returns a slice of location-based groups, each containing grouped suggestions for that location.
func GroupActionableSuggestions(suggestions []ActionableSuggestion, structure *DocumentStructure) []LocationGroupedSuggestions {
	if len(suggestions) == 0 {
		return []LocationGroupedSuggestions{}
	}

	// First, group suggestions by location
	locationGroups := make(map[string][]ActionableSuggestion)
	locationMap := make(map[string]SuggestionLocation) // Track the actual location object

	for _, sugg := range suggestions {
		locationKey := getLocationKey(sugg.Location)
		locationGroups[locationKey] = append(locationGroups[locationKey], sugg)
		locationMap[locationKey] = sugg.Location
	}

	// Process each location group
	var result []LocationGroupedSuggestions
	for locationKey, locationSuggestions := range locationGroups {
		// Within this location, group by suggestion ID
		groupedSuggestions := groupSuggestionsByID(locationSuggestions, structure)

		// Sort suggestions within this location by position
		sort.Slice(groupedSuggestions, func(i, j int) bool {
			return groupedSuggestions[i].Position.StartIndex < groupedSuggestions[j].Position.StartIndex
		})

		result = append(result, LocationGroupedSuggestions{
			Location:    locationMap[locationKey],
			Suggestions: groupedSuggestions,
		})
	}

	// Sort location groups by the first suggestion's position in each group
	sort.Slice(result, func(i, j int) bool {
		if len(result[i].Suggestions) == 0 {
			return false
		}
		if len(result[j].Suggestions) == 0 {
			return true
		}
		return result[i].Suggestions[0].Position.StartIndex < result[j].Suggestions[0].Position.StartIndex
	})

	return result
}

// groupSuggestionsByID groups suggestions by their ID and merges contiguous atomic operations.
// Suggestions with the same ID that are contiguous in position are merged into a single
// GroupedActionableSuggestion. Non-contiguous suggestions with the same ID are kept separate.
func groupSuggestionsByID(suggestions []ActionableSuggestion, structure *DocumentStructure) []GroupedActionableSuggestion {
	if len(suggestions) == 0 {
		return []GroupedActionableSuggestion{}
	}

	// Group by suggestion ID
	groupsBySuggestionID := make(map[string][]ActionableSuggestion)
	for _, sugg := range suggestions {
		groupsBySuggestionID[sugg.ID] = append(groupsBySuggestionID[sugg.ID], sugg)
	}

	// Process each ID group
	var grouped []GroupedActionableSuggestion
	for id, group := range groupsBySuggestionID {
		// Sort by start position to ensure correct ordering
		sort.Slice(group, func(i, j int) bool {
			return group[i].Position.StartIndex < group[j].Position.StartIndex
		})

		// Verify contiguity (atomic operations should be adjacent or overlapping)
		if !areContiguous(group) {
			// If not contiguous, treat each as separate (shouldn't happen, but defensive)
			for _, sugg := range group {
				grouped = append(grouped, convertSingleSuggestion(sugg))
			}
			continue
		}

		// Group is valid - merge the suggestions
		merged := mergeSuggestions(id, group, structure)
		grouped = append(grouped, merged)
	}

	// Sort final result by position for consistent output
	sort.Slice(grouped, func(i, j int) bool {
		return grouped[i].Position.StartIndex < grouped[j].Position.StartIndex
	})

	return grouped
}

// getLocationKey creates a unique key for a location to enable grouping.
// Two locations are considered the same if they share the same section, heading, and table context.
func getLocationKey(loc SuggestionLocation) string {
	key := loc.Section

	if loc.ParentHeading != "" {
		key += "|heading:" + loc.ParentHeading + "|level:" + string(rune(loc.HeadingLevel))
	}

	if loc.InTable && loc.Table != nil {
		key += "|table:" + loc.Table.TableID
		if loc.Table.TableTitle != "" {
			key += "|title:" + loc.Table.TableTitle
		}
	}

	if loc.InMetadata {
		key += "|metadata:true"
	}

	return key
}

// areContiguous checks if suggestions are adjacent or overlapping in position.
// This validates that they're truly part of the same logical change.
func areContiguous(suggestions []ActionableSuggestion) bool {
	if len(suggestions) <= 1 {
		return true
	}

	for i := 0; i < len(suggestions)-1; i++ {
		current := suggestions[i]
		next := suggestions[i+1]

		// Next suggestion should start at or before current ends (allowing for overlap/adjacency)
		// We allow a small gap (1 char) for edge cases
		if next.Position.StartIndex > current.Position.EndIndex+1 {
			return false
		}
	}

	return true
}

// convertSingleSuggestion converts a single ActionableSuggestion to GroupedActionableSuggestion.
// Used for suggestions that don't need grouping.
func convertSingleSuggestion(sugg ActionableSuggestion) GroupedActionableSuggestion {
	return GroupedActionableSuggestion{
		ID:     sugg.ID,
		Anchor: sugg.Anchor,
		Change: sugg.Change,
		Verification: SuggestionVerification{
			TextBeforeChange: sugg.Verification.TextBeforeChange,
			TextAfterChange:  sugg.Verification.TextAfterChange,
		},
		Position: struct {
			StartIndex int64 `json:"start_index"`
			EndIndex   int64 `json:"end_index"`
		}{
			StartIndex: sugg.Position.StartIndex,
			EndIndex:   sugg.Position.EndIndex,
		},
		AtomicChanges: []SuggestionChange{sugg.Change},
		AtomicCount:   1,
	}
}

// mergeSuggestions combines multiple atomic suggestions into a single grouped suggestion.
func mergeSuggestions(id string, suggestions []ActionableSuggestion, structure *DocumentStructure) GroupedActionableSuggestion {
	if len(suggestions) == 1 {
		return convertSingleSuggestion(suggestions[0])
	}

	first := suggestions[0]
	last := suggestions[len(suggestions)-1]

	// Extract anchors with increased length (120 chars) for better context
	const groupedAnchorLength = 120
	precedingText, followingText := getTextAround(structure, first.Position.StartIndex, last.Position.EndIndex, groupedAnchorLength)

	// Collect atomic changes
	atomicChanges := make([]SuggestionChange, len(suggestions))
	for i, sugg := range suggestions {
		atomicChanges[i] = sugg.Change
	}

	// Merge the changes to compute the net effect
	mergedChange := mergeChanges(suggestions)

	// Build verification texts
	var originalText, newText string
	if mergedChange.Type == "insert" {
		originalText = ""
		newText = mergedChange.NewText
	} else if mergedChange.Type == "delete" {
		originalText = mergedChange.OriginalText
		newText = ""
	} else { // "replace"
		originalText = mergedChange.OriginalText
		newText = mergedChange.NewText
	}

	verification := SuggestionVerification{
		TextBeforeChange: precedingText + originalText + followingText,
		TextAfterChange:  precedingText + newText + followingText,
	}

	return GroupedActionableSuggestion{
		ID: id,
		Anchor: SuggestionAnchor{
			PrecedingText: precedingText,
			FollowingText: followingText,
		},
		Change:       mergedChange,
		Verification: verification,
		Position: struct {
			StartIndex int64 `json:"start_index"`
			EndIndex   int64 `json:"end_index"`
		}{
			StartIndex: first.Position.StartIndex,
			EndIndex:   last.Position.EndIndex,
		},
		AtomicChanges: atomicChanges,
		AtomicCount:   len(suggestions),
	}
}

// mergeChanges combines multiple atomic changes into a single net change.
// Handles sequences like: insert "Build " + delete "Y" + insert "y" -> replace "Y" with "Build y"
func mergeChanges(suggestions []ActionableSuggestion) SuggestionChange {
	var originalParts []string
	var newParts []string
	hasInsertions := false
	hasDeletions := false
	var linkMeta *SuggestionChange // preserved if any atom carries link info

	// Process each atomic change in order
	for _, sugg := range suggestions {
		switch sugg.Change.Type {
		case "insert":
			hasInsertions = true
			newParts = append(newParts, sugg.Change.NewText)
			if sugg.Change.NewURL != "" {
				c := sugg.Change
				linkMeta = &c
			}
		case "delete":
			hasDeletions = true
			originalParts = append(originalParts, sugg.Change.OriginalText)
		case "style":
			// Style changes don't affect text content
			// Keep the text in both original and new
			if sugg.Change.OriginalText != "" {
				originalParts = append(originalParts, sugg.Change.OriginalText)
				newParts = append(newParts, sugg.Change.OriginalText)
			}
		case "link_add", "link_change", "link_remove":
			c := sugg.Change
			linkMeta = &c
		}
	}

	originalText := strings.Join(originalParts, "")
	newText := strings.Join(newParts, "")

	// Determine the type of the merged change
	changeType := "replace"
	if !hasDeletions && hasInsertions {
		changeType = "insert"
	} else if hasDeletions && !hasInsertions {
		changeType = "delete"
	} else if !hasDeletions && !hasInsertions {
		changeType = "style"
	}

	merged := SuggestionChange{
		Type:         changeType,
		OriginalText: originalText,
		NewText:      newText,
	}

	// Preserve link metadata if any atomic change carried it.
	if linkMeta != nil {
		merged.LinkText = linkMeta.LinkText
		merged.OldURL = linkMeta.OldURL
		merged.NewURL = linkMeta.NewURL
		// With no text insert/delete, this is a pure link operation.
		if !hasInsertions && !hasDeletions {
			merged.Type = linkMeta.Type
		}
	}

	return merged
}
