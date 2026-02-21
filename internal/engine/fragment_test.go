package engine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/ai"
	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestSplitFragments_Basic(t *testing.T) {
	html := `<p>First fragment about Go.</p>
<p>Second fragment about Rust.</p>
<p>Third fragment.</p>
<hr>`

	frags := SplitFragments(html)
	if len(frags) != 3 {
		t.Fatalf("got %d fragments, want 3", len(frags))
	}
	if !strings.Contains(frags[0].Title, "First fragment about Go") {
		t.Errorf("frag[0].Title = %q", frags[0].Title)
	}
	if !strings.Contains(frags[1].Title, "Second fragment about Rust") {
		t.Errorf("frag[1].Title = %q", frags[1].Title)
	}
}

func TestSplitFragments_BlockquoteStaysWithParagraph(t *testing.T) {
	html := `<p>David Crawshaw on X:</p>
<blockquote><p><em>The age of malleable software is close.</em></p></blockquote>
<p>Google WebMCP</p>`

	frags := SplitFragments(html)
	if len(frags) != 2 {
		t.Fatalf("got %d fragments, want 2", len(frags))
	}
	if !strings.Contains(frags[0].HTML, "David Crawshaw") {
		t.Error("frag[0] should contain the paragraph")
	}
	if !strings.Contains(frags[0].HTML, "blockquote") {
		t.Error("frag[0] should contain the blockquote")
	}
	if !strings.Contains(frags[1].HTML, "Google WebMCP") {
		t.Error("frag[1] should be the next paragraph")
	}
}

func TestSplitFragments_ListStaysWithParagraph(t *testing.T) {
	html := `<p>Again so subtle:</p>
<ul><li>item one</li><li>item two</li></ul>
<p>Next topic.</p>`

	frags := SplitFragments(html)
	if len(frags) != 2 {
		t.Fatalf("got %d fragments, want 2", len(frags))
	}
	if !strings.Contains(frags[0].HTML, "<ul>") {
		t.Error("frag[0] should contain the list")
	}
}

func TestSplitFragments_HrIgnored(t *testing.T) {
	html := `<p>Before hr.</p><hr><p>After hr.</p>`

	frags := SplitFragments(html)
	if len(frags) != 2 {
		t.Fatalf("got %d fragments, want 2", len(frags))
	}
	for _, f := range frags {
		if strings.Contains(f.HTML, "<hr") {
			t.Error("fragment should not contain <hr>")
		}
	}
}

func TestSplitFragments_EmptyInput(t *testing.T) {
	frags := SplitFragments("")
	if len(frags) != 0 {
		t.Errorf("got %d fragments for empty input, want 0", len(frags))
	}
}

func TestSplitFragments_TitleTruncated(t *testing.T) {
	long := "<p>" + strings.Repeat("x", 200) + "</p>"
	frags := SplitFragments(long)
	if len(frags) != 1 {
		t.Fatalf("got %d fragments, want 1", len(frags))
	}
	if len(frags[0].Title) > 125 { // 120 + "…"
		t.Errorf("title too long: %d chars", len(frags[0].Title))
	}
	if !strings.HasSuffix(frags[0].Title, "…") {
		t.Error("long title should end with ellipsis")
	}
}

func TestSplitFragments_RealWorldMoments(t *testing.T) {
	html := `<p>Via Mario Zechner on <a href="https://x.com">X</a>: The Self-Healing PR</p>
<p>Large PR's are not beneficial to the agent's context window.</p>
<p>Via Martin Fowler fragments, Margaret-Anne Storey: <a href="https://example.com">Cognitive Debt</a>, summary:</p>
<blockquote><p>As AI accelerates development, the bigger risk shifts from technical debt to cognitive debt.</p></blockquote>
<p>Context engineering pattern, <em>compliancy gate pattern</em>:</p>
<blockquote><ul><li>ask the model to do something</li></ul></blockquote>
<p>I know, but it works. Sometimes.</p>
<p>Cloudflare has an <a href="https://example.com/llms.txt">llms.txt</a> for their docs.</p>
<hr>`

	frags := SplitFragments(html)

	if len(frags) != 6 {
		t.Fatalf("got %d fragments, want 6", len(frags))
	}

	// Fragment with blockquote should include it
	found := false
	for _, f := range frags {
		if strings.Contains(f.HTML, "Martin Fowler") && strings.Contains(f.HTML, "blockquote") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a fragment combining Martin Fowler paragraph with its blockquote")
	}
}

