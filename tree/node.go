// Package tree is the filesystem boundary of the pantheon module: it scans a
// volume root into an in-memory tree of nodes, resolves codes to paths,
// reports grammar deviations as Issues (never errors — the forgiving scanner,
// SPEC §9), and provides the atomic write primitives every consumer must use
// (SPEC §11).
package tree

import (
	"os"
	"path/filepath"

	"github.com/LinusNyman/pantheon/prefix"
)

// Node is one directory in the pantheon tree.
type Node struct {
	Code  string // full/working prefix ("asb"); for Mismatched nodes, the *claimed* prefix
	Name  string // descriptive name ("bibliotheca"); may be "" (assefqf_12)
	Path  string // absolute path on disk
	Disc  prefix.Discriminator
	Depth int // 1 at the volume root's children

	Parent   *Node
	Children []*Node // sorted: numbers numerically, otherwise by value, then name

	// HasMeta reports a conforming <code>__/ directory on disk.
	HasMeta bool
	// Mismatched marks a node whose inherited prefix does not match its
	// parent (a PrefixMismatch issue was recorded); its claimed prefix is
	// adopted as Code so the subtree stays addressable mid-restructure.
	Mismatched bool
	// IsRepo marks a directory containing .git; the scanner does not descend
	// into repos (their interior follows ecosystem conventions, not the
	// grammar) unless ScanOpts.DescendRepos is set.
	IsRepo bool
}

// MetaDir returns the path of the node's meta directory <path>/<code>__,
// whether or not it exists on disk.
func (n *Node) MetaDir() string {
	return filepath.Join(n.Path, n.Code+"__")
}

// EnsureMetaDir creates the meta directory if missing and returns its path.
func (n *Node) EnsureMetaDir() (string, error) {
	dir := n.MetaDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	n.HasMeta = true
	return dir, nil
}

// AppFile returns the conventional path of a tool's per-node file inside the
// meta dir: <path>/<code>__/<code>_<descriptor>.<ext> (SPEC §5). It does not
// create anything.
func (n *Node) AppFile(descriptor, ext string) string {
	return filepath.Join(n.MetaDir(), prefix.FormatFile(n.Code, descriptor, ext))
}

// Walk visits n and all descendants depth-first in sorted order — which, per
// SPEC ("Sorting" §4), is the same order as sorting all full prefixes.
func (n *Node) Walk(fn func(*Node)) {
	fn(n)
	for _, c := range n.Children {
		c.Walk(fn)
	}
}
