package orchestrator

import (
	"testing"

	"bauer/internal/gdocs"
)

func TestBuildParseResultSummary_LinkTypes(t *testing.T) {
	suggestions := []SimplifiedActionableSuggestion{
		{ID: "1", File: "a", Change: gdocs.SuggestionChange{Type: "insert"}},
		{ID: "2", File: "a", Change: gdocs.SuggestionChange{Type: "link_add"}},
		{ID: "3", File: "a", Change: gdocs.SuggestionChange{Type: "link_change"}},
		{ID: "4", File: "b", Change: gdocs.SuggestionChange{Type: "link_remove"}},
	}

	summary := buildParseResultSummary(suggestions, map[string]*FileMapping{})

	if summary.TotalSuggestions != 4 {
		t.Fatalf("total = %d, want 4", summary.TotalSuggestions)
	}
	if summary.ByType.Insert != 1 || summary.ByType.LinkAdd != 1 ||
		summary.ByType.LinkChange != 1 || summary.ByType.LinkRemove != 1 {
		t.Errorf("by_type wrong: %+v", summary.ByType)
	}

	// total_suggestions must reconcile with the sum of by_type buckets.
	sum := summary.ByType.Insert + summary.ByType.Delete + summary.ByType.Replace +
		summary.ByType.LinkAdd + summary.ByType.LinkChange + summary.ByType.LinkRemove
	if sum != summary.TotalSuggestions {
		t.Errorf("by_type sum %d != total %d", sum, summary.TotalSuggestions)
	}

	if f := summary.ByFile["a"]; f.LinkAdds != 1 || f.LinkChanges != 1 {
		t.Errorf("by_file[a] = %+v", f)
	}
	if f := summary.ByFile["b"]; f.LinkRemoves != 1 {
		t.Errorf("by_file[b] = %+v", f)
	}
}
