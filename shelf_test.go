package main

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// docFromHTML is a test helper that creates a goquery document from an HTML string.
func docFromHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}
	return doc
}

func TestParseBooksFromHTML(t *testing.T) {
	t.Run("single book with all fields", func(t *testing.T) {
		html := `<table>
		<tr class="bookalike review">
			<td class="field title"><a title="Project Hail Mary">Project Hail Mary</a></td>
			<td class="field author"><a>Weir, Andy</a></td>
			<td class="field rating"><span class="staticStars" title="really liked it"></span></td>
			<td class="field avg_rating"><div class="value">4.52</div></td>
			<td class="field date_read"><span class="date_read_value">Jan 15, 2024</span></td>
			<td class="field date_added"><span class="date_added_value">Dec 01, 2023</span></td>
		</tr>
		</table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		b := books[0]
		if b.Title != "Project Hail Mary" {
			t.Errorf("Title = %q, want %q", b.Title, "Project Hail Mary")
		}
		if b.Author != "Andy Weir" {
			t.Errorf("Author = %q, want %q", b.Author, "Andy Weir")
		}
		if b.Rating != "really liked it" {
			t.Errorf("Rating = %q, want %q", b.Rating, "really liked it")
		}
		if b.AvgRating != "4.52" {
			t.Errorf("AvgRating = %q, want %q", b.AvgRating, "4.52")
		}
		if b.DateRead != "Jan 15, 2024" {
			t.Errorf("DateRead = %q, want %q", b.DateRead, "Jan 15, 2024")
		}
		if b.DateAdded != "Dec 01, 2023" {
			t.Errorf("DateAdded = %q, want %q", b.DateAdded, "Dec 01, 2023")
		}
	})

	t.Run("author without comma is not flipped", func(t *testing.T) {
		html := `<table>
		<tr class="bookalike review">
			<td class="field title"><a title="Dune">Dune</a></td>
			<td class="field author"><a>Frank Herbert</a></td>
		</tr>
		</table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		if books[0].Author != "Frank Herbert" {
			t.Errorf("Author = %q, want %q", books[0].Author, "Frank Herbert")
		}
	})

	t.Run("multiple books", func(t *testing.T) {
		html := `<table>
		<tr class="bookalike review">
			<td class="field title"><a title="Book One">Book One</a></td>
			<td class="field author"><a>Smith, John</a></td>
		</tr>
		<tr class="bookalike review">
			<td class="field title"><a title="Book Two">Book Two</a></td>
			<td class="field author"><a>Doe, Jane</a></td>
		</tr>
		</table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 2 {
			t.Fatalf("got %d books, want 2", len(books))
		}
		if books[0].Title != "Book One" {
			t.Errorf("books[0].Title = %q, want %q", books[0].Title, "Book One")
		}
		if books[1].Title != "Book Two" {
			t.Errorf("books[1].Title = %q, want %q", books[1].Title, "Book Two")
		}
	})

	t.Run("row without title is skipped", func(t *testing.T) {
		html := `<table>
		<tr class="bookalike review">
			<td class="field author"><a>Some Author</a></td>
		</tr>
		</table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 0 {
			t.Fatalf("got %d books, want 0", len(books))
		}
	})

	t.Run("empty table", func(t *testing.T) {
		html := `<table></table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 0 {
			t.Fatalf("got %d books, want 0", len(books))
		}
	})

	t.Run("title falls back to link text", func(t *testing.T) {
		html := `<table>
		<tr class="bookalike review">
			<td class="field title"><a>Fallback Title</a></td>
		</tr>
		</table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		if books[0].Title != "Fallback Title" {
			t.Errorf("Title = %q, want %q", books[0].Title, "Fallback Title")
		}
	})

	t.Run("optional fields can be empty", func(t *testing.T) {
		html := `<table>
		<tr class="bookalike review">
			<td class="field title"><a title="Minimal Book">Minimal Book</a></td>
		</tr>
		</table>`

		doc := docFromHTML(t, html)
		books := parseBooksFromHTML(doc)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		b := books[0]
		if b.Author != "" {
			t.Errorf("Author = %q, want empty", b.Author)
		}
		if b.Rating != "" {
			t.Errorf("Rating = %q, want empty", b.Rating)
		}
		if b.AvgRating != "" {
			t.Errorf("AvgRating = %q, want empty", b.AvgRating)
		}
	})
}
