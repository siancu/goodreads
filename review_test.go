package main

import "testing"

func TestParseStarRating(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"five stars", "Rating 5 out of 5", 5},
		{"one star", "Rating 1 out of 5", 1},
		{"three stars", "Rating 3 out of 5", 3},
		{"empty string", "", 0},
		{"garbage", "some random text", 0},
		{"case insensitive", "rating 4 out of 5", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStarRating(tt.input)
			if got != tt.want {
				t.Errorf("parseStarRating(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseReviews(t *testing.T) {
	t.Run("single review with all fields", func(t *testing.T) {
		html := `<html><body>
			<article class="ReviewCard">
				<a href="/review/show/12345">Like</a>
				<div class="ReviewerProfile__name">Alice</div>
				<span class="RatingStars RatingStars__small" aria-label="Rating 5 out of 5"></span>
				<section class="ReviewCard__row">
					<span class="Text Text__body3">January 15, 2024</span>
				</section>
				<section class="ReviewText">
					<span class="Formatted">This book was fantastic!</span>
				</section>
			</article>
		</body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 1 {
			t.Fatalf("got %d reviews, want 1", len(reviews))
		}
		r := reviews[0]
		if r.ID != "12345" {
			t.Errorf("ID = %q, want %q", r.ID, "12345")
		}
		if r.ReviewerName != "Alice" {
			t.Errorf("ReviewerName = %q, want %q", r.ReviewerName, "Alice")
		}
		if r.Rating != 5 {
			t.Errorf("Rating = %d, want 5", r.Rating)
		}
		if r.Date != "January 15, 2024" {
			t.Errorf("Date = %q, want %q", r.Date, "January 15, 2024")
		}
		if r.Text != "This book was fantastic!" {
			t.Errorf("Text = %q, want %q", r.Text, "This book was fantastic!")
		}
	})

	t.Run("multiple reviews with IDs", func(t *testing.T) {
		html := `<html><body>
			<article class="ReviewCard">
				<a href="/review/show/111">Like</a>
				<div class="ReviewerProfile__name">Alice</div>
				<span class="RatingStars" aria-label="Rating 5 out of 5"></span>
				<section class="ReviewText"><span class="Formatted">Great!</span></section>
			</article>
			<article class="ReviewCard">
				<a href="/review/show/222">Like</a>
				<div class="ReviewerProfile__name">Bob</div>
				<span class="RatingStars" aria-label="Rating 2 out of 5"></span>
				<section class="ReviewText"><span class="Formatted">Not great.</span></section>
			</article>
			<article class="ReviewCard">
				<a href="/review/show/333">Like</a>
				<div class="ReviewerProfile__name">Carol</div>
				<span class="RatingStars" aria-label="Rating 4 out of 5"></span>
				<section class="ReviewText"><span class="Formatted">Pretty good.</span></section>
			</article>
		</body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 3 {
			t.Fatalf("got %d reviews, want 3", len(reviews))
		}
		if reviews[0].ReviewerName != "Alice" || reviews[0].Rating != 5 || reviews[0].ID != "111" {
			t.Errorf("reviews[0] = %+v", reviews[0])
		}
		if reviews[1].ReviewerName != "Bob" || reviews[1].Rating != 2 || reviews[1].ID != "222" {
			t.Errorf("reviews[1] = %+v", reviews[1])
		}
		if reviews[2].ReviewerName != "Carol" || reviews[2].Rating != 4 || reviews[2].ID != "333" {
			t.Errorf("reviews[2] = %+v", reviews[2])
		}
	})

	t.Run("review without ID", func(t *testing.T) {
		html := `<html><body>
			<article class="ReviewCard">
				<div class="ReviewerProfile__name">NoID</div>
				<span class="RatingStars" aria-label="Rating 3 out of 5"></span>
			</article>
		</body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 1 {
			t.Fatalf("got %d reviews, want 1", len(reviews))
		}
		if reviews[0].ID != "" {
			t.Errorf("ID = %q, want empty", reviews[0].ID)
		}
	})

	t.Run("review without rating", func(t *testing.T) {
		html := `<html><body>
			<article class="ReviewCard">
				<div class="ReviewerProfile__name">Dave</div>
				<section class="ReviewText"><span class="Formatted">No stars here.</span></section>
			</article>
		</body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 1 {
			t.Fatalf("got %d reviews, want 1", len(reviews))
		}
		if reviews[0].Rating != 0 {
			t.Errorf("Rating = %d, want 0", reviews[0].Rating)
		}
	})

	t.Run("review without text", func(t *testing.T) {
		html := `<html><body>
			<article class="ReviewCard">
				<div class="ReviewerProfile__name">Eve</div>
				<span class="RatingStars" aria-label="Rating 3 out of 5"></span>
			</article>
		</body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 1 {
			t.Fatalf("got %d reviews, want 1", len(reviews))
		}
		if reviews[0].Text != "" {
			t.Errorf("Text = %q, want empty", reviews[0].Text)
		}
	})

	t.Run("no reviews", func(t *testing.T) {
		html := `<html><body><div>No reviews here</div></body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 0 {
			t.Fatalf("got %d reviews, want 0", len(reviews))
		}
	})

	t.Run("skips WriteReviewCTA", func(t *testing.T) {
		html := `<html><body>
			<article class="WriteReviewCTA">
				<div>Write a review</div>
			</article>
			<article class="ReviewCard">
				<div class="ReviewerProfile__name">Frank</div>
				<span class="RatingStars" aria-label="Rating 4 out of 5"></span>
			</article>
		</body></html>`
		doc := docFromHTML(t, html)
		reviews := parseReviews(doc)

		if len(reviews) != 1 {
			t.Fatalf("got %d reviews, want 1", len(reviews))
		}
		if reviews[0].ReviewerName != "Frank" {
			t.Errorf("ReviewerName = %q, want %q", reviews[0].ReviewerName, "Frank")
		}
	})
}
