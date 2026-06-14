package prefix

import "strings"

// NextLetter allocates a Letter discriminator for a new child named name
// (already sanitized), given its siblings' discriminator values (SPEC §6):
//
//  1. the first letter of the name
//  2. each subsequent consonant of the name, in order
//  3. each remaining rune of the name, in order
//  4. the first free letter of the alphabet (a–z, then 0–9)
//
// Returns ErrAlphabetExhausted when every letter is taken — the caller should
// fall back to a Word or Number discriminator.
func NextLetter(name string, taken []string, opts Opts) (string, error) {
	used := make(map[string]bool, len(taken))
	for _, t := range taken {
		used[t] = true
	}
	free := func(r rune) (string, bool) {
		s := string(r)
		return s, !used[s]
	}

	var runes []rune
	for _, r := range name {
		if isNameRune(r) && (opts.AllowSwedish || r < 'å') {
			runes = append(runes, r)
		}
	}
	if len(runes) == 0 {
		return "", ErrEmpty
	}

	if s, ok := free(runes[0]); ok {
		return s, nil
	}
	for _, r := range runes[1:] {
		if !isVowel(r) && r >= 'a' {
			if s, ok := free(r); ok {
				return s, nil
			}
		}
	}
	for _, r := range runes[1:] {
		if s, ok := free(r); ok {
			return s, nil
		}
	}
	alphabet := "abcdefghijklmnopqrstuvwxyz0123456789"
	if opts.AllowSwedish {
		alphabet += "åäö"
	}
	for _, r := range alphabet {
		if s, ok := free(r); ok {
			return s, nil
		}
	}
	return "", ErrAlphabetExhausted
}

func isVowel(r rune) bool {
	return strings.ContainsRune("aeiouyåäö", r)
}

// NextIndex allocates the next sequential index discriminator — a, b, …, z, aa,
// ab, …, zz, aaa, … — the enrollment-ordered counterpart to NextLetter's
// name-derived choice. Both produce a Letter discriminator; the two are
// indistinguishable on disk (SPEC §4). Use it when the value carries no meaning
// and only the order matters (e.g. children created in registration order).
//
// taken lists the discriminator values already in use among the siblings, of
// any kind, so the result never collides. Allocation is monotonic: it extends
// past the highest index already present rather than recycling a removed
// child's letter. The bijective base-26 sequence over a–z never exhausts, so —
// unlike NextLetter — NextIndex cannot fail.
func NextIndex(taken []string) string {
	used := make(map[string]bool, len(taken))
	maxVal := ""
	for _, t := range taken {
		used[t] = true
		if isIndex(t) && indexGT(t, maxVal) {
			maxVal = t
		}
	}
	next := "a"
	if maxVal != "" {
		next = incIndex(maxVal)
	}
	for used[next] {
		next = incIndex(next)
	}
	return next
}

// isIndex reports whether s is a non-empty run of a–z, i.e. a value the index
// sequence can compare and increment.
func isIndex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// indexGT reports whether a sorts after b in the bijective base-26 order:
// longer values always sort later; equal-length values compare lexically
// (a–z is already in order at each position).
func indexGT(a, b string) bool {
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	return a > b
}

// incIndex returns the successor in the bijective base-26 sequence:
// a→b, …, z→aa, az→ba, zz→aaa. The input is assumed to be a–z (see isIndex).
func incIndex(s string) string {
	r := []rune(s)
	i := len(r) - 1
	for i >= 0 && r[i] == 'z' {
		r[i] = 'a'
		i--
	}
	if i < 0 {
		return strings.Repeat("a", len(r)+1)
	}
	r[i]++
	return string(r)
}