func TestFragmentGUID(t *testing.T) {
	guid1 := FragmentGUID("parent-123", "<p>Fragment one</p>")
	guid2 := FragmentGUID("parent-123", "<p>Fragment two</p>")
	guid3 := FragmentGUID("parent-123", "<p>Fragment one</p>")

	if guid1 == guid2 {
		t.Error("different content should produce different GUIDs")
	}
	if guid1 != guid3 {
		t.Error("same content should produce same GUID")
	}
	if !strings.HasPrefix(guid1, "parent-123#frag-") {
		t.Errorf("GUID should start with parent GUID, got %q", guid1)
	}
}

func TestSplitFragmentsWithAI(t *testing.T) {
	html := `<p>Via Mario Zechner on X: The Self-Healing PR</p>
<p>Large PR's are not beneficial to the agent's context window.</p>
<p>Context engineering pattern, compliancy gate pattern:</p>
<blockquote><ul><li>ask the model to do something</li></ul></blockquote>
<p>I know, but it works. Sometimes.</p>
<p>Cloudflare has an llms.txt for their docs.</p>`

	// Heuristic produces 4 fragments: [0] Mario, [1] Large PRs, [2] Context+blockquote, [3] I know, [4] Cloudflare
	// AI should group [0,1], [2,3], [4]
	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return `{"groups": [[0, 1], [2, 3], [4]]}`, nil
	})
	defer restore()

	frags := SplitFragmentsWithAI(html, "test-key", "test-model")

	if len(frags) != 3 {
		t.Fatalf("got %d fragments, want 3", len(frags))
	}
	if !strings.Contains(frags[0].HTML, "Mario Zechner") || !strings.Contains(frags[0].HTML, "not beneficial") {
		t.Error("first fragment should combine Mario + Large PRs")
	}
	if !strings.Contains(frags[1].HTML, "compliancy") || !strings.Contains(frags[1].HTML, "I know") {
		t.Error("second fragment should combine pattern + commentary")
	}
	if !strings.Contains(frags[2].HTML, "Cloudflare") {
		t.Error("third fragment should be Cloudflare")
	}
}

func TestSplitFragmentsWithAI_FallbackOnError(t *testing.T) {
	html := `<p>First.</p><p>Second.</p>`

	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return "", fmt.Errorf("API unavailable")
	})
	defer restore()

	frags := SplitFragmentsWithAI(html, "test-key", "test-model")

	// Should fall back to heuristic (2 fragments)
	if len(frags) != 2 {
		t.Fatalf("got %d fragments, want 2 (heuristic fallback)", len(frags))
	}
}

func TestSplitFragmentsWithAI_FallbackOnBadJSON(t *testing.T) {
	html := `<p>First.</p><p>Second.</p>`

	restore := SetFragmentCompleteFunc(func(apiKey, model string, messages []ai.Message) (string, error) {
		return "not json", nil
	})
	defer restore()

	frags := SplitFragmentsWithAI(html, "test-key", "test-model")

	if len(frags) != 2 {
		t.Fatalf("got %d fragments, want 2 (heuristic fallback)", len(frags))
	}
}

func TestParseFragmentGroups(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxIndex int
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "valid groups",
			input:    `{"groups": [[0, 1], [2], [3, 4]]}`,
			maxIndex: 5,
			wantLen:  3,
		},
		{
			name:     "JSON in code block",
			input:    "```json\n{\"groups\": [[0], [1]]}\n```",
			maxIndex: 2,
			wantLen:  2,
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			maxIndex: 2,
			wantErr:  true,
		},
		{
			name:     "index out of range",
			input:    `{"groups": [[0, 5]]}`,
			maxIndex: 3,
			wantErr:  true,
		},
		{
			name:     "empty groups",
			input:    `{"groups": []}`,
			maxIndex: 2,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := parseFragmentGroups(tt.input, tt.maxIndex)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(groups) != tt.wantLen {
				t.Errorf("got %d groups, want %d", len(groups), tt.wantLen)
			}
		})
	}
}

func TestMergeFragments(t *testing.T) {
	initial := []Fragment{
		{HTML: "<p>A</p>", Title: "A"},
		{HTML: "<p>B</p>", Title: "B"},
		{HTML: "<p>C</p>", Title: "C"},
		{HTML: "<p>D</p>", Title: "D"},
	}

	groups := [][]int{{0, 1}, {2, 3}}
	result := mergeFragments(initial, groups)

	if len(result) != 2 {
		t.Fatalf("got %d fragments, want 2", len(result))
	}
	if !strings.Contains(result[0].HTML, "<p>A</p>") || !strings.Contains(result[0].HTML, "<p>B</p>") {
		t.Error("first fragment should contain A and B")
	}
	if !strings.Contains(result[1].HTML, "<p>C</p>") || !strings.Contains(result[1].HTML, "<p>D</p>") {
		t.Error("second fragment should contain C and D")
	}
}


