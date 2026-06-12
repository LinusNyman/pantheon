package tree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/LinusNyman/pantheon/prefix"
)

// ScanOpts configures Scan. The zero value scans every directory to unlimited
// depth, stops at git-repo boundaries, and skips file checking.
type ScanOpts struct {
	// ScanFiles also parses every file basename and reports StrayFile issues
	// (slower; used by pan doctor). Node files are never stored on the tree —
	// apps address them through AppFile and their own conventions.
	ScanFiles bool
	// MaxDepth limits how deep nodes are created (0 = unlimited).
	MaxDepth int
	// SkipDirs are directory basenames skipped entirely, without an Issue
	// (e.g. "node_modules"). Dotfiles are always skipped silently.
	SkipDirs []string
	// DescendRepos scans inside directories containing .git. Off by default:
	// a repo's interior follows its ecosystem's conventions, not the grammar.
	DescendRepos bool
}

// Tree is the result of one Scan.
type Tree struct {
	RootPath string
	Roots    []*Node // depth-1 nodes, sorted
	Issues   []Issue
	byCode   map[string]*Node
}

// Scan reads the volume at rootPath into a Tree. The only fatal error is
// failing to read rootPath itself; everything found *inside* is either a Node
// or an Issue (SPEC §11.5).
func Scan(rootPath string, opts ScanOpts) (*Tree, error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("tree: reading root: %w", err)
	}
	t := &Tree{RootPath: abs, byCode: make(map[string]*Node)}
	t.Roots = t.scanLevel(abs, "", nil, entries, opts, 1)
	return t, nil
}

// Find resolves a full prefix to its node ("asb" → bibliotheca), or nil.
// When duplicate codes exist on disk the first node in scan order wins (a
// DuplicateDiscriminator issue names the loser).
func (t *Tree) Find(code string) *Node { return t.byCode[code] }

// Walk visits every node depth-first in sorted order.
func (t *Tree) Walk(fn func(*Node)) {
	for _, r := range t.Roots {
		r.Walk(fn)
	}
}

func (t *Tree) issue(kind IssueKind, path, msg string) {
	t.Issues = append(t.Issues, Issue{Kind: kind, Path: path, Msg: msg})
}

// scanLevel parses one directory's entries into child nodes of parent (nil at
// the volume root, where parentCode is ""), records issues, recurses, and
// returns the sorted children.
func (t *Tree) scanLevel(dirPath string, parentCode string, parent *Node, entries []os.DirEntry, opts ScanOpts, depth int) []*Node {
	var kids []*Node
	for _, ent := range entries {
		name := ent.Name()
		path := filepath.Join(dirPath, name)

		if !ent.IsDir() {
			if opts.ScanFiles {
				t.checkFile(name, path, parentCode)
			}
			continue
		}
		if slices.Contains(opts.SkipDirs, name) {
			continue
		}

		e, err := prefix.ParseDir(name, parentCode)
		var mismatched bool
		switch {
		case err == nil && e.Disc.Kind == prefix.Meta:
			if parent != nil {
				parent.HasMeta = true
			}
			if opts.ScanFiles {
				t.checkMetaFiles(path, parentCode)
			}
			continue
		case err == nil:
		case errors.Is(err, prefix.ErrDotfile):
			continue
		case errors.Is(err, prefix.ErrPrefixMismatch):
			t.issue(PrefixMismatch, path, fmt.Sprintf("claims prefix %q under parent prefix %q", e.FullPrefix(), parentCode))
			mismatched = true
		case errors.Is(err, prefix.ErrBadMetaName):
			t.issue(BadMetaName, path, fmt.Sprintf("meta dir stem %q is not the parent prefix %q", e.Inherited, parentCode))
			continue
		default: // ErrMalformed, ErrCharset, ErrEmpty
			t.issue(Orphan, path, err.Error())
			continue
		}

		node := &Node{
			Code:       e.FullPrefix(),
			Name:       e.Name,
			Path:       path,
			Disc:       e.Disc,
			Depth:      depth,
			Parent:     parent,
			Mismatched: mismatched,
		}
		kids = append(kids, node)
	}

	t.checkMixedScheme(kids)
	sortNodes(kids)

	for _, n := range kids {
		if first, dup := t.byCode[n.Code]; dup {
			t.issue(DuplicateDiscriminator, n.Path, fmt.Sprintf("code %q already used by %s", n.Code, first.Path))
		} else {
			t.byCode[n.Code] = n
		}

		children, err := os.ReadDir(n.Path)
		if err != nil {
			t.issue(Orphan, n.Path, "unreadable directory: "+err.Error())
			continue
		}
		if slices.ContainsFunc(children, func(e os.DirEntry) bool { return e.Name() == ".git" }) {
			n.IsRepo = true
			if !opts.DescendRepos {
				// A repo's interior is ecosystem territory; but still note a
				// conforming meta dir (repos keep <code>__/ at their top level).
				if i, err := os.Stat(filepath.Join(n.Path, n.Code+"__")); err == nil && i.IsDir() {
					n.HasMeta = true
				}
				continue
			}
		}
		if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
			continue
		}
		n.Children = t.scanLevel(n.Path, n.Code, n, children, opts, depth+1)
	}
	return kids
}

