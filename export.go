package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// reCoverSize matches Goodreads cover image size markers like ._SX50_, ._SY75_.
var reCoverSize = regexp.MustCompile(`\._S[XY]\d+_`)

// reSeries matches a trailing parenthetical containing a series reference like "(Culture, #5)".
var reSeries = regexp.MustCompile(`\s*\(([^)]*#\d+[^)]*)\)\s*$`)

// exportBook is the JSON-serializable representation of a book for export.
type exportBook struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Series    string   `json:"series,omitempty"`
	Author    string   `json:"author"`
	AvgRating string   `json:"avg_rating"`
	CoverURL  string   `json:"cover_url"`
	Shelves   []string `json:"shelves"`
	DateRead  string   `json:"date_read,omitempty"`
	DateAdded string   `json:"date_added,omitempty"`
	URL       string   `json:"url"`
}

// extractSeries splits a title like "Excession (Culture, #5)" into
// clean title "Excession" and series "Culture, #5".
// If no series pattern is found, returns the original title and empty series.
func extractSeries(title string) (string, string) {
	m := reSeries.FindStringSubmatch(title)
	if m == nil {
		return title, ""
	}
	clean := strings.TrimSpace(reSeries.ReplaceAllString(title, ""))
	return clean, m[1]
}

// collectAllBooks fetches books from the given shelves (or all shelves if none specified)
// for the given user, deduplicating by book ID and merging shelf membership.
func collectAllBooks(userID string, filterShelves []string) []exportBook {
	client := newClient()

	// Fetch shelf list page to get shelf names.
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

	shelfEntries := parseShelves(doc)
	if len(shelfEntries) == 0 {
		fatal("no shelves found. Check if your cookies are valid.")
	}

	// Filter to requested shelves if specified.
	if len(filterShelves) > 0 {
		filterSet := make(map[string]bool, len(filterShelves))
		for _, s := range filterShelves {
			filterSet[s] = true
		}
		var filtered []shelfEntry
		for _, shelf := range shelfEntries {
			if filterSet[shelf.Name] {
				filtered = append(filtered, shelf)
			}
		}
		if len(filtered) == 0 {
			fatal("none of the requested shelves were found")
		}
		shelfEntries = filtered
	}

	// Collect books from each shelf.
	seen := make(map[string]int) // book ID -> index in result
	var result []exportBook

	for _, shelf := range shelfEntries {
		for page := 1; ; page++ {
			fmt.Fprintf(os.Stderr, "Fetching shelf '%s' (page %d)...\n", shelf.Name, page)

			u := fmt.Sprintf("%s/review/list/%s?shelf=%s&page=%d", baseURL, userID, url.QueryEscape(shelf.Name), page)
			resp, err := doGet(client, u)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch %s page %d: %v\n", shelf.Name, page, err)
				break
			}
			if resp.StatusCode != 200 {
				fmt.Fprintf(os.Stderr, "Warning: HTTP %d fetching %s page %d\n", resp.StatusCode, shelf.Name, page)
				break
			}

			doc, err := parseHTML(resp)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse %s page %d: %v\n", shelf.Name, page, err)
				break
			}

			books := parseBooksFromHTML(doc)
			if len(books) == 0 {
				break
			}

			for _, b := range books {
				if b.ID == "" {
					continue
				}
				if idx, ok := seen[b.ID]; ok {
					// Merge shelf into existing entry.
					eb := &result[idx]
					if !containsString(eb.Shelves, shelf.Name) {
						eb.Shelves = append(eb.Shelves, shelf.Name)
					}
					// Fill in any missing fields from this shelf's data.
					if eb.CoverURL == "" && b.CoverURL != "" {
						eb.CoverURL = b.CoverURL
					}
					if eb.DateRead == "" && b.DateRead != "" {
						eb.DateRead = b.DateRead
					}
					if eb.DateAdded == "" && b.DateAdded != "" {
						eb.DateAdded = b.DateAdded
					}
				} else {
					shelves := b.Shelves
					if !containsString(shelves, shelf.Name) {
						shelves = append(shelves, shelf.Name)
					}
					if len(shelves) == 0 {
						shelves = []string{shelf.Name}
					}
					cleanTitle, series := extractSeries(b.Title)
					seen[b.ID] = len(result)
					result = append(result, exportBook{
						ID:        b.ID,
						Title:     cleanTitle,
						Series:    series,
						Author:    b.Author,
						AvgRating: b.AvgRating,
						CoverURL:  upscaleCoverURL(b.CoverURL),
						Shelves:   shelves,
						DateRead:  b.DateRead,
						DateAdded: b.DateAdded,
						URL:       fmt.Sprintf("%s/book/show/%s", baseURL, b.ID),
					})
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Collected %d unique books from %d shelves.\n", len(result), len(shelfEntries))
	return result
}

// upscaleCoverURL replaces the small thumbnail size marker in a Goodreads cover URL
// with a larger one suitable for Notion page covers.
func upscaleCoverURL(u string) string {
	if u == "" {
		return ""
	}
	return reCoverSize.ReplaceAllString(u, "._SY475_")
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// deduplicateBooks merges books from multiple shelves by ID.
// This is the testable core of the deduplication logic.
func deduplicateBooks(shelfBooks map[string][]book) []exportBook {
	seen := make(map[string]int)
	var result []exportBook

	for shelfName, books := range shelfBooks {
		for _, b := range books {
			if b.ID == "" {
				continue
			}
			if idx, ok := seen[b.ID]; ok {
				eb := &result[idx]
				if !containsString(eb.Shelves, shelfName) {
					eb.Shelves = append(eb.Shelves, shelfName)
				}
				if eb.CoverURL == "" && b.CoverURL != "" {
					eb.CoverURL = b.CoverURL
				}
				if eb.DateRead == "" && b.DateRead != "" {
					eb.DateRead = b.DateRead
				}
				if eb.DateAdded == "" && b.DateAdded != "" {
					eb.DateAdded = b.DateAdded
				}
			} else {
				cleanTitle, series := extractSeries(b.Title)
				seen[b.ID] = len(result)
				result = append(result, exportBook{
					ID:        b.ID,
					Title:     cleanTitle,
					Series:    series,
					Author:    b.Author,
					AvgRating: b.AvgRating,
					CoverURL:  upscaleCoverURL(b.CoverURL),
					Shelves:   []string{shelfName},
					DateRead:  b.DateRead,
					DateAdded: b.DateAdded,
					URL:       fmt.Sprintf("%s/book/show/%s", baseURL, b.ID),
				})
			}
		}
	}

	return result
}

// cmdExportJSON collects all books and outputs JSON to stdout.
func cmdExportJSON(userID string, shelves []string, limit int) {
	books := collectAllBooks(userID, shelves)
	if limit > 0 && limit < len(books) {
		books = books[:limit]
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(books); err != nil {
		fatal("encoding JSON: %v", err)
	}
}

// parseGoodreadsDate converts Goodreads date formats to ISO 8601 (YYYY-MM-DD).
// Handles "Jan 15, 2024", "Jan 2024", and returns "" for unparseable dates.
func parseGoodreadsDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return ""
	}
	for _, layout := range []string{"Jan 02, 2006", "Jan 2006", "2006"} {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// ratingToStars converts a numeric rating string to a display string with stars.
// e.g. "4.52" -> "4.52 ★★★★½", "3.00" -> "3.00 ★★★"
func ratingToStars(avgRating string) string {
	avgRating = strings.TrimSpace(avgRating)
	if avgRating == "" {
		return ""
	}
	f, err := strconv.ParseFloat(avgRating, 64)
	if err != nil {
		return avgRating
	}
	full := int(f)
	half := f-float64(full) >= 0.25
	stars := strings.Repeat("★", full)
	if half {
		stars += "½"
	}
	return fmt.Sprintf("%s %s", avgRating, stars)
}

// flagStrings collects all values for a repeated flag (e.g. --shelf a --shelf b).
func flagStrings(args []string, long, short string) []string {
	var result []string
	for i, a := range args {
		if (a == long || a == short) && i+1 < len(args) {
			result = append(result, args[i+1])
		}
	}
	return result
}

// runExportCommand parses flags and dispatches the export subcommand.
func runExportCommand(args []string) {
	jsonMode := hasFlag(args, "--json", "-j")

	userID := getUserID()

	limit := flagInt(args, "--limit", "-l", 0)
	shelves := flagStrings(args, "--shelf", "-s")

	if jsonMode {
		cmdExportJSON(userID, shelves, limit)
	} else {
		printExportUsage()
		os.Exit(1)
	}
}

func printExportUsage() {
	fmt.Println(`Usage:
  goodreads export [options]

Options:
  --json, -j                    Export all books as JSON to stdout
  --shelf, -s <name>            Only export from this shelf (repeatable)
  --limit, -l <n>               Limit number of books exported

Examples:
  goodreads export --json > books.json
  goodreads export --json --shelf read --shelf to-read --shelf currently-reading`)
}
