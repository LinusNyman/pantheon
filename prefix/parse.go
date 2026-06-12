package prefix

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// ParseDir interprets a directory basename relative to its parent's full
// prefix ("" at the volume root). Context-aware parsing is what makes
// multi-char discriminators unambiguous (SPEC §2, §4).
//
//	depth 1:  "<disc>_<name>"             a_actio
//	depth ≥2: "<parent>_<disc>[_<name>]"  as_b_bibliotheca, assefqf_12
//	meta:     "<parent>__"                asb__
//
// On ErrPrefixMismatch and ErrBadMetaName the returned entry is a best-effort
// context-free reading (its FullPrefix is the name's *claimed* prefix), so a
// forgiving scanner can keep descending — e.g. into an old subtree that has
// not been renamed yet.
func ParseDir(basename, parentPrefix string) (DirEntry, error) {
	if basename == "" {
		return DirEntry{}, ErrEmpty
	}
	if strings.HasPrefix(basename, ".") {
		return DirEntry{}, ErrDotfile
	}
	if !validRunes(basename) {
		return DirEntry{}, ErrCharset
	}

	// Meta directory: exactly <parentPrefix>__.
	if strings.HasSuffix(basename, "__") {
		stem := basename[:len(basename)-2]
		if stem == "" || strings.Contains(stem, "_") {
			return DirEntry{}, ErrMalformed
		}
		e := DirEntry{Inherited: stem, Disc: Discriminator{Kind: Meta, Value: "__", Padded: "__"}}
		if parentPrefix != "" && stem == parentPrefix {
			return e, nil
		}
		return e, ErrBadMetaName
	}

	segs := strings.Split(basename, "_")
	if slices.Contains(segs, "") {
		return DirEntry{}, ErrMalformed // internal "__", leading or trailing "_"
	}

	// Depth 1: <disc>_<name>, name required (a_actio, never bare "a").
	if parentPrefix == "" {
		if len(segs) < 2 {
			return DirEntry{}, ErrMalformed
		}
		return DirEntry{
			Disc: classify(segs[0]),
			Name: strings.Join(segs[1:], "_"),
		}, nil
	}

	// Depth ≥2 with matching inherited prefix.
	if rest, ok := strings.CutPrefix(basename, parentPrefix+"_"); ok {
		rsegs := strings.Split(rest, "_")
		return DirEntry{
			Inherited: parentPrefix,
			Disc:      classify(rsegs[0]),
			Name:      strings.Join(rsegs[1:], "_"),
		}, nil
	}

	// Mismatch: best-effort context-free reading.
	if len(segs) >= 3 {
		return DirEntry{
			Inherited: segs[0],
			Disc:      classify(segs[1]),
			Name:      strings.Join(segs[2:], "_"),
		}, ErrPrefixMismatch
	}
	if len(segs) == 2 {
		return DirEntry{
			Disc: classify(segs[0]),
			Name: segs[1],
		}, ErrPrefixMismatch
	}
	return DirEntry{}, ErrMalformed // a single bare word can't claim any prefix
}

// ParseFile interprets a file basename relative to the working prefix of the
// directory it sits in (SPEC §3).
//
//	node's own file: "<prefix>.<ext>"               asb.md          (Descriptor "")
//	supplementary:   "<prefix>_<descriptor>.<ext>"  asb_todo.md
//	volume root:     "README[.md]" and "_<name>.*"  (documented exceptions)
//
// The extension is split at the *first* dot, so "asb_data.tar.gz" yields
// descriptor "data" and ext "tar.gz". (Sanitized names never contain dots.)
func ParseFile(basename, workingPrefix string) (FileEntry, error) {
	if basename == "" {
		return FileEntry{}, ErrEmpty
	}
	if strings.HasPrefix(basename, ".") {
		return FileEntry{}, ErrDotfile
	}

	stem, ext, _ := strings.Cut(basename, ".")
	if stem == "" {
		return FileEntry{}, ErrMalformed
	}

	// Volume root: README and underscore-led meta documents (SPEC §3).
	if workingPrefix == "" {
		if stem == "README" {
			return FileEntry{Descriptor: stem, Ext: ext}, nil
		}
		if rest, ok := strings.CutPrefix(stem, "_"); ok && rest != "" && validRunes(rest) {
			return FileEntry{Descriptor: rest, Ext: ext}, nil
		}
		return FileEntry{}, ErrPrefixMismatch
	}

	if !validRunes(stem) {
		return FileEntry{}, ErrCharset
	}
	if stem == workingPrefix {
		return FileEntry{Prefix: workingPrefix, Ext: ext}, nil
	}
	if desc, ok := strings.CutPrefix(stem, workingPrefix+"_"); ok {
		if slices.Contains(strings.Split(desc, "_"), "") {
			return FileEntry{}, ErrMalformed
		}
		return FileEntry{Prefix: workingPrefix, Descriptor: desc, Ext: ext}, nil
	}
	return FileEntry{}, ErrPrefixMismatch
}

// classify infers a Discriminator from a discriminator segment alone
// (SPEC §4): all-digits → Number, single rune → Letter, multi-char → Word.
// The segment is assumed non-empty and charset-checked by the caller.
func classify(s string) Discriminator {
	allDigits := true
	for _, r := range s {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return Discriminator{Kind: Number, Value: unpad(s), Padded: s}
	}
	if utf8.RuneCountInString(s) == 1 {
		return Discriminator{Kind: Letter, Value: s, Padded: s}
	}
	return Discriminator{Kind: Word, Value: s, Padded: s}
}

// unpad strips cosmetic zero-padding from a numeric segment ("03" → "3").
func unpad(s string) string {
	t := strings.TrimLeft(s, "0")
	if t == "" {
		return "0"
	}
	return t
}

// validRunes reports whether every rune is permitted in a name: ASCII
// lowercase letters, digits, underscore, and the Swedish letters åäö.
// Parsing tolerates åäö unconditionally (a forgiving scanner must read what
// is on disk); AllowSwedish gates creation only (Sanitize, NextLetter).
func validRunes(s string) bool {
	for _, r := range s {
		if !isNameRune(r) && r != '_' {
			return false
		}
	}
	return true
}

func isNameRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == 'å' || r == 'ä' || r == 'ö'
}
