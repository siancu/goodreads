package main

import "testing"

func TestParseUserList(t *testing.T) {
	t.Run("extracts users and deduplicates", func(t *testing.T) {
		html := `<div>
			<a href="/user/show/100.Alice">Alice</a>
			<a href="/user/show/100.Alice">Alice Again</a>
			<a href="/user/show/200.Bob">Bob</a>
		</div>`
		doc := docFromHTML(t, html)
		users := parseUserList(doc, "999")

		if len(users) != 2 {
			t.Fatalf("got %d users, want 2", len(users))
		}
		if users[0].ID != "100" || users[0].Name != "Alice" {
			t.Errorf("users[0] = %+v, want {ID:100, Name:Alice}", users[0])
		}
		if users[1].ID != "200" || users[1].Name != "Bob" {
			t.Errorf("users[1] = %+v, want {ID:200, Name:Bob}", users[1])
		}
	})

	t.Run("excludes self", func(t *testing.T) {
		html := `<div>
			<a href="/user/show/42.Me">Me</a>
			<a href="/user/show/99.Friend">Friend</a>
		</div>`
		doc := docFromHTML(t, html)
		users := parseUserList(doc, "42")

		if len(users) != 1 {
			t.Fatalf("got %d users, want 1", len(users))
		}
		if users[0].ID != "99" {
			t.Errorf("ID = %q, want %q", users[0].ID, "99")
		}
	})

	t.Run("skips links without name text", func(t *testing.T) {
		html := `<div>
			<a href="/user/show/50"></a>
			<a href="/user/show/60.Named">Named</a>
		</div>`
		doc := docFromHTML(t, html)
		users := parseUserList(doc, "0")

		if len(users) != 1 {
			t.Fatalf("got %d users, want 1", len(users))
		}
	})

	t.Run("no users", func(t *testing.T) {
		html := `<div><p>No friends found</p></div>`
		doc := docFromHTML(t, html)
		users := parseUserList(doc, "0")

		if len(users) != 0 {
			t.Fatalf("got %d users, want 0", len(users))
		}
	})
}

func TestParseUserProfile(t *testing.T) {
	t.Run("all fields present", func(t *testing.T) {
		html := `<html><body>
			<h1 class="userProfileName">Jane Doe</h1>
			<div class="infoBoxRowItem" itemprop="address">London, UK</div>
			<div class="infoBoxRowTitle">Joined</div>
			<div class="infoBoxRowItem">March 2015</div>
			<a href="/review/list/123">42 books</a>
			<a href="/friend/123">15 friends</a>
		</body></html>`
		doc := docFromHTML(t, html)
		p := parseUserProfile(doc)

		if p.Name != "Jane Doe" {
			t.Errorf("Name = %q, want %q", p.Name, "Jane Doe")
		}
		if p.Location != "London, UK" {
			t.Errorf("Location = %q, want %q", p.Location, "London, UK")
		}
		if p.JoinDate != "March 2015" {
			t.Errorf("JoinDate = %q, want %q", p.JoinDate, "March 2015")
		}
		if p.BooksCount != "42" {
			t.Errorf("BooksCount = %q, want %q", p.BooksCount, "42")
		}
		if p.FriendsCount != "15" {
			t.Errorf("FriendsCount = %q, want %q", p.FriendsCount, "15")
		}
	})

	t.Run("name from title fallback", func(t *testing.T) {
		html := `<html><head><title>John Smith | Goodreads</title></head><body></body></html>`
		doc := docFromHTML(t, html)
		p := parseUserProfile(doc)

		if p.Name != "John Smith" {
			t.Errorf("Name = %q, want %q", p.Name, "John Smith")
		}
	})

	t.Run("location from infoBoxRowTitle fallback", func(t *testing.T) {
		html := `<html><body>
			<div class="infoBoxRowTitle">Location</div>
			<div class="infoBoxRowItem">Berlin, Germany</div>
		</body></html>`
		doc := docFromHTML(t, html)
		p := parseUserProfile(doc)

		if p.Location != "Berlin, Germany" {
			t.Errorf("Location = %q, want %q", p.Location, "Berlin, Germany")
		}
	})

	t.Run("empty page returns zero values", func(t *testing.T) {
		html := `<html><body></body></html>`
		doc := docFromHTML(t, html)
		p := parseUserProfile(doc)

		if p.Name != "" || p.Location != "" || p.JoinDate != "" || p.BooksCount != "" || p.FriendsCount != "" {
			t.Errorf("expected all empty, got %+v", p)
		}
	})
}

