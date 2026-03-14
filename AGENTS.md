# AGENTS.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

After adding a new feature or changing how a feature works, and if the feature works as expected, update README.md (if necessary), update AGENTS.md (if necessary), commit and push.

## Project Overview

A Go CLI tool for interacting with Goodreads (since Goodreads deprecated their API in 2020). This is a port of the [Python version](https://github.com/siancu/goodreads).

## Architecture

- **Multi-file CLI** in a single `main` package:
  - `main.go` — CLI entry point, subcommand routing, flag parsing
  - `client.go` — HTTP client, cookie persistence, HTML parsing helpers
  - `auth.go` — login/logout commands
  - `shelf.go` — shelf management commands (list, show, add, delete)
  - `book.go` — book commands (search, show, add, remove, rate, similar, status, progress)
  - `author.go` — author commands (search, show, books)
  - `user.go` — user commands (list, show, shelves, books, stats) and top-level stats
  - `review.go` — book review commands (reviews with best/worst sorting, --full, --review N)
  - `export.go` — export commands (JSON export)
- **Authentication**: Login via Amazon SSO, cookies saved to `~/.goodreads-cookies.json`
- **HTTP Client**: `net/http` with `net/http/cookiejar`
- **HTML Parsing**: `github.com/PuerkitoBio/goquery` (CSS selectors, like BeautifulSoup)

## Build & Run

```bash
go build -o goodreads .
./goodreads login
```

## Releasing

Releases are automated via GoReleaser + GitHub Actions. To create a release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers `.github/workflows/release.yml`, which builds binaries for macOS and Linux (amd64 + arm64) and creates a GitHub Release with the archives attached.

Configuration: `.goreleaser.yaml`

## Key Patterns

### Adding New Commands

1. Create a new file (e.g., `shelf.go`) with command functions:

```go
func cmdShelfList() {
    userID := getUserID()
    client := newClient()

    resp, err := doGet(client, baseURL+"/some/path")
    if err != nil {
        fatal("fetching page: %v", err)
    }

    doc, err := parseHTML(resp)
    if err != nil {
        fatal("parsing page: %v", err)
    }

    // Extract data with goquery selectors and print
}
```

2. Register the command in `main.go`:
   - Add a `case` in the `switch os.Args[1]` block
   - Parse any subcommand-specific flags
   - Update `printUsage()`

3. Update README.md and AGENTS.md if needed.

### Helper Functions (client.go)

| Function | Purpose |
|----------|---------|
| `getUserID()` | Get saved user ID or exit with error |
| `newClient()` | Create authenticated HTTP client |
| `doGet(client, url)` | GET with browser headers |
| `doPost(client, url, data, referer)` | POST form data with browser headers |
| `parseHTML(resp)` | Parse response body into goquery document |
| `extractFormData(doc, formName)` | Extract form action + all input fields |
| `csrfToken(doc)` | Extract Rails CSRF token from meta tag |
| `fatal(format, args...)` | Print error to stderr and exit |

### Shared Helpers (shelf.go)

| Function | Purpose |
|----------|---------|
| `doPostWithCSRF(client, url, data, referer, token)` | POST as Rails AJAX with CSRF headers |
| `doDelete(client, url, token, referer)` | DELETE as Rails AJAX call |

### Shared Helpers (book.go)

| Function | Purpose |
|----------|---------|
| `getCSRFToken(userID)` | Fetch CSRF token from review list page |
| `fetchBookTitle(bookID)` | Fetch a book's title (returns "" on failure) |

### Export Helpers (export.go)

| Function | Purpose |
|----------|---------|
| `collectAllBooks(userID)` | Fetch all books across all shelves, deduplicated |
| `deduplicateBooks(shelfBooks)` | Merge books from multiple shelves by ID |
| `parseGoodreadsDate(dateStr)` | Convert Goodreads dates to ISO 8601 |
| `ratingToStars(avgRating)` | Convert numeric rating to star display |
| `upscaleCoverURL(url)` | Replace thumbnail size marker with larger size |

### Finding Goodreads Endpoints

1. Open Chrome DevTools → Network tab
2. Perform the action on goodreads.com
3. Look at the request URL and response HTML
4. Use goquery CSS selectors to extract data

## Dependencies

Managed via `go.mod`. Add new dependencies with:

```bash
go get github.com/some/package
```

Current external dependencies:
- `github.com/PuerkitoBio/goquery` — HTML parsing with CSS selectors
- `golang.org/x/term` — Terminal password input without echo

## Testing

### Unit tests

```bash
go test -v ./...
```

Test files follow Go conventions (`*_test.go` alongside source):
- `main_test.go` — flag parsing (`hasFlag`, `flagInt`, `parseLoginFlags`)
- `shelf_test.go` — `parseBooksFromHTML` with HTML fixtures
- `client_test.go` — `csrfToken`, `extractFormData`, `resolveURL`
- `book_test.go` — `printWrapped`, `pickBestList`, `validateProgressArgs`
- `author_test.go` — `parseAuthorSearchResults`, `parseAuthorInfo`, `parseAuthorBooks`
- `user_test.go` — `parseUserList`, `parseUserProfile`, `parseShelves`, `parseReadingStats`, `formatCommas`, `flagString`
- `review_test.go` — `parseStarRating`, `parseReviews`
- `export_test.go` — `parseGoodreadsDate`, `ratingToStars`, `upscaleCoverURL`, `deduplicateBooks`

### Manual integration tests

First login, then run commands:

```bash
./goodreads login
./goodreads shelf list
./goodreads book search "project hail mary"
./goodreads book show 54493401
./goodreads book reviews 54493401 --best 3
./goodreads book progress 54493401 --page 150
./goodreads book progress 54493401 --percent 45
./goodreads author search "Adrian Tchaikovsky"
./goodreads author show 1445909
./goodreads author books 1445909
./goodreads user list
./goodreads user show <user-id>
./goodreads user shelves <user-id>
./goodreads user books <user-id> --shelf read --limit 3
./goodreads stats
./goodreads stats 2023
./goodreads export --json > /tmp/books.json
./goodreads export --json --shelf read --shelf to-read --shelf currently-reading > /tmp/books.json
./goodreads logout
```
