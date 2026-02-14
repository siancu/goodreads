package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// book holds data parsed from a shelf's book table row.
// Fields are optional — not every shelf page includes all columns.
type book struct {
	Title     string
	Author    string
	Rating    string // user's rating (e.g. "really liked it")
	AvgRating string // community average (e.g. "4.52")
	DateRead  string
	DateAdded string
}

// parseBooksFromHTML extracts book data from a Goodreads shelf page.
// Each book is a <tr class="bookalike review"> with fields in <td> cells.
func parseBooksFromHTML(doc *goquery.Document) []book {
	var books []book

	doc.Find("tr.bookalike.review").Each(func(_ int, row *goquery.Selection) {
		b := book{}

		// Title: prefer the title attribute (full title), fall back to link text.
		if el := row.Find("td.field.title a").First(); el.Length() > 0 {
			if title, ok := el.Attr("title"); ok && strings.TrimSpace(title) != "" {
				b.Title = strings.TrimSpace(title)
			} else {
				b.Title = strings.TrimSpace(el.Text())
			}
		}

		// Author: Goodreads sometimes uses "Last, First" — flip it.
		if el := row.Find("td.field.author a").First(); el.Length() > 0 {
			author := strings.TrimSpace(el.Text())
			if parts := strings.SplitN(author, ", ", 2); len(parts) == 2 {
				author = parts[1] + " " + parts[0]
			}
			b.Author = author
		}

		// User's rating.
		if el := row.Find("td.field.rating span.staticStars").First(); el.Length() > 0 {
			b.Rating, _ = el.Attr("title")
		}

		// Community average rating.
		if el := row.Find("td.field.avg_rating div.value").First(); el.Length() > 0 {
			b.AvgRating = strings.TrimSpace(el.Text())
		}

		// Date read.
		if el := row.Find("td.field.date_read span.date_read_value").First(); el.Length() > 0 {
			b.DateRead = strings.TrimSpace(el.Text())
		}

		// Date added.
		if el := row.Find("td.field.date_added span.date_added_value").First(); el.Length() > 0 {
			b.DateAdded = strings.TrimSpace(el.Text())
		}

		if b.Title != "" {
			books = append(books, b)
		}
	})

	return books
}

// cmdShelfList lists all shelves for the current user.
func cmdShelfList() {
	userID := getUserID()
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/review/list/%s", baseURL, userID))
	if err != nil {
		fatal("fetching shelves: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching shelves: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing shelves page: %v", err)
	}

	// Goodreads renders shelves in several possible containers.
	// We try multiple CSS selectors from most specific to least.
	shelves := doc.Find("a.actionLinkLite.bookPageGenreLink")
	if shelves.Length() == 0 {
		shelves = doc.Find("#paginatedShelfList .selectedShelf a, #paginatedShelfList a")
	}
	if shelves.Length() == 0 {
		if container := doc.Find("#paginatedShelfList"); container.Length() > 0 {
			shelves = container.Find("a")
		}
	}
	if shelves.Length() == 0 {
		shelves = doc.Find("a[href*='/review/list/'][href*='shelf=']")
	}

	// Deduplicate and print.
	seen := make(map[string]bool)
	fmt.Println("Your Goodreads Shelves:")
	fmt.Println(strings.Repeat("-", 40))

	shelves.Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.Contains(href, "shelf=") {
			return
		}

		// Extract shelf name from the URL query string.
		// e.g. "/review/list/12345?shelf=to-read&page=1" → "to-read"
		parts := strings.SplitN(href, "shelf=", 2)
		if len(parts) < 2 {
			return
		}
		name := strings.SplitN(parts[1], "&", 2)[0]
		if name == "" || seen[name] {
			return
		}
		seen[name] = true

		// Check for a book count inside the link.
		count := ""
		if span := s.Find("span.smallText"); span.Length() > 0 {
			count = strings.TrimSpace(span.Text())
		}

		if count != "" {
			fmt.Printf("  %s (%s)\n", name, count)
		} else {
			fmt.Printf("  %s\n", name)
		}
	})

	if len(seen) == 0 {
		fmt.Fprintln(os.Stderr, "  No shelves found. Check if your cookies are valid.")
	}
}

