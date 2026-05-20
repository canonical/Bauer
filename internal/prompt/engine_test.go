package prompt

import (
	"os"
	"strings"
	"testing"

	"bauer/internal/gdocs"
	"bauer/internal/source/mapping"
)

func TestNewEngine(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine == nil {
		t.Fatal("NewEngine() returned nil engine")
	}
}

func TestChunkLocations(t *testing.T) {
	tests := []struct {
		name           string
		groups         []gdocs.LocationGroupedSuggestions
		chunkSize      int
		expectedChunks int
	}{
		{
			name: "single location - request 1 chunk",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(5)},
			},
			chunkSize:      1,
			expectedChunks: 1,
		},
		{
			name: "3 locations - request 1 chunk",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(3)},
				{Suggestions: makeTestSuggestions(4)},
				{Suggestions: makeTestSuggestions(2)},
			},
			chunkSize:      1,
			expectedChunks: 1,
		},
		{
			name: "6 locations - request 3 chunks",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(5)},
				{Suggestions: makeTestSuggestions(3)},
				{Suggestions: makeTestSuggestions(8)},
				{Suggestions: makeTestSuggestions(2)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(7)},
			},
			chunkSize:      3, // 6 locations / 3 chunks = 2 locations per chunk
			expectedChunks: 3,
		},
		{
			name: "5 locations - request 2 chunks",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(2)},
				{Suggestions: makeTestSuggestions(3)},
				{Suggestions: makeTestSuggestions(4)},
				{Suggestions: makeTestSuggestions(5)},
			},
			chunkSize:      2, // 5 locations / 2 chunks = 3 locs in chunk 1, 2 in chunk 2
			expectedChunks: 2,
		},
		{
			name:           "empty groups",
			groups:         []gdocs.LocationGroupedSuggestions{},
			chunkSize:      10,
			expectedChunks: 1,
		},
		{
			name: "25 locations - request 1 chunk",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
			},
			chunkSize:      1,
			expectedChunks: 1,
		},
		{
			name: "25 locations - request 5 chunks",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
			},
			chunkSize:      5,
			expectedChunks: 5,
		},
		{
			name: "3 locations - request 10 chunks (more than locations)",
			groups: []gdocs.LocationGroupedSuggestions{
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
				{Suggestions: makeTestSuggestions(1)},
			},
			chunkSize:      10,
			expectedChunks: 3, // Should cap at number of locations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkLocations(tt.groups, tt.chunkSize)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}

			// Verify all locations preserved
			originalCount := len(tt.groups)
			chunkedCount := 0
			for _, chunk := range chunks {
				chunkedCount += len(chunk)
			}

			if chunkedCount != originalCount {
				t.Errorf("Lost locations during chunking: original=%d, chunked=%d", originalCount, chunkedCount)
			}

			// Verify chunk distribution is reasonable
			if originalCount > 0 && tt.chunkSize > 0 && tt.chunkSize < originalCount {
				expectedLocPerChunk := (originalCount + tt.chunkSize - 1) / tt.chunkSize
				for i, chunk := range chunks {
					// Each chunk should have roughly the expected number (may vary by 1)
					if len(chunk) > expectedLocPerChunk+1 || len(chunk) < 1 {
						t.Errorf("Chunk %d has %d locations, expected around %d", i, len(chunk), expectedLocPerChunk)
					}
				}
			}
		})
	}
}

func TestChunkLocationsPractical(t *testing.T) {
	// This test demonstrates the new chunk size semantics with a real-world scenario
	// Simulating: 25 locations with chunk-size=1 should create 1 chunk (not 25)

	// Create 25 locations
	locations := make([]gdocs.LocationGroupedSuggestions, 25)
	for i := range locations {
		locations[i] = gdocs.LocationGroupedSuggestions{
			Location:    gdocs.SuggestionLocation{Section: "Body"},
			Suggestions: makeTestSuggestions(1),
		}
	}

	// Test 1: chunk-size=1 means "create 1 chunk total"
	chunks := ChunkLocations(locations, 1)
	if len(chunks) != 1 {
		t.Errorf("With 25 locations and chunk-size=1, expected 1 chunk, got %d", len(chunks))
	}
	if len(chunks) > 0 && len(chunks[0]) != 25 {
		t.Errorf("Expected first chunk to contain all 25 locations, got %d", len(chunks[0]))
	}

	// Test 2: chunk-size=5 means "create 5 chunks"
	chunks = ChunkLocations(locations, 5)
	if len(chunks) != 5 {
		t.Errorf("With 25 locations and chunk-size=5, expected 5 chunks, got %d", len(chunks))
	}

	// Each chunk should have 5 locations (25/5 = 5)
	for i, chunk := range chunks {
		if len(chunk) != 5 {
			t.Errorf("Chunk %d has %d locations, expected 5", i+1, len(chunk))
		}
	}

	// Test 3: chunk-size=3 means "create 3 chunks" (25/3 = 9, 8, 8)
	chunks = ChunkLocations(locations, 3)
	if len(chunks) != 3 {
		t.Errorf("With 25 locations and chunk-size=3, expected 3 chunks, got %d", len(chunks))
	}

	// Verify total is still 25
	total := 0
	for _, chunk := range chunks {
		total += len(chunk)
	}
	if total != 25 {
		t.Errorf("Expected total of 25 locations across chunks, got %d", total)
	}
}

