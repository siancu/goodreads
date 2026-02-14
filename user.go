package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// userListEntry holds a user from the friends/following list.
type userListEntry struct {
	ID, Name string
}

// userProfile holds parsed data from a user's profile page.
type userProfile struct {
	Name         string
	Location     string
	JoinDate     string
	BooksCount   string
	FriendsCount string
}

// shelfEntry holds a shelf name and optional book count.
type shelfEntry struct {
	Name  string
	Count string
}

// readingStats holds parsed year/page stats from a stats page.
type readingStats struct {
	YearStats map[string]int // year → books count
	PageStats map[string]int // year → pages count
}

// parseUserList extracts users from a friends page, excluding the given user ID.
func parseUserList(doc *goquery.Document, selfID string) []userListEntry {
	var users []userListEntry
	seenIDs := make(map[string]bool)
	reUserID := regexp.MustCompile(`/user/show/(\d+)`)

	doc.Find("a[href*='/user/show/']").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		m := reUserID.FindStringSubmatch(href)
		if m == nil {
			return
		}
		uid := m[1]
		if uid == selfID || seenIDs[uid] {
			return
		}
		name := strings.TrimSpace(s.Text())
		if name == "" {
			return
		}
		seenIDs[uid] = true
		users = append(users, userListEntry{ID: uid, Name: name})
	})

	return users
}

// parseUserProfile extracts profile info from a user show page.
func parseUserProfile(doc *goquery.Document) userProfile {
	var p userProfile

	// User name.
	if el := doc.Find("h1.userProfileName"); el.Length() > 0 {
		// Get the direct text node, excluding child elements like "edit profile".
		p.Name = strings.TrimSpace(el.Contents().First().Text())
		if p.Name == "" {
			p.Name = strings.TrimSpace(el.Text())
			p.Name = regexp.MustCompile(`\s*\(edit\s+profile\)\s*$`).ReplaceAllString(p.Name, "")
		}
	}
	if p.Name == "" {
		if el := doc.Find("title"); el.Length() > 0 {
			p.Name = strings.TrimSpace(el.Text())
			p.Name = regexp.MustCompile(`\s*\|\s*Goodreads\s*$`).ReplaceAllString(p.Name, "")
		}
	}

	// Location.
	if el := doc.Find("div.infoBoxRowItem[itemprop=address]"); el.Length() > 0 {
		p.Location = strings.TrimSpace(el.Text())
	} else {
		doc.Find("div.infoBoxRowTitle").Each(func(_ int, row *goquery.Selection) {
			if p.Location != "" {
				return
			}
			if strings.Contains(strings.ToLower(row.Text()), "location") {
				if val := row.Next(); val.Length() > 0 {
					p.Location = strings.TrimSpace(val.Text())
				}
			}
		})
	}

	// Join date.
	doc.Find("div.infoBoxRowTitle").Each(func(_ int, row *goquery.Selection) {
		if p.JoinDate != "" {
			return
		}
		if strings.Contains(strings.ToLower(row.Text()), "joined") {
			if val := row.Next(); val.Length() > 0 {
				p.JoinDate = strings.TrimSpace(val.Text())
			}
		}
	})

	// Books and friends counts from profile stat links.
	reDigits := regexp.MustCompile(`(\d[\d,]*)`)
	doc.Find("a[href*='/review/list/'], a[href*='/friend/']").Each(func(_ int, s *goquery.Selection) {
		text := strings.ToLower(strings.TrimSpace(s.Text()))
		if m := reDigits.FindStringSubmatch(s.Text()); m != nil {
			if strings.Contains(text, "book") {
				p.BooksCount = m[1]
			} else if strings.Contains(text, "friend") {
				p.FriendsCount = m[1]
			}
		}
	})

	return p
}

// parseShelves extracts shelf names and counts from a review list page.
func parseShelves(doc *goquery.Document) []shelfEntry {
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

	var result []shelfEntry
	seen := make(map[string]bool)

	shelves.Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.Contains(href, "shelf=") {
			return
		}
		parts := strings.SplitN(href, "shelf=", 2)
		if len(parts) < 2 {
			return
		}
		name := strings.SplitN(parts[1], "&", 2)[0]
		if name == "" || seen[name] {
			return
		}
		seen[name] = true

		count := ""
		if span := s.Find("span.smallText"); span.Length() > 0 {
			count = strings.TrimSpace(span.Text())
		}

		result = append(result, shelfEntry{Name: name, Count: count})
	})

	return result
}

