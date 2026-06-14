package tree

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LinusNyman/pantheon/prefix"
)

// Rename is one basename change: two absolute paths differing only in their
// final component.
type Rename struct {
	From string
	To   string
}

// RenamePlan is a computed, not-yet-applied cascading rename of a node and its
// whole subtree. Build it with PlanRename, inspect it (String / dry-run), then
// Apply it. This is the prefix-aware replacement for the old `renp` shell
// function: it understands that a descendant's code has the node's code as a
// string prefix, so changing the node's code ripples down deterministically.
type RenamePlan struct {
	Node    *Node
	OldCode string
	NewCode string
	Renames []Rename // ordered deepest-first (safe to apply one component at a time)
}

// Empty reports whether the plan changes nothing.
func (p *RenamePlan) Empty() bool { return len(p.Renames) == 0 }

// PlanRename computes the renames needed to give node a new inherited prefix,
// discriminator, and/or descriptive name — cascading the resulting code change
// to every descendant directory and file, and to the node's own files.
//
//   - keep the discriminator: pass node.Disc
//   - keep the name:          pass node.Name
//   - keep the inherited:     pass node's current inherited (CurrentInherited)
//   - fix a PrefixMismatch:   pass the real parent's code as newInherited (reroot)
//
// It validates the new discriminator against the node's on-disk siblings and
// pre-checks that no target path already exists (no silent overwrite, SPEC
// §11.4). It does not touch the disk.
func (t *Tree) PlanRename(node *Node, newInherited string, newDisc prefix.Discriminator, newName string) (*RenamePlan, error) {
	if node == nil {
		return nil, fmt.Errorf("tree: PlanRename: nil node")
	}
	oldCode := node.Code
	newCode := newInherited + newDisc.Value

	plan := &RenamePlan{Node: node, OldCode: oldCode, NewCode: newCode}

	// Sibling-uniqueness: the new discriminator must not collide with a
	// sibling (excluding the node itself).
	for _, sib := range t.siblingsOf(node) {
		if sib == node {
			continue
		}
		if sib.Disc.Value == newDisc.Value && newDisc.Kind != prefix.Meta {
			return nil, fmt.Errorf("%w: sibling %s already uses %q", prefix.ErrDuplicateDiscriminator, sib.Path, newDisc.Value)
		}
	}

	// The node directory itself is renamed via the grammar (its basename does
	// not start with the code — it is inherited_own_name).
	parentDir := filepath.Dir(node.Path)
	newNodeBase := prefix.FormatDir(prefix.DirEntry{Inherited: newInherited, Disc: newDisc, Name: newName})
	if newNodeBase != filepath.Base(node.Path) {
		plan.Renames = append(plan.Renames, Rename{From: node.Path, To: filepath.Join(parentDir, newNodeBase)})
	}

	// Every descendant basename that begins with the old code gets that
	// leading code swapped for the new code. Only basenames change, so walking
	// deepest-first and renaming one component at a time keeps every `From`
	// valid (ancestors are still at their original names when a child is moved).
	if newCode != oldCode {
		walkRoot := node.Path
		err := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // forgiving: skip unreadable entries
			}
			if path == walkRoot {
				return nil // the node dir is handled above
			}
			base := d.Name()
			if d.IsDir() && (base == ".git" || base == "node_modules") {
				return filepath.SkipDir
			}
			if strings.HasPrefix(base, oldCode) {
				newBase := newCode + base[len(oldCode):]
				plan.Renames = append(plan.Renames, Rename{From: path, To: filepath.Join(filepath.Dir(path), newBase)})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Deepest-first: rename children before the directories that contain them.
	sort.SliceStable(plan.Renames, func(i, j int) bool {
		return strings.Count(plan.Renames[i].From, string(os.PathSeparator)) >
			strings.Count(plan.Renames[j].From, string(os.PathSeparator))
	})

	if err := plan.checkCollisions(); err != nil {
		return nil, err
	}
	return plan, nil
}

// checkCollisions rejects the plan if any target already exists on disk and is
// not itself a source being vacated, or if two renames target the same path.
func (p *RenamePlan) checkCollisions() error {
	sources := make(map[string]bool, len(p.Renames))
	for _, r := range p.Renames {
		sources[r.From] = true
	}
	targets := make(map[string]bool, len(p.Renames))
	for _, r := range p.Renames {
		if targets[r.To] {
			return fmt.Errorf("tree: rename plan maps two entries onto %s", r.To)
		}
		targets[r.To] = true
		if _, err := os.Lstat(r.To); err == nil && !sources[r.To] {
			return fmt.Errorf("tree: rename target already exists: %s (refusing to overwrite)", r.To)
		}
	}
	return nil
}

// Apply executes the plan deepest-first. On any failure it rolls back the
// renames already performed (reverse order) and returns the original error, so
// a partial cascade never leaves the tree half-renamed (the transactional
// property; recovery does not depend on git).
func (p *RenamePlan) Apply() error {
	var done []Rename
	for _, r := range p.Renames {
		if err := os.Rename(r.From, r.To); err != nil {
			for i := len(done) - 1; i >= 0; i-- {
				_ = os.Rename(done[i].To, done[i].From)
			}
			return fmt.Errorf("tree: applying rename %s → %s: %w (rolled back %d)", r.From, r.To, err, len(done))
		}
		done = append(done, r)
	}
	return nil
}

// String renders the plan for a dry-run: the code change plus each basename
// rename, deepest-first.
func (p *RenamePlan) String() string {
	if p.Empty() {
		return "(nothing to rename)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "rename %s → %s  (%d path(s))\n", p.OldCode, p.NewCode, len(p.Renames))
	for _, r := range p.Renames {
		fmt.Fprintf(&b, "  %s → %s\n", filepath.Base(r.From), filepath.Base(r.To))
	}
	return strings.TrimRight(b.String(), "\n")
}

// CurrentInherited returns a node's inherited prefix (its code minus its own
// discriminator value) — the value to pass unchanged to PlanRename when only
// the discriminator or name is changing.
func (n *Node) CurrentInherited() string {
	return strings.TrimSuffix(n.Code, n.Disc.Value)
}

// siblingsOf returns the nodes at the same level as node.
func (t *Tree) siblingsOf(node *Node) []*Node {
	if node.Parent != nil {
		return node.Parent.Children
	}
	return t.Roots
}
