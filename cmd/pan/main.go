// pan is the pantheon spine CLI: inspect the tree, resolve codes, detect
// deviations, create conforming nodes, and consult the ontology. Output is
// plain, stable, and grep-friendly; --json where a tool might consume it.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LinusNyman/pantheon/ontology"
	"github.com/LinusNyman/pantheon/prefix"
	"github.com/LinusNyman/pantheon/tree"
)

const version = "0.2.0"

const usage = `pan — the pantheon spine

Usage:
  pan tree [code]                      render the (sub)tree
  pan cd <keys>                        path of the node reached by a typeahead jump
  pan resolve <code|path>              code → path, or path → code
  pan doctor [code] [--json] [--files] list grammar deviations (exit 1 if any)
  pan mk <parent-code> <name>          create a conforming child dir
         [--kind letter|number|word] [--disc x] [--range A-B] [--meta] [--swedish]
  pan mv <code>                        rename a node, cascading to its whole subtree
         [--disc x] [--name n] [--reroot] [--dry-run]
  pan onto [code]                      the ontology table / one domain's lineage
  pan version

Options (every command):
  --root <path>   volume root (default: $PAN_ROOT → $PANTHEON_ROOT → ~/vol_f)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "tree":
		err = cmdTree(args)
	case "cd":
		err = cmdCd(args)
	case "resolve":
		err = cmdResolve(args)
	case "doctor":
		err = cmdDoctor(args)
	case "mk":
		err = cmdMk(args)
	case "mv":
		err = cmdMv(args)
	case "onto":
		err = cmdOnto(args)
	case "version":
		fmt.Println("pan", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "pan: unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "pan:", err)
		os.Exit(1)
	}
}

// newFlags returns a FlagSet with the shared --root flag pre-registered.
func newFlags(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	root := fs.String("root", "", "volume root")
	return fs, root
}

// parseFlexible parses args allowing flags before, between, or after
// positional arguments (stdlib flag stops at the first positional, which
// would silently ignore a trailing --root — unacceptable for commands that
// write to the tree). Returns the positional arguments in order.
func parseFlexible(fs *flag.FlagSet, args []string) []string {
	var pos []string
	for {
		fs.Parse(args) // ExitOnError: exits on bad flags
		rest := fs.Args()
		if len(rest) == 0 {
			return pos
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// skipDirs are ecosystem directories pan never looks inside.
var skipDirs = []string{"node_modules"}

func argN(pos []string, n int) string {
	if n < len(pos) {
		return pos[n]
	}
	return ""
}

func resolveRoot(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	return tree.Root("PAN_ROOT", "")
}

func scan(root string, opts tree.ScanOpts) (*tree.Tree, error) {
	t, err := tree.Scan(root, opts)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// mustFind resolves a code or exits with the conventional message.
func mustFind(t *tree.Tree, code string) (*tree.Node, error) {
	n := t.Find(code)
	if n == nil {
		return nil, fmt.Errorf("no node with code %q under %s", code, t.RootPath)
	}
	return n, nil
}

func cmdTree(args []string) error {
	fs, root := newFlags("tree")
	pos := parseFlexible(fs, args)
	t, err := scan(resolveRoot(*root), tree.ScanOpts{SkipDirs: skipDirs})
	if err != nil {
		return err
	}

	nodes := t.Roots
	base := 0
	if code := argN(pos, 0); code != "" {
		n, err := mustFind(t, code)
		if err != nil {
			return err
		}
		nodes = []*tree.Node{n}
		base = n.Depth - 1
	}
	for _, r := range nodes {
		r.Walk(func(n *tree.Node) {
			var marks []string
			if n.HasMeta {
				marks = append(marks, "meta")
			}
			if n.IsRepo {
				marks = append(marks, "repo")
			}
			if n.Mismatched {
				marks = append(marks, "MISMATCH")
			}
			line := fmt.Sprintf("%s%-12s %s", strings.Repeat("  ", n.Depth-1-base), n.Code, n.Name)
			if len(marks) > 0 {
				line += "  [" + strings.Join(marks, ",") + "]"
			}
			fmt.Println(strings.TrimRight(line, " "))
		})
	}
	return nil
}

func cmdResolve(args []string) error {
	fs, root := newFlags("resolve")
	pos := parseFlexible(fs, args)
	arg := argN(pos, 0)
	if arg == "" {
		return fmt.Errorf("usage: pan resolve <code|path>")
	}
	t, err := scan(resolveRoot(*root), tree.ScanOpts{})
	if err != nil {
		return err
	}

	// Path → code when the argument points at an existing directory.
	if strings.ContainsRune(arg, os.PathSeparator) || dirExists(arg) {
		abs, err := filepath.Abs(arg)
		if err != nil {
			return err
		}
		var found *tree.Node
		t.Walk(func(n *tree.Node) {
			if n.Path == abs {
				found = n
			}
		})
		if found == nil {
			return fmt.Errorf("%s is not a node under %s", abs, t.RootPath)
		}
		fmt.Println(found.Code)
		return nil
	}

	n, err := mustFind(t, arg)
	if err != nil {
		return err
	}
	fmt.Println(n.Path)
	return nil
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func cmdDoctor(args []string) error {
	fs, root := newFlags("doctor")
	asJSON := fs.Bool("json", false, "machine-readable output")
	files := fs.Bool("files", false, "also check file names (StrayFile; noisy in project dirs)")
	pos := parseFlexible(fs, args)
	t, err := scan(resolveRoot(*root), tree.ScanOpts{ScanFiles: *files, SkipDirs: skipDirs})
	if err != nil {
		return err
	}

	issues := t.Issues
	if code := argN(pos, 0); code != "" {
		n, err := mustFind(t, code)
		if err != nil {
			return err
		}
		var filtered []tree.Issue
		for _, is := range issues {
			if is.Path == n.Path || strings.HasPrefix(is.Path, n.Path+string(os.PathSeparator)) {
				filtered = append(filtered, is)
			}
		}
		issues = filtered
	}

	if *asJSON {
		type jsonIssue struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
			Msg  string `json:"msg"`
		}
		out := make([]jsonIssue, 0, len(issues))
		for _, is := range issues {
			out = append(out, jsonIssue{is.Kind.String(), is.Path, is.Msg})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
	} else {
		byKind := map[tree.IssueKind][]tree.Issue{}
		for _, is := range issues {
			byKind[is.Kind] = append(byKind[is.Kind], is)
		}
		for kind := tree.Orphan; kind <= tree.StrayFile; kind++ {
			group := byKind[kind]
			if len(group) == 0 {
				continue
			}
			fmt.Printf("%s (%d)\n", kind, len(group))
			for _, is := range group {
				fmt.Printf("  %s — %s\n", is.Path, is.Msg)
			}
		}
		fmt.Printf("%d issue(s)\n", len(issues))
	}
	if len(issues) > 0 {
		os.Exit(1)
	}
	return nil
}

func cmdMk(args []string) error {
	fs, root := newFlags("mk")
	kind := fs.String("kind", "letter", "discriminator kind: letter|number|word")
	disc := fs.String("disc", "", "explicit discriminator value")
	rangeFlag := fs.String("range", "", "create numbered children A-B (e.g. 1-12); name optional")
	meta := fs.Bool("meta", false, "also create the <code>__ meta dir")
	swedish := fs.Bool("swedish", false, "allow åäö in names")
	pos := parseFlexible(fs, args)
	opts := prefix.Opts{AllowSwedish: *swedish}

	parentCode := argN(pos, 0)
	if parentCode == "" {
		return fmt.Errorf("usage: pan mk <parent-code> <name>  (or --range A-B with optional name)")
	}

	t, err := scan(resolveRoot(*root), tree.ScanOpts{})
	if err != nil {
		return err
	}
	parent, err := mustFind(t, parentCode)
	if err != nil {
		return err
	}

	var discs []prefix.Discriminator // existing siblings, for uniqueness
	for _, c := range parent.Children {
		if !c.Mismatched {
			discs = append(discs, c.Disc)
		}
	}

	// --range: bulk numbered children (absorbs the old `mudir` shell function).
	// A descriptive name is optional here (nameless lectures: assc_1, assc_2).
	if *rangeFlag != "" {
		lo, hi, err := parseRange(*rangeFlag)
		if err != nil {
			return err
		}
		name := ""
		if raw := argN(pos, 1); raw != "" {
			if name, err = prefix.Sanitize(raw, opts); err != nil {
				return fmt.Errorf("name %q: %w", raw, err)
			}
		}
		for n := lo; n <= hi; n++ {
			v := strconv.Itoa(n)
			d := prefix.Discriminator{Kind: prefix.Number, Value: v, Padded: v}
			if err := prefix.ValidateSiblings(append(discs, d)); err != nil {
				return fmt.Errorf("range stopped at %d: %w", n, err)
			}
			path, err := mkChild(parent, d, name, *meta)
			if err != nil {
				return err
			}
			fmt.Println(path)
			discs = append(discs, d)
		}
		return nil
	}

	rawName := argN(pos, 1)
	if rawName == "" {
		return fmt.Errorf("usage: pan mk <parent-code> <name>")
	}
	name, err := prefix.Sanitize(rawName, opts)
	if err != nil {
		return fmt.Errorf("name %q: %w", rawName, err)
	}

	var taken []string
	for _, d := range discs {
		taken = append(taken, d.Value)
	}

	d := prefix.Discriminator{}
	switch {
	case *disc != "":
		d = classifyArg(*disc)
	case *kind == "letter":
		v, err := prefix.NextLetter(name, taken, opts)
		if err != nil {
			return err
		}
		d = prefix.Discriminator{Kind: prefix.Letter, Value: v, Padded: v}
	case *kind == "number":
		next := 1
		for _, ex := range discs {
			if ex.Kind == prefix.Number {
				if v, err := strconv.Atoi(ex.Value); err == nil && v >= next {
					next = v + 1
				}
			}
		}
		v := strconv.Itoa(next)
		d = prefix.Discriminator{Kind: prefix.Number, Value: v, Padded: v}
	case *kind == "word":
		return fmt.Errorf("--kind word needs an explicit --disc")
	default:
		return fmt.Errorf("unknown --kind %q", *kind)
	}
	if err := prefix.ValidateSiblings(append(discs, d)); err != nil {
		return err
	}

	path, err := mkChild(parent, d, name, *meta)
	if err != nil {
		return err
	}
	fmt.Println(path)
	if *meta {
		fmt.Println(filepath.Join(path, parent.Code+d.Value+"__"))
	}
	return nil
}

// mkChild creates one conforming child directory (and optionally its meta dir)
// under parent, returning the new directory's path. It refuses to overwrite.
func mkChild(parent *tree.Node, d prefix.Discriminator, name string, meta bool) (string, error) {
	entry := prefix.DirEntry{Inherited: parent.Code, Disc: d, Name: name}
	path := filepath.Join(parent.Path, prefix.FormatDir(entry))
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists", path)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return "", err
	}
	if meta {
		if err := os.Mkdir(filepath.Join(path, entry.FullPrefix()+"__"), 0o755); err != nil {
			return "", err
		}
	}
	return path, nil
}

// parseRange parses "A-B" into two ascending integers.
func parseRange(s string) (int, int, error) {
	lo, hi, ok := strings.Cut(s, "-")
	if !ok {
		return 0, 0, fmt.Errorf("range %q must be A-B", s)
	}
	a, err := strconv.Atoi(strings.TrimSpace(lo))
	if err != nil {
		return 0, 0, fmt.Errorf("range start %q: %w", lo, err)
	}
	b, err := strconv.Atoi(strings.TrimSpace(hi))
	if err != nil {
		return 0, 0, fmt.Errorf("range end %q: %w", hi, err)
	}
	if a > b {
		return 0, 0, fmt.Errorf("range %q: start must be ≤ end", s)
	}
	return a, b, nil
}

// classifyArg builds a Discriminator from an explicit --disc value, keeping
// cosmetic zero-padding ("03") in the directory name only.
func classifyArg(s string) prefix.Discriminator {
	allDigits := s != ""
	for _, r := range s {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	switch {
	case allDigits:
		v := strings.TrimLeft(s, "0")
		if v == "" {
			v = "0"
		}
		return prefix.Discriminator{Kind: prefix.Number, Value: v, Padded: s}
	case len([]rune(s)) == 1:
		return prefix.Discriminator{Kind: prefix.Letter, Value: s, Padded: s}
	default:
		return prefix.Discriminator{Kind: prefix.Word, Value: s, Padded: s}
	}
}

// cmdCd resolves a typeahead key sequence to a node and prints its absolute
// path to stdout (for `cd "$(pan cd asb)"`); the breadcrumb goes to stderr so
// it stays visible without polluting the path. Replaces the f/o/p shell
// functions — point a shell wrapper at any volume via --root.
func cmdCd(args []string) error {
	fs, root := newFlags("cd")
	rootLabel := fs.String("label", "", "breadcrumb root label (default: the root dir name)")
	pos := parseFlexible(fs, args)
	keys := argN(pos, 0)
	if keys == "" {
		return fmt.Errorf("usage: pan cd <keys>")
	}
	resolved := resolveRoot(*root)
	t, err := scan(resolved, tree.ScanOpts{})
	if err != nil {
		return err
	}
	chain := t.Navigate(keys)
	if len(chain) == 0 {
		return fmt.Errorf("no node reached for %q under %s", keys, t.RootPath)
	}
	label := *rootLabel
	if label == "" {
		label = filepath.Base(resolved)
	}
	fmt.Fprintln(os.Stderr, tree.Breadcrumb(label, chain))
	fmt.Println(chain[len(chain)-1].Path)
	return nil
}

// cmdMv renames a node, cascading the code change through its whole subtree.
// The prefix-aware, tested replacement for the old `renp` shell function.
func cmdMv(args []string) error {
	fs, root := newFlags("mv")
	disc := fs.String("disc", "", "new discriminator value")
	name := fs.String("name", "", "new descriptive name")
	reroot := fs.Bool("reroot", false, "set inherited prefix to the real parent's code (fixes a PrefixMismatch)")
	dryRun := fs.Bool("dry-run", false, "print the plan without applying it")
	swedish := fs.Bool("swedish", false, "allow åäö in the new name")
	pos := parseFlexible(fs, args)
	code := argN(pos, 0)
	if code == "" {
		return fmt.Errorf("usage: pan mv <code> [--disc x] [--name n] [--reroot]")
	}
	nameSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "name" {
			nameSet = true
		}
	})
	if *disc == "" && !nameSet && !*reroot {
		return fmt.Errorf("pan mv: nothing to change (give --disc, --name, and/or --reroot)")
	}

	t, err := scan(resolveRoot(*root), tree.ScanOpts{})
	if err != nil {
		return err
	}
	n, err := mustFind(t, code)
	if err != nil {
		return err
	}

	newDisc := n.Disc
	if *disc != "" {
		newDisc = classifyArg(*disc)
	}
	newName := n.Name
	if nameSet {
		if newName, err = prefix.Sanitize(*name, prefix.Opts{AllowSwedish: *swedish}); err != nil {
			return fmt.Errorf("name %q: %w", *name, err)
		}
	}
	newInherited := n.CurrentInherited()
	if *reroot {
		if n.Parent == nil {
			return fmt.Errorf("pan mv --reroot: %q is a top-level node with no parent to reroot under", code)
		}
		newInherited = n.Parent.Code
	}

	plan, err := t.PlanRename(n, newInherited, newDisc, newName)
	if err != nil {
		return err
	}
	if plan.Empty() {
		fmt.Println("(nothing to rename)")
		return nil
	}
	fmt.Fprintln(os.Stderr, plan.String())
	if *dryRun {
		return nil
	}
	if err := plan.Apply(); err != nil {
		return err
	}
	fmt.Printf("renamed %s → %s (%d path(s))\n", plan.OldCode, plan.NewCode, len(plan.Renames))
	return nil
}

func cmdOnto(args []string) error {
	fs, root := newFlags("onto")
	pos := parseFlexible(fs, args)

	path := ontology.DefaultPath(resolveRoot(*root))
	tbl, err := ontology.Load(path)
	if errors.Is(err, iofs.ErrNotExist) {
		return fmt.Errorf("no ontology at %s — the table is personal data each user keeps in their own volume; create it as TSV: code, parent, latin, greek, symbol, optional deity (tab-separated, # comments)", path)
	}
	if err != nil {
		return err
	}

	if code := argN(pos, 0); code != "" {
		n := tbl.Find(code)
		if n == nil {
			return fmt.Errorf("no ontology domain with code %q (the disk tree may still use it — the ontology is a map, not a law)", code)
		}
		fmt.Printf("%s  %s · %s · %s", n.Code, n.Latin, n.Greek, n.Symbol)
		if n.Deity != "" {
			fmt.Printf(" · %s", n.Deity)
		}
		fmt.Println()
		var latins []string
		for _, a := range n.Lineage() {
			latins = append(latins, a.Latin)
		}
		fmt.Println("      " + strings.Join(latins, " › "))
		for _, c := range n.Children {
			fmt.Printf("      → %-6s %s\n", c.Code, c.Latin)
		}
		return nil
	}

	for _, r := range tbl.Roots() {
		printOnto(r, 0)
	}
	return nil
}

func printOnto(n *ontology.Node, depth int) {
	line := fmt.Sprintf("%s%-8s %-12s %-14s %s", strings.Repeat("  ", depth), n.Code, n.Latin, n.Greek, n.Symbol)
	if n.Deity != "" {
		line += "  " + n.Deity
	}
	fmt.Println(strings.TrimRight(line, " "))
	for _, c := range n.Children {
		printOnto(c, depth+1)
	}
}
