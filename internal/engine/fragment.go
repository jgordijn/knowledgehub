package engine

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"

	"github.com/jgordijn/knowledgehub/internal/ai"
)

// Fragment represents a single snippet extracted from a fragment feed entry.
type Fragment struct {
	HTML  string // The HTML content of this fragment
	Title string // Auto-generated title from the text content
}

// fragmentCompleteMu protects fragmentCompleteFunc from concurrent test modifications.
var fragmentCompleteMu sync.RWMutex

// fragmentCompleteFunc is the function used to call the AI for fragment grouping.
// It can be overridden in tests via SetFragmentCompleteFunc.
var fragmentCompleteFunc = func(apiKey, model string, messages []ai.Message) (string, error) {
	client := ai.NewClient(apiKey, model)
	return client.Complete(messages)
}

// callFragmentComplete invokes fragmentCompleteFunc with read-lock protection.
func callFragmentComplete(apiKey, model string, messages []ai.Message) (string, error) {
	fragmentCompleteMu.RLock()
	fn := fragmentCompleteFunc
	fragmentCompleteMu.RUnlock()
	return fn(apiKey, model, messages)
}

// SetFragmentCompleteFunc replaces fragmentCompleteFunc for testing and returns a restore function.
func SetFragmentCompleteFunc(fn func(apiKey, model string, messages []ai.Message) (string, error)) func() {
	fragmentCompleteMu.Lock()
	orig := fragmentCompleteFunc
	fragmentCompleteFunc = fn
	fragmentCompleteMu.Unlock()
	return func() {
		fragmentCompleteMu.Lock()
		fragmentCompleteFunc = orig
		fragmentCompleteMu.Unlock()
	}
}

// SplitFragments splits HTML content into individual fragments using a heuristic.
// A new fragment starts at each <p> element. Block-level elements like
// <blockquote>, <ul>, <ol>, and <pre> are appended to the current fragment.
// <hr> elements are discarded.
func SplitFragments(html string) []Fragment {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var fragments []Fragment
	var current strings.Builder

	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		tagName := goquery.NodeName(s)

		if tagName == "hr" {
			return
		}

		// A <p> starts a new fragment; flush any accumulated content first
		if tagName == "p" {
			if current.Len() > 0 {
				fragments = append(fragments, newFragment(current.String()))
				current.Reset()
			}
		}

		h, _ := goquery.OuterHtml(s)
		current.WriteString(h)
	})

	// Flush the last fragment
	if current.Len() > 0 {
		fragments = append(fragments, newFragment(current.String()))
	}

	return fragments
}

