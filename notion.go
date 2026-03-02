package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const notionAPIBase = "https://api.notion.com/v1"
const notionVersion = "2022-06-28"

// notionRequest sends a request to the Notion API and returns the parsed JSON response.
func notionRequest(method, path string, body interface{}, token string) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, notionAPIBase+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg, _ := result["message"].(string)
		return result, fmt.Errorf("Notion API error %d: %s", resp.StatusCode, msg)
	}

	return result, nil
}

// createNotionDatabase creates a new database under the given parent page.
func createNotionDatabase(parentPageID, token string) (string, error) {
	payload := map[string]interface{}{
		"parent": map[string]interface{}{
			"type":    "page_id",
			"page_id": parentPageID,
		},
		"icon": map[string]interface{}{
			"type":  "emoji",
			"emoji": "📚",
		},
		"title": []map[string]interface{}{
			{"type": "text", "text": map[string]string{"content": "Goodreads Books"}},
		},
		"properties": map[string]interface{}{
			"Title": map[string]interface{}{
				"title": map[string]interface{}{},
			},
			"Author": map[string]interface{}{
				"rich_text": map[string]interface{}{},
			},
			"Rating": map[string]interface{}{
				"rich_text": map[string]interface{}{},
			},
			"Shelves": map[string]interface{}{
				"multi_select": map[string]interface{}{},
			},
			"Goodreads URL": map[string]interface{}{
				"url": map[string]interface{}{},
			},
			"Description": map[string]interface{}{
				"rich_text": map[string]interface{}{},
			},
		},
	}

	result, err := notionRequest("POST", "/databases", payload, token)
	if err != nil {
		return "", fmt.Errorf("creating database: %w", err)
	}

	id, _ := result["id"].(string)
	if id == "" {
		return "", fmt.Errorf("no database ID in response")
	}

	return id, nil
}

// buildNotionPagePayload constructs the Notion API payload for creating a book page.
func buildNotionPagePayload(dbID string, b exportBook) map[string]interface{} {
	properties := map[string]interface{}{
		"Title": map[string]interface{}{
			"title": []map[string]interface{}{
				{"type": "text", "text": map[string]string{"content": b.Title}},
			},
		},
	}

	if b.Author != "" {
		properties["Author"] = map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": map[string]string{"content": b.Author}},
			},
		}
	}

	if b.AvgRating != "" {
		properties["Rating"] = map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": map[string]string{"content": ratingToStars(b.AvgRating)}},
			},
		}
	}

	if len(b.Shelves) > 0 {
		tags := make([]map[string]string, len(b.Shelves))
		for i, s := range b.Shelves {
			tags[i] = map[string]string{"name": s}
		}
		properties["Shelves"] = map[string]interface{}{
			"multi_select": tags,
		}
	}

	if b.URL != "" {
		properties["Goodreads URL"] = map[string]interface{}{
			"url": b.URL,
		}
	}

	page := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": dbID,
		},
		"properties": properties,
	}

	if b.CoverURL != "" {
		page["cover"] = map[string]interface{}{
			"type": "external",
			"external": map[string]string{
				"url": b.CoverURL,
			},
		}
		page["icon"] = map[string]interface{}{
			"type": "external",
			"external": map[string]string{
				"url": b.CoverURL,
			},
		}
	}

	return page
}

// createNotionPage creates a single page in the Notion database for a book.
func createNotionPage(dbID string, b exportBook, token string) error {
	payload := buildNotionPagePayload(dbID, b)
	_, err := notionRequest("POST", "/pages", payload, token)
	return err
}

// cmdExportNotion collects all books and exports them to a new Notion database.
func cmdExportNotion(userID, parentPageID, token string) {
	books := collectAllBooks(userID)

	if len(books) == 0 {
		fmt.Fprintln(os.Stderr, "No books to export.")
		return
	}

	fmt.Fprintf(os.Stderr, "Creating Notion database...\n")
	dbID, err := createNotionDatabase(parentPageID, token)
	if err != nil {
		fatal("creating Notion database: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Database created: https://www.notion.so/%s\n", dbID)

	failures := 0
	for i, b := range books {
		fmt.Fprintf(os.Stderr, "Creating page %d/%d: %s...\n", i+1, len(books), b.Title)

		err := createNotionPage(dbID, b, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to create page for %q: %v\n", b.Title, err)
			failures++

			// Retry once after a longer pause for rate limit errors.
			time.Sleep(1 * time.Second)
			if err2 := createNotionPage(dbID, b, token); err2 != nil {
				fmt.Fprintf(os.Stderr, "  Retry also failed: %v\n", err2)
			} else {
				fmt.Fprintf(os.Stderr, "  Retry succeeded.\n")
				failures--
			}
		}

		// Rate limit: ~2.8 req/s, safely under Notion's 3 req/s limit.
		time.Sleep(350 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "\nDone! Created %d/%d pages.", len(books)-failures, len(books))
	if failures > 0 {
		fmt.Fprintf(os.Stderr, " %d failed.", failures)
	}
	fmt.Fprintln(os.Stderr)
}
