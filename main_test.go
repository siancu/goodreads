package main

import "testing"

func TestHasFlag(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		long  string
		short string
		want  bool
	}{
		{"long flag present", []string{"--debug"}, "--debug", "-d", true},
		{"short flag present", []string{"-d"}, "--debug", "-d", true},
		{"flag absent", []string{"--other"}, "--debug", "-d", false},
		{"empty args", nil, "--debug", "-d", false},
		{"mixed args with flag", []string{"--limit", "5", "--debug"}, "--debug", "-d", true},
		{"mixed args without flag", []string{"--limit", "5"}, "--debug", "-d", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFlag(tt.args, tt.long, tt.short)
			if got != tt.want {
				t.Errorf("hasFlag(%v, %q, %q) = %v, want %v", tt.args, tt.long, tt.short, got, tt.want)
			}
		})
	}
}

func TestFlagInt(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		long       string
		short      string
		defaultVal int
		want       int
	}{
		{"long flag", []string{"--limit", "20"}, "--limit", "-n", 10, 20},
		{"short flag", []string{"-n", "5"}, "--limit", "-n", 10, 5},
		{"missing flag returns default", []string{"--other", "3"}, "--limit", "-n", 10, 10},
		{"empty args returns default", nil, "--limit", "-n", 10, 10},
		{"non-numeric value returns default", []string{"--limit", "abc"}, "--limit", "-n", 10, 10},
		{"flag at end without value returns default", []string{"--limit"}, "--limit", "-n", 10, 10},
		{"flag among other args", []string{"--debug", "--limit", "42", "--force"}, "--limit", "-n", 10, 42},
		{"zero value", []string{"--limit", "0"}, "--limit", "-n", 10, 0},
		{"negative value", []string{"--limit", "-1"}, "--limit", "-n", 10, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flagInt(tt.args, tt.long, tt.short, tt.defaultVal)
			if got != tt.want {
				t.Errorf("flagInt(%v, %q, %q, %d) = %d, want %d", tt.args, tt.long, tt.short, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestParseLoginFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantEmail string
		wantPass  string
		wantDebug bool
	}{
		{"all long flags", []string{"--email", "a@b.com", "--password", "secret", "--debug"},
			"a@b.com", "secret", true},
		{"all short flags", []string{"-e", "a@b.com", "-p", "secret", "-d"},
			"a@b.com", "secret", true},
		{"no flags", nil, "", "", false},
		{"email only", []string{"--email", "a@b.com"}, "a@b.com", "", false},
		{"debug only", []string{"--debug"}, "", "", true},
		{"flag without value is ignored", []string{"--email"}, "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, password, debug := parseLoginFlags(tt.args)
			if email != tt.wantEmail {
				t.Errorf("email = %q, want %q", email, tt.wantEmail)
			}
			if password != tt.wantPass {
				t.Errorf("password = %q, want %q", password, tt.wantPass)
			}
			if debug != tt.wantDebug {
				t.Errorf("debug = %v, want %v", debug, tt.wantDebug)
			}
		})
	}
}