// SplitFragmentsWithAI uses the heuristic splitter as a first pass, then asks
// the LLM to re-group fragments that belong to the same topic.
// Falls back to the heuristic result on any AI error.
func SplitFragmentsWithAI(html, apiKey, model string) []Fragment {
	initial := SplitFragments(html)
	if len(initial) <= 1 {
		return initial
	}

	// Build a numbered text preview for the LLM
	var sb strings.Builder
	for i, f := range initial {
		text := extractText(f.HTML)
		if len(text) > 300 {
			text = text[:300] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i, text))
	}

	prompt := fmt.Sprintf(`These numbered blocks are extracted from a blog post that contains multiple short topics/moments. Group consecutive blocks that belong to the same topic into fragments. A commentary paragraph about a preceding quote belongs with that quote.

%s
Return JSON only: {"groups": [[0, 1], [2], ...]}`, sb.String())

	response, err := callFragmentComplete(apiKey, model, []ai.Message{
		{Role: "system", Content: "You group content blocks into coherent fragments. Always respond with valid JSON."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		log.Printf("AI fragment grouping failed, using heuristic: %v", err)
		return initial
	}

	groups, err := parseFragmentGroups(response, len(initial))
	if err != nil {
		log.Printf("Failed to parse AI fragment groups, using heuristic: %v", err)
		return initial
	}

	return mergeFragments(initial, groups)
}

// parseFragmentGroups parses the LLM response into groups of indices.
func parseFragmentGroups(response string, maxIndex int) ([][]int, error) {
	response = strings.TrimSpace(response)

	// Extract JSON from markdown code blocks if present
	if idx := strings.Index(response, "```json"); idx >= 0 {
		start := idx + 7
		end := strings.Index(response[start:], "```")
		if end >= 0 {
			response = response[start : start+end]
		}
	} else if idx := strings.Index(response, "```"); idx >= 0 {
		start := idx + 3
		end := strings.Index(response[start:], "```")
		if end >= 0 {
			response = response[start : start+end]
		}
	}

	response = strings.TrimSpace(response)

	var result struct {
		Groups [][]int `json:"groups"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON %q: %w", response, err)
	}

	if len(result.Groups) == 0 {
		return nil, fmt.Errorf("empty groups")
	}

	// Validate indices
	for _, group := range result.Groups {
		if len(group) == 0 {
			return nil, fmt.Errorf("empty group in response")
		}
		for _, idx := range group {
			if idx < 0 || idx >= maxIndex {
				return nil, fmt.Errorf("index %d out of range [0, %d)", idx, maxIndex)
			}
		}
	}

	return result.Groups, nil
}

// mergeFragments combines initial fragments according to the AI-determined groups.
func mergeFragments(initial []Fragment, groups [][]int) []Fragment {
	var result []Fragment
	for _, group := range groups {
		var htmlParts []string
		for _, idx := range group {
			htmlParts = append(htmlParts, initial[idx].HTML)
		}
		merged := strings.Join(htmlParts, "\n")
		result = append(result, newFragment(merged))
	}
	return result
}

func newFragment(html string) Fragment {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	title := ""
	if err == nil {
		title = strings.TrimSpace(doc.Text())
	}
	// Collapse whitespace for a clean title
	title = strings.Join(strings.Fields(title), " ")
	if len(title) > 120 {
		title = title[:120] + "…"
	}
	return Fragment{HTML: strings.TrimSpace(html), Title: title}
}

func extractText(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}
	text := strings.TrimSpace(doc.Text())
	return strings.Join(strings.Fields(text), " ")
}

// FragmentGUID generates a stable, unique GUID for a fragment using
// the parent entry's GUID and a hash of the fragment content.
func FragmentGUID(parentGUID, fragmentHTML string) string {
	h := sha256.Sum256([]byte(fragmentHTML))
	return fmt.Sprintf("%s#frag-%x", parentGUID, h[:6])
}

// titleSimilarity returns the word-level Jaccard similarity between two titles.
// Returns a value between 0.0 (no overlap) and 1.0 (identical words).
func titleSimilarity(a, b string) float64 {
	wordsA := titleWords(a)
	wordsB := titleWords(b)
	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}

	union := len(wordsA)
	for w := range wordsB {
		if !wordsA[w] {
			union++
		}
	}

	return float64(intersection) / float64(union)
}

func titleWords(s string) map[string]bool {
	words := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(s)) {
		w = strings.TrimRight(w, ".,;:!?\"'")
		if w != "" {
			words[w] = true
		}
	}
	return words
}



// resolveContentLinks resolves relative href and src attributes in HTML content
// to absolute URLs using the given base URL. This prevents relative links from
// resolving against the knowledgehub domain when rendered in the browser.
func resolveContentLinks(html, baseURLStr string) string {
	base, err := url.Parse(baseURLStr)
	if err != nil {
		return html
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	doc.Find("[href]").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			resolved := resolveURL(base, href)
			if resolved != "" {
				s.SetAttr("href", resolved)
			}
		}
	})

	doc.Find("[src]").Each(func(_ int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			resolved := resolveURL(base, src)
			if resolved != "" {
				s.SetAttr("src", resolved)
			}
		}
	})

	result, err := doc.Find("body").Html()
	if err != nil {
		return html
	}
	return strings.TrimSpace(result)
}