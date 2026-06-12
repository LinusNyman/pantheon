package tree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// golden builds a miniature vol_f in a temp dir, deliberately including every
// deviation the Issue taxonomy must catch (SPEC §9, design doc §6).
func golden(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		// conforming spine
		"a_actio/a_s_scientia/as_b_bibliotheca/asb__",
		"a_actio/a_s_scientia/as_s_studium/ass_a_ludus",
		"a_actio/a_s_scientia/as_s_studium/ass_b_elementa",
		"a_actio/a_s_scientia/as_s_studium/ass_c_gymnasium/assc_01_intro",
		"a_actio/a_s_scientia/as_s_studium/ass_c_gymnasium/assc_02_algebra",
		"a_actio/a_s_scientia/as_s_studium/ass_c_gymnasium/assc_10_late",
		"a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os",
		"e_ego/e_m_mens/em__",
		// deviations
		"sort", // Orphan
		"a_actio/a_s_scientia/as_s_studium/ass_9_deviant",                     // MixedScheme (Number among Letters)
		"a_actio/a_s_scientia/as_b_duplicate",                                 // DuplicateDiscriminator (code asb)
		"a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/asbo_p_pantheon/asbop__", // PrefixMismatch, meta inside still conforms
		"a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/asbo_p_pantheon/asbop_docs",
		"a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/zz__", // BadMetaName
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"README.md":    "# vol\n",
		"_pan_v2_1.md": "ontology\n",
		".DS_Store":    "junk",
		"a_actio/a_s_scientia/as_b_bibliotheca/asb.md":                    "main\n",
		"a_actio/a_s_scientia/as_b_bibliotheca/asb_meditations_marcus.md": "notes\n",
		"a_actio/a_s_scientia/as_b_bibliotheca/asb__/asb_todo.md":         "- [ ] x\n",
		"a_actio/a_s_scientia/as_b_bibliotheca/asb__/notes.md":            "stray in meta\n",
		"a_actio/a_s_scientia/as_b_bibliotheca/stray.txt":                 "stray\n",
	}
	for p, content := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestScanGoldenTree(t *testing.T) {
	root := golden(t)
	tr, err := Scan(root, ScanOpts{ScanFiles: true})
	if err != nil {
		t.Fatal(err)
	}

	// --- resolution
	bib := tr.Find("asb")
	if bib == nil || bib.Name != "bibliotheca" {
		t.Fatalf("Find(asb) = %+v, want bibliotheca (first wins over duplicate)", bib)
	}
	if !bib.HasMeta {
		t.Error("asb should have HasMeta")
	}
	if got := tr.Find("aoap"); got == nil || got.Name != "pantheon_os" {
		t.Fatalf("Find(aoap) = %+v", got)
	}

	// --- the restructuring case: mismatched subtree stays addressable
	pan := tr.Find("asbop")
	if pan == nil {
		t.Fatal("Find(asbop) = nil; mismatched node not adopted")
	}
	if !pan.Mismatched || pan.Name != "pantheon" {
		t.Errorf("asbop = %+v, want Mismatched pantheon", pan)
	}
	if !pan.HasMeta {
		t.Error("asbop should see its conforming asbop__ meta dir")
	}
	if docs := tr.Find("asbopdocs"); docs == nil || docs.Parent != pan {
		t.Errorf("Find(asbopdocs) = %+v, want child of asbop", docs)
	}

	// --- padded numbers: code unpadded, numeric sort
	if n := tr.Find("assc1"); n == nil || n.Name != "intro" || n.Disc.Padded != "01" {
		t.Errorf("Find(assc1) = %+v", n)
	}
	gym := tr.Find("assc")
	var order []string
	for _, c := range gym.Children {
		order = append(order, c.Code)
	}
	if got := strings.Join(order, ","); got != "assc1,assc2,assc10" {
		t.Errorf("numeric sibling sort = %s, want assc1,assc2,assc10", got)
	}

	// --- issues: exactly the planted deviations
	wantIssues := map[IssueKind][]string{
		Orphan:                 {filepath.Join(root, "sort")},
		MixedScheme:            {filepath.Join(root, "a_actio/a_s_scientia/as_s_studium/ass_9_deviant")},
		DuplicateDiscriminator: {filepath.Join(root, "a_actio/a_s_scientia/as_b_duplicate")},
		PrefixMismatch:         {filepath.Join(root, "a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/asbo_p_pantheon")},
		BadMetaName:            {filepath.Join(root, "a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/zz__")},
		StrayFile: {
			filepath.Join(root, "a_actio/a_s_scientia/as_b_bibliotheca/asb__/notes.md"),
			filepath.Join(root, "a_actio/a_s_scientia/as_b_bibliotheca/stray.txt"),
		},
	}
	got := map[IssueKind][]string{}
	for _, is := range tr.Issues {
		got[is.Kind] = append(got[is.Kind], is.Path)
	}
	for kind, wantPaths := range wantIssues {
		gp := got[kind]
		if len(gp) != len(wantPaths) {
			t.Errorf("%s: got %d issues %v, want %v", kind, len(gp), gp, wantPaths)
			continue
		}
		for _, w := range wantPaths {
			found := false
			for _, g := range gp {
				if g == w {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: missing expected issue at %s (got %v)", kind, w, gp)
			}
		}
	}
	if len(tr.Issues) != 7 {
		for _, is := range tr.Issues {
			t.Log(is)
		}
		t.Errorf("total issues = %d, want 7", len(tr.Issues))
	}

	// --- deterministic walk order: depth-first, a before e
	var codes []string
	tr.Walk(func(n *Node) { codes = append(codes, n.Code) })
	if codes[0] != "a" || codes[len(codes)-1] != "em" {
		t.Errorf("walk order: %v", codes)
	}
}

func TestScanRepoBoundary(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{
		"a_actio/a_t_tool/.git",
		"a_actio/a_t_tool/at__",
		"a_actio/a_t_tool/cmd", // ecosystem dir: must NOT become an orphan issue
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tr, err := Scan(root, ScanOpts{ScanFiles: true})
	if err != nil {
		t.Fatal(err)
	}
	n := tr.Find("at")
	if n == nil || !n.IsRepo {
		t.Fatalf("Find(at) = %+v, want IsRepo", n)
	}
	if !n.HasMeta {
		t.Error("repo node should still see its top-level meta dir")
	}
	if len(n.Children) != 0 {
		t.Errorf("repo interior scanned: %v", n.Children)
	}
	if len(tr.Issues) != 0 {
		t.Errorf("repo interior produced issues: %v", tr.Issues)
	}

	tr2, _ := Scan(root, ScanOpts{DescendRepos: true})
	if len(tr2.Issues) == 0 {
		t.Error("DescendRepos should surface the repo interior (cmd/ as an orphan)")
	}
}

func TestMetaHelpers(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a_actio"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr, err := Scan(root, ScanOpts{})
	if err != nil {
		t.Fatal(err)
	}
	n := tr.Find("a")
	if n.HasMeta {
		t.Error("no meta dir yet")
	}
	want := filepath.Join(root, "a_actio", "a__")
	if n.MetaDir() != want {
		t.Errorf("MetaDir = %s, want %s", n.MetaDir(), want)
	}
	dir, err := n.EnsureMetaDir()
	if err != nil || dir != want || !n.HasMeta {
		t.Fatalf("EnsureMetaDir = %s, %v", dir, err)
	}
	if got := n.AppFile("todo", "md"); got != filepath.Join(want, "a_todo.md") {
		t.Errorf("AppFile = %s", got)
	}
}

func TestWriteFileAndUniquify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "asb_todo.md")

	if err := WriteFile(path, []byte("v1\n")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("v2\n")); err != nil { // overwrite is fine for content writes
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "v2\n" {
		t.Fatalf("content = %q, %v", data, err)
	}

	// no temp litter
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".pantheon-tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}

	// Uniquify never overwrites
	if got := Uniquify(path); got != filepath.Join(dir, "asb_todo_2.md") {
		t.Errorf("Uniquify = %s", got)
	}
	if err := WriteFile(filepath.Join(dir, "asb_todo_2.md"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	if got := Uniquify(path); got != filepath.Join(dir, "asb_todo_3.md") {
		t.Errorf("Uniquify second = %s", got)
	}
	free := filepath.Join(dir, "fresh.md")
	if got := Uniquify(free); got != free {
		t.Errorf("Uniquify(fresh) = %s", got)
	}
}

func TestRootResolution(t *testing.T) {
	t.Setenv("PANTHEON_ROOT", "")
	t.Setenv("TESTAPP_ROOT", "/tmp/app")
	if got := Root("TESTAPP_ROOT", "/tmp/fb"); got != "/tmp/app" {
		t.Errorf("app env: %s", got)
	}
	t.Setenv("TESTAPP_ROOT", "")
	t.Setenv("PANTHEON_ROOT", "/tmp/pan")
	if got := Root("TESTAPP_ROOT", "/tmp/fb"); got != "/tmp/pan" {
		t.Errorf("pantheon env: %s", got)
	}
	t.Setenv("PANTHEON_ROOT", "")
	if got := Root("TESTAPP_ROOT", "/tmp/fb"); got != "/tmp/fb" {
		t.Errorf("fallback: %s", got)
	}
	if got := Root("TESTAPP_ROOT", ""); !strings.HasSuffix(got, "vol_f") {
		t.Errorf("default: %s", got)
	}
}
