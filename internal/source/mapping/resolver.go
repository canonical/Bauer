package mapping

import (
	"strings"
	"unicode"

	"bauer/internal/figma"
	"bauer/internal/gdocs"
)

// Resolver builds ResolvedChunk values by joining gdocs groups with figma design data.
type Resolver struct{}

// Build returns one ResolvedChunk per gdocs LocationGroupedSuggestions.
// If design is nil (no figma URL was supplied), each chunk has empty design fields
// and Mapping.Method == "none".
func (r *Resolver) Build(
	groups []gdocs.LocationGroupedSuggestions,
	design *figma.NormalizedDesign,
) []ResolvedChunk {
	chunks := make([]ResolvedChunk, len(groups))
	for i, group := range groups {
		chunks[i] = ResolvedChunk{
			Locations: []gdocs.LocationGroupedSuggestions{group},
			Mapping:   MappingMetadata{Method: "none", Confidence: 0, Status: "none"},
		}
		if design != nil {
			anchors, meta := r.resolveAnchor(group, design)
			chunks[i].DesignAnchors = anchors
			chunks[i].Mapping = meta
			chunks[i].ScreenshotPaths = r.screenshotsForAnchors(anchors, design)
			chunks[i].Comments = r.commentsForAnchors(anchors, design)
		}
	}
	return chunks
}

func (r *Resolver) resolveAnchor(
	group gdocs.LocationGroupedSuggestions,
	design *figma.NormalizedDesign,
) ([]DesignAnchorRef, MappingMetadata) {
	// Strategy 1: user-supplied node ID from URL
	if design.RootNodeID != "" {
		for _, anchor := range design.Anchors {
			if anchor.NodeID == design.RootNodeID {
				return []DesignAnchorRef{{
					FileKey:  design.FileKey,
					NodeID:   anchor.NodeID,
					NodeName: anchor.NodeName,
				}}, MappingMetadata{Method: "url", Confidence: 1.0, Status: "healthy"}
			}
		}
	}

	// Strategy 2: text layer matching (Jaccard)
	if anchor, conf := matchByTextLayers(group, design.Anchors); anchor != nil {
		return []DesignAnchorRef{{
			FileKey:  design.FileKey,
			NodeID:   anchor.NodeID,
			NodeName: anchor.NodeName,
		}}, MappingMetadata{Method: "text", Confidence: conf, Status: "healthy"}
	}

	// Strategy 3: frame name matching
	if anchor, conf := matchByFrameName(group.Location, design.Anchors); anchor != nil {
		return []DesignAnchorRef{{
			FileKey:  design.FileKey,
			NodeID:   anchor.NodeID,
			NodeName: anchor.NodeName,
		}}, MappingMetadata{Method: "name", Confidence: conf, Status: "healthy"}
	}

	// Strategy 4: fallback to first anchor (root node)
	if len(design.Anchors) > 0 {
		return []DesignAnchorRef{{
			FileKey:  design.FileKey,
			NodeID:   design.Anchors[0].NodeID,
			NodeName: design.Anchors[0].NodeName,
		}}, MappingMetadata{Method: "fallback", Confidence: 0.50, Status: "unresolved"}
	}

	return nil, MappingMetadata{Method: "none", Confidence: 0, Status: "unresolved"}
}

