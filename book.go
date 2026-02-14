package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var reBookID = regexp.MustCompile(`/book/show/(\d+)`)

// listEntry represents a Goodreads list (name + URL).
type listEntry struct {
	name string
	href string
}

// cmdBookSearch searches for books by query string.
func cmdBookSearch(query string, limit int) {
	client := newClient()

	u := fmt.Sprintf("%s/search?q=%s", baseURL, url.QueryEscape(query))
	resp, err := doGet(client, u)
	if err != nil {
		fatal("searching books: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("searching books: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing search results: %v", err)
	}

	table := doc.Find("table.tableList")
	if table.Length() == 0 {
		fmt.Printf("No results found for '%s'.\n", query)
		return
	}

	rows := table.Find("tr")
	if rows.Length() == 0 {
		fmt.Printf("No results found for '%s'.\n", query)
		return
	}

	fmt.Printf("Search results for '%s':\n", query)
	fmt.Println(strings.Repeat("-", 60))

	count := 0
	rows.Each(func(_ int, row *goquery.Selection) {
		if count >= limit {
			return
		}

		titleEl := row.Find("a.bookTitle")
		if titleEl.Length() == 0 {
			return
		}

		title := strings.TrimSpace(titleEl.Text())
		author := strings.TrimSpace(row.Find("a.authorName").Text())
		if author == "" {
			author = "Unknown"
		}
		rating := strings.TrimSpace(row.Find("span.minirating").Text())

		href, _ := titleEl.Attr("href")
		m := reBookID.FindStringSubmatch(href)
		bookID := ""
		if m != nil {
			bookID = m[1]
		}

		fmt.Printf("  [%s] %s\n", bookID, title)
		fmt.Printf("    by %s\n", author)
		if rating != "" {
			fmt.Printf("    %s\n", rating)
		}
		fmt.Println()
		count++
	})

	if count == 0 {
		fmt.Printf("No results found for '%s'.\n", query)
	}
}

// cmdBookShow displays detailed information about a book.
func cmdBookShow(bookID string) {
	client := newClient()

	u := fmt.Sprintf("%s/book/show/%s", baseURL, bookID)
	resp, err := doGet(client, u)
	if err != nil {
		fatal("fetching book: %v", err)
	}
	if resp.StatusCode == 404 {
		fatal("book %s not found", bookID)
	}
	if resp.StatusCode != 200 {
		fatal("fetching book: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing book page: %v", err)
	}

	// Title
	title := ""
	if el := doc.Find("h1[data-testid=bookTitle]"); el.Length() > 0 {
		title = strings.TrimSpace(el.Text())
	} else if el := doc.Find("meta[property='og:title']"); el.Length() > 0 {
		title, _ = el.Attr("content")
		title = strings.TrimSpace(title)
	}

	// Authors
	var authors []string
	doc.Find("span.ContributorLink__name[data-testid=name]").Each(func(_ int, s *goquery.Selection) {
		authors = append(authors, strings.TrimSpace(s.Text()))
	})
	if len(authors) == 0 {
		if el := doc.Find("a.authorName"); el.Length() > 0 {
			authors = append(authors, strings.TrimSpace(el.First().Text()))
		}
	}

	// Rating
	rating := ""
	if el := doc.Find("div.RatingStatistics__rating"); el.Length() > 0 {
		rating = strings.TrimSpace(el.Text())
	}

	// Ratings & reviews count
	ratingsCount := ""
	reviewsCount := ""
	doc.Find("span[data-testid=ratingsCount], span[data-testid=reviewsCount]").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		lower := strings.ToLower(text)
		if strings.Contains(lower, "rating") {
			ratingsCount = text
		} else if strings.Contains(lower, "review") {
			reviewsCount = text
		}
	})

	// Description
	description := ""
	if el := doc.Find("div.DetailsLayoutRightParagraph__widthConstrained span.Formatted"); el.Length() > 0 {
		description = strings.TrimSpace(el.Text())
	} else if el := doc.Find("meta[property='og:description']"); el.Length() > 0 {
		description, _ = el.Attr("content")
		description = strings.TrimSpace(description)
	}

	// Genres
	var genres []string
	doc.Find("span.BookPageMetadataSection__genreButton a.Button--tag").Each(func(i int, s *goquery.Selection) {
		if i < 5 {
			genres = append(genres, strings.TrimSpace(s.Text()))
		}
	})

	// Pages
	pages := ""
	if el := doc.Find("p[data-testid=pagesFormat]"); el.Length() > 0 {
		pages = strings.TrimSpace(el.Text())
	}

	// Publication info
	publication := ""
	if el := doc.Find("p[data-testid=publicationInfo]"); el.Length() > 0 {
		publication = strings.TrimSpace(el.Text())
	}

	// Series
	series := ""
	if el := doc.Find("div.BookPageTitleSection__title h3.Text__italic a"); el.Length() > 0 {
		series = strings.TrimSpace(el.Text())
	}

	// Print
	fmt.Println(strings.Repeat("=", 60))
	if title != "" {
		fmt.Printf("  %s\n", title)
	} else {
		fmt.Printf("  Book ID: %s\n", bookID)
	}
	if series != "" {
		fmt.Printf("  (%s)\n", series)
	}
	if len(authors) > 0 {
		fmt.Printf("  by %s\n", strings.Join(authors, ", "))
	}

	fmt.Println(strings.Repeat("-", 60))

	if rating != "" {
		line := fmt.Sprintf("  Rating: %s/5", rating)
		if ratingsCount != "" {
			line += fmt.Sprintf("  (%s", ratingsCount)
			if reviewsCount != "" {
				line += fmt.Sprintf(", %s", reviewsCount)
			}
			line += ")"
		}
		fmt.Println(line)
	}
	if pages != "" {
		fmt.Printf("  Format: %s\n", pages)
	}
	if publication != "" {
		fmt.Printf("  Published: %s\n", publication)
	}
	if len(genres) > 0 {
		fmt.Printf("  Genres: %s\n", strings.Join(genres, ", "))
	}

	if description != "" {
		fmt.Println(strings.Repeat("-", 60))
		printWrapped(description, 70)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  URL: %s/book/show/%s\n", baseURL, bookID)
}

