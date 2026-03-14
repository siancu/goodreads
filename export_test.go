package main

import (
	"sort"
	"testing"
)

func TestParseGoodreadsDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full date", "Jan 15, 2024", "2024-01-15"},
		{"month and year", "Jan 2024", "2024-01-01"},
		{"year only", "2024", "2024-01-01"},
		{"empty string", "", ""},
		{"whitespace only", "  ", ""},
		{"unparseable", "garbage", ""},
		{"different month", "Dec 25, 2023", "2023-12-25"},
		{"with whitespace", "  Jan 15, 2024  ", "2024-01-15"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGoodreadsDate(tt.input)
			if got != tt.want {
				t.Errorf("parseGoodreadsDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRatingToStars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"high rating", "4.52", "4.52 ★★★★½"},
		{"exact three", "3.00", "3.00 ★★★"},
		{"five stars", "5.00", "5.00 ★★★★★"},
		{"one star", "1.00", "1.00 ★"},
		{"with half", "3.50", "3.50 ★★★½"},
		{"low fraction no half", "4.10", "4.10 ★★★★"},
		{"just above half threshold", "4.25", "4.25 ★★★★½"},
		{"empty string", "", ""},
		{"not a number", "N/A", "N/A"},
		{"zero", "0.00", "0.00 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ratingToStars(tt.input)
			if got != tt.want {
				t.Errorf("ratingToStars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUpscaleCoverURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"SX50 thumbnail", "https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/1320560129l/7468160._SX50_.jpg", "https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/1320560129l/7468160._SY475_.jpg"},
		{"SY75 thumbnail", "https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/1386920544l/17865._SY75_.jpg", "https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/1386920544l/17865._SY475_.jpg"},
		{"SX98 thumbnail", "https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/54493401._SX98_.jpg", "https://i.gr-assets.com/images/S/compressed.photo.goodreads.com/books/54493401._SY475_.jpg"},
		{"no size marker", "https://example.com/cover.jpg", "https://example.com/cover.jpg"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upscaleCoverURL(tt.input)
			if got != tt.want {
				t.Errorf("upscaleCoverURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSeries(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTitle  string
		wantSeries string
	}{
		{"simple series", "Excession (Culture, #5)", "Excession", "Culture, #5"},
		{"series with semicolon", "Snuff (Discworld, #39; City Watch, #8)", "Snuff", "Discworld, #39; City Watch, #8"},
		{"multiple series", "Gridlinked (Agent Cormac #1, Polity Universe #3)", "Gridlinked", "Agent Cormac #1, Polity Universe #3"},
		{"no series", "To Kill a Mockingbird", "To Kill a Mockingbird", ""},
		{"parens without series number", "A Book (Some Subtitle)", "A Book (Some Subtitle)", ""},
		{"empty string", "", "", ""},
		{"series #1", "Children of Time (Children of Time, #1)", "Children of Time", "Children of Time, #1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotSeries := extractSeries(tt.input)
			if gotTitle != tt.wantTitle {
				t.Errorf("extractSeries(%q) title = %q, want %q", tt.input, gotTitle, tt.wantTitle)
			}
			if gotSeries != tt.wantSeries {
				t.Errorf("extractSeries(%q) series = %q, want %q", tt.input, gotSeries, tt.wantSeries)
			}
		})
	}
}

func TestDeduplicateBooks(t *testing.T) {
	t.Run("merges shelves for same book", func(t *testing.T) {
		shelfBooks := map[string][]book{
			"read": {
				{ID: "123", Title: "Book A", Author: "Author A", AvgRating: "4.00"},
				{ID: "456", Title: "Book B", Author: "Author B", AvgRating: "3.50"},
			},
			"top-5": {
				{ID: "123", Title: "Book A", Author: "Author A", AvgRating: "4.00"},
			},
		}

		result := deduplicateBooks(shelfBooks)

		if len(result) != 2 {
			t.Fatalf("got %d books, want 2", len(result))
		}

		// Find Book A (ID 123).
		var bookA *exportBook
		for i := range result {
			if result[i].ID == "123" {
				bookA = &result[i]
				break
			}
		}
		if bookA == nil {
			t.Fatal("book 123 not found in results")
		}

		sort.Strings(bookA.Shelves)
		if len(bookA.Shelves) != 2 {
			t.Errorf("book 123 has %d shelves, want 2", len(bookA.Shelves))
		}
	})

	t.Run("skips books without ID", func(t *testing.T) {
		shelfBooks := map[string][]book{
			"read": {
				{ID: "", Title: "No ID Book"},
				{ID: "789", Title: "Has ID"},
			},
		}

		result := deduplicateBooks(shelfBooks)

		if len(result) != 1 {
			t.Fatalf("got %d books, want 1", len(result))
		}
		if result[0].ID != "789" {
			t.Errorf("ID = %q, want %q", result[0].ID, "789")
		}
	})

	t.Run("fills missing fields from later shelves", func(t *testing.T) {
		shelfBooks := map[string][]book{
			"to-read": {
				{ID: "100", Title: "Book", Author: "Author"},
			},
			"read": {
				{ID: "100", Title: "Book", Author: "Author", CoverURL: "https://example.com/cover.jpg", DateRead: "Jan 15, 2024"},
			},
		}

		result := deduplicateBooks(shelfBooks)

		if len(result) != 1 {
			t.Fatalf("got %d books, want 1", len(result))
		}
		if result[0].CoverURL != "https://example.com/cover.jpg" {
			t.Errorf("CoverURL = %q, want cover URL", result[0].CoverURL)
		}
		if result[0].DateRead != "Jan 15, 2024" {
			t.Errorf("DateRead = %q, want %q", result[0].DateRead, "Jan 15, 2024")
		}
	})

	t.Run("constructs goodreads URL", func(t *testing.T) {
		shelfBooks := map[string][]book{
			"read": {{ID: "54493401", Title: "Test"}},
		}

		result := deduplicateBooks(shelfBooks)

		if len(result) != 1 {
			t.Fatalf("got %d books, want 1", len(result))
		}
		want := "https://www.goodreads.com/book/show/54493401"
		if result[0].URL != want {
			t.Errorf("URL = %q, want %q", result[0].URL, want)
		}
	})
}

func TestFlagStrings(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"single long flag", []string{"--shelf", "read"}, []string{"read"}},
		{"single short flag", []string{"-s", "read"}, []string{"read"}},
		{"multiple flags", []string{"--shelf", "read", "--shelf", "to-read", "--shelf", "currently-reading"}, []string{"read", "to-read", "currently-reading"}},
		{"mixed long and short", []string{"--shelf", "read", "-s", "to-read"}, []string{"read", "to-read"}},
		{"no flags", []string{"--json"}, nil},
		{"empty args", []string{}, nil},
		{"flag at end without value", []string{"--shelf"}, nil},
		{"interleaved with other flags", []string{"--json", "--shelf", "read", "--limit", "10", "--shelf", "to-read"}, []string{"read", "to-read"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flagStrings(tt.args, "--shelf", "-s")
			if len(got) != len(tt.want) {
				t.Fatalf("flagStrings() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("flagStrings()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
