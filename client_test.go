package main

import "testing"

func TestCsrfToken(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"token present",
			`<html><head><meta name="csrf-token" content="abc123xyz"></head></html>`,
			"abc123xyz",
		},
		{
			"no meta tag",
			`<html><head><title>No Token</title></head></html>`,
			"",
		},
		{
			"empty content",
			`<html><head><meta name="csrf-token" content=""></head></html>`,
			"",
		},
		{
			"different meta tag",
			`<html><head><meta name="viewport" content="width=device-width"></head></html>`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := docFromHTML(t, tt.html)
			got := csrfToken(doc)
			if got != tt.want {
				t.Errorf("csrfToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFormData(t *testing.T) {
	t.Run("form by name", func(t *testing.T) {
		html := `<html><body>
		<form name="login" action="/sign_in">
			<input name="email" value="test@example.com">
			<input name="token" value="abc">
			<input name="submit" value="Log In">
		</form>
		</body></html>`

		doc := docFromHTML(t, html)
		action, fields := extractFormData(doc, "login")
		if action != "/sign_in" {
			t.Errorf("action = %q, want %q", action, "/sign_in")
		}
		if fields.Get("email") != "test@example.com" {
			t.Errorf("email = %q, want %q", fields.Get("email"), "test@example.com")
		}
		if fields.Get("token") != "abc" {
			t.Errorf("token = %q, want %q", fields.Get("token"), "abc")
		}
	})

	t.Run("form by id ap_signin_form", func(t *testing.T) {
		html := `<html><body>
		<form id="ap_signin_form" action="/ap/signin">
			<input name="appActionToken" value="xyz">
		</form>
		</body></html>`

		doc := docFromHTML(t, html)
		action, fields := extractFormData(doc, "nonexistent")
		if action != "/ap/signin" {
			t.Errorf("action = %q, want %q", action, "/ap/signin")
		}
		if fields.Get("appActionToken") != "xyz" {
			t.Errorf("appActionToken = %q, want %q", fields.Get("appActionToken"), "xyz")
		}
	})

	t.Run("falls back to first form", func(t *testing.T) {
		html := `<html><body>
		<form action="/fallback">
			<input name="field1" value="val1">
		</form>
		</body></html>`

		doc := docFromHTML(t, html)
		action, fields := extractFormData(doc, "nonexistent")
		if action != "/fallback" {
			t.Errorf("action = %q, want %q", action, "/fallback")
		}
		if fields.Get("field1") != "val1" {
			t.Errorf("field1 = %q, want %q", fields.Get("field1"), "val1")
		}
	})

	t.Run("no forms returns empty", func(t *testing.T) {
		html := `<html><body><p>No forms here</p></body></html>`

		doc := docFromHTML(t, html)
		action, fields := extractFormData(doc, "login")
		if action != "" {
			t.Errorf("action = %q, want empty", action)
		}
		if len(fields) != 0 {
			t.Errorf("fields has %d entries, want 0", len(fields))
		}
	})

	t.Run("inputs without name are skipped", func(t *testing.T) {
		html := `<html><body>
		<form name="test" action="/test">
			<input name="included" value="yes">
			<input value="no-name">
			<input name="" value="empty-name">
		</form>
		</body></html>`

		doc := docFromHTML(t, html)
		_, fields := extractFormData(doc, "test")
		if fields.Get("included") != "yes" {
			t.Errorf("included = %q, want %q", fields.Get("included"), "yes")
		}
		if len(fields) != 1 {
			t.Errorf("fields has %d entries, want 1", len(fields))
		}
	})
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		ref  string
		want string
	}{
		{
			"absolute ref returned as-is",
			"https://www.goodreads.com/page",
			"https://www.amazon.com/login",
			"https://www.amazon.com/login",
		},
		{
			"relative path resolved",
			"https://www.goodreads.com/review/list",
			"/user_shelves",
			"https://www.goodreads.com/user_shelves",
		},
		{
			"relative path without leading slash",
			"https://www.goodreads.com/some/path",
			"other",
			"https://www.goodreads.com/some/other",
		},
		{
			"http ref is absolute",
			"https://www.goodreads.com/",
			"http://example.com/page",
			"http://example.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveURL(tt.base, tt.ref)
			if got != tt.want {
				t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.ref, got, tt.want)
			}
		})
	}
}
