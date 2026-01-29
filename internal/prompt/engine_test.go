package prompt

import (
	"os"
	"testing"

	"bauer/internal/gdocs"
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
		"BAU Implementation Instructions",
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
		"BAU Implementation Instructions",
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
