package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var reAuthorID = regexp.MustCompile(`/author/show/(\d+)`)

// authorSearchResult holds an author from search results.
type authorSearchResult struct {
	ID, Name string
}

// authorInfo holds parsed data from an author's profile page.
type authorInfo struct {
	Name    string
	Bio     string
	Website string
	Born    string
	Genres  []string
}

// authorBookEntry holds a book from an author's book list.
type authorBookEntry struct {
	ID, Title, Rating string
}

// parseAuthorSearchResults extracts author results from a search page.
func parseAuthorSearchResults(doc *goquery.Document, limit int) []authorSearchResult {
	var authors []authorSearchResult
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
		authors = append(authors, authorSearchResult{ID: id, Name: strings.TrimSpace(s.Text())})
	})

	return authors
}

// parseAuthorInfo extracts author details from an author profile page.
func parseAuthorInfo(doc *goquery.Document) authorInfo {
	var info authorInfo

	// Author name.
	if el := doc.Find("h1.authorName span[itemprop=name]"); el.Length() > 0 {
		info.Name = strings.TrimSpace(el.Text())
	} else if el := doc.Find("title"); el.Length() > 0 {
		titleText := strings.TrimSpace(el.Text())
		re := regexp.MustCompile(`^(.+?)\s*\(Author`)
		if m := re.FindStringSubmatch(titleText); m != nil {
			info.Name = strings.TrimSpace(m[1])
		}
	}

	// Bio.
	if el := doc.Find("div.aboutAuthorInfo span"); el.Length() > 0 {
		info.Bio = strings.TrimSpace(el.Text())
		info.Bio = regexp.MustCompile(`\s+`).ReplaceAllString(info.Bio, " ")
	}

	// Website and born/genres from dataTitle/dataItem pairs.
	doc.Find("div.dataTitle").Each(func(_ int, item *goquery.Selection) {
		label := strings.ToLower(strings.TrimSpace(item.Text()))
		valueEl := item.Next()
		if valueEl.Length() == 0 {
			return
		}

		if strings.Contains(label, "website") {
			if link := valueEl.Find("a[href*='http']"); link.Length() > 0 {
				info.Website, _ = link.Attr("href")
			}
		} else if strings.Contains(label, "born") {
			info.Born = strings.TrimSpace(valueEl.Text())
		} else if strings.Contains(label, "genre") {
			valueEl.Find("a").Each(func(i int, g *goquery.Selection) {
				if i < 5 {
					info.Genres = append(info.Genres, strings.TrimSpace(g.Text()))
				}
			})
		}
	})

	return info
}

// parseAuthorBooks extracts books from an author's profile page.
func parseAuthorBooks(doc *goquery.Document, limit int) []authorBookEntry {
	var books []authorBookEntry

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

		books = append(books, authorBookEntry{ID: m[1], Title: title, Rating: rating})
	})

	return books
}

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

	authors := parseAuthorSearchResults(doc, limit)

	if len(authors) == 0 {
		fmt.Printf("No authors found for '%s'.\n", query)
		return
	}

	fmt.Printf("Author search results for '%s':\n", query)
	fmt.Println(strings.Repeat("-", 60))
	for _, a := range authors {
		fmt.Printf("  [%s] %s\n", a.ID, a.Name)
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

	info := parseAuthorInfo(doc)

	// Print.
	fmt.Println(strings.Repeat("=", 60))
	if info.Name != "" {
		fmt.Printf("  %s\n", info.Name)
	} else {
		fmt.Printf("  Author ID: %s\n", authorID)
	}
	fmt.Println(strings.Repeat("-", 60))

	if info.Born != "" {
		fmt.Printf("  Born: %s\n", info.Born)
	}
	if len(info.Genres) > 0 {
		fmt.Printf("  Genres: %s\n", strings.Join(info.Genres, ", "))
	}
	if info.Website != "" {
		fmt.Printf("  Website: %s\n", info.Website)
	}

	if info.Bio != "" {
		fmt.Println(strings.Repeat("-", 60))
		printWrapped(info.Bio, 70)
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

	books := parseAuthorBooks(doc, limit)

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
		fmt.Printf("  [%s] %s\n", b.ID, b.Title)
		if b.Rating != "" {
			fmt.Printf("    %s\n", b.Rating)
		}
		fmt.Println()
	}
}
