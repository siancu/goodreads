package main

import "testing"

func TestParseAuthorSearchResults(t *testing.T) {
	t.Run("single author", func(t *testing.T) {
		html := `<div>
			<a class="authorName" href="/author/show/1445909.Adrian_Tchaikovsky">Adrian Tchaikovsky</a>
		</div>`
		doc := docFromHTML(t, html)
		results := parseAuthorSearchResults(doc, 10)

		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if results[0].ID != "1445909" {
			t.Errorf("ID = %q, want %q", results[0].ID, "1445909")
		}
		if results[0].Name != "Adrian Tchaikovsky" {
			t.Errorf("Name = %q, want %q", results[0].Name, "Adrian Tchaikovsky")
		}
	})

	t.Run("deduplicates by ID", func(t *testing.T) {
		html := `<div>
			<a class="authorName" href="/author/show/123.John_Smith">John Smith</a>
			<a class="authorName" href="/author/show/123.John_Smith">John Smith</a>
			<a class="authorName" href="/author/show/456.Jane_Doe">Jane Doe</a>
		</div>`
		doc := docFromHTML(t, html)
		results := parseAuthorSearchResults(doc, 10)

		if len(results) != 2 {
			t.Fatalf("got %d results, want 2", len(results))
		}
		if results[0].ID != "123" {
			t.Errorf("results[0].ID = %q, want %q", results[0].ID, "123")
		}
		if results[1].ID != "456" {
			t.Errorf("results[1].ID = %q, want %q", results[1].ID, "456")
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		html := `<div>
			<a class="authorName" href="/author/show/1.A">Author A</a>
			<a class="authorName" href="/author/show/2.B">Author B</a>
			<a class="authorName" href="/author/show/3.C">Author C</a>
		</div>`
		doc := docFromHTML(t, html)
		results := parseAuthorSearchResults(doc, 2)

		if len(results) != 2 {
			t.Fatalf("got %d results, want 2", len(results))
		}
	})

	t.Run("no results", func(t *testing.T) {
		html := `<div><p>No results found</p></div>`
		doc := docFromHTML(t, html)
		results := parseAuthorSearchResults(doc, 10)

		if len(results) != 0 {
			t.Fatalf("got %d results, want 0", len(results))
		}
	})

	t.Run("skips links without valid author ID", func(t *testing.T) {
		html := `<div>
			<a class="authorName" href="/author/show/">No ID</a>
			<a class="authorName" href="/author/show/99.Valid">Valid Author</a>
		</div>`
		doc := docFromHTML(t, html)
		results := parseAuthorSearchResults(doc, 10)

		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if results[0].ID != "99" {
			t.Errorf("ID = %q, want %q", results[0].ID, "99")
		}
	})
}

func TestParseAuthorInfo(t *testing.T) {
	t.Run("all fields present", func(t *testing.T) {
		html := `<html><body>
			<h1 class="authorName"><span itemprop="name">Adrian Tchaikovsky</span></h1>
			<div class="aboutAuthorInfo"><span>Born in Lincolnshire,   studied   zoology.</span></div>
			<div class="dataTitle">Born</div>
			<div class="dataItem">November 7, 1972</div>
			<div class="dataTitle">Website</div>
			<div class="dataItem"><a href="http://shadowsoftheapt.com/">shadowsoftheapt.com</a></div>
			<div class="dataTitle">Genre</div>
			<div class="dataItem"><a>Science Fiction</a><a>Fantasy</a></div>
		</body></html>`
		doc := docFromHTML(t, html)
		info := parseAuthorInfo(doc)

		if info.Name != "Adrian Tchaikovsky" {
			t.Errorf("Name = %q, want %q", info.Name, "Adrian Tchaikovsky")
		}
		if info.Bio != "Born in Lincolnshire, studied zoology." {
			t.Errorf("Bio = %q, want %q", info.Bio, "Born in Lincolnshire, studied zoology.")
		}
		if info.Born != "November 7, 1972" {
			t.Errorf("Born = %q, want %q", info.Born, "November 7, 1972")
		}
		if info.Website != "http://shadowsoftheapt.com/" {
			t.Errorf("Website = %q, want %q", info.Website, "http://shadowsoftheapt.com/")
		}
		if len(info.Genres) != 2 || info.Genres[0] != "Science Fiction" || info.Genres[1] != "Fantasy" {
			t.Errorf("Genres = %v, want [Science Fiction, Fantasy]", info.Genres)
		}
	})

	t.Run("name from title fallback", func(t *testing.T) {
		html := `<html><head><title>Frank Herbert (Author of Dune)</title></head><body></body></html>`
		doc := docFromHTML(t, html)
		info := parseAuthorInfo(doc)

		if info.Name != "Frank Herbert" {
			t.Errorf("Name = %q, want %q", info.Name, "Frank Herbert")
		}
	})

	t.Run("no name found", func(t *testing.T) {
		html := `<html><head><title>Goodreads</title></head><body></body></html>`
		doc := docFromHTML(t, html)
		info := parseAuthorInfo(doc)

		if info.Name != "" {
			t.Errorf("Name = %q, want empty", info.Name)
		}
	})

	t.Run("bio whitespace is collapsed", func(t *testing.T) {
		html := `<html><body>
			<div class="aboutAuthorInfo"><span>Line one.
			Line two.    Extra   spaces.</span></div>
		</body></html>`
		doc := docFromHTML(t, html)
		info := parseAuthorInfo(doc)

		if info.Bio != "Line one. Line two. Extra spaces." {
			t.Errorf("Bio = %q, want %q", info.Bio, "Line one. Line two. Extra spaces.")
		}
	})

	t.Run("genres limited to 5", func(t *testing.T) {
		html := `<html><body>
			<div class="dataTitle">Genre</div>
			<div class="dataItem"><a>G1</a><a>G2</a><a>G3</a><a>G4</a><a>G5</a><a>G6</a><a>G7</a></div>
		</body></html>`
		doc := docFromHTML(t, html)
		info := parseAuthorInfo(doc)

		if len(info.Genres) != 5 {
			t.Fatalf("got %d genres, want 5", len(info.Genres))
		}
	})

	t.Run("empty page returns zero values", func(t *testing.T) {
		html := `<html><body></body></html>`
		doc := docFromHTML(t, html)
		info := parseAuthorInfo(doc)

		if info.Name != "" || info.Bio != "" || info.Website != "" || info.Born != "" || len(info.Genres) != 0 {
			t.Errorf("expected all zero values, got %+v", info)
		}
	})
}

func TestParseAuthorBooks(t *testing.T) {
	t.Run("single book with rating", func(t *testing.T) {
		html := `<table>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/25499718"><span>Children of Time</span></a></td>
			<td><span class="minirating">4.30 avg rating — 176,816 ratings</span></td>
		</tr>
		</table>`
		doc := docFromHTML(t, html)
		books := parseAuthorBooks(doc, 10)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		if books[0].ID != "25499718" {
			t.Errorf("ID = %q, want %q", books[0].ID, "25499718")
		}
		if books[0].Title != "Children of Time" {
			t.Errorf("Title = %q, want %q", books[0].Title, "Children of Time")
		}
		if books[0].Rating != "4.30 avg rating — 176,816 ratings" {
			t.Errorf("Rating = %q, want %q", books[0].Rating, "4.30 avg rating — 176,816 ratings")
		}
	})

	t.Run("title without span", func(t *testing.T) {
		html := `<table>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/77711">A Fire Upon the Deep</a></td>
		</tr>
		</table>`
		doc := docFromHTML(t, html)
		books := parseAuthorBooks(doc, 10)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		if books[0].Title != "A Fire Upon the Deep" {
			t.Errorf("Title = %q, want %q", books[0].Title, "A Fire Upon the Deep")
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		html := `<table>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/1"><span>Book A</span></a></td>
		</tr>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/2"><span>Book B</span></a></td>
		</tr>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/3"><span>Book C</span></a></td>
		</tr>
		</table>`
		doc := docFromHTML(t, html)
		books := parseAuthorBooks(doc, 2)

		if len(books) != 2 {
			t.Fatalf("got %d books, want 2", len(books))
		}
	})

	t.Run("skips rows without bookTitle link", func(t *testing.T) {
		html := `<table>
		<tr itemtype="http://schema.org/Book">
			<td>No link here</td>
		</tr>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/42"><span>Real Book</span></a></td>
		</tr>
		</table>`
		doc := docFromHTML(t, html)
		books := parseAuthorBooks(doc, 10)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		if books[0].ID != "42" {
			t.Errorf("ID = %q, want %q", books[0].ID, "42")
		}
	})

	t.Run("empty table", func(t *testing.T) {
		html := `<table></table>`
		doc := docFromHTML(t, html)
		books := parseAuthorBooks(doc, 10)

		if len(books) != 0 {
			t.Fatalf("got %d books, want 0", len(books))
		}
	})

	t.Run("book without rating", func(t *testing.T) {
		html := `<table>
		<tr itemtype="http://schema.org/Book">
			<td><a class="bookTitle" href="/book/show/999"><span>Obscure Title</span></a></td>
		</tr>
		</table>`
		doc := docFromHTML(t, html)
		books := parseAuthorBooks(doc, 10)

		if len(books) != 1 {
			t.Fatalf("got %d books, want 1", len(books))
		}
		if books[0].Rating != "" {
			t.Errorf("Rating = %q, want empty", books[0].Rating)
		}
	})
}
