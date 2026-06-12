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