// printWrapped word-wraps text to the given width with 2-space indent.
func printWrapped(text string, width int) {
	words := strings.Fields(text)
	line := "  "
	for _, word := range words {
		if len(line)+len(word)+1 > width && line != "  " {
			fmt.Println(line)
			line = "  " + word
		} else {
			if line == "  " {
				line += word
			} else {
				line += " " + word
			}
		}
	}
	if strings.TrimSpace(line) != "" {
		fmt.Println(line)
	}
}

// getCSRFToken fetches a CSRF token from the user's review list page.
func getCSRFToken(userID string) (string, *goquery.Document) {
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/review/list/%s", baseURL, userID))
	if err != nil {
		fatal("fetching CSRF token: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching CSRF token: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing page for CSRF token: %v", err)
	}

	token := csrfToken(doc)
	if token == "" {
		fatal("could not find CSRF token. Try logging in again.")
	}

	return token, doc
}

// fetchBookTitle fetches just the title of a book, returning "" on failure.
func fetchBookTitle(bookID string) string {
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/book/show/%s", baseURL, bookID))
	if err != nil || resp.StatusCode != 200 {
		return ""
	}

	doc, err := parseHTML(resp)
	if err != nil {
		return ""
	}

	if el := doc.Find("h1[data-testid=bookTitle]"); el.Length() > 0 {
		return strings.TrimSpace(el.Text())
	}
	if el := doc.Find("meta[property='og:title']"); el.Length() > 0 {
		content, _ := el.Attr("content")
		return strings.TrimSpace(content)
	}
	return ""
}

// cmdBookAdd adds a book to a shelf.
func cmdBookAdd(bookID, shelfName string) {
	userID := getUserID()
	token, _ := getCSRFToken(userID)

	bookTitle := fetchBookTitle(bookID)

	client := newClient()
	data := url.Values{
		"book_id":            {bookID},
		"name":               {shelfName},
		"authenticity_token": {token},
	}

	referer := fmt.Sprintf("%s/review/list/%s", baseURL, userID)
	resp, err := doPostWithCSRF(client, baseURL+"/shelf/add_to_shelf", data, referer, token)
	if err != nil {
		fatal("adding book to shelf: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		if bookTitle != "" {
			fmt.Printf("'%s' added to shelf '%s'.\n", bookTitle, shelfName)
		} else {
			fmt.Printf("Book %s added to shelf '%s'.\n", bookID, shelfName)
		}
	case 401:
		fatal("not authorized. Try logging in again.")
	case 404:
		fatal("book %s not found", bookID)
	default:
		fatal("failed to add book (HTTP %d)", resp.StatusCode)
	}
}