// matchByTextLayers uses weighted Jaccard similarity on token bags.
// Returns the best matching anchor and its confidence, or nil if no match meets threshold.
func matchByTextLayers(group gdocs.LocationGroupedSuggestions, anchors []figma.DesignAnchor) (*figma.DesignAnchor, float64) {
	// Build token bag from gdocs suggestion group
	gdocsTokens := tokenize(group.Location.ParentHeading + " " + group.Location.Section)
	for _, sug := range group.Suggestions {
		gdocsTokens = append(gdocsTokens, tokenizeFromSuggestion(sug)...)
	}
	gdocsSet := toSet(gdocsTokens)

	if len(gdocsSet) == 0 {
		return nil, 0
	}

	var best *figma.DesignAnchor
	bestConf := 0.0

	for i := range anchors {
		figmaTokens := tokenize(strings.Join(anchors[i].NearestText, " "))
		figmaSet := toSet(figmaTokens)

		shared := intersect(gdocsSet, figmaSet)
		union := unionSets(gdocsSet, figmaSet)
		if len(union) == 0 {
			continue
		}

		jacc := float64(len(shared)) / float64(len(union))
		conf := 0.50 + (jacc * 0.45)

		if jacc >= 0.30 && conf > bestConf {
			bestConf = conf
			best = &anchors[i]
		}
	}
	return best, bestConf
}

// matchByFrameName compares the gdocs heading to the Figma frame name.
func matchByFrameName(loc gdocs.SuggestionLocation, anchors []figma.DesignAnchor) (*figma.DesignAnchor, float64) {
	headingTokens := toSet(tokenize(loc.ParentHeading))
	if len(headingTokens) == 0 {
		return nil, 0
	}

	var best *figma.DesignAnchor
	bestConf := 0.0

	for i := range anchors {
		frameTokens := toSet(tokenize(anchors[i].NodeName))
		shared := intersect(headingTokens, frameTokens)

		maxLen := len(headingTokens)
		if len(frameTokens) > maxLen {
			maxLen = len(frameTokens)
		}
		if maxLen == 0 {
			continue
		}

		overlap := float64(len(shared)) / float64(maxLen)
		conf := 0.50 + (overlap * 0.35)

		if overlap >= 0.50 && conf > bestConf {
			bestConf = conf
			best = &anchors[i]
		}
	}
	return best, bestConf
}

func (r *Resolver) screenshotsForAnchors(anchors []DesignAnchorRef, design *figma.NormalizedDesign) []string {
	var paths []string
	for _, anchor := range anchors {
		for _, shot := range design.Screenshots {
			if shot.NodeID == anchor.NodeID {
				paths = append(paths, shot.LocalPath)
			}
		}
	}
	return paths
}

func (r *Resolver) commentsForAnchors(anchors []DesignAnchorRef, design *figma.NormalizedDesign) []DesignCommentRef {
	anchorIDs := map[string]bool{}
	for _, a := range anchors {
		anchorIDs[a.NodeID] = true
	}
	var refs []DesignCommentRef
	for _, c := range design.Comments {
		if c.Resolved {
			continue // resolved comments not included in prompt context
		}
		if anchorIDs[c.NodeID] {
			refs = append(refs, DesignCommentRef{
				CommentID: c.ID,
				Message:   c.Message,
				Author:    c.Author,
				NodeID:    c.NodeID,
			})
		}
	}
	return refs
}

// tokenizeFromSuggestion returns tokenized original text from a suggestion's change.
func tokenizeFromSuggestion(sug gdocs.GroupedActionableSuggestion) []string {
	return tokenize(sug.Change.OriginalText)
}

// tokenize normalizes text into a lowercase token slice, removing stop words and short tokens.
func tokenize(text string) []string {
	stop := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"in": true, "of": true, "to": true, "for": true, "is": true, "are": true,
		"it": true, "at": true, "on": true, "by": true, "be": true,
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var result []string
	for _, w := range words {
		if len(w) >= 3 && !stop[w] {
			result = append(result, w)
		}
	}
	return result
}

func toSet(tokens []string) map[string]bool {
	s := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		s[t] = true
	}
	return s
}

func intersect(a, b map[string]bool) map[string]bool {
	result := map[string]bool{}
	for k := range a {
		if b[k] {
			result[k] = true
		}
	}
	return result
}

func unionSets(a, b map[string]bool) map[string]bool {
	result := map[string]bool{}
	for k := range a {
		result[k] = true
	}
	for k := range b {
		result[k] = true
	}
	return result
}