func TestRenderChunk(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	data := PromptData{
		DocumentTitle:   "Test Document",
		SuggestedURL:    "ubuntu.com/test/page",
		ChunkNumber:     1,
		TotalChunks:     2,
		LocationCount:   2,
		SuggestionsJSON: `[{"location":{"section":"Body"},"suggestions":[{"id":"test-1"}]}]`,
	}

	content, err := engine.RenderChunk(data)
	if err != nil {
		t.Fatalf("RenderChunk() failed: %v", err)
	}

	// Verify content contains expected sections
	expectedStrings := []string{
		"BAU Copy Update Implementation Instructions",
		"Test Document",
		"ubuntu.com/test/page",
		"Chunk 1 of 2",
		"Suggestions Data",
		"```json",
		"Vanilla Framework Patterns Reference",
	}

	for _, expected := range expectedStrings {
		if !contains(content, expected) {
			t.Errorf("Rendered content missing expected string: %q", expected)
		}
	}
}

func TestRenderChunkWithPageRefresh(t *testing.T) {
	// Test with PageRefresh enabled
	engine, err := NewEngine(true)
	if err != nil {
		t.Fatalf("Failed to create engine with PageRefresh: %v", err)
	}

	data := PromptData{
		DocumentTitle:   "Test Document",
		SuggestedURL:    "ubuntu.com/test/page",
		ChunkNumber:     1,
		TotalChunks:     1,
		LocationCount:   1,
		SuggestionsJSON: `[{"location":{"section":"Body"},"suggestions":[{"id":"test-1"}]}]`,
	}

	content, err := engine.RenderChunk(data)
	if err != nil {
		t.Fatalf("RenderChunk() with PageRefresh failed: %v", err)
	}

	// Verify content still contains expected sections
	// (Both templates should have the same structure for now)
	expectedStrings := []string{
		"BAU Page Refresh Implementation Instructions",
		"Test Document",
		"ubuntu.com/test/page",
		"Chunk 1 of 1",
		"Suggestions Data",
		"```json",
		"Vanilla Framework Patterns Reference",
	}

	for _, expected := range expectedStrings {
		if !contains(content, expected) {
			t.Errorf("Rendered content with PageRefresh missing expected string: %q", expected)
		}
	}

	// Test with PageRefresh disabled
	engineNormal, err := NewEngine(false)
	if err != nil {
		t.Fatalf("Failed to create engine without PageRefresh: %v", err)
	}

	contentNormal, err := engineNormal.RenderChunk(data)
	if err != nil {
		t.Fatalf("RenderChunk() without PageRefresh failed: %v", err)
	}

	// For now, both templates are identical, so content should be the same
	// In the future, they may differ
	if len(content) == 0 || len(contentNormal) == 0 {
		t.Error("Rendered content should not be empty")
	}
}

func TestGenerateAllChunks(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create temporary output directory
	tmpDir := t.TempDir()

	result := &gdocs.ProcessingResult{
		DocumentTitle: "Test Document",
		DocumentID:    "test-456",
		Metadata: &gdocs.MetadataTable{
			SuggestedUrl: "ubuntu.com/test/page",
		},
		GroupedSuggestions: []gdocs.LocationGroupedSuggestions{
			{
				Location:    gdocs.SuggestionLocation{Section: "Body"},
				Suggestions: makeTestSuggestions(5),
			},
			{
				Location:    gdocs.SuggestionLocation{Section: "Body"},
				Suggestions: makeTestSuggestions(8),
			},
			{
				Location:    gdocs.SuggestionLocation{Section: "Body"},
				Suggestions: makeTestSuggestions(3),
			},
		},
	}

	chunks, err := engine.GenerateAllChunks(
		result,
		2, // Request 2 chunks total (3 locations will be split into 2 chunks)
		tmpDir,
	)
	if err != nil {
		t.Fatalf("GenerateAllChunks() failed: %v", err)
	}

	// Verify correct number of chunks: requested 2 chunks for 3 locations
	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	// Verify files were created
	for _, chunk := range chunks {
		if _, err := os.Stat(chunk.Filename); os.IsNotExist(err) {
			t.Errorf("Chunk file not created: %s", chunk.Filename)
		}

		// Verify file content is not empty
		content, err := os.ReadFile(chunk.Filename)
		if err != nil {
			t.Errorf("Failed to read chunk file: %v", err)
		}
		if len(content) == 0 {
			t.Errorf("Chunk file is empty: %s", chunk.Filename)
		}
	}

	// Verify total location count matches
	totalFromChunks := 0
	for _, chunk := range chunks {
		totalFromChunks += chunk.LocationCount
	}

	totalOriginal := len(result.GroupedSuggestions)
	if totalFromChunks != totalOriginal {
		t.Errorf("Location count mismatch: chunks=%d, original=%d", totalFromChunks, totalOriginal)
	}
}