// cmdShelfShow displays all books on a named shelf, paginating automatically.
// Books are printed as each page is fetched so you see results immediately.
func cmdShelfShow(shelfName string) {
	userID := getUserID()
	client := newClient()

	total := 0
	headerPrinted := false

	for page := 1; ; page++ {
		u := fmt.Sprintf("%s/review/list/%s?shelf=%s&page=%d", baseURL, userID, url.QueryEscape(shelfName), page)
		resp, err := doGet(client, u)
		if err != nil {
			fatal("fetching shelf page %d: %v", page, err)
		}
		if resp.StatusCode != 200 {
			fatal("fetching shelf: HTTP %d", resp.StatusCode)
		}

		doc, err := parseHTML(resp)
		if err != nil {
			fatal("parsing shelf page %d: %v", page, err)
		}

		books := parseBooksFromHTML(doc)
		if len(books) == 0 {
			break
		}

		// Print header before the first batch of results.
		if !headerPrinted {
			fmt.Printf("Books on '%s':\n", shelfName)
			fmt.Println(strings.Repeat("-", 60))
			headerPrinted = true
		}

		for _, b := range books {
			fmt.Printf("  %s\n", b.Title)
			fmt.Printf("    by %s", b.Author)
			if b.Rating != "" {
				fmt.Printf(" | Your rating: %s", b.Rating)
			}
			if b.AvgRating != "" {
				fmt.Printf(" | Avg: %s", b.AvgRating)
			}
			fmt.Println()
		}

		total += len(books)
	}

	if total == 0 {
		fmt.Fprintf(os.Stderr, "No books found on shelf '%s'.\n", shelfName)
	} else {
		fmt.Printf("\n%d books total\n", total)
	}
}

// cmdShelfAdd creates a new shelf by submitting the "add shelf" form.
func cmdShelfAdd(shelfName string, debug bool) {
	userID := getUserID()
	client := newClient()

	// Fetch the shelf list page which contains the "add shelf" form.
	listURL := fmt.Sprintf("%s/review/list/%s", baseURL, userID)
	resp, err := doGet(client, listURL)
	if err != nil {
		fatal("fetching shelf page: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching shelf page: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing shelf page: %v", err)
	}

	if debug {
		saveDebug(doc, "/tmp/goodreads_shelf_page.html")
	}

	// Find the add-shelf form. Goodreads uses several possible IDs/classes.
	form := doc.Find("form#addShelfForm")
	if form.Length() == 0 {
		form = doc.Find("form#shelf_name_form")
	}
	if form.Length() == 0 {
		form = doc.Find("form.addShelfForm")
	}
	if form.Length() == 0 {
		// Match by action, but only the exact /user_shelves endpoint (not /user_shelves/ID).
		doc.Find("form").Each(func(_ int, s *goquery.Selection) {
			if form.Length() > 0 {
				return
			}
			action, _ := s.Attr("action")
			if action == "/user_shelves" {
				form = s
			}
		})
	}

	if debug {
		count := doc.Find("form").Length()
		fmt.Printf("Debug: Found %d forms on page\n", count)
		doc.Find("form").Each(func(i int, s *goquery.Selection) {
			action, _ := s.Attr("action")
			id, _ := s.Attr("id")
			fmt.Printf("  Form %d: action=%s, id=%s\n", i, action, id)
		})
	}

	// If no form found, look for a link to an "add shelf" page.
	if form.Length() == 0 {
		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			if strings.Contains(href, "/shelf/new") || strings.Contains(href, "/user_shelves/new") {
				if !strings.HasPrefix(href, "http") {
					href = baseURL + href
				}
				r, e := doGet(client, href)
				if e == nil && r.StatusCode == 200 {
					d, e2 := parseHTML(r)
					if e2 == nil {
						form = d.Find("form").First()
						doc = d // update doc for CSRF extraction below
					}
				}
			}
		})
	}

	if form.Length() == 0 {
		fatal("could not find the add shelf form. Try adding a shelf manually on goodreads.com.")
	}

	// Extract form action and fields.
	action, _ := form.Attr("action")
	action = resolveURL(resp.Request.URL.String(), action)

	token := csrfToken(doc)

	fields := url.Values{}
	form.Find("input").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		if name == "" {
			return
		}
		value, _ := s.Attr("value")
		fields.Set(name, value)
	})

	if token != "" {
		fields.Set("authenticity_token", token)
	}

	// Set the shelf name in the text input field.
	nameField := "user_shelf[name]"
	if textInput := form.Find("input[type=text]"); textInput.Length() > 0 {
		if n, ok := textInput.Attr("name"); ok && n != "" {
			nameField = n
		}
	}
	fields.Set(nameField, shelfName)

	if debug {
		fmt.Printf("Debug: Form action: %s\n", action)
		if token != "" {
			fmt.Printf("Debug: CSRF token: %s...\n", token[:min(20, len(token))])
		}
	}

	// Submit as an AJAX request (Rails remote form).
	resp, err = doPostWithCSRF(client, action, fields, listURL, token)
	if err != nil {
		fatal("creating shelf: %v", err)
	}
	defer resp.Body.Close()

	if debug {
		fmt.Printf("Debug: Response status: %d\n", resp.StatusCode)
	}

	switch resp.StatusCode {
	case 200, 201, 302, 303:
		fmt.Printf("Shelf '%s' created successfully!\n", shelfName)
	default:
		fatal("failed to create shelf (HTTP %d)", resp.StatusCode)
	}
}

