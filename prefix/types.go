// Package prefix implements the pantheon prefix grammar: parsing, formatting,
// sanitizing, and discriminator allocation for directory and file names.
//
// All functions are pure (no I/O); the filesystem boundary lives in package
// tree. See SPEC.md for the authoritative grammar.
//
// Parsing is forgiving about alphabet (it reads åäö from disk regardless of
// configuration); the AllowSwedish restriction is enforced only where names
// are created: Sanitize and NextLetter.
package prefix

// Kind classifies a discriminator by its parse-time structure (SPEC §4):
// all-digits → Number, single rune → Letter, multi-char otherwise → Word.
type Kind int

const (
	// Letter is a single [a-z0-9åäö] character — whether chosen as the
	// child's characteristic letter or allocated as a sequential index;
	// the two are indistinguishable on disk.
	Letter Kind = iota
	// Number is an integer. The directory name may be cosmetically
	// zero-padded; the working-prefix value is always unpadded (SPEC §4).
	Number
	// Word is a full multi-character word.
	Word
	// Meta is the "__" directory; it does NOT extend the working prefix.
	Meta
)

func (k Kind) String() string {
	switch k {
	case Letter:
		return "Letter"
	case Number:
		return "Number"
	case Word:
		return "Word"
	case Meta:
		return "Meta"
	default:
		return "Unknown"
	}
}

// Discriminator is the segment of a directory name between the inherited
// prefix and the optional descriptive name.
type Discriminator struct {
	Kind   Kind
	Value  string // canonical, unpadded form used in the working prefix ("b", "12", "aar", "__")
	Padded string // form written in the directory name ("03" when padded), otherwise == Value
}

// DirEntry is a parsed directory basename, interpreted relative to the parent
// directory's full prefix (SPEC §2).
type DirEntry struct {
	Inherited string // parent's full prefix; "" at depth 1
	Disc      Discriminator
	Name      string // descriptive name; may be "" at depth ≥2 (assefqf_12)
}

// FullPrefix returns the working prefix this directory's children and files
// inherit: Inherited + Disc.Value — except for the meta directory, which
// keeps the parent's prefix unchanged (SPEC §5).
func (e DirEntry) FullPrefix() string {
	if e.Disc.Kind == Meta {
		return e.Inherited
	}
	return e.Inherited + e.Disc.Value
}

// FileEntry is a parsed file basename, interpreted relative to the working
// prefix of the directory it sits in (SPEC §3).
type FileEntry struct {
	Prefix     string // the working prefix carried by the file
	Descriptor string // "" for the node's own file (asb.md)
	Ext        string // without the leading dot; "" when extensionless; may itself contain dots ("tar.gz")
}
