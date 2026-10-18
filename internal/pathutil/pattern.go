package pathutil

import "strings"

// Pattern は検証・正規化済みのルールパス（DESIGN §6.3）。RelPath と同じ規則に従い、
// 加えてグロブ文字 "*" と "?" を含められる。"**" は要素全体としてのみ使える。
type Pattern struct {
	p string
}

// ParsePattern はポリシーのルールパスを検証する。RelPath の規則に加え、"*"・"**"・"?"
// 以外のグロブ構文を書けないよう "["、"]"、"{"、"}" を拒否し、1 つの要素内で "**" と
// 他の文字を混ぜたもの（"a**" など）も拒否する。
func ParsePattern(s string) (Pattern, error) {
	elems, err := split(s, checkPatternElement)
	if err != nil {
		return Pattern{}, err
	}
	return Pattern{p: join(elems)}, nil
}

// String は正規化済みのパターンを返す。
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