func TestParseShelves(t *testing.T) {
	t.Run("extracts shelves with counts", func(t *testing.T) {
		html := `<div id="paginatedShelfList">
			<a href="/review/list/123?shelf=read">read <span class="smallText">(42)</span></a>
			<a href="/review/list/123?shelf=to-read">to-read <span class="smallText">(10)</span></a>
		</div>`
		doc := docFromHTML(t, html)
		shelves := parseShelves(doc)

		if len(shelves) != 2 {
			t.Fatalf("got %d shelves, want 2", len(shelves))
		}
		if shelves[0].Name != "read" || shelves[0].Count != "(42)" {
			t.Errorf("shelves[0] = %+v, want {Name:read, Count:(42)}", shelves[0])
		}
		if shelves[1].Name != "to-read" || shelves[1].Count != "(10)" {
			t.Errorf("shelves[1] = %+v, want {Name:to-read, Count:(10)}", shelves[1])
		}
	})

	t.Run("deduplicates shelves", func(t *testing.T) {
		html := `<div>
			<a class="actionLinkLite bookPageGenreLink" href="/review/list/1?shelf=read">read</a>
			<a class="actionLinkLite bookPageGenreLink" href="/review/list/1?shelf=read&page=2">read</a>
		</div>`
		doc := docFromHTML(t, html)
		shelves := parseShelves(doc)

		if len(shelves) != 1 {
			t.Fatalf("got %d shelves, want 1", len(shelves))
		}
	})

	t.Run("shelves without count", func(t *testing.T) {
		html := `<div>
			<a class="actionLinkLite bookPageGenreLink" href="/review/list/1?shelf=custom">custom</a>
		</div>`
		doc := docFromHTML(t, html)
		shelves := parseShelves(doc)

		if len(shelves) != 1 {
			t.Fatalf("got %d shelves, want 1", len(shelves))
		}
		if shelves[0].Count != "" {
			t.Errorf("Count = %q, want empty", shelves[0].Count)
		}
	})

	t.Run("no shelves", func(t *testing.T) {
		html := `<div><p>No shelves</p></div>`
		doc := docFromHTML(t, html)
		shelves := parseShelves(doc)

		if len(shelves) != 0 {
			t.Fatalf("got %d shelves, want 0", len(shelves))
		}
	})
}

func TestParseReadingStats(t *testing.T) {
	t.Run("parses year and page stats", func(t *testing.T) {
		body := `<script>
			var year_stats = {"2023":9,"2022":3,"2021":5};
			var page_stats = {"2023":4602,"2022":1538,"2021":1702};
		</script>`
		stats := parseReadingStats(body)

		if len(stats.YearStats) != 3 {
			t.Fatalf("got %d year entries, want 3", len(stats.YearStats))
		}
		if stats.YearStats["2023"] != 9 {
			t.Errorf("YearStats[2023] = %d, want 9", stats.YearStats["2023"])
		}
		if stats.PageStats["2023"] != 4602 {
			t.Errorf("PageStats[2023] = %d, want 4602", stats.PageStats["2023"])
		}
	})

	t.Run("no stats returns empty maps", func(t *testing.T) {
		body := `<html><body>no stats here</body></html>`
		stats := parseReadingStats(body)

		if len(stats.YearStats) != 0 {
			t.Errorf("got %d year entries, want 0", len(stats.YearStats))
		}
		if len(stats.PageStats) != 0 {
			t.Errorf("got %d page entries, want 0", len(stats.PageStats))
		}
	})

	t.Run("only year stats present", func(t *testing.T) {
		body := `var year_stats = {"2024":2};`
		stats := parseReadingStats(body)

		if stats.YearStats["2024"] != 2 {
			t.Errorf("YearStats[2024] = %d, want 2", stats.YearStats["2024"])
		}
		if len(stats.PageStats) != 0 {
			t.Errorf("got %d page entries, want 0", len(stats.PageStats))
		}
	})
}

func TestFormatCommas(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{41085, "41,085"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCommas(tt.input)
			if got != tt.want {
				t.Errorf("formatCommas(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFlagString(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		long, short string
		defaultVal string
		want       string
	}{
		{"long flag", []string{"--shelf", "to-read"}, "--shelf", "-s", "read", "to-read"},
		{"short flag", []string{"-s", "custom"}, "--shelf", "-s", "read", "custom"},
		{"missing flag returns default", []string{"--other", "x"}, "--shelf", "-s", "read", "read"},
		{"empty args returns default", []string{}, "--shelf", "-s", "read", "read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flagString(tt.args, tt.long, tt.short, tt.defaultVal)
			if got != tt.want {
				t.Errorf("flagString() = %q, want %q", got, tt.want)
			}
		})
	}
}
