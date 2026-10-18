package policy

import "testing"

func TestMatchSegment(t *testing.T) {
	tests := []struct {
		seg, name string
		want      bool
	}{
		{"src", "src", true},
		{"src", "src-old", false},
		{"src", "sr", false},
		{".env*", ".env", true},
		{".env*", ".env.local", true},
		{".env*", "env", false},
		{"*", "", true},
		{"*", "anything", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "axxbyyc", true},
		{"a*b*c", "acb", false},
		{"?", "a", true},
		{"?", "", false},
		{"?", "ab", false},
		{"?.go", "a.go", true},
		{"?.go", ".go", false},
		// "?" は 1 文字（rune）に一致する。
		{"?", "日", true},
		{"日?", "日本", true},
		// パス名に含まれる "*" や "?" は普通の文字として扱う。
		{"a?", "a*", true},
		{"*", "a?b", true},
	}
	for _, tt := range tests {
		if got := matchSegment(tt.seg, tt.name); got != tt.want {
			t.Errorf("matchSegment(%q, %q) = %v, want %v", tt.seg, tt.name, got, tt.want)
		}
	}
}
