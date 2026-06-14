package tree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LinusNyman/pantheon/prefix"
)

func build(t *testing.T, dirs []string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for p, c := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p)), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// studiumTree is a conforming subtree rooted at scientia (as), used by the
// cascade tests.
func studiumTree(t *testing.T) string {
	return build(t,
		[]string{
			"a_actio/a_s_scientia/as_b_bibliotheca",
			"a_actio/a_s_scientia/as_s_studium/ass__",
			"a_actio/a_s_scientia/as_s_studium/ass_a_ludus",
			"a_actio/a_s_scientia/as_s_studium/ass_c_gymnasium/assc_01_intro",
		},
		map[string]string{
			"a_actio/a_s_scientia/as_s_studium/ass.md":                                 "main\n",
			"a_actio/a_s_scientia/as_s_studium/ass__/ass_todo.md":                      "- [ ] x\n",
			"a_actio/a_s_scientia/as_s_studium/ass_a_ludus/assa.md":                    "ludus\n",
			"a_actio/a_s_scientia/as_s_studium/ass_c_gymnasium/assc_01_intro/assc1.md": "intro\n",
		},
	)
}

func TestPlanRenameCascade(t *testing.T) {
	root := studiumTree(t)
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("ass")

	plan, err := tr.PlanRename(n, n.CurrentInherited(), prefix.Discriminator{Kind: prefix.Letter, Value: "x", Padded: "x"}, n.Name)
	if err != nil {
		t.Fatal(err)
	}
	if plan.OldCode != "ass" || plan.NewCode != "asx" {
		t.Fatalf("codes: %s → %s", plan.OldCode, plan.NewCode)
	}
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}

	tr2, _ := Scan(root, ScanOpts{})
	if tr2.Find("ass") != nil {
		t.Error("old code ass still resolves")
	}
	for _, code := range []string{"asx", "asxa", "asxc", "asxc1"} {
		if tr2.Find(code) == nil {
			t.Errorf("new code %s does not resolve after cascade", code)
		}
	}
	// node dir and files renamed
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as_x_studium/asx.md"))
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as_x_studium/asx__/asx_todo.md"))
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as_x_studium/asx_c_gymnasium/asxc_01_intro/asxc1.md"))
	mustNotExist(t, filepath.Join(root, "a_actio/a_s_scientia/as_s_studium"))
	// sibling untouched
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as_b_bibliotheca"))
}

func TestPlanRenameNameOnly(t *testing.T) {
	root := studiumTree(t)
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("ass")

	plan, err := tr.PlanRename(n, n.CurrentInherited(), n.Disc, "studies")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Renames) != 1 { // only the node dir; no cascade when code is unchanged
		t.Fatalf("name-only plan has %d renames, want 1: %s", len(plan.Renames), plan)
	}
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	tr2, _ := Scan(root, ScanOpts{})
	if got := tr2.Find("ass"); got == nil || got.Name != "studies" {
		t.Fatalf("after name-only rename: %+v", got)
	}
	if tr2.Find("assa") == nil { // children keep their codes
		t.Error("child assa lost after name-only rename")
	}
}

func TestPlanRenameReroot(t *testing.T) {
	// A mismatched node (the mid-restructure case): asbo_t_todo physically
	// under aoap but claiming prefix asbo. Reroot fixes it to aoapt.
	root := build(t,
		[]string{
			"a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/asbo_t_todo/asbot__",
		},
		map[string]string{
			"a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/asbo_t_todo/asbot__/asbot_todo.md": "- [ ] x\n",
		},
	)
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("asbot")
	if n == nil || !n.Mismatched {
		t.Fatalf("setup: asbot = %+v", n)
	}
	parent := tr.Find("aoap")

	plan, err := tr.PlanRename(n, parent.Code, n.Disc, n.Name)
	if err != nil {
		t.Fatal(err)
	}
	if plan.NewCode != "aoapt" {
		t.Fatalf("reroot newCode = %s, want aoapt", plan.NewCode)
	}
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	tr2, _ := Scan(root, ScanOpts{})
	got := tr2.Find("aoapt")
	if got == nil || got.Mismatched {
		t.Fatalf("after reroot: %+v (mismatch should be gone)", got)
	}
	mustExist(t, filepath.Join(root, "a_actio/a_o_opus/ao_a_ars/aoa_p_pantheon_os/aoap_t_todo/aoapt__/aoapt_todo.md"))
}

func TestPlanRenamePaddedNumber(t *testing.T) {
	root := studiumTree(t)
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("assc1")

	plan, err := tr.PlanRename(n, n.CurrentInherited(), prefix.Discriminator{Kind: prefix.Number, Value: "2", Padded: "02"}, n.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as_s_studium/ass_c_gymnasium/assc_02_intro/assc2.md"))
	tr2, _ := Scan(root, ScanOpts{})
	if tr2.Find("assc2") == nil {
		t.Error("renumbered node assc2 does not resolve")
	}
}

func TestPlanRenameSiblingCollision(t *testing.T) {
	root := studiumTree(t)
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("ass")
	// sibling as_b_bibliotheca already uses discriminator b
	_, err := tr.PlanRename(n, n.CurrentInherited(), prefix.Discriminator{Kind: prefix.Letter, Value: "b", Padded: "b"}, n.Name)
	if !errors.Is(err, prefix.ErrDuplicateDiscriminator) {
		t.Fatalf("err = %v, want ErrDuplicateDiscriminator", err)
	}
}

func TestPlanRenamePreexistingTarget(t *testing.T) {
	root := studiumTree(t)
	// pre-create a conflicting node dir target
	if err := os.MkdirAll(filepath.Join(root, "a_actio/a_s_scientia/as_x_studium"), 0o755); err != nil {
		t.Fatal(err)
	}
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("ass")
	_, err := tr.PlanRename(n, n.CurrentInherited(), prefix.Discriminator{Kind: prefix.Letter, Value: "x", Padded: "x"}, n.Name)
	if err == nil {
		t.Fatal("expected collision error for pre-existing target")
	}
}

func TestApplyRollback(t *testing.T) {
	root := build(t,
		[]string{"a_actio/a_s_scientia"},
		map[string]string{"a_actio/a_s_scientia/as.md": "x\n"},
	)
	tr, _ := Scan(root, ScanOpts{})
	n := tr.Find("as")

	plan, err := tr.PlanRename(n, n.CurrentInherited(), prefix.Discriminator{Kind: prefix.Letter, Value: "x", Padded: "x"}, n.Name)
	if err != nil {
		t.Fatal(err)
	}
	// After planning, externally create a NON-EMPTY dir at the node-dir target
	// so the final (shallowest) rename fails and rollback must reverse the
	// already-applied file rename.
	conflict := filepath.Join(root, "a_actio/a_x_scientia")
	if err := os.MkdirAll(conflict, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conflict, "blocker"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := plan.Apply(); err == nil {
		t.Fatal("expected Apply to fail on the blocked node-dir rename")
	}
	// rollback restored the original layout
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as.md"))
	mustNotExist(t, filepath.Join(root, "a_actio/a_s_scientia/ax.md"))
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Errorf("expected to exist: %s (%v)", p, err)
	}
}

func mustNotExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err == nil {
		t.Errorf("expected NOT to exist: %s", p)
	}
}