func TestReplaceVar(t *testing.T) {
	tests := []struct {
		name     string
		template string
		key      string
		value    string
		expected string
	}{
		{
			name:     "single replacement",
			template: "Hello {{.Name}}!",
			key:      "Name",
			value:    "World",
			expected: "Hello World!",
		},
		{
			name:     "multiple replacements",
			template: "{{.Greeting}} {{.Name}}, {{.Greeting}} again!",
			key:      "Greeting",
			value:    "Hi",
			expected: "Hi {{.Name}}, Hi again!",
		},
		{
			name:     "no replacement",
			template: "Hello World",
			key:      "Name",
			value:    "Test",
			expected: "Hello World",
		},
		{
			name:     "empty value",
			template: "Value: {{.Value}}",
			key:      "Value",
			value:    "",
			expected: "Value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceVar(tt.template, tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper functions

func makeResolvedChunk(heading, section string, anchors []mapping.DesignAnchorRef, screenshots []string, comments []mapping.DesignCommentRef, method string, conf float64) mapping.ResolvedChunk {
	return mapping.ResolvedChunk{
		Locations: []gdocs.LocationGroupedSuggestions{
			{
				Location:    gdocs.SuggestionLocation{Section: section, ParentHeading: heading},
				Suggestions: makeTestSuggestions(1),
			},
		},
		DesignAnchors:   anchors,
		ScreenshotPaths: screenshots,
		Comments:        comments,
		Mapping:         mapping.MappingMetadata{Method: method, Confidence: conf, Status: "healthy"},
	}
}

func TestGenerateChunksFromResolved_NoFigma(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	chunks := []mapping.ResolvedChunk{
		makeResolvedChunk("Heading 1", "Body", nil, nil, nil, "none", 0),
		makeResolvedChunk("Heading 2", "Body", nil, nil, nil, "none", 0),
	}

	prompts, err := engine.GenerateChunksFromResolved("Test Doc", "ubuntu.com/page", "", chunks, 10)
	if err != nil {
		t.Fatalf("GenerateChunksFromResolved() failed: %v", err)
	}

	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt batch, got %d", len(prompts))
	}

	p := prompts[0]
	if p.FigmaContextJSON != "" {
		t.Errorf("expected empty FigmaContextJSON, got %q", p.FigmaContextJSON)
	}
	if p.ChunkNumber != 1 {
		t.Errorf("ChunkNumber = %d, want 1", p.ChunkNumber)
	}
	if p.TotalChunks != 1 {
		t.Errorf("TotalChunks = %d, want 1", p.TotalChunks)
	}
	if p.LocationCount != 2 {
		t.Errorf("LocationCount = %d, want 2", p.LocationCount)
	}
}

func TestGenerateChunksFromResolved_WithFigma(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	anchors := []mapping.DesignAnchorRef{{FileKey: "f1", NodeID: "1:1", NodeName: "Login Frame"}}
	screenshots := []string{"/tmp/screenshot.png"}
	comments := []mapping.DesignCommentRef{{CommentID: "c1", Message: "check spacing", Author: "alice", NodeID: "1:1"}}

	chunks := []mapping.ResolvedChunk{
		makeResolvedChunk("Login", "Body", anchors, screenshots, comments, "url", 1.0),
	}

	prompts, err := engine.GenerateChunksFromResolved("Test Doc", "ubuntu.com/page", "https://figma.com/file/f1", chunks, 10)
	if err != nil {
		t.Fatalf("GenerateChunksFromResolved() failed: %v", err)
	}

	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(prompts))
	}

	p := prompts[0]
	if p.FigmaContextJSON == "" {
		t.Fatal("expected non-empty FigmaContextJSON")
	}
	if p.FigmaURL != "https://figma.com/file/f1" {
		t.Errorf("FigmaURL = %q, want %q", p.FigmaURL, "https://figma.com/file/f1")
	}
	if !strings.Contains(p.FigmaContextJSON, "Login Frame") {
		t.Errorf("FigmaContextJSON missing anchor name: %s", p.FigmaContextJSON)
	}
	if !strings.Contains(p.FigmaContextJSON, "/tmp/screenshot.png") {
		t.Errorf("FigmaContextJSON missing screenshot: %s", p.FigmaContextJSON)
	}
	if !strings.Contains(p.FigmaContextJSON, "check spacing") {
		t.Errorf("FigmaContextJSON missing comment: %s", p.FigmaContextJSON)
	}
}