// cmdShelfDelete deletes a user-created shelf by name.
func cmdShelfDelete(shelfName string, force, debug bool) {
	getUserID() // ensure logged in
	client := newClient()

	// Protect default shelves.
	protected := map[string]bool{"read": true, "currently-reading": true, "to-read": true}
	if protected[strings.ToLower(shelfName)] {
		fatal("cannot delete the '%s' shelf — it is a default Goodreads shelf", shelfName)
	}

	// Fetch the shelf edit page to find the shelf's numeric ID.
	resp, err := doGet(client, baseURL+"/shelf/edit")
	if err != nil {
		fatal("fetching shelf edit page: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching shelf edit page: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing shelf edit page: %v", err)
	}

	if debug {
		saveDebug(doc, "/tmp/goodreads_shelf_edit.html")
	}

	token := csrfToken(doc)

	// Find the shelf's numeric ID by matching its name in the edit form.
	// Structure: <form action="/user_shelves/ID"> ... <input name="user_shelf[name]" value="shelf-name">
	reShelfID := regexp.MustCompile(`/user_shelves/(\d+)`)
	reFormID := regexp.MustCompile(`user_shelf_(\d+)`)
	var shelfID string

	doc.Find("input[name='user_shelf[name]']").Each(func(_ int, s *goquery.Selection) {
		if shelfID != "" {
			return // already found
		}
		val, _ := s.Attr("value")
		if !strings.EqualFold(val, shelfName) {
			return
		}

		// Walk up to the parent <form> to get the shelf ID from its action or id.
		for p := s.Parent(); p.Length() > 0; p = p.Parent() {
			if goquery.NodeName(p) != "form" {
				continue
			}
			if action, ok := p.Attr("action"); ok {
				if m := reShelfID.FindStringSubmatch(action); m != nil {
					shelfID = m[1]
					return
				}
			}
			if id, ok := p.Attr("id"); ok {
				if m := reFormID.FindStringSubmatch(id); m != nil {
					shelfID = m[1]
					return
				}
			}
			break
		}
	})

	if debug {
		fmt.Printf("Debug: Looking for shelf '%s'\n", shelfName)
		fmt.Printf("Debug: Found shelf ID: %s\n", shelfID)
	}

	if shelfID == "" {
		fatal("could not find shelf '%s'. Make sure the shelf exists.", shelfName)
	}

	// Confirm unless --force.
	if !force {
		fmt.Printf("Are you sure you want to delete the shelf '%s'? This cannot be undone. [y/N]: ", shelfName)
		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Deletion cancelled.")
			return
		}
	}

	// Send DELETE request.
	deleteURL := fmt.Sprintf("%s/user_shelves/%s", baseURL, shelfID)

	if debug {
		fmt.Printf("Debug: Deleting shelf at: %s\n", deleteURL)
	}

	resp, err = doDelete(client, deleteURL, token, baseURL+"/shelf/edit")
	if err != nil {
		fatal("deleting shelf: %v", err)
	}
	defer resp.Body.Close()

	if debug {
		fmt.Printf("Debug: Response status: %d\n", resp.StatusCode)
	}

	switch resp.StatusCode {
	case 200, 204, 302, 303:
		fmt.Printf("Shelf '%s' deleted successfully!\n", shelfName)
	default:
		fatal("failed to delete shelf (HTTP %d)", resp.StatusCode)
	}
}

// --- HTTP helpers specific to shelf operations ---

// doPostWithCSRF sends a POST as a Rails AJAX request with CSRF token headers.
func doPostWithCSRF(client *http.Client, rawURL string, data url.Values, referer, token string) (*http.Response, error) {
	req, err := http.NewRequest("POST", rawURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/javascript, application/javascript, application/ecmascript, application/x-ecmascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", referer)
	req.Header.Set("X-CSRF-Token", token)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	return client.Do(req)
}

// doDelete sends a DELETE request as a Rails AJAX call.
func doDelete(client *http.Client, rawURL, token, referer string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/javascript, application/javascript, application/ecmascript, application/x-ecmascript, */*; q=0.01")
	req.Header.Set("X-CSRF-Token", token)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", referer)
	return client.Do(req)
}