// parseReadingStats extracts year_stats and page_stats JS variables from a stats page body.
func parseReadingStats(body string) readingStats {
	var stats readingStats
	stats.YearStats = make(map[string]int)
	stats.PageStats = make(map[string]int)

	reYear := regexp.MustCompile(`year_stats\s*=\s*(\{[^}]+\})`)
	rePage := regexp.MustCompile(`page_stats\s*=\s*(\{[^}]+\})`)

	if m := reYear.FindStringSubmatch(body); m != nil {
		json.Unmarshal([]byte(m[1]), &stats.YearStats)
	}
	if m := rePage.FindStringSubmatch(body); m != nil {
		json.Unmarshal([]byte(m[1]), &stats.PageStats)
	}

	return stats
}

// cmdUserList lists users you follow (friends).
func cmdUserList() {
	userID := getUserID()
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/friend/user/%s", baseURL, userID))
	if err != nil {
		fatal("fetching friends: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching friends: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing friends page: %v", err)
	}

	users := parseUserList(doc, userID)

	fmt.Println("Users You Follow:")
	fmt.Println(strings.Repeat("-", 40))

	if len(users) == 0 {
		fmt.Println("  No users found.")
		return
	}

	for _, u := range users {
		fmt.Printf("  [%s] %s\n", u.ID, u.Name)
	}
}