// cmdBookRemove removes a book from all shelves.
func cmdBookRemove(bookID string) {
	userID := getUserID()
	token, _ := getCSRFToken(userID)

	bookTitle := fetchBookTitle(bookID)

	client := newClient()
	data := url.Values{
		"authenticity_token": {token},
	}

	referer := fmt.Sprintf("%s/review/list/%s", baseURL, userID)
	resp, err := doPostWithCSRF(client, fmt.Sprintf("%s/review/destroy/%s", baseURL, bookID), data, referer, token)
	if err != nil {
		fatal("removing book: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		if bookTitle != "" {
			fmt.Printf("'%s' removed from your shelves.\n", bookTitle)
		} else {
			fmt.Printf("Book %s removed from your shelves.\n", bookID)
		}
	case 401:
		fatal("not authorized. Try logging in again.")
	case 404:
		fatal("book %s not found in your shelves", bookID)
	default:
		fatal("failed to remove book (HTTP %d)", resp.StatusCode)
	}
}

// cmdBookRate rates a book (1-5 stars).
func cmdBookRate(bookID string, rating int) {
	if rating < 1 || rating > 5 {
		fatal("rating must be between 1 and 5")
	}

	userID := getUserID()
	token, _ := getCSRFToken(userID)

	bookTitle := fetchBookTitle(bookID)

	client := newClient()
	data := url.Values{
		"rating":             {fmt.Sprintf("%d", rating)},
		"authenticity_token": {token},
	}

	resp, err := doPostWithCSRF(client, fmt.Sprintf("%s/review/rate/%s", baseURL, bookID), data, baseURL, token)
	if err != nil {
		fatal("rating book: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200, 204:
		stars := strings.Repeat("*", rating)
		if bookTitle != "" {
			fmt.Printf("'%s' rated %s (%d/5).\n", bookTitle, stars, rating)
		} else {
			fmt.Printf("Book %s rated %s (%d/5).\n", bookID, stars, rating)
		}
	case 401:
		fatal("not authorized. Try logging in again.")
	case 404:
		fatal("book %s not found. Add it to a shelf first.", bookID)
	default:
		fatal("failed to rate book (HTTP %d)", resp.StatusCode)
	}
}

// cmdBookSimilar finds similar books using Goodreads lists.
func cmdBookSimilar(bookID string, limit int, showLists bool, listIndex int) {
	client := newClient()

	// Get book title for display.
	bookTitle := fetchBookTitle(bookID)

	// Get lists containing this book.
	resp, err := doGet(client, fmt.Sprintf("%s/list/book/%s", baseURL, bookID))
	if err != nil {
		fatal("fetching lists: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching lists: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing lists page: %v", err)
	}

	// Find list links.
	listLinks := doc.Find("div.leftContainer a[href*='/list/show/']")
	if listLinks.Length() == 0 {
		listLinks = doc.Find("a[href*='/list/show/']")
	}
	if listLinks.Length() == 0 {
		fmt.Println("No lists found containing this book.")
		return
	}

	// Build deduplicated list of available lists.
	reListID := regexp.MustCompile(`/list/show/(\d+)`)
	var available []listEntry
	seenIDs := make(map[string]bool)

	listLinks.Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		m := reListID.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id := m[1]
		if seenIDs[id] {
			return
		}
		seenIDs[id] = true

		name := strings.TrimSpace(s.Text())
		if name == "" {
			// Try extracting from URL slug.
			re := regexp.MustCompile(`/list/show/\d+\.([^#?]+)`)
			if sm := re.FindStringSubmatch(href); sm != nil {
				name = strings.ReplaceAll(sm[1], "_", " ")
			}
		}
		if name != "" {
			available = append(available, listEntry{name: name, href: href})
		}
	})

	if len(available) == 0 {
		fmt.Println("No lists found containing this book.")
		return
	}

	// --show-lists: display available lists and exit.
	if showLists {
		if bookTitle != "" {
			fmt.Printf("Lists containing '%s':\n", bookTitle)
		} else {
			fmt.Printf("Lists containing book %s:\n", bookID)
		}
		fmt.Println(strings.Repeat("-", 60))
		max := 20
		if len(available) < max {
			max = len(available)
		}
		for i := 0; i < max; i++ {
			fmt.Printf("  %2d. %s\n", i+1, available[i].name)
		}
		fmt.Println()
		fmt.Println("Use --list N to select a specific list.")
		return
	}

	// Select a list.
	var selected listEntry
	if listIndex > 0 {
		if listIndex > len(available) {
			fatal("list number must be between 1 and %d", len(available))
		}
		selected = available[listIndex-1]
	} else {
		selected = pickBestList(available)
	}

	// Fetch the list page.
	listURL := selected.href
	if !strings.HasPrefix(listURL, "http") {
		listURL = baseURL + listURL
	}
	// Remove anchor.
	if idx := strings.Index(listURL, "#"); idx != -1 {
		listURL = listURL[:idx]
	}

	resp, err = doGet(client, listURL)
	if err != nil {
		fatal("fetching list: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching list: HTTP %d", resp.StatusCode)
	}

	doc, err = parseHTML(resp)
	if err != nil {
		fatal("parsing list page: %v", err)
	}

	// Extract books from the list.
	type similarBook struct {
		id, title, author, rating string
	}
	var results []similarBook

	doc.Find("tr[itemtype='http://schema.org/Book']").Each(func(_ int, row *goquery.Selection) {
		if len(results) >= limit {
			return
		}

		titleLink := row.Find("a.bookTitle")
		if titleLink.Length() == 0 {
			return
		}

		href, _ := titleLink.Attr("href")
		m := reBookID.FindStringSubmatch(href)
		if m == nil {
			return
		}
		rowBookID := m[1]

		// Skip the original book.
		if rowBookID == bookID {
			return
		}

		results = append(results, similarBook{
			id:     rowBookID,
			title:  strings.TrimSpace(titleLink.Text()),
			author: strings.TrimSpace(row.Find("a.authorName").Text()),
			rating: strings.TrimSpace(row.Find("span.minirating").Text()),
		})
	})

	// Print results.
	if bookTitle != "" {
		fmt.Printf("Books similar to '%s':\n", bookTitle)
	} else {
		fmt.Printf("Books similar to book %s:\n", bookID)
	}
	fmt.Printf("(from list: %s)\n", selected.name)
	fmt.Println(strings.Repeat("-", 60))

	if len(results) == 0 {
		fmt.Println("  No similar books found.")
		return
	}

	for _, b := range results {
		fmt.Printf("  [%s] %s\n", b.id, b.title)
		if b.author != "" {
			fmt.Printf("    by %s\n", b.author)
		}
		if b.rating != "" {
			fmt.Printf("    %s\n", b.rating)
		}
		fmt.Println()
	}
}