func TestGenerateChunksFromResolved_MultiChunkBatching(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// 5 chunks, batch size 2 → 3 batches (2+2+1)
	var chunks []mapping.ResolvedChunk
	for i := range 5 {
		chunks = append(chunks, makeResolvedChunk("Heading", "Body", nil, nil, nil, "none", 0))
		_ = i
	}

	prompts, err := engine.GenerateChunksFromResolved("Test Doc", "ubuntu.com/page", "", chunks, 2)
	if err != nil {
		t.Fatalf("GenerateChunksFromResolved() failed: %v", err)
	}

	if len(prompts) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(prompts))
	}

	// Verify ChunkNumber and TotalChunks
	for i, p := range prompts {
		if p.ChunkNumber != i+1 {
			t.Errorf("prompts[%d].ChunkNumber = %d, want %d", i, p.ChunkNumber, i+1)
		}
		if p.TotalChunks != 3 {
			t.Errorf("prompts[%d].TotalChunks = %d, want 3", i, p.TotalChunks)
		}
	}

	// First two batches have 2 locations each, last has 1
	if prompts[0].LocationCount != 2 {
		t.Errorf("batch 0: LocationCount = %d, want 2", prompts[0].LocationCount)
	}
	if prompts[1].LocationCount != 2 {
		t.Errorf("batch 1: LocationCount = %d, want 2", prompts[1].LocationCount)
	}
	if prompts[2].LocationCount != 1 {
		t.Errorf("batch 2: LocationCount = %d, want 1", prompts[2].LocationCount)
	}
}

func TestRenderChunk_NoFigma_NoFigmaSection(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	data := PromptData{
		DocumentTitle:   "Test Doc",
		SuggestedURL:    "ubuntu.com/page",
		ChunkNumber:     1,
		TotalChunks:     1,
		LocationCount:   1,
		SuggestionsJSON: `[]`,
	}

	content, err := engine.RenderChunk(data)
	if err != nil {
		t.Fatalf("RenderChunk() failed: %v", err)
	}

	if strings.Contains(content, "Design Context") {
		t.Error("expected no 'Design Context' section when FigmaContextJSON is empty")
	}
}

func TestRenderChunk_WithFigma_IncludesFigmaSection(t *testing.T) {
	engine, err := NewEngine(false)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	data := PromptData{
		DocumentTitle:    "Test Doc",
		SuggestedURL:     "ubuntu.com/page",
		ChunkNumber:      1,
		TotalChunks:      1,
		LocationCount:    1,
		SuggestionsJSON:  `[]`,
		FigmaContextJSON: `{"anchors":[{"file_key":"f1","node_id":"1:1","node_name":"Hero Frame"}],"screenshots":["/tmp/hero.png"],"comments":[{"comment_id":"c1","message":"align left","author":"bob","node_id":"1:1"}]}`,
		FigmaURL:         "https://figma.com/file/f1",
	}

	content, err := engine.RenderChunk(data)
	if err != nil {
		t.Fatalf("RenderChunk() failed: %v", err)
	}

	expectedStrings := []string{
		"Design Context",
		"Hero Frame",
		"/tmp/hero.png",
		"align left",
		"bob",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(content, expected) {
			t.Errorf("rendered content missing expected string: %q", expected)
		}
	}
}

func makeTestSuggestions(count int) []gdocs.GroupedActionableSuggestion {
	suggestions := make([]gdocs.GroupedActionableSuggestion, count)
	for i := range count {
		suggestions[i] = gdocs.GroupedActionableSuggestion{
			ID: string(rune('a' + i)),
			Anchor: gdocs.SuggestionAnchor{
				PrecedingText: "before",
				FollowingText: "after",
			},
			Change: gdocs.SuggestionChange{
				Type:    "insert",
				NewText: "test",
			},
			Verification: gdocs.SuggestionVerification{
				TextBeforeChange: "before after",
				TextAfterChange:  "before test after",
			},
			AtomicCount: 1,
		}
	}
	return suggestions
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
