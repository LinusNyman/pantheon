package prefix

import "strings"

// FormatDir renders a DirEntry back to a directory basename — the inverse of
// ParseDir. The discriminator's Padded form is used when set, so cosmetic
// number padding round-trips (SPEC §4).
func FormatDir(e DirEntry) string {
	if e.Disc.Kind == Meta {
		return e.Inherited + "__"
	}
	own := e.Disc.Padded
	if own == "" {
		own = e.Disc.Value
	}
	var b strings.Builder
	if e.Inherited != "" {
		b.WriteString(e.Inherited)
		b.WriteByte('_')
	}
	b.WriteString(own)
	if e.Name != "" {
		b.WriteByte('_')
		b.WriteString(e.Name)
	}
	return b.String()
}

// FormatFile renders a file basename from its parts — the inverse of
// ParseFile. An empty descriptor yields the node's own file (asb.md); a
// leading dot on ext is tolerated.
func FormatFile(workingPrefix, descriptor, ext string) string {
	s := workingPrefix
	if descriptor != "" {
		s += "_" + descriptor
	}
	if ext != "" {
		s += "." + strings.TrimPrefix(ext, ".")
	}
	return s
}