// checkFile reports a StrayFile when a basename doesn't carry the directory's
// working prefix. At the volume root, README and _<name>.* are exceptions
// (SPEC §3).
func (t *Tree) checkFile(name, path, workingPrefix string) {
	if _, err := prefix.ParseFile(name, workingPrefix); err != nil && !errors.Is(err, prefix.ErrDotfile) {
		t.issue(StrayFile, path, fmt.Sprintf("does not carry working prefix %q: %v", workingPrefix, err))
	}
}

// checkMetaFiles checks the files inside a meta dir, whose working prefix is
// the parent's full prefix unchanged (SPEC §5).
func (t *Tree) checkMetaFiles(metaPath, workingPrefix string) {
	entries, err := os.ReadDir(metaPath)
	if err != nil {
		t.issue(Orphan, metaPath, "unreadable directory: "+err.Error())
		return
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		t.checkFile(ent.Name(), filepath.Join(metaPath, ent.Name()), workingPrefix)
	}
}

// checkMixedScheme implements the convention-only deviation detector
// (SPEC §4): infer the majority discriminator kind among conforming siblings
// and flag the minority. Mismatched nodes sit in a different prefix space and
// are excluded; ties flag nothing.
func (t *Tree) checkMixedScheme(kids []*Node) {
	counts := map[prefix.Kind]int{}
	conforming := 0
	for _, n := range kids {
		if n.Mismatched {
			continue
		}
		counts[n.Disc.Kind]++
		conforming++
	}
	if len(counts) < 2 {
		return
	}
	var majority prefix.Kind
	best := 0
	for k, c := range counts {
		if c > best {
			majority, best = k, c
		}
	}
	if best*2 <= conforming { // no strict majority — ambiguous, stay quiet
		return
	}
	for _, n := range kids {
		if !n.Mismatched && n.Disc.Kind != majority {
			t.issue(MixedScheme, n.Path, fmt.Sprintf("discriminator kind %s among %s siblings", n.Disc.Kind, majority))
		}
	}
}

// sortNodes orders siblings by discriminator (numbers numerically, otherwise
// by value), then by name — the deterministic traversal of SPEC "Sorting".
func sortNodes(ns []*Node) {
	slices.SortFunc(ns, func(a, b *Node) int {
		if a.Disc.Kind == prefix.Number && b.Disc.Kind == prefix.Number {
			x, _ := strconv.Atoi(a.Disc.Value)
			y, _ := strconv.Atoi(b.Disc.Value)
			if x != y {
				return x - y
			}
		}
		if c := strings.Compare(a.Disc.Value, b.Disc.Value); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
}
