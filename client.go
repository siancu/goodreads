package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const baseURL = "https://www.goodreads.com"

var cookieFile = filepath.Join(homeDir(), ".goodreads-cookies.json")

// userAgent is the browser User-Agent we send with every request.
// Goodreads blocks requests that don't look like a real browser.
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"

// cookieData is the JSON structure we persist to disk.
// It stores both the session cookies and the user's Goodreads ID.
type cookieData struct {
	Cookies map[string]string `json:"cookies"`
	UserID  string            `json:"user_id,omitempty"`
}

// homeDir returns the user's home directory, or panics if it can't be determined.
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("cannot determine home directory: %v", err))
	}
	return home
}

// loadCookieData reads the full cookie file from disk.
// Returns nil if the file doesn't exist or is malformed.
func loadCookieData() *cookieData {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return nil
	}
	var cd cookieData
	if err := json.Unmarshal(data, &cd); err != nil {
		return nil
	}
	return &cd
}

// saveCookieData writes cookie data to disk with restricted permissions (0600).
func saveCookieData(cd *cookieData) error {
	data, err := json.MarshalIndent(cd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cookies: %w", err)
	}
	return os.WriteFile(cookieFile, data, 0600)
}

// getUserID reads the saved user ID from the cookie file.
// Exits the program if not logged in — this is intentional:
// most commands can't work without a user ID.
func getUserID() string {
	cd := loadCookieData()
	if cd == nil {
		fmt.Fprintln(os.Stderr, "Error: Not logged in. Run 'goodreads-go login' first.")
		os.Exit(1)
	}
	if cd.UserID == "" {
		fmt.Fprintln(os.Stderr, "Error: No user ID saved. Run 'goodreads-go login' again.")
		os.Exit(1)
	}
	return cd.UserID
}

// newClient creates an HTTP client configured with:
//   - the saved session cookies
//   - a browser-like User-Agent header
//   - automatic redirect following
//
// This is the client used by all authenticated commands.
func newClient() *http.Client {
	cd := loadCookieData()
	if cd == nil || len(cd.Cookies) == 0 {
		fmt.Fprintln(os.Stderr, "Error: Not logged in. Run 'goodreads-go login' first.")
		os.Exit(1)
	}

	// Build the cookie jar and pre-populate it with our saved cookies.
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(baseURL)
	var cookies []*http.Cookie
	for name, value := range cd.Cookies {
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
	}
	jar.SetCookies(u, cookies)

	return &http.Client{Jar: jar}
}

// doGet performs a GET request with browser-like headers.
// It's a convenience wrapper that adds the User-Agent and other headers
// that Goodreads expects.
func doGet(client *http.Client, rawURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", baseURL+"/")
	return client.Do(req)
}

// doPost performs a POST request with form-encoded data and browser-like headers.
func doPost(client *http.Client, rawURL string, data url.Values, referer string) (*http.Response, error) {
	req, err := http.NewRequest("POST", rawURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	return client.Do(req)
}

// parseHTML parses an HTTP response body into a goquery document.
// The caller should still close resp.Body after calling this.
func parseHTML(resp *http.Response) (*goquery.Document, error) {
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

// extractFormData finds a form in the document and returns its action URL
// and all input field name/value pairs. It tries several strategies to find
// the form: by name, by ID "ap_signin_form", or just the first form.
func extractFormData(doc *goquery.Document, formName string) (action string, fields url.Values) {
	fields = url.Values{}

	// Try to find the form by name attribute, then by ID, then first form.
	form := doc.Find(fmt.Sprintf("form[name=%s]", formName))
	if form.Length() == 0 {
		form = doc.Find("form#ap_signin_form")
	}
	if form.Length() == 0 {
		form = doc.Find("form").First()
	}
	if form.Length() == 0 {
		return "", fields
	}

	action, _ = form.Attr("action")

	// Collect all <input> fields — this includes hidden fields like CSRF tokens.
	form.Find("input").Each(func(_ int, s *goquery.Selection) {
		name, exists := s.Attr("name")
		if !exists || name == "" {
			return
		}
		value, _ := s.Attr("value")
		fields.Set(name, value)
	})

	return action, fields
}

// csrfToken extracts the Rails CSRF token from a <meta name="csrf-token"> tag.
func csrfToken(doc *goquery.Document) string {
	token, _ := doc.Find(`meta[name="csrf-token"]`).Attr("content")
	return token
}
