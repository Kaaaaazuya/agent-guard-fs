package policy

import "slices"

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

// matchPrefixes は、パターン要素 pat が path の先頭 j 要素（j = 0 .. len(path)）に
// 一致するかを返す。結果の長さは len(path)+1 で、result[j] が対応する。
// "**" はパス要素全体として書かれ、0 個以上の要素に一致する。
// 動的計画法で計算するので、"**" が多くても計算量はパターン長とパス長の積に比例する。
func matchPrefixes(pat, path []string) []bool {
	// prev[j] は、ここまでのパターンが path[:j] に一致するかを表す。
	prev := make([]bool, len(path)+1)
	prev[0] = true
	for _, seg := range pat {
		cur := make([]bool, len(path)+1)
		for j := 0; j <= len(path); j++ {
			if seg == "**" {
				// 0 要素で一致するか、path[j-1] を吸収して一致するか。
				cur[j] = prev[j] || (j > 0 && cur[j-1])
			} else {
				cur[j] = j > 0 && prev[j-1] && matchSegment(seg, path[j-1])
			}
		}
		prev = cur
	}
	return prev
}

// matchRule は、ルールのパターンがパス path を対象とするかを返す。
// recursive が false なら、パターンがパス全体に一致する必要がある。
// true なら、パス自身またはいずれかの祖先に一致すればよい。
func matchRule(pat, path []string, recursive bool) bool {
	prefixes := matchPrefixes(pat, path)
	if recursive {
		return slices.Contains(prefixes, true)
	}
	return prefixes[len(path)]
}
