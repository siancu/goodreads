package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns everything it wrote to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintWrapped(t *testing.T) {
	t.Run("short text fits on one line", func(t *testing.T) {
		got := captureStdout(t, func() { printWrapped("hello world", 70) })
		// Should be "  hello world\n"
		if got != "  hello world\n" {
			t.Errorf("got %q, want %q", got, "  hello world\n")
		}
	})

	t.Run("wraps at width boundary", func(t *testing.T) {
		got := captureStdout(t, func() { printWrapped("aaa bbb ccc ddd", 12) })
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		// Each line should start with 2-space indent
		for i, line := range lines {
			if !strings.HasPrefix(line, "  ") {
				t.Errorf("line %d = %q, missing 2-space indent", i, line)
			}
		}
		// Should wrap into multiple lines
		if len(lines) < 2 {
			t.Errorf("expected multiple lines for width 12, got %d: %q", len(lines), got)
		}
	})

	t.Run("empty text produces no output", func(t *testing.T) {
		got := captureStdout(t, func() { printWrapped("", 70) })
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("whitespace-only text produces no output", func(t *testing.T) {
		got := captureStdout(t, func() { printWrapped("   \t\n   ", 70) })
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestPickBestList(t *testing.T) {
	t.Run("returns first non-generic list", func(t *testing.T) {
		lists := []listEntry{
			{name: "Best Science Fiction Books", href: "/list/show/1"},
			{name: "Best Books of the Decade", href: "/list/show/2"},
			{name: "Hard Sci-Fi Space Operas", href: "/list/show/3"},
			{name: "Fiction & Fantasy", href: "/list/show/4"},
		}
		got := pickBestList(lists)
		if got.name != "Hard Sci-Fi Space Operas" {
			t.Errorf("got %q, want %q", got.name, "Hard Sci-Fi Space Operas")
		}
	})

	t.Run("falls back to first when all generic", func(t *testing.T) {
		lists := []listEntry{
			{name: "Best Science Fiction Books", href: "/list/show/1"},
			{name: "Best Books Ever", href: "/list/show/2"},
			{name: "Must Read Novels", href: "/list/show/3"},
		}
		got := pickBestList(lists)
		if got.name != "Best Science Fiction Books" {
			t.Errorf("got %q, want %q", got.name, "Best Science Fiction Books")
		}
	})

	t.Run("skips movie/film lists", func(t *testing.T) {
		lists := []listEntry{
			{name: "Best Movie Adaptations", href: "/list/show/1"},
			{name: "Great Space Adventures", href: "/list/show/2"},
		}
		got := pickBestList(lists)
		if got.name != "Great Space Adventures" {
			t.Errorf("got %q, want %q", got.name, "Great Space Adventures")
		}
	})

	t.Run("skips reading challenge lists", func(t *testing.T) {
		lists := []listEntry{
			{name: "2024 Reading Challenge", href: "/list/show/1"},
			{name: "Cyberpunk Novels", href: "/list/show/2"},
		}
		got := pickBestList(lists)
		if got.name != "Cyberpunk Novels" {
			t.Errorf("got %q, want %q", got.name, "Cyberpunk Novels")
		}
	})

	t.Run("skips bucket list and 1001 books", func(t *testing.T) {
		lists := []listEntry{
			{name: "Bucket List Books", href: "/list/show/1"},
			{name: "1001 Books to Read", href: "/list/show/2"},
			{name: "Underrated Gems", href: "/list/show/3"},
		}
		got := pickBestList(lists)
		if got.name != "Underrated Gems" {
			t.Errorf("got %q, want %q", got.name, "Underrated Gems")
		}
	})

	t.Run("single list is returned", func(t *testing.T) {
		lists := []listEntry{
			{name: "Best Fantasy Books", href: "/list/show/1"},
		}
		got := pickBestList(lists)
		if got.name != "Best Fantasy Books" {
			t.Errorf("got %q, want %q", got.name, "Best Fantasy Books")
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		lists := []listEntry{
			{name: "BEST SCIENCE FICTION & Fantasy", href: "/list/show/1"},
			{name: "Specific Niche List", href: "/list/show/2"},
		}
		got := pickBestList(lists)
		if got.name != "Specific Niche List" {
			t.Errorf("got %q, want %q", got.name, "Specific Niche List")
		}
	})
}

func TestValidateProgressArgs(t *testing.T) {
	tests := []struct {
		name    string
		page    int
		percent int
		wantErr string
	}{
		{"page only", 42, 0, ""},
		{"percent only", 0, 50, ""},
		{"neither specified", 0, 0, "specify --page or --percent"},
		{"both specified", 42, 50, "specify --page or --percent, not both"},
		{"negative page", -1, 0, "page must be a positive number"},
		{"percent too high", 0, 101, "percent must be between 1 and 100"},
		{"percent negative", 0, -5, "percent must be between 1 and 100"},
		{"percent at 100", 0, 100, ""},
		{"percent at 1", 0, 1, ""},
		{"page at 1", 1, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateProgressArgs(tt.page, tt.percent)
			if got != tt.wantErr {
				t.Errorf("validateProgressArgs(%d, %d) = %q, want %q", tt.page, tt.percent, got, tt.wantErr)
			}
		})
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"reading", "reading", "currently-reading"},
		{"currently-reading", "currently-reading", "currently-reading"},
		{"read", "read", "read"},
		{"finished", "finished", "read"},
		{"to-read", "to-read", "to-read"},
		{"want-to-read", "want-to-read", "to-read"},
		{"case insensitive", "Reading", "currently-reading"},
		{"unknown status", "dropped", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeStatus(tt.input)
			if got != tt.want {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