func TestResolveContentLinks(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		baseURL string
		want    string
	}{
		{
			name:    "empty href resolves to base URL",
			html:    `<p>Check this <a href="">Link</a></p>`,
			baseURL: "https://example.com/blog/",
			want:    `href="https://example.com/blog/"`,
		},
		{
			name:    "relative href resolves to absolute",
			html:    `<p>Read <a href="/article/1">more</a></p>`,
			baseURL: "https://example.com/blog/",
			want:    `href="https://example.com/article/1"`,
		},
		{
			name:    "absolute href unchanged",
			html:    `<p>See <a href="https://other.com/page">this</a></p>`,
			baseURL: "https://example.com/blog/",
			want:    `href="https://other.com/page"`,
		},
		{
			name:    "relative src resolves to absolute",
			html:    `<p><img src="/img/photo.jpg"/></p>`,
			baseURL: "https://example.com/blog/",
			want:    `src="https://example.com/img/photo.jpg"`,
		},
		{
			name:    "invalid base URL returns original",
			html:    `<p><a href="/test">link</a></p>`,
			baseURL: "://invalid",
			want:    `href="/test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveContentLinks(tt.html, tt.baseURL)
			if !strings.Contains(result, tt.want) {
				t.Errorf("resolveContentLinks() = %q, want substring %q", result, tt.want)
			}
		})
	}
}

func TestTitleSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		min  float64
		max  float64
	}{
		{
			name: "identical",
			a:    "Sonnet 4.6 is here. Link",
			b:    "Sonnet 4.6 is here. Link",
			min:  1.0, max: 1.0,
		},
		{
			name: "case difference",
			a:    "Sonnet 4.6 is here. link",
			b:    "Sonnet 4.6 is here. Link",
			min:  1.0, max: 1.0,
		},
		{
			name: "extra words",
			a:    "Need to read: Harness Engineering, via Martin Fowler.",
			b:    "Need to read: Harness Engineering",
			min:  0.6, max: 0.7,
		},
		{
			name: "completely different",
			a:    "Sonnet 4.6 is here",
			b:    "Context engineering pattern",
			min:  0.0, max: 0.15,
		},
		{
			name: "both empty",
			a:    "",
			b:    "",
			min:  1.0, max: 1.0,
		},
		{
			name: "one empty",
			a:    "hello world",
			b:    "",
			min:  0.0, max: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := titleSimilarity(tt.a, tt.b)
			if score < tt.min || score > tt.max {
				t.Errorf("titleSimilarity(%q, %q) = %.3f, want [%.3f, %.3f]", tt.a, tt.b, score, tt.min, tt.max)
			}
		})
	}
}


func TestContentSHA256(t *testing.T) {
	h1 := contentSHA256("<p>Hello</p>")
	h2 := contentSHA256("<p>Hello</p>")
	h3 := contentSHA256("<p>World</p>")

	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(h1))
	}
}

func TestLoadFragmentHashes(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	// Empty field returns empty map
	hashes := loadFragmentHashes(resource)
	if len(hashes) != 0 {
		t.Errorf("expected empty map, got %d entries", len(hashes))
	}

	// Valid JSON
	resource.Set("fragment_hashes", `{"guid-1":"abc123"}`)
	hashes = loadFragmentHashes(resource)
	if hashes["guid-1"] != "abc123" {
		t.Errorf("expected abc123, got %q", hashes["guid-1"])
	}

	// Invalid JSON returns empty map
	resource.Set("fragment_hashes", "not-json")
	hashes = loadFragmentHashes(resource)
	if len(hashes) != 0 {
		t.Errorf("expected empty map for invalid JSON, got %d entries", len(hashes))
	}
}

func TestSaveFragmentHashes(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	resource := testutil.CreateResource(t, app, "test", "https://example.com", "rss", "healthy", 0, true)

	hashes := map[string]string{"guid-1": "hash1", "guid-2": "hash2"}
	err := saveFragmentHashes(app, resource, hashes)
	if err != nil {
		t.Fatalf("saveFragmentHashes returned error: %v", err)
	}

	// Reload and verify
	updated, _ := app.FindRecordById("resources", resource.Id)
	loaded := loadFragmentHashes(updated)
	if loaded["guid-1"] != "hash1" || loaded["guid-2"] != "hash2" {
		t.Errorf("round-trip failed: got %v", loaded)
	}
}