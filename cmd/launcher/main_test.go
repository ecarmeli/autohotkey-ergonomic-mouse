package main

import "testing"

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain version", input: "1.0.6", want: "1.0.6"},
		{name: "lowercase v prefix", input: "v1.0.6", want: "1.0.6"},
		{name: "uppercase V prefix", input: "V1.0.6", want: "1.0.6"},
		{name: "surrounding whitespace", input: "  v1.0.6  ", want: "1.0.6"},
		{name: "empty value", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeVersion(tt.input)
			if got != tt.want {
				t.Errorf("normalizeVersion(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [3]int
		wantErr bool
	}{
		{name: "regular version", input: "1.0.6", want: [3]int{1, 0, 6}},
		{name: "version with v prefix", input: "v1.2.3", want: [3]int{1, 2, 3}},
		{name: "double-digit components", input: "10.20.30", want: [3]int{10, 20, 30}},
		{name: "missing patch component", input: "1.2", wantErr: true},
		{name: "extra component", input: "1.2.3.4", wantErr: true},
		{name: "non-numeric component", input: "1.2.x", wantErr: true},
		{name: "development version", input: "dev", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersion(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseVersion(%q) returned no error; want an error", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseVersion(%q) returned unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseVersion(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{name: "newer patch", latest: "v1.0.7", current: "1.0.6", want: true},
		{name: "newer minor", latest: "v1.1.0", current: "1.0.6", want: true},
		{name: "newer major", latest: "v2.0.0", current: "1.9.9", want: true},
		{name: "same version", latest: "v1.0.6", current: "1.0.6", want: false},
		{name: "older version", latest: "v1.0.5", current: "1.0.6", want: false},
		{name: "numeric patch comparison", latest: "v1.0.10", current: "1.0.9", want: true},
		{name: "numeric minor comparison", latest: "v1.10.0", current: "1.9.0", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isNewerVersion(tt.latest, tt.current)
			if err != nil {
				t.Fatalf("isNewerVersion(%q, %q) returned unexpected error: %v", tt.latest, tt.current, err)
			}
			if got != tt.want {
				t.Errorf("isNewerVersion(%q, %q) = %v; want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestIsNewerVersionRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
	}{
		{name: "development current version", latest: "v1.0.6", current: "dev"},
		{name: "invalid latest version", latest: "latest", current: "1.0.6"},
		{name: "empty latest version", latest: "", current: "1.0.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isNewerVersion(tt.latest, tt.current)
			if err == nil {
				t.Fatalf("isNewerVersion(%q, %q) returned no error; want an error", tt.latest, tt.current)
			}
			if got {
				t.Errorf("isNewerVersion(%q, %q) returned true with an error; want false", tt.latest, tt.current)
			}
		})
	}
}
