package tree

import (
	"errors"
	"testing"
)

func TestCodeOfFile(t *testing.T) {
	tests := map[string]string{
		"mvy_cool_video.mp4": "mvy",
		"asb.md":             "asb",
		"asb_todo.md":        "asb",
		"assefqf12_data.csv": "assefqf12",
		"asb_data.tar.gz":    "asb",
		"noprefix":           "noprefix",
		"":                   "",
	}
	for in, want := range tests {
		if got := CodeOfFile(in); got != want {
			t.Errorf("CodeOfFile(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNodeForFile(t *testing.T) {
	root := build(t, []string{
		"a_actio/a_s_scientia/as_b_bibliotheca",
	}, nil)
	tr, _ := Scan(root, ScanOpts{})

	n, err := tr.NodeForFile("asb_meditations.md")
	if err != nil || n == nil || n.Code != "asb" {
		t.Fatalf("NodeForFile(asb_…) = %+v, %v", n, err)
	}
	if _, err := tr.NodeForFile("My Video.mp4"); !errors.Is(err, ErrNoNode) {
		t.Errorf("non-conforming name: err = %v, want ErrNoNode", err)
	}
	if _, err := tr.NodeForFile("zzz_x.md"); !errors.Is(err, ErrNoNode) {
		t.Errorf("unknown code: err = %v, want ErrNoNode", err)
	}
}
