package logging

import "testing"

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "<unset>"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcde", "abcd..."},
		{"ghp_abc123xyz", "ghp_..."},
	}
	for _, tc := range tests {
		got := MaskSecret(tc.input)
		if got != tc.want {
			t.Errorf("MaskSecret(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMaskPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "<unset>"},
		{"/home/user/secrets/creds.json", ".../creds.json"},
		{"creds.json", ".../creds.json"},
		{"/tmp/service-account.json", ".../service-account.json"},
	}
	for _, tc := range tests {
		got := MaskPath(tc.input)
		if got != tc.want {
			t.Errorf("MaskPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
