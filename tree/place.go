package tree

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNoNode is returned when a code or file prefix resolves to no node.
var ErrNoNode = errors.New("tree: no node for code")

// CodeOfFile extracts the working-prefix code a file basename carries: the
// leading token before the first underscore, or the whole stem when there is
// none. "mvy_cool_video.mp4" → "mvy"; "asb.md" → "asb". The code is the part
// before the first '.' and the first '_' (a working prefix never contains an
// underscore — SPEC §2).
func CodeOfFile(basename string) string {
	stem := basename
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	code, _, _ := strings.Cut(stem, "_")
	return code
}

// NodeForFile resolves the node a file belongs to from the code embedded in
// its basename (CodeOfFile). Returns ErrNoNode when the name is not conforming
// or no node has that code — sanitize/rename the file first, then place it.
func (t *Tree) NodeForFile(basename string) (*Node, error) {
	code := CodeOfFile(basename)
	if code == "" {
		return nil, fmt.Errorf("%w: %q has no prefix", ErrNoNode, basename)
	}
	n := t.Find(code)
	if n == nil {
		return nil, fmt.Errorf("%w: %q (code %q)", ErrNoNode, basename, code)
	}
	return n, nil
}
