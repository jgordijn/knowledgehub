package engine

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/stealth"
	"github.com/pocketbase/pocketbase/core"
)

// BrowserExtractFunc extracts article content using a headless browser.
// Override in tests to avoid needing a real browser.
var BrowserExtractFunc = defaultBrowserExtract

func defaultBrowserExtract(articleURL string) (ExtractedContent, error) {
	browser, cleanup, err := launchBrowser()
	if err != nil {
		return ExtractedContent{}, err
	}
	defer cleanup()

	page, err := openStealthPage(browser, articleURL)
	if err != nil {
		return ExtractedContent{}, err
	}
	defer page.Close()

	waitForContent(page, articleURL)

	htmlPage := page.Timeout(10 * time.Second)
	html, err := htmlPage.HTML()
	if err != nil {
		return ExtractedContent{}, fmt.Errorf("getting HTML from %s: %w", articleURL, err)
	}

	return ExtractContentFromHTML(html, articleURL), nil
}

// extractWithBrowserFallback tries plain HTTP extraction first, falling back to
// browser-based extraction when bot protection is detected. If the browser
// succeeds, the resource is marked with use_browser=true for future calls.
func extractWithBrowserFallback(app core.App, resource *core.Record, articleURL string, client *http.Client) (ExtractedContent, error) {
	useBrowser := resource.GetBool("use_browser")

	if !useBrowser {
		extracted, err := ExtractContent(articleURL, client)
		if err == nil {
			return extracted, nil
		}
		if !looksLikeBotProtection(err) {
			return extracted, err
		}
		log.Printf("Bot protection detected for %s, trying browser extraction", articleURL)
	}

	extracted, err := BrowserExtractFunc(articleURL)
	if err != nil {
		return ExtractedContent{}, err
	}

	// Auto-learn: mark resource for browser extraction on future fetches
	if !useBrowser {
		resource.Set("use_browser", true)
		if saveErr := app.Save(resource); saveErr != nil {
			log.Printf("Failed to set use_browser for resource %s: %v", resource.Id, saveErr)
		}
		log.Printf("Marked resource %q for browser extraction", resource.GetString("name"))
	}

	return extracted, nil
}

// BrowserFetchBodyFunc fetches a URL using a headless browser and returns the
// raw page content (XML for feeds, HTML for web pages). Override in tests.
var BrowserFetchBodyFunc = defaultBrowserFetchBody

func defaultBrowserFetchBody(targetURL string) (string, error) {
	browser, cleanup, err := launchBrowser()
	if err != nil {
		return "", err
	}
	defer cleanup()

	page, err := openStealthPage(browser, targetURL)
	if err != nil {
		return "", err
	}
	defer page.Close()

	waitForContent(page, targetURL)

	htmlPage := page.Timeout(10 * time.Second)
	// XMLSerializer preserves the original XML for RSS/Atom feeds and
	// returns valid HTML for regular web pages.
	result, err := htmlPage.Eval(`() => new XMLSerializer().serializeToString(document)`)
	if err != nil {
		return "", fmt.Errorf("getting content from %s: %w", targetURL, err)
	}

	body := result.Value.Str()
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("browser returned empty content from %s", targetURL)
	}

	return body, nil
}

// launchBrowser launches a headless Chrome with anti-bot-detection settings.
// NoSandbox is required for LXC containers where Chrome's sandbox namespaces
// are not available. This is safe for a private app behind Tailscale.
func launchBrowser() (*rod.Browser, func(), error) {
	u, err := launcher.New().
		NoSandbox(true).
		Set(flags.Flag("disable-blink-features"), "AutomationControlled").
		// Use Chrome's new headless mode which is indistinguishable from
		// headful Chrome (same TLS fingerprint, User-Agent, etc.).
		Set(flags.Flag("headless"), "new").
		Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, nil, fmt.Errorf("connecting to browser: %w", err)
	}
	return browser, func() { browser.Close() }, nil
}

// openStealthPage creates a browser page with comprehensive anti-detection
// scripts (via go-rod/stealth), then navigates to the target URL.
func openStealthPage(browser *rod.Browser, targetURL string) (*rod.Page, error) {
	page, err := stealth.Page(browser)
	if err != nil {
		return nil, fmt.Errorf("creating stealth page: %w", err)
	}

	if err := page.Navigate(targetURL); err != nil {
		page.Close()
		return nil, fmt.Errorf("navigating to %s: %w", targetURL, err)
	}

	return page, nil
}

// waitForContent waits for the page to load, handling JS challenge pages
// (e.g., nitter proof-of-work) that compute a cookie and reload.
func waitForContent(page *rod.Page, targetURL string) {
	waitPage := page.Timeout(40 * time.Second)
	_ = waitPage.WaitLoad()

	// Brief pause for onload JS (e.g., proof-of-work challenges) to execute
	// and set verification cookies before we inspect the content.
	time.Sleep(2 * time.Second)

	// Check whether the loaded page is a JS challenge rather than real content.
	checkPage := page.Timeout(5 * time.Second)
	html, err := checkPage.HTML()
	if err != nil || !looksLikeChallengePage(html) {
		// Normal page — wait for DOM stability (AJAX, etc.) and return.
		stablePage := page.Timeout(5 * time.Second)
		_ = stablePage.WaitStable(2 * time.Second)
		return
	}

	// Challenge detected — the JS should have set the verification cookie.
	// Navigate again so the server sees the cookie and returns real content.
	log.Printf("Challenge page detected for %s, re-navigating after cookie set", targetURL)
	_ = page.Navigate(targetURL)

	reloadPage := page.Timeout(30 * time.Second)
	_ = reloadPage.WaitLoad()
	_ = reloadPage.WaitStable(2 * time.Second)
}

// looksLikeChallengePage checks if the HTML looks like a browser-verification
// challenge rather than actual content.
func looksLikeChallengePage(html string) bool {
	lower := strings.ToLower(html)
	return strings.Contains(lower, "verifying your browser") ||
		strings.Contains(lower, "checking your browser") ||
		strings.Contains(lower, "just a moment") ||
		strings.Contains(lower, "challenge-platform")
}


// looksLikeBotProtection checks if an HTTP error suggests the server is
// blocking automated access (Cloudflare, anti-bot challenges, etc).
func looksLikeBotProtection(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 403") ||
		strings.Contains(msg, "HTTP 429") ||
		strings.Contains(msg, "HTTP 503")
}

// looksLikeFeedProtection checks if an RSS fetch error suggests bot protection
// that a browser might bypass (e.g., Cloudflare challenge pages).
// Empty responses are excluded — they indicate a hard block where even a real
// browser gets nothing, so attempting browser fallback just wastes time.
func looksLikeFeedProtection(err error) bool {
	return looksLikeBotProtection(err)
}
