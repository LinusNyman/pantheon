// Package ontology loads a user's life-domain table — the map that gives the
// prefix codes their meaning.
//
// The table is personal data: it lives in the user's own volume (by
// convention <root>/_ontology.tsv, see DefaultPath), never in this module.
// It is reference data, not law: codes are *suggested* filesystem prefixes
// (SPEC §10); the scanner never validates a disk tree against them.
//
// File format — tab-separated, "#" comments and blank lines ignored:
//
//	code  parent  latin  greek  symbol  [deity]
//
// where parent is empty for top-level domains, and rows come parents-first.
package ontology

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Node is one life domain.
type Node struct {
	Code   string // suggested filesystem prefix ("csng")
	Latin  string
	Greek  string
	Symbol string
	Deity  string // verbatim; "" when none

	Parent   *Node
	Children []*Node // document order
}

// Table is one loaded ontology.
type Table struct {
	roots  []*Node
	all    []*Node
	byCode map[string]*Node
}

// DefaultPath returns the conventional location of the table inside a volume:
// <root>/_ontology.tsv — an underscore-led volume meta document (SPEC §3).
func DefaultPath(root string) string {
	return filepath.Join(root, "_ontology.tsv")
}

// Load reads a table from a TSV file. A missing file surfaces as an
// fs.ErrNotExist-wrapped error so callers can offer guidance.
func Load(path string) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return t, nil
}

// Parse reads a table from TSV content.
func Parse(r io.Reader) (*Table, error) {
	t := &Table{byCode: make(map[string]*Node)}
	sc := bufio.NewScanner(r)
	for lineNo := 1; sc.Scan(); lineNo++ {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 5 || len(f) > 6 { // the deity column is optional
			return nil, fmt.Errorf("ontology: line %d: %d fields, want 5–6 (code parent latin greek symbol [deity])", lineNo, len(f))
		}
		n := &Node{Code: f[0], Latin: f[2], Greek: f[3], Symbol: f[4]}
		if len(f) == 6 {
			n.Deity = f[5]
		}
		if n.Code == "" {
			return nil, fmt.Errorf("ontology: line %d: empty code", lineNo)
		}
		if t.byCode[n.Code] != nil {
			return nil, fmt.Errorf("ontology: line %d: duplicate code %q", lineNo, n.Code)
		}
		t.byCode[n.Code] = n
		t.all = append(t.all, n)
		if f[1] == "" {
			t.roots = append(t.roots, n)
			continue
		}
		parent := t.byCode[f[1]]
		if parent == nil {
			return nil, fmt.Errorf("ontology: line %d: %q references unknown parent %q (parents must come first)", lineNo, n.Code, f[1])
		}
		n.Parent = parent
		parent.Children = append(parent.Children, n)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return t, nil
}

// Roots returns the top-level domains in document order.
func (t *Table) Roots() []*Node { return t.roots }

// Find resolves a suggested code to its domain, or nil.
func (t *Table) Find(code string) *Node { return t.byCode[code] }

// All returns every domain in document order (parents before children).
func (t *Table) All() []*Node { return t.all }

// Lineage returns the path from the top-level domain down to n, inclusive.
func (n *Node) Lineage() []*Node {
	var path []*Node
	for c := n; c != nil; c = c.Parent {
		path = append(path, c)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
