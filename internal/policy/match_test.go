package policy

import (
	"strings"
	"testing"
)

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

// splitTest は "a/b" 形式を要素列にする。"." はルート（空の列）になる。
func splitTest(s string) []string {
	if s == "." {
		return nil
	}
	return strings.Split(s, "/")
}

func TestMatchRule(t *testing.T) {
	tests := []struct {
		name      string
		pat, path string
		recursive bool
		want      bool
	}{
		{"exact", "src", "src", false, true},
		{"not recursive excludes children", "src", "src/a", false, false},
		{"recursive includes children", "src", "src/a/b", true, true},
		{"recursive includes itself", "src", "src", true, true},
		{"element boundary, not prefix", "src", "src-old", true, false},
		{"element boundary, not prefix of child", "src", "src-old/a", true, false},
		{"root only matches root", ".", "src", false, false},
		{"root only matches root itself", ".", ".", false, true},
		{"root recursive matches everything", ".", "a/b/c", true, true},
		{"double star matches zero elements", "src/**", "src", false, true},
		{"double star matches one element", "src/**", "src/a", false, true},
		{"double star matches many elements", "src/**", "src/a/b/c", false, true},
		{"double star leading", "**/.env*", ".env", false, true},
		{"double star leading nested", "**/.env*", "src/.env", false, true},
		{"double star leading deep", "**/.env*", "src/a/.env.local", false, true},
		{"double star does not match a different name", "**/.env*", "src/env", false, false},
		{"double star in middle", "a/**/z", "a/z", false, true},
		{"double star in middle deep", "a/**/z", "a/b/c/z", false, true},
		{"double star in middle wrong tail", "a/**/z", "a/b/y", false, false},
		{"glob within element", "src/*.go", "src/main.go", false, true},
		{"glob does not cross elements", "src/*.go", "src/a/main.go", false, false},
		{"recursive glob on prefix", "src/*", "src/a/b", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchRule(splitTest(tt.pat), splitTest(tt.path), tt.recursive)
			if got != tt.want {
				t.Errorf("matchRule(%q, %q, recursive=%v) = %v, want %v", tt.pat, tt.path, tt.recursive, got, tt.want)
			}
		})
	}
}
