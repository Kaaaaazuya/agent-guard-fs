package pathutil

import (
	"testing"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

func TestParsePatternNormalizes(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"src", "src"},
		{".", "."},
		{"**/.env*", "**/.env*"},
		{"./src/**", "src/**"},
		{"src//*.go", "src/*.go"},
		{"a?c/", "a?c"},
		{"*/*", "*/*"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParsePattern(tt.in)
			if err != nil {
				t.Fatalf("ParsePattern(%q) error: %v", tt.in, err)
			}
			if got.String() != tt.want {
				t.Errorf("ParsePattern(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParsePatternRejects(t *testing.T) {
	tests := []struct {
		name, in string
	}{
		{"empty", ""},
		{"absolute", "/src"},
		{"parent", "src/../.env"},
		{"backslash escape", `\*`},
		{"colon", "c:/x"},
		{"NUL", "a\x00"},
		{"bracket", "[ab]"},
		{"closing bracket", "a]"},
		{"brace", "{a,b}"},
		{"closing brace", "a}"},
		{"double star prefix", "a**"},
		{"double star suffix", "**a"},
		{"triple star", "***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePattern(tt.in)
			if code := errs.CodeOf(err); err == nil || code != errs.InvalidArgument {
				t.Errorf("ParsePattern(%q) error = %v, want %s", tt.in, err, errs.InvalidArgument)
			}
		})
	}
}

// Glob characters are ordinary characters in input paths.
func TestParseAcceptsGlobCharacters(t *testing.T) {
	for _, in := range []string{"a*b", "**", "what?.txt", "[x]", "{y}"} {
		if _, err := Parse(in); err != nil {
			t.Errorf("Parse(%q) error: %v", in, err)
		}
	}
}
