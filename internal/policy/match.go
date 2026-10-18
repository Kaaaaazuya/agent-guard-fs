package policy

// matchSegment は、1 つのパス要素 name がパターン要素 seg に一致するかを返す。
// seg の "*" は任意の文字列（空を含む）、"?" は任意の 1 文字に一致する。
// 文字は rune 単位で比較するので、"?" がマルチバイト文字の途中で切れることはない。
func matchSegment(seg, name string) bool {
	p, s := []rune(seg), []rune(name)
	pi, si := 0, 0
	star, mark := -1, 0
	for si < len(s) {
		switch {
		case pi < len(p) && p[pi] != '*' && (p[pi] == '?' || p[pi] == s[si]):
			pi++
			si++
		case pi < len(p) && p[pi] == '*':
			star = pi
			mark = si
			pi++
		case star >= 0:
			// 直前の "*" に、もう 1 文字ぶん余計に吸収させて再試行する。
			pi = star + 1
			mark++
			si = mark
		default:
			return false
		}
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}