// pickBestList selects the best list by skipping overly generic ones.
func pickBestList(lists []listEntry) listEntry {
	skipPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^best\s+(science\s+fiction|fantasy|sci-?fi)(\s|&|$)`),
		regexp.MustCompile(`(?i)^best\s+books?\s*(of|ever|$)`),
		regexp.MustCompile(`(?i)^best\s+books?\s+of\s+(the\s+)?(decade|\d{2,4})`),
		regexp.MustCompile(`(?i)^(science\s+fiction|fantasy|fiction)(\s+&\s+|\s+)(fantasy|science\s+fiction|books?|novels?)`),
		regexp.MustCompile(`(?i)^best\s+\w+\s+books?\s*$`),
		regexp.MustCompile(`(?i)reading\s+challenge`),
		regexp.MustCompile(`(?i)^recommended\s+by\s+reading`),
		regexp.MustCompile(`(?i)^top\s+\d+\s+for\s+the\s+reading`),
		regexp.MustCompile(`(?i)movie|film|watched|adaptation|screen`),
		regexp.MustCompile(`(?i)books?\s+everyone`),
		regexp.MustCompile(`(?i)books?\s+to\s+read\s+before`),
		regexp.MustCompile(`(?i)must\s+read`),
		regexp.MustCompile(`(?i)bucket\s+list`),
		regexp.MustCompile(`(?i)1001\s+books?`),
		regexp.MustCompile(`(?i)everyone.s\s+read\s+it`),
	}

	for _, l := range lists {
		generic := false
		for _, p := range skipPatterns {
			if p.MatchString(l.name) {
				generic = true
				break
			}
		}
		if !generic {
			return l
		}
	}

	// All lists are generic — fall back to first.
	return lists[0]
}
