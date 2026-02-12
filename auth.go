package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/term"
)

// cmdLogin authenticates with Goodreads via Amazon's sign-in flow.
//
// The flow is:
//  1. GET goodreads.com/user/sign_in → find the email sign-in link
//  2. Follow that link to Amazon's auth page
//  3. Submit email (and possibly password) via the form
//  4. If Amazon asked for email only, submit password on the next page
//  5. Verify login by fetching /user/edit and extracting the user ID
//  6. Save all cookies to disk
func cmdLogin(email, password string, debug bool) {
	// Resolve credentials: flags → env vars → interactive prompt
	if email == "" {
		email = os.Getenv("GOODREADS_EMAIL")
	}
	if password == "" {
		password = os.Getenv("GOODREADS_PASSWORD")
	}
	if email == "" {
		email = prompt("Email: ")
	}
	if password == "" {
		password = promptPassword("Password: ")
	}

	// Create a fresh client with an empty cookie jar.
	// The jar accumulates cookies across all requests during login.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Step 1: Fetch the Goodreads sign-in page.
	fmt.Println("Fetching sign-in page...")
	resp, err := doGet(client, baseURL+"/user/sign_in")
	if err != nil {
		fatal("fetching sign-in page: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("sign-in page returned HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing sign-in page: %v", err)
	}

	if debug {
		saveDebug(doc, "/tmp/goodreads_signin.html")
	}

	// Step 2: Find the email sign-in link.
	// Goodreads shows multiple sign-in options (Google, Apple, Amazon, email).
	// The email link goes to Amazon's ap/signin without an identityProvider param.
	var signinLink string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, "ap/signin") && !strings.Contains(href, "identityProvider") {
			signinLink = href
		}
	})

	if signinLink == "" {
		fatal("could not find email sign-in link")
	}

	// Make the link absolute if it's relative.
	if !strings.HasPrefix(signinLink, "http") {
		signinLink = baseURL + signinLink
	}

	// Step 3: Follow the link to Amazon's auth page.
	fmt.Println("Following sign-in link to Amazon auth...")
	resp, err = doGet(client, signinLink)
	if err != nil {
		fatal("fetching Amazon auth page: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("Amazon auth page returned HTTP %d", resp.StatusCode)
	}

	doc, err = parseHTML(resp)
	if err != nil {
		fatal("parsing Amazon auth page: %v", err)
	}
	currentURL := resp.Request.URL.String()

	if debug {
		saveDebug(doc, "/tmp/goodreads_amazon_auth.html")
	}

	// Step 4: Fill in the sign-in form.
	// Amazon has two possible flows:
	//   - Email + password on the same page
	//   - Email first, then password on a second page
	form := doc.Find("form[name=signIn]")
	if form.Length() == 0 {
		fatal("could not find login form on Amazon auth page")
	}

	action, fields := extractFormData(doc, "signIn")
	if action == "" {
		fatal("could not find form action URL")
	}
	action = resolveURL(currentURL, action)

	// Check if the password field is visible (not hidden with a "hide" class).
	passwordInput := form.Find("input[name=password]")
	passwordVisible := passwordInput.Length() > 0 &&
		!passwordInput.HasClass("hide")

	fields.Set("email", email)

	if passwordVisible {
		// Both fields visible — submit together.
		fields.Set("password", password)
		fmt.Println("Submitting credentials...")
	} else {
		// Email-only step — remove the password placeholder.
		fields.Del("password")
		fmt.Println("Submitting email...")
	}

	resp, err = doPost(client, action, fields, currentURL)
	if err != nil {
		fatal("submitting credentials: %v", err)
	}

	doc, err = parseHTML(resp)
	if err != nil {
		fatal("parsing response: %v", err)
	}
	currentURL = resp.Request.URL.String()

	if debug {
		saveDebug(doc, "/tmp/goodreads_step1_response.html")
	}

	// Step 5: If we only submitted email, now submit the password.
	if !passwordVisible {
		checkCaptcha(doc)

		action, fields = extractFormData(doc, "signIn")
		if action == "" {
			fatal("could not find password form")
		}
		action = resolveURL(currentURL, action)

		fields.Set("password", password)
		fmt.Println("Submitting password...")

		resp, err = doPost(client, action, fields, currentURL)
		if err != nil {
			fatal("submitting password: %v", err)
		}

		doc, err = parseHTML(resp)
		if err != nil {
			fatal("parsing password response: %v", err)
		}

		if debug {
			saveDebug(doc, "/tmp/goodreads_step2_response.html")
		}
	}

	// Step 6: Check for errors.
	checkLoginErrors(doc)

	// Step 7: Verify login by fetching the user's edit page.
	fmt.Println("Verifying login...")
	resp, err = doGet(client, baseURL+"/user/edit")
	if err != nil {
		fatal("verifying login: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("login verification failed (HTTP %d)", resp.StatusCode)
	}

	doc, err = parseHTML(resp)
	if err != nil {
		fatal("parsing verification page: %v", err)
	}

	if strings.Contains(strings.ToLower(resp.Request.URL.String()), "sign_in") {
		fatal("login failed — still on sign-in page")
	}

	// Step 8: Extract user ID from the page.
	userID := extractUserID(doc)

	// Step 9: Collect all cookies from the jar and save them.
	allCookies := make(map[string]string)
	u, _ := url.Parse(baseURL)
	for _, c := range jar.Cookies(u) {
		allCookies[c.Name] = c.Value
	}
	// Also grab Amazon cookies (some are needed for auth).
	amazonURL, _ := url.Parse("https://www.amazon.com")
	for _, c := range jar.Cookies(amazonURL) {
		allCookies[c.Name] = c.Value
	}

	if len(allCookies) == 0 {
		fatal("no cookies captured — login may have failed")
	}

	if err := saveCookieData(&cookieData{Cookies: allCookies, UserID: userID}); err != nil {
		fatal("saving cookies: %v", err)
	}

	fmt.Printf("Login successful! Cookies saved to %s\n", cookieFile)
	if userID != "" {
		fmt.Printf("User ID: %s\n", userID)
	} else {
		fmt.Fprintln(os.Stderr, "Warning: Could not determine user ID. Some commands may not work.")
	}
}

// cmdLogout removes the saved cookie file.
func cmdLogout() {
	if err := os.Remove(cookieFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Not logged in.")
			return
		}
		fatal("removing cookie file: %v", err)
	}
	fmt.Println("Logged out. Cookies removed.")
}

