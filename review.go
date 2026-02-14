package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// review holds data parsed from a single Goodreads review card.
type review struct {
	ID           string // Goodreads review ID (from /review/show/<id>)
	ReviewerName string
	Rating       int // 1-5, or 0 if unknown
	Date         string
	Text         string
}

var reStarRating = regexp.MustCompile(`(?i)Rating\s+(\d)\s+out\s+of\s+5`)
var reReviewID = regexp.MustCompile(`/review/show/(\d+)`)

// parseStarRating extracts a numeric rating from an aria-label like "Rating 4 out of 5".
// Returns 0 if the rating cannot be determined.
func parseStarRating(ariaLabel string) int {
	m := reStarRating.FindStringSubmatch(ariaLabel)
	if m == nil {
		return 0
	}
	return int(m[1][0] - '0')
}

// parseReviews extracts reviews from a Goodreads book show page.
func parseReviews(doc *goquery.Document) []review {
	var reviews []review

	doc.Find("article.ReviewCard").Each(func(_ int, card *goquery.Selection) {
		var r review

		// Review ID from the first /review/show/ link.
		card.Find("a[href*='/review/show/']").Each(func(_ int, a *goquery.Selection) {
			if r.ID != "" {
				return
			}
			href, _ := a.Attr("href")
			if m := reReviewID.FindStringSubmatch(href); m != nil {
				r.ID = m[1]
			}
		})

		// Reviewer name.
		if el := card.Find("div.ReviewerProfile__name"); el.Length() > 0 {
			r.ReviewerName = strings.TrimSpace(el.Text())
		}

		// Star rating from aria-label.
		if el := card.Find("span.RatingStars"); el.Length() > 0 {
			if label, ok := el.Attr("aria-label"); ok {
				r.Rating = parseStarRating(label)
			}
		}

		// Date.
		if el := card.Find("section.ReviewCard__row span.Text.Text__body3"); el.Length() > 0 {
			r.Date = strings.TrimSpace(el.Text())
		}

		// Review text.
		if el := card.Find("section.ReviewText span.Formatted"); el.Length() > 0 {
			r.Text = strings.TrimSpace(el.Text())
		}

		reviews = append(reviews, r)
	})

	return reviews
}

// cmdBookReviews displays reviews for a book, optionally filtered to best/worst by rating.
func cmdBookReviews(bookID string, bestN, worstN, limit int, full bool, reviewIndex int) {
	client := newClient()

	resp, err := doGet(client, fmt.Sprintf("%s/book/show/%s", baseURL, bookID))
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

	// Book title for display.
	title := ""
	if el := doc.Find("h1[data-testid=bookTitle]"); el.Length() > 0 {
		title = strings.TrimSpace(el.Text())
	}

	reviews := parseReviews(doc)
	if len(reviews) == 0 {
		fmt.Println("No reviews found.")
		return
	}

	// Show a single review by index.
	if reviewIndex > 0 {
		if reviewIndex > len(reviews) {
			fatal("review number must be between 1 and %d", len(reviews))
		}
		r := reviews[reviewIndex-1]
		printSingleReview(title, bookID, r, reviewIndex, len(reviews))
		return
	}

	useBestWorst := bestN > 0 || worstN > 0

	if useBestWorst {
		if bestN > 0 {
			printReviewSection(title, bookID, reviews, bestN, "best", full)
		}
		if worstN > 0 {
			if bestN > 0 {
				fmt.Println()
			}
			printReviewSection(title, bookID, reviews, worstN, "worst", full)
		}
	} else {
		printReviewSection(title, bookID, reviews, limit, "default", full)
	}
}

// printSingleReview displays a single review in full.
func printSingleReview(title, bookID string, r review, index, total int) {
	if title != "" {
		fmt.Printf("Review #%d for '%s'\n", index, title)
	} else {
		fmt.Printf("Review #%d for book %s\n", index, bookID)
	}
	fmt.Println(strings.Repeat("=", 60))

	// Rating and reviewer.
	if r.Rating > 0 {
		stars := strings.Repeat("*", r.Rating)
		fmt.Printf("  %s (%d/5)", stars, r.Rating)
	} else {
		fmt.Print("  (no rating)")
	}
	if r.ReviewerName != "" {
		fmt.Printf(" - %s", r.ReviewerName)
	}
	fmt.Println()

	if r.Date != "" {
		fmt.Printf("  %s\n", r.Date)
	}

	if r.Text != "" {
		fmt.Println()
		printWrapped(r.Text, 70)
	}

	fmt.Println(strings.Repeat("=", 60))
	if r.ID != "" {
		fmt.Printf("  URL: %s/review/show/%s\n", baseURL, r.ID)
	}
}

// printReviewSection prints a sorted/limited section of reviews.
// mode is "best", "worst", or "default" (page order).
func printReviewSection(title, bookID string, reviews []review, n int, mode string, full bool) {
	sorted := make([]review, len(reviews))
	copy(sorted, reviews)

	label := "Reviews"
	switch mode {
	case "best":
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].Rating > sorted[j].Rating
		})
		label = "Best Reviews"
	case "worst":
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].Rating < sorted[j].Rating
		})
		label = "Worst Reviews"
	}

	if n > len(sorted) {
		n = len(sorted)
	}
	selected := sorted[:n]

	if title != "" {
		fmt.Printf("%s for '%s'\n", label, title)
	} else {
		fmt.Printf("%s for book %s\n", label, bookID)
	}
	fmt.Println(strings.Repeat("=", 60))

	for i, r := range selected {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 60))
		}

		// Review number, rating, and reviewer.
		fmt.Printf("  #%d", i+1)
		if r.Rating > 0 {
			stars := strings.Repeat("*", r.Rating)
			fmt.Printf("  %s (%d/5)", stars, r.Rating)
		} else {
			fmt.Print("  (no rating)")
		}
		if r.ReviewerName != "" {
			fmt.Printf(" - %s", r.ReviewerName)
		}
		fmt.Println()

		// Date.
		if r.Date != "" {
			fmt.Printf("  %s\n", r.Date)
		}

		// Review text.
		if r.Text != "" {
			fmt.Println()
			text := r.Text
			if !full && len(text) > 500 {
				text = text[:500] + "..."
			}
			printWrapped(text, 70)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Showing %d of %d reviews found\n", n, len(reviews))
}