// cmdUserShow displays a user's profile.
func cmdUserShow(uid string) {
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/user/show/%s", baseURL, uid))
	if err != nil {
		fatal("fetching user: %v", err)
	}
	if resp.StatusCode == 404 {
		fatal("user %s not found", uid)
	}
	if resp.StatusCode != 200 {
		fatal("fetching user: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing user page: %v", err)
	}

	p := parseUserProfile(doc)

	fmt.Println(strings.Repeat("=", 60))
	if p.Name != "" {
		fmt.Printf("  %s\n", p.Name)
	} else {
		fmt.Printf("  User ID: %s\n", uid)
	}
	fmt.Println(strings.Repeat("-", 60))

	if p.Location != "" {
		fmt.Printf("  Location: %s\n", p.Location)
	}
	if p.JoinDate != "" {
		fmt.Printf("  Joined:   %s\n", p.JoinDate)
	}
	if p.BooksCount != "" {
		fmt.Printf("  Books:    %s\n", p.BooksCount)
	}
	if p.FriendsCount != "" {
		fmt.Printf("  Friends:  %s\n", p.FriendsCount)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  URL: %s/user/show/%s\n", baseURL, uid)
}

// cmdUserShelves lists shelves for a given user.
func cmdUserShelves(uid string) {
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/review/list/%s", baseURL, uid))
	if err != nil {
		fatal("fetching shelves: %v", err)
	}
	if resp.StatusCode == 404 {
		fatal("user %s not found", uid)
	}
	if resp.StatusCode != 200 {
		fatal("fetching shelves: HTTP %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp)
	if err != nil {
		fatal("parsing shelves page: %v", err)
	}

	shelves := parseShelves(doc)

	fmt.Printf("Shelves for user %s:\n", uid)
	fmt.Println(strings.Repeat("-", 40))

	if len(shelves) == 0 {
		fmt.Println("  No shelves found.")
		return
	}

	for _, s := range shelves {
		if s.Count != "" {
			fmt.Printf("  %s (%s)\n", s.Name, s.Count)
		} else {
			fmt.Printf("  %s\n", s.Name)
		}
	}
}

// cmdUserBooks shows books on a user's shelf.
func cmdUserBooks(uid, shelfName string, limit int) {
	client := newClient()

	total := 0
	headerPrinted := false

	for page := 1; ; page++ {
		u := fmt.Sprintf("%s/review/list/%s?shelf=%s&page=%d", baseURL, uid, url.QueryEscape(shelfName), page)
		resp, err := doGet(client, u)
		if err != nil {
			fatal("fetching books page %d: %v", page, err)
		}
		if resp.StatusCode == 404 {
			fatal("user %s not found", uid)
		}
		if resp.StatusCode != 200 {
			fatal("fetching books: HTTP %d", resp.StatusCode)
		}

		doc, err := parseHTML(resp)
		if err != nil {
			fatal("parsing books page %d: %v", page, err)
		}

		books := parseBooksFromHTML(doc)
		if len(books) == 0 {
			break
		}

		if !headerPrinted {
			fmt.Printf("Books on '%s' for user %s:\n", shelfName, uid)
			fmt.Println(strings.Repeat("-", 60))
			headerPrinted = true
		}

		for _, b := range books {
			if limit > 0 && total >= limit {
				break
			}
			fmt.Printf("  %s\n", b.Title)
			fmt.Printf("    by %s", b.Author)
			if b.Rating != "" {
				fmt.Printf(" | Your rating: %s", b.Rating)
			}
			if b.AvgRating != "" {
				fmt.Printf(" | Avg: %s", b.AvgRating)
			}
			fmt.Println()
			total++
		}

		if limit > 0 && total >= limit {
			break
		}
	}

	if total == 0 {
		fmt.Printf("No books found on shelf '%s' for user %s.\n", shelfName, uid)
	} else {
		fmt.Printf("\n%d books total\n", total)
	}
}

// cmdUserStats shows reading statistics for a user.
func cmdUserStats(uid string) {
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/review/stats/%s", baseURL, uid))
	if err != nil {
		fatal("fetching stats: %v", err)
	}
	if resp.StatusCode == 404 {
		fatal("user %s not found", uid)
	}
	if resp.StatusCode != 200 {
		fatal("fetching stats: HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fatal("reading stats page: %v", err)
	}

	stats := parseReadingStats(string(bodyBytes))
	printStats(fmt.Sprintf("Reading Stats for user %s", uid), stats)
}

// cmdStats shows reading statistics for the current user.
func cmdStats(yearFilter string) {
	userID := getUserID()
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/review/stats/%s", baseURL, userID))
	if err != nil {
		fatal("fetching stats: %v", err)
	}
	if resp.StatusCode != 200 {
		fatal("fetching stats: HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fatal("reading stats page: %v", err)
	}

	stats := parseReadingStats(string(bodyBytes))

	if yearFilter != "" {
		books, hasYear := stats.YearStats[yearFilter]
		if !hasYear {
			fmt.Printf("No reading data found for %s.\n", yearFilter)
			return
		}
		pages := stats.PageStats[yearFilter]
		fmt.Printf("Reading Stats for %s\n", yearFilter)
		fmt.Println(strings.Repeat("=", 40))
		fmt.Printf("  Books read:     %d\n", books)
		fmt.Printf("  Pages read:     %s\n", formatCommas(pages))
		if books > 0 {
			fmt.Printf("  Avg pages/book: %d\n", pages/books)
		}
		return
	}

	printStats("Reading Stats", stats)
}

// printStats displays reading stats with totals and per-year breakdown.
func printStats(header string, stats readingStats) {
	if len(stats.YearStats) == 0 {
		fmt.Println("No reading statistics found.")
		return
	}

	totalBooks := 0
	totalPages := 0
	for _, v := range stats.YearStats {
		totalBooks += v
	}
	for _, v := range stats.PageStats {
		totalPages += v
	}

	// Sort years descending.
	var years []string
	for yr := range stats.YearStats {
		years = append(years, yr)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(years)))

	fmt.Println(header)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("  Total books read: %d\n", totalBooks)
	fmt.Printf("  Total pages read: %s\n", formatCommas(totalPages))
	if totalBooks > 0 {
		fmt.Printf("  Avg pages/book:   %d\n", totalPages/totalBooks)
	}
	fmt.Println()
	fmt.Println("By Year:")
	fmt.Println(strings.Repeat("-", 40))

	max := 10
	if len(years) < max {
		max = len(years)
	}
	for i := 0; i < max; i++ {
		yr := years[i]
		books := stats.YearStats[yr]
		pages := stats.PageStats[yr]
		barLen := books
		if barLen > 30 {
			barLen = 30
		}
		bar := strings.Repeat("*", barLen)
		fmt.Printf("  %s: %3d books, %5s pages  %s\n", yr, books, formatCommas(pages), bar)
	}

	if len(years) > 10 {
		fmt.Printf("  ... and %d more years\n", len(years)-10)
	}
}

// formatCommas formats an integer with comma separators.
func formatCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
