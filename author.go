package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var reAuthorID = regexp.MustCompile(`/author/show/(\d+)`)

// cmdAuthorSearch searches for authors by name.
func cmdAuthorSearch(query string, limit int) {
	client := newClient()

	u := fmt.Sprintf("%s/search?q=%s&search_type=authors", baseURL, url.QueryEscape(query))
	resp, err := doGet(client, u)
	if err != nil {
		fatal("searching authors: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("searching authors: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing search results: %v", err)
	}

	// Find author links in search results and deduplicate by ID.
	type authorResult struct {
		id, name string
	}
	var authors []authorResult
	seenIDs := make(map[string]bool)

	doc.Find("a.authorName[href*='/author/show/']").Each(func(_ int, s *goquery.Selection) {
		if len(authors) >= limit {
			return
		}
		href, _ := s.Attr("href")
		m := reAuthorID.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id := m[1]
		if seenIDs[id] {
			return
		}
		seenIDs[id] = true
		authors = append(authors, authorResult{id: id, name: strings.TrimSpace(s.Text())})
	})

	if len(authors) == 0 {
		fmt.Printf("No authors found for '%s'.\n", query)
		return
	}

	fmt.Printf("Author search results for '%s':\n", query)
	fmt.Println(strings.Repeat("-", 60))
	for _, a := range authors {
		fmt.Printf("  [%s] %s\n", a.id, a.name)
	}
}

// cmdAuthorShow displays detailed information about an author.
func cmdAuthorShow(authorID string) {
	client := newClient()

	u := fmt.Sprintf("%s/author/show/%s", baseURL, authorID)
	resp, err := doGet(client, u)
	if err != nil {
		fatal("fetching author: %v", err)
	}
	if resp.StatusCode == 404 {
		fatal("author %s not found", authorID)
	}
	if resp.StatusCode != 200 {
		fatal("fetching author: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing author page: %v", err)
	}

	// Author name.
	name := ""
	if el := doc.Find("h1.authorName span[itemprop=name]"); el.Length() > 0 {
		name = strings.TrimSpace(el.Text())
	} else if el := doc.Find("title"); el.Length() > 0 {
		titleText := strings.TrimSpace(el.Text())
		re := regexp.MustCompile(`^(.+?)\s*\(Author`)
		if m := re.FindStringSubmatch(titleText); m != nil {
			name = strings.TrimSpace(m[1])
		}
	}

	// Bio.
	bio := ""
	if el := doc.Find("div.aboutAuthorInfo span"); el.Length() > 0 {
		bio = strings.TrimSpace(el.Text())
		bio = regexp.MustCompile(`\s+`).ReplaceAllString(bio, " ")
	}

	// Website and born/genres from dataTitle/dataItem pairs.
	website := ""
	born := ""
	var genres []string

	doc.Find("div.dataTitle").Each(func(_ int, item *goquery.Selection) {
		label := strings.ToLower(strings.TrimSpace(item.Text()))
		valueEl := item.Next()
		if valueEl.Length() == 0 {
			return
		}

		if strings.Contains(label, "website") {
			if link := valueEl.Find("a[href*='http']"); link.Length() > 0 {
				website, _ = link.Attr("href")
			}
		} else if strings.Contains(label, "born") {
			born = strings.TrimSpace(valueEl.Text())
		} else if strings.Contains(label, "genre") {
			valueEl.Find("a").Each(func(i int, g *goquery.Selection) {
				if i < 5 {
					genres = append(genres, strings.TrimSpace(g.Text()))
				}
			})
		}
	})

	// Print.
	fmt.Println(strings.Repeat("=", 60))
	if name != "" {
		fmt.Printf("  %s\n", name)
	} else {
		fmt.Printf("  Author ID: %s\n", authorID)
	}
	fmt.Println(strings.Repeat("-", 60))

	if born != "" {
		fmt.Printf("  Born: %s\n", born)
	}
	if len(genres) > 0 {
		fmt.Printf("  Genres: %s\n", strings.Join(genres, ", "))
	}
	if website != "" {
		fmt.Printf("  Website: %s\n", website)
	}

	if bio != "" {
		fmt.Println(strings.Repeat("-", 60))
		printWrapped(bio, 70)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  URL: %s/author/show/%s\n", baseURL, authorID)
}

// cmdAuthorBooks lists books by an author.
func cmdAuthorBooks(authorID string, limit int) {
	client := newClient()

	u := fmt.Sprintf("%s/author/show/%s", baseURL, authorID)
	resp, err := doGet(client, u)
	if err != nil {
		fatal("fetching author: %v", err)
	}
	if resp.StatusCode == 404 {
		fatal("author %s not found", authorID)
	}
	if resp.StatusCode != 200 {
		fatal("fetching author: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing author page: %v", err)
	}

	// Author name for display.
	name := ""
	if el := doc.Find("h1.authorName span[itemprop=name]"); el.Length() > 0 {
		name = strings.TrimSpace(el.Text())
	}

	// Find books in the table.
	type authorBook struct {
		id, title, rating string
	}
	var books []authorBook

	doc.Find("tr[itemtype='http://schema.org/Book']").Each(func(_ int, row *goquery.Selection) {
		if len(books) >= limit {
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

		// Use innermost span if present to avoid duplicated text.
		title := ""
		if span := titleLink.Find("span").First(); span.Length() > 0 {
			title = strings.TrimSpace(span.Text())
		} else {
			title = strings.TrimSpace(titleLink.Text())
		}

		rating := ""
		if el := row.Find("span.minirating"); el.Length() > 0 {
			rating = strings.TrimSpace(el.Text())
		}

		books = append(books, authorBook{id: m[1], title: title, rating: rating})
	})

	// Print.
	if name != "" {
		fmt.Printf("Books by %s:\n", name)
	} else {
		fmt.Printf("Books by author %s:\n", authorID)
	}
	fmt.Println(strings.Repeat("-", 60))

	if len(books) == 0 {
		fmt.Println("  No books found.")
		return
	}

	for _, b := range books {
		fmt.Printf("  [%s] %s\n", b.id, b.title)
		if b.rating != "" {
			fmt.Printf("    %s\n", b.rating)
		}
		fmt.Println()
	}
}
