package tree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LinusNyman/pantheon/prefix"
)

func letter(v string) prefix.Discriminator {
	return prefix.Discriminator{Kind: prefix.Letter, Value: v, Padded: v}
}

func TestCreateChildRoot(t *testing.T) {
	root := t.TempDir()
	tr, _ := Scan(root, ScanOpts{})

	n, err := tr.CreateChild("", letter("a"), "actio", false)
	if err != nil {
		t.Fatal(err)
	}
	if n.Code != "a" || n.Depth != 1 || n.Parent != nil {
		t.Fatalf("root child = %+v", n)
	}
	mustExist(t, filepath.Join(root, "a_actio"))
	if tr.Find("a") != n {
		t.Error("new root node not registered in Find")
	}
}

func TestCreateChildNestedWithMeta(t *testing.T) {
	root := build(t, []string{"a_actio"}, nil)
	tr, _ := Scan(root, ScanOpts{})

	n, err := tr.CreateChild("a", letter("s"), "scientia", true)
	if err != nil {
		t.Fatal(err)
	}
	if n.Code != "as" || n.Depth != 2 || n.Parent != tr.Find("a") {
		t.Fatalf("nested child = %+v", n)
	}
	if !n.HasMeta {
		t.Error("HasMeta not set")
	}
	mustExist(t, filepath.Join(root, "a_actio/a_s_scientia/as__"))
	// wired into the parent's children
	if tr.Find("a").Children[0] != n {
		t.Error("child not appended to parent")
	}
}

func TestCreateChildDuplicate(t *testing.T) {
	root := build(t, []string{"a_actio/a_s_scientia"}, nil)
	tr, _ := Scan(root, ScanOpts{})
	_, err := tr.CreateChild("a", letter("s"), "somnium", false)
	if !errors.Is(err, prefix.ErrDuplicateDiscriminator) {
		t.Fatalf("err = %v, want ErrDuplicateDiscriminator", err)
	}
}

func TestCreateChildExistingPath(t *testing.T) {
	root := build(t, []string{"a_actio"}, nil)
	tr, _ := Scan(root, ScanOpts{})
	// A dir appears on disk after the scan (not registered as a sibling), so
	// the dup check passes but the overwrite guard must still fire.
	if err := os.Mkdir(filepath.Join(root, "a_actio", "a_s_scientia"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := tr.CreateChild("a", letter("s"), "scientia", false)
	if err == nil {
		t.Fatal("expected error creating over an existing directory")
	}
}

func TestCreateChildNumberPadding(t *testing.T) {
	root := build(t, []string{"a_actio"}, nil)
	tr, _ := Scan(root, ScanOpts{})
	d := prefix.Discriminator{Kind: prefix.Number, Value: "3", Padded: "03"}
	n, err := tr.CreateChild("a", d, "intro", false)
	if err != nil {
		t.Fatal(err)
	}
	if n.Code != "a3" {
		t.Fatalf("code = %s, want a3", n.Code)
	}
	mustExist(t, filepath.Join(root, "a_actio/a_03_intro")) // padded on disk
}

func TestCreateChildUnknownParent(t *testing.T) {
	root := build(t, []string{"a_actio"}, nil)
	tr, _ := Scan(root, ScanOpts{})
	_, err := tr.CreateChild("zzz", letter("a"), "x", false)
	if !errors.Is(err, ErrNoNode) {
		t.Fatalf("err = %v, want ErrNoNode", err)
	}
}
