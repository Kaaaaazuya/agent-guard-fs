package pathutil

import (
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

func TestParseNormalizes(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"a", "a"},
		{"src/main.go", "src/main.go"},
		{"./a", "a"},
		{"a//b", "a/b"},
		{"a/./b", "a/b"},
		{"a/", "a"},
		{".", "."},
		{"./", "."},
		{".env", ".env"},
		{"a..b", "a..b"},
		{"...", "..."},
		{"日本語/ファイル.txt", "日本語/ファイル.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.in, err)
			}
			if got.String() != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		name, in string
	}{
		{"empty", ""},
		{"absolute", "/etc/passwd"},
		{"parent", ".."},
		{"parent prefix", "../a"},
		{"parent inside root", "a/../b"},
		{"parent trailing", "a/.."},
		{"backslash", `a\b`},
		{"colon", "a:b"},
		{"NUL", "a\x00b"},
		{"newline", "a\nb"},
		{"tab", "a\tb"},
		{"DEL", "a\x7fb"},
		{"C1 control", "a\u0085b"},
		{"invalid UTF-8", "a\xffb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.in)
			if code := errs.CodeOf(err); err == nil || code != errs.InvalidArgument {
				t.Errorf("Parse(%q) error = %v, want %s", tt.in, err, errs.InvalidArgument)
			}
		})
	}
}

func TestIsRoot(t *testing.T) {
	for in, want := range map[string]bool{".": true, "./": true, "a": false} {
		p, err := Parse(in)
		if err != nil {
			t.Fatal(err)
		}
		if p.IsRoot() != want {
			t.Errorf("Parse(%q).IsRoot() = %v, want %v", in, p.IsRoot(), want)
		}
	}
}