// --- helpers ---

// extractUserID tries to find the user's Goodreads ID from a page.
// It checks profile links first, then falls back to searching <script> tags.
func extractUserID(doc *goquery.Document) string {
	re := regexp.MustCompile(`/user/show/(\d+)`)

	// Try profile links.
	var userID string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if userID != "" {
			return // already found
		}
		href, _ := s.Attr("href")
		if m := re.FindStringSubmatch(href); m != nil {
			userID = m[1]
		}
	})
	if userID != "" {
		return userID
	}

	// Fallback: search JavaScript for currentUserId.
	reJS := regexp.MustCompile(`"currentUserId"\s*:\s*"?(\d+)"?`)
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if userID != "" {
			return
		}
		if m := reJS.FindStringSubmatch(s.Text()); m != nil {
			userID = m[1]
		}
	})

	return userID
}

// checkCaptcha aborts if a CAPTCHA is detected on the page.
func checkCaptcha(doc *goquery.Document) {
	if doc.Find("#auth-captcha-image-container").Length() > 0 ||
		doc.Find("#auth-captcha-image").Length() > 0 {
		fatal("CAPTCHA required. Please use browser-based login.")
	}
}

// checkLoginErrors aborts if an error message, CAPTCHA, or 2FA prompt is found.
func checkLoginErrors(doc *goquery.Document) {
	// Error message box.
	errBox := doc.Find("#auth-error-message-box")
	if errBox.Length() > 0 {
		fatal("login failed — %s", strings.TrimSpace(errBox.Text()))
	}

	// CAPTCHA.
	checkCaptcha(doc)

	// Two-factor auth.
	if doc.Find("input[name=otpCode]").Length() > 0 {
		fatal("two-factor authentication required. Please use browser-based login.")
	}
}

// resolveURL makes a potentially-relative URL absolute using the given base.
func resolveURL(base, ref string) string {
	if strings.HasPrefix(ref, "http") {
		return ref
	}
	baseU, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refU, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseU.ResolveReference(refU).String()
}

// prompt reads a line of text from stdin with the given label.
func prompt(label string) string {
	fmt.Print(label)
	var s string
	fmt.Scanln(&s)
	return s
}

// promptPassword reads a password from the terminal without echoing it.
// This uses the golang.org/x/term package to disable echo.
func promptPassword(label string) string {
	fmt.Print(label)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // newline after hidden input
	if err != nil {
		fatal("reading password: %v", err)
	}
	return string(b)
}

// saveDebug writes a goquery document's HTML to a file for debugging.
func saveDebug(doc *goquery.Document, path string) {
	html, err := doc.Html()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Debug: could not serialize HTML: %v\n", err)
		return
	}
	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Debug: could not write %s: %v\n", path, err)
		return
	}
	fmt.Printf("Debug: Saved HTML to %s\n", path)
}

// fatal prints an error message and exits.
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
