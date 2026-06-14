package tree

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LinusNyman/pantheon/prefix"
)

// CreateChild creates one conforming child directory under the node with code
// parentCode (or directly under the volume root when parentCode is ""), and
// returns the new node wired into the in-memory tree.
//
// It is the single node-creation primitive for the whole suite: `pan mk` and
// every tool (pensum's `pen new`, …) go through it instead of re-deriving the
// FormatDir + sibling-uniqueness + mkdir sequence. It:
//
//   - resolves the parent (ErrNoNode if parentCode names nothing),
//   - validates the new discriminator against existing siblings (SPEC §8;
//     ErrDuplicateDiscriminator on a clash),
//   - formats the directory name from the grammar (prefix.FormatDir),
//   - refuses to overwrite an existing path,
//   - creates the directory and, when meta is set, its <code>__ meta dir,
//   - registers the node so Find and the parent's Children see it immediately.
func (t *Tree) CreateChild(parentCode string, disc prefix.Discriminator, name string, meta bool) (*Node, error) {
	if disc.Value == "" || disc.Kind == prefix.Meta {
		return nil, fmt.Errorf("tree: CreateChild needs a non-empty, non-meta discriminator")
	}

	var (
		parent     *Node
		parentPath string
		inherited  string
		siblings   []*Node
		depth      int
	)
	if parentCode == "" {
		parentPath = t.RootPath
		siblings = t.Roots
		depth = 1
	} else {
		parent = t.Find(parentCode)
		if parent == nil {
			return nil, fmt.Errorf("%w: %q", ErrNoNode, parentCode)
		}
		parentPath = parent.Path
		inherited = parent.Code
		siblings = parent.Children
		depth = parent.Depth + 1
	}

	// Sibling-uniqueness against conforming siblings (SPEC §8). Mismatched
	// siblings sit in a different prefix space; a literal name clash is caught
	// by the on-disk existence check below regardless.
	discs := make([]prefix.Discriminator, 0, len(siblings)+1)
	for _, s := range siblings {
		if !s.Mismatched {
			discs = append(discs, s.Disc)
		}
	}
	discs = append(discs, disc)
	if err := prefix.ValidateSiblings(discs); err != nil {
		return nil, err
	}

	entry := prefix.DirEntry{Inherited: inherited, Disc: disc, Name: name}
	path := filepath.Join(parentPath, prefix.FormatDir(entry))
	if _, err := os.Lstat(path); err == nil {
		return nil, fmt.Errorf("tree: %s already exists (refusing to overwrite)", path)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return nil, err
	}
	if meta {
		if err := os.Mkdir(filepath.Join(path, entry.FullPrefix()+"__"), 0o755); err != nil {
			return nil, err
		}
	}

	node := &Node{
		Code:    entry.FullPrefix(),
		Name:    name,
		Path:    path,
		Disc:    disc,
		Depth:   depth,
		Parent:  parent,
		HasMeta: meta,
	}
	if parent == nil {
		t.Roots = append(t.Roots, node)
		sortNodes(t.Roots)
	} else {
		parent.Children = append(parent.Children, node)
		sortNodes(parent.Children)
	}
	t.byCode[node.Code] = node
	return node, nil
}
