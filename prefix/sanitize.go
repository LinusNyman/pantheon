package prefix

import (
	"strings"
	"unicode"
)

// Opts configures name creation. The zero value is the portable core:
// [a-z0-9_] only.
type Opts struct {
	// AllowSwedish admits åäö (SPEC §4; off by default — decision §12#4).
	AllowSwedish bool
}

// Sanitize turns free text into a valid on-disk name (SPEC §7). It is applied
// once at creation; the name is then frozen on disk. Characters with no
// mapping are stripped, not transliterated.
func Sanitize(raw string, opts Opts) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))

	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsSpace(r) || r == '-':
			b.WriteByte('_')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_':
			b.WriteRune(r)
		case opts.AllowSwedish && (r == 'å' || r == 'ä' || r == 'ö'):
			b.WriteRune(r)
		}
	}

	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if out == "" {
		return "", ErrEmpty
	}
	return out, nil
}

// SanitizeFilename sanitizes a file basename while preserving a single
// trailing extension, which is lowercased: "My Photo.JPG" → "my_photo.jpg".
// The stem is run through Sanitize (SPEC §7). It is the in-package version of
// the user's normalize_name shell helper. A leading dot (dotfile) is not
// treated as an extension separator. Returns ErrEmpty if the stem sanitizes
// away to nothing.
func SanitizeFilename(basename string, opts Opts) (string, error) {
	stem, ext := basename, ""
	if i := strings.LastIndex(basename, "."); i > 0 {
		stem, ext = basename[:i], basename[i+1:]
	}
	s, err := Sanitize(stem, opts)
	if err != nil {
		return "", err
	}
	if ext != "" {
		if e, err := Sanitize(ext, opts); err == nil && e != "" {
			return s + "." + e, nil
		}
	}
	return s, nil
}
