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

// exportBook is the JSON-serializable representation of a book for export.
type exportBook struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Author    string   `json:"author"`
	AvgRating string   `json:"avg_rating"`
	CoverURL  string   `json:"cover_url"`
	Shelves   []string `json:"shelves"`
	DateRead  string   `json:"date_read,omitempty"`
	DateAdded string   `json:"date_added,omitempty"`
	URL       string   `json:"url"`
}

// collectAllBooks fetches all books across all shelves for the given user,
// deduplicating by book ID and merging shelf membership.
func collectAllBooks(userID string) []exportBook {
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

	// Collect books from each shelf.
	type bookKey struct{}
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
					seen[b.ID] = len(result)
					result = append(result, exportBook{
						ID:        b.ID,
						Title:     b.Title,
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
				seen[b.ID] = len(result)
				result = append(result, exportBook{
					ID:        b.ID,
					Title:     b.Title,
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
func cmdExportJSON(userID string) {
	books := collectAllBooks(userID)
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

// runExportCommand parses flags and dispatches the export subcommand.
func runExportCommand(args []string) {
	jsonMode := hasFlag(args, "--json", "-j")
	notionMode := hasFlag(args, "--notion", "-N")
	parentID := flagString(args, "--parent", "-P", "")
	token := flagString(args, "--token", "-t", "")
	if token == "" {
		token = os.Getenv("NOTION_TOKEN")
	}

	userID := getUserID()

	if jsonMode {
		cmdExportJSON(userID)
	} else if notionMode {
		if parentID == "" {
			fatal("--parent is required for Notion export. Provide the parent page ID.")
		}
		if token == "" {
			fatal("Notion token required. Set NOTION_TOKEN env var or use --token.")
		}
		cmdExportNotion(userID, parentID, token)
	} else {
		printExportUsage()
		os.Exit(1)
	}
}

func printExportUsage() {
	fmt.Println(`Usage:
  goodreads export <mode> [options]

Modes:
  --json, -j                    Export all books as JSON to stdout
  --notion, -N                  Export to a Notion database

Notion options:
  --parent, -P <page-id>       Parent page ID for the Notion database (required)
  --token, -t <token>          Notion integration token (or set NOTION_TOKEN env var)

Examples:
  goodreads export --json > books.json
  goodreads export --notion --parent abc123def456`)
}
