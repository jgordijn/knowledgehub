package engine

import (
	"fmt"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestParseFragmentGroups_Valid(t *testing.T) {
	response := `{"groups": [[0, 1], [2], [3, 4]]}`
	groups, err := parseFragmentGroups(response, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups[0]) != 2 || groups[0][0] != 0 || groups[0][1] != 1 {
		t.Errorf("group 0 = %v, want [0, 1]", groups[0])
	}
}

func TestParseFragmentGroups_InCodeBlock(t *testing.T) {
	response := "```json\n{\"groups\": [[0], [1, 2]]}\n```"
	groups, err := parseFragmentGroups(response, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestParseFragmentGroups_InPlainCodeBlock(t *testing.T) {
	response := "```\n{\"groups\": [[0], [1]]}\n```"
	groups, err := parseFragmentGroups(response, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestParseFragmentGroups_InvalidJSON(t *testing.T) {
	_, err := parseFragmentGroups("not json", 5)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFragmentGroups_EmptyGroupsList(t *testing.T) {
	_, err := parseFragmentGroups(`{"groups": []}`, 5)
	if err == nil {
		t.Error("expected error for empty groups")
	}
}

func TestParseFragmentGroups_EmptyGroupInList(t *testing.T) {
	_, err := parseFragmentGroups(`{"groups": [[]]}`, 5)
	if err == nil {
		t.Error("expected error for empty group in response")
	}
}

func TestParseFragmentGroups_IndexOutOfRange(t *testing.T) {
	_, err := parseFragmentGroups(`{"groups": [[0, 10]]}`, 5)
	if err == nil {
		t.Error("expected error for out of range index")
	}
}

func TestParseFragmentGroups_NegativeIndex(t *testing.T) {
	_, err := parseFragmentGroups(`{"groups": [[-1]]}`, 5)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestSaveFragmentHashes_PersistsCorrectly(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	resource.Set("fragment_feed", true)
	app.Save(resource)

	hashes := map[string]string{
		"guid-1": "hash-1",
		"guid-2": "hash-2",
	}

	err := saveFragmentHashes(app, resource, hashes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := app.FindRecordById("resources", resource.Id)
	loaded := loadFragmentHashes(updated)
	if len(loaded) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(loaded))
	}
	if loaded["guid-1"] != "hash-1" {
		t.Errorf("hash for guid-1 = %q, want 'hash-1'", loaded["guid-1"])
	}
}

func TestLoadFragmentHashes_EmptyField(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	hashes := loadFragmentHashes(resource)
	if len(hashes) != 0 {
		t.Errorf("expected empty map, got %d entries", len(hashes))
	}
}

func TestLoadFragmentHashes_MalformedJSON(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)
	resource.Set("fragment_hashes", "not-json-at-all")
	app.Save(resource)

	hashes := loadFragmentHashes(resource)
	if len(hashes) != 0 {
		t.Errorf("expected empty map for invalid JSON, got %d entries", len(hashes))
	}
}

func TestExtractText_FromHTML(t *testing.T) {
	html := "<p>Hello <strong>World</strong></p>"
	text := extractText(html)
	if text != "Hello World" {
		t.Errorf("extractText = %q, want 'Hello World'", text)
	}
}

func TestExtractText_EmptyInput(t *testing.T) {
	text := extractText("")
	_ = text // Just ensure no panic
}

func TestResolveContentLinks_RelativeHrefs(t *testing.T) {
	html := `<a href="/path/to/article">Link</a>`
	result := resolveContentLinks(html, "https://example.com/page")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if result != `<a href="https://example.com/path/to/article">Link</a>` {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestResolveContentLinks_RelativeSrc(t *testing.T) {
	html := `<img src="/images/photo.jpg"/>`
	result := resolveContentLinks(html, "https://example.com/page")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if result == html {
		t.Errorf("src should have been resolved, got: %s", result)
	}
}

func TestResolveContentLinks_InvalidBase(t *testing.T) {
	html := `<a href="/path">Link</a>`
	result := resolveContentLinks(html, "://invalid")
	if result != html {
		t.Errorf("should return original html for invalid base URL, got: %s", result)
	}
}

func TestResolveContentLinks_AbsoluteURLUnchanged(t *testing.T) {
	html := `<a href="https://other.com/page">Link</a>`
	result := resolveContentLinks(html, "https://example.com")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if result != `<a href="https://other.com/page">Link</a>` {
		t.Errorf("absolute URL should remain unchanged: %s", result)
	}
}

func TestSplitFragmentsWithAI_SingleFragmentNoAI(t *testing.T) {
	html := "<p>Single paragraph</p>"
	fragments := SplitFragmentsWithAI(html, "key", "model")
	if len(fragments) != 1 {
		t.Errorf("expected 1 fragment for single paragraph, got %d", len(fragments))
	}
}

func TestSplitFragmentsWithAI_AIUnavailable(t *testing.T) {
	html := "<p>Paragraph one</p><p>Paragraph two</p>"

	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return "", fmt.Errorf("AI unavailable")
	})
	defer restore()

	fragments := SplitFragmentsWithAI(html, "key", "model")
	if len(fragments) != 2 {
		t.Errorf("expected 2 fragments (heuristic fallback), got %d", len(fragments))
	}
}

func TestSplitFragmentsWithAI_AIBadResponse(t *testing.T) {
	html := "<p>Paragraph one</p><p>Paragraph two</p>"

	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return "completely invalid json response", nil
	})
	defer restore()

	fragments := SplitFragmentsWithAI(html, "key", "model")
	if len(fragments) != 2 {
		t.Errorf("expected 2 fragments (heuristic fallback), got %d", len(fragments))
	}
}

func TestSplitFragmentsWithAI_SuccessfulGrouping(t *testing.T) {
	html := "<p>Paragraph one about Go</p><p>More about Go</p><p>Unrelated topic</p>"

	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"groups": [[0, 1], [2]]}`, nil
	})
	defer restore()

	fragments := SplitFragmentsWithAI(html, "key", "model")
	if len(fragments) != 2 {
		t.Errorf("expected 2 merged fragments, got %d", len(fragments))
	}
}

func TestMergeFragments_Groups(t *testing.T) {
	initial := []Fragment{
		{HTML: "<p>First</p>", Title: "First"},
		{HTML: "<p>Second</p>", Title: "Second"},
		{HTML: "<p>Third</p>", Title: "Third"},
	}
	groups := [][]int{{0, 1}, {2}}
	result := mergeFragments(initial, groups)
	if len(result) != 2 {
		t.Fatalf("expected 2 merged fragments, got %d", len(result))
	}
}

func TestContentSHA256_Deterministic(t *testing.T) {
	hash1 := contentSHA256("hello")
	hash2 := contentSHA256("hello")
	hash3 := contentSHA256("world")

	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different content should produce different hashes")
	}
	if len(hash1) != 64 {
		t.Errorf("hash length = %d, want 64 hex chars", len(hash1))
	}
}

func TestFragmentGUID_Unique(t *testing.T) {
	guid1 := FragmentGUID("parent-1", "<p>Content A</p>")
	guid2 := FragmentGUID("parent-1", "<p>Content B</p>")

	if guid1 == guid2 {
		t.Error("different content should produce different GUIDs")
	}
	if guid1[:len("parent-1#frag-")] != "parent-1#frag-" {
		t.Errorf("GUID should start with 'parent-1#frag-': %s", guid1)
	}
}

func TestSplitFragments_HRDiscarded(t *testing.T) {
	html := "<p>First</p><hr/><p>Second</p>"
	fragments := SplitFragments(html)
	if len(fragments) != 2 {
		t.Errorf("expected 2 fragments (HR discarded), got %d", len(fragments))
	}
}

func TestSplitFragments_BlockElementsAttach(t *testing.T) {
	html := "<p>Topic intro</p><blockquote>A quote</blockquote><p>New topic</p>"
	fragments := SplitFragments(html)
	if len(fragments) != 2 {
		t.Errorf("expected 2 fragments, got %d", len(fragments))
	}
}

func TestSplitFragments_EmptyString(t *testing.T) {
	fragments := SplitFragments("")
	if len(fragments) != 0 {
		t.Errorf("expected 0 fragments for empty HTML, got %d", len(fragments))
	}
}

func TestNewFragment_LongTitleTruncation(t *testing.T) {
	longText := ""
	for i := 0; i < 200; i++ {
		longText += "word "
	}
	html := "<p>" + longText + "</p>"
	frag := newFragment(html)
	if len(frag.Title) > 123 { // 120 chars + "…" (multi-byte)
		t.Errorf("title should be truncated, length = %d", len(frag.Title))
	}
}

func TestTitleSimilarity_Cases(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		min  float64
		max  float64
	}{
		{"identical words", "hello world", "hello world", 1.0, 1.0},
		{"case insensitive match", "Hello World", "hello world", 1.0, 1.0},
		{"completely different", "hello world", "foo bar", 0.0, 0.0},
		{"partial overlap", "hello world foo", "hello world bar", 0.4, 0.7},
		{"both empty strings", "", "", 1.0, 1.0},
		{"one empty string", "hello", "", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := titleSimilarity(tt.a, tt.b)
			if score < tt.min || score > tt.max {
				t.Errorf("titleSimilarity(%q, %q) = %f, want in [%f, %f]", tt.a, tt.b, score, tt.min, tt.max)
			}
		})
	}
}

func TestSetFragmentCompleteFunc_Restores(t *testing.T) {
	called := false
	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		called = true
		return `{"groups":[[0]]}`, nil
	})
	defer restore()

	_, _ = callFragmentComplete("key", "model", []ai.Message{{Role: "user", Content: "test"}})
	if !called {
		t.Error("custom function should have been called")
	}
}
