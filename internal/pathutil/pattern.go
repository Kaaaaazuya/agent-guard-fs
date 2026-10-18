package pathutil

import "strings"

// Pattern is a validated, normalized rule path (DESIGN §6.3). It follows
// the same rules as RelPath and may additionally contain the glob
// characters "*" and "?". "**" is allowed only as a whole element.
type Pattern struct {
	p string
}

// ParsePattern validates a rule path from the policy. Besides the RelPath
// rules, it rejects "[", "]", "{" and "}" so that no glob syntax beyond
// "*", "**" and "?" can be written, and "**" mixed with other characters
// in one element (such as "a**").
func ParsePattern(s string) (Pattern, error) {
	elems, err := split(s, checkPatternElement)
	if err != nil {
		return Pattern{}, err
	}
	return Pattern{p: join(elems)}, nil
}

// String returns the normalized pattern.
func (p Pattern) String() string { return p.p }

func checkPatternElement(e string) error {
	if strings.ContainsAny(e, "[]{}") {
		return invalid(`rule path must not contain "[", "]", "{" or "}"`)
	}
	if e != "**" && strings.Contains(e, "**") {
		return invalid(`"**" must be a whole path element`)
	}
	return nil
}
