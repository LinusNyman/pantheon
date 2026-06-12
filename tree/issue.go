package tree

import "fmt"

// IssueKind classifies a grammar deviation found by the scanner (SPEC §9).
type IssueKind int

const (
	// Orphan: a basename that doesn't parse under its parent's prefix at all
	// (also used for directories the scanner could not read).
	Orphan IssueKind = iota
	// PrefixMismatch: parses, but the inherited part is not the parent's full
	// prefix. The node is still scanned under its claimed prefix.
	PrefixMismatch
	// DuplicateDiscriminator: two nodes resolve to the same code; the first
	// keeps the code in Find.
	DuplicateDiscriminator
	// MixedScheme: a sibling deviates from the majority discriminator kind of
	// its parent — the convention-only deviation detector (SPEC §4).
	MixedScheme
	// BadMetaName: a *__ directory whose stem is not its parent's full prefix.
	BadMetaName
	// StrayFile: a file whose prefix part is not the working prefix of the
	// directory it sits in (only reported when ScanOpts.ScanFiles is set).
	StrayFile
)

func (k IssueKind) String() string {
	switch k {
	case Orphan:
		return "Orphan"
	case PrefixMismatch:
		return "PrefixMismatch"
	case DuplicateDiscriminator:
		return "DuplicateDiscriminator"
	case MixedScheme:
		return "MixedScheme"
	case BadMetaName:
		return "BadMetaName"
	case StrayFile:
		return "StrayFile"
	default:
		return "Unknown"
	}
}

// Issue is one deviation, attached to the scan result — disk content is never
// a fatal error (SPEC §11.5).
type Issue struct {
	Kind IssueKind
	Path string // absolute path of the offending entry
	Msg  string // one human/LLM-readable line
}

func (i Issue) String() string {
	return fmt.Sprintf("%s: %s — %s", i.Kind, i.Path, i.Msg)
}
