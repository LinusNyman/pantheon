package tree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteFile writes data to path atomically: temp file in the same directory,
// fsync, rename (SPEC §11.2). Every content write in the suite must go
// through here — never raw os.WriteFile over user content.
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pantheon-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	// Best-effort: persist the rename itself.
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}

// Uniquify returns path if nothing exists there, otherwise the first free
// "<stem>_2.<ext>", "<stem>_3.<ext>", … variant — the one conflict resolver
// (SPEC §11.4): no code path may silently overwrite.
func Uniquify(path string) string {
	if _, err := os.Lstat(path); err != nil {
		return path
	}
	dir, base := filepath.Dir(path), filepath.Base(path)
	stem, ext, hasExt := strings.Cut(base, ".")
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s_%d", stem, n)
		if hasExt {
			candidate += "." + ext
		}
		candidate = filepath.Join(dir, candidate)
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}
