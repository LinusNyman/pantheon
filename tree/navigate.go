package tree

import "strings"

// Navigate resolves a sequence of characters to a node by descending the tree
// one level per character, choosing at each level the child whose descriptive
// name best matches the character. It is the prefix-aware port of the old
// `f`/`o`/`p` shell functions: a forgiving typeahead jump ("asb" → actio →
// scientia → bibliotheca) for interactive `cd`, complementing the exact
// Find(code) lookup.
//
// It returns the chain of matched nodes (empty if the first character matches
// nothing). The chain may be shorter than keys when a character matches no
// child at some level — navigation stops there and returns what it reached.
func (t *Tree) Navigate(keys string) []*Node {
	return navigate(t.Roots, keys)
}

// Navigate descends from n (n itself is not included in the result).
func (n *Node) Navigate(keys string) []*Node {
	return navigate(n.Children, keys)
}

func navigate(level []*Node, keys string) []*Node {
	var chain []*Node
	for _, ch := range strings.ToLower(keys) {
		best := bestMatch(level, ch)
		if best == nil {
			break
		}
		chain = append(chain, best)
		level = best.Children
	}
	return chain
}

// bestMatch picks the node whose Name matches r with the lowest score
// (earliest first occurrence, with a bonus when the character appears twice —
// the density heuristic from the original `f`). Ties go to the earlier node in
// the already-sorted slice. Returns nil if no name contains r.
func bestMatch(level []*Node, r rune) *Node {
	var best *Node
	bestScore := 1 << 30
	for _, n := range level {
		name := strings.ToLower(n.Name)
		idx := strings.IndexRune(name, r)
		if idx < 0 {
			continue
		}
		score := idx * 100
		if strings.IndexRune(name[idx+len(string(r)):], r) >= 0 {
			score -= 50 // appears at least twice: denser match
		}
		if score < bestScore {
			best, bestScore = n, score
		}
	}
	return best
}

// Breadcrumb renders a matched chain as "root ➜ name ➜ name" using display
// names, for the navigation UX the shell functions printed.
func Breadcrumb(rootLabel string, chain []*Node) string {
	parts := make([]string, 0, len(chain)+1)
	parts = append(parts, rootLabel)
	for _, n := range chain {
		parts = append(parts, n.Display())
	}
	return strings.Join(parts, " ➜ ")
}
