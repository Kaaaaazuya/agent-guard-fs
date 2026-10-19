package policy

import (
	"fmt"
	"slices"
	"strings"
)

// Shadowed は、deny ルールにすっかり覆われていて効かない allow ルール。
// deny が優先なので、書いた人の意図と違うことが多い（DESIGN §6.2）。
type Shadowed struct {
	// Rule は覆われている allow ルールの名前。
	Rule string
	// By は、その allow ルールの操作を拒否しているルールの名前（重複なし、名前順）。
	By []string
}

func (s Shadowed) String() string {
	return fmt.Sprintf("allow rule %q never takes effect: everything it allows is denied by %s", s.Rule, quoteAll(s.By))
}

func quoteAll(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(q, ", ")
}

// Shadowed は、deny ルールに覆われて効かない allow ルールを返す。
// グロブを含む allow ルールは、覆われているかを確実には言えないので調べない。
// したがって、返らないルールが効くとは限らない。見つかったものは確実に効かない。
func (e *Engine) Shadowed() []Shadowed {
	var out []Shadowed
	for _, a := range e.allow {
		if hasGlob(a.elems) {
			continue
		}
		path := splitPath(fold(joinPath(a.elems)))
		var by []string
		covered := true
		for _, action := range a.actions {
			name, ok := e.denyCovering(action, path, a.recursive)
			if !ok {
				covered = false
				break
			}
			if !slices.Contains(by, name) {
				by = append(by, name)
			}
		}
		if covered {
			slices.Sort(by)
			out = append(out, Shadowed{Rule: a.name, By: by})
		}
	}
	return out
}

// denyCovering は、action について path を覆う deny ルールの名前を返す。
// 配下も含めて覆う必要があるとき（subtree）は、配下まで及ぶ deny ルールだけを数える。
func (e *Engine) denyCovering(action Action, path []string, subtree bool) (string, bool) {
	for _, d := range e.deny {
		if !slices.Contains(d.actions, action) {
			continue
		}
		reachesBelow := d.recursive || (len(d.elems) > 0 && d.elems[len(d.elems)-1] == "**")
		if subtree && !reachesBelow {
			continue
		}
		if matchRule(d.elems, path, d.recursive) {
			return d.name, true
		}
	}
	return "", false
}

func hasGlob(elems []string) bool {
	return slices.ContainsFunc(elems, func(e string) bool { return strings.ContainsAny(e, "*?") })
}

// joinPath は splitPath の逆。ルートは空の列なので "." に戻す。
func joinPath(elems []string) string {
	if len(elems) == 0 {
		return "."
	}
	return strings.Join(elems, "/")
}
