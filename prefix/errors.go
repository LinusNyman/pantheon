package prefix

import "errors"

// Typed sentinels (SPEC §11 — never (x, bool) returns). The tree scanner maps
// these onto its Issue taxonomy; ErrPrefixMismatch and ErrBadMetaName are
// returned *alongside* a best-effort entry so a forgiving scanner can keep
// descending with the claimed prefix.
var (
	// ErrEmpty: the name (or a required part of it) is empty.
	ErrEmpty = errors.New("prefix: empty name")
	// ErrDotfile: dotfiles are outside the grammar and are skipped silently.
	ErrDotfile = errors.New("prefix: dotfile")
	// ErrCharset: a rune outside [a-z0-9_] (åäö are tolerated when parsing).
	ErrCharset = errors.New("prefix: character outside [a-z0-9_]")
	// ErrMalformed: the structure does not follow the grammar at all.
	ErrMalformed = errors.New("prefix: name does not follow the pantheon grammar")
	// ErrPrefixMismatch: parses as a pantheon name, but its inherited part is
	// not the parent's full prefix. The returned entry is the best-effort
	// context-free reading.
	ErrPrefixMismatch = errors.New("prefix: inherited prefix does not match parent")
	// ErrBadMetaName: a *__ directory whose stem is not the parent's full
	// prefix. The returned entry carries the claimed stem.
	ErrBadMetaName = errors.New("prefix: meta dir stem does not match parent prefix")
	// ErrDuplicateDiscriminator: two siblings share a discriminator value.
	ErrDuplicateDiscriminator = errors.New("prefix: duplicate sibling discriminator")
	// ErrAlphabetExhausted: NextLetter found no free letter.
	ErrAlphabetExhausted = errors.New("prefix: all letters taken")
)
