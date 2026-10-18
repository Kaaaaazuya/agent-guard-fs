// Package pathutil validates and normalizes paths relative to the root
// (DESIGN §6.4). Policy and filesystem code accept only the types defined
// here, so no unchecked string path crosses a package boundary.
package pathutil

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Kaaaaazuya/agent-guard-fs/internal/errs"
)

// RelPath is a validated, normalized path relative to the root. Its
// elements are separated by "/", and the root itself is ".". The zero
// value is not valid; use Parse.
type RelPath struct {
	p string
}

// Parse validates s and returns it in normalized form. It rejects empty
// and absolute paths, any ".." element (even one that would stay inside
// the root), "\" and ":", NUL and other control characters, and invalid
// UTF-8. "./a", "a//b" and a trailing "/" are normalized.
func Parse(s string) (RelPath, error) {
	elems, err := split(s)
	if err != nil {
		return RelPath{}, err
	}
	return RelPath{p: join(elems)}, nil
}

// String returns the normalized path.
func (r RelPath) String() string { return r.p }

// IsRoot reports whether r is the root itself.
func (r RelPath) IsRoot() bool { return r.p == "." }

// split validates s and returns its elements with "." and empty elements
// removed.
func split(s string) ([]string, error) {
	if s == "" {
		return nil, invalid("path is empty")
	}
	if !utf8.ValidString(s) {
		return nil, invalid("path is not valid UTF-8")
	}
	if strings.HasPrefix(s, "/") {
		return nil, invalid("path must be relative to the root")
	}
	for _, r := range s {
		switch {
		case r == '\\' || r == ':':
			return nil, invalid(`path must not contain "\" or ":"`)
		case unicode.IsControl(r):
			return nil, invalid("path must not contain control characters")
		}
	}
	var elems []string
	for e := range strings.SplitSeq(s, "/") {
		switch e {
		case "", ".":
			continue
		case "..":
			return nil, invalid(`path must not contain ".."`)
		}
		elems = append(elems, e)
	}
	return elems, nil
}

func join(elems []string) string {
	if len(elems) == 0 {
		return "."
	}
	return strings.Join(elems, "/")
}

func invalid(msg string) error {
	return errs.New(errs.InvalidArgument, msg)
}
