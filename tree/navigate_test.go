package tree

import (
	"strings"
	"testing"
)

func navTree(t *testing.T) *Tree {
	root := build(t,
		[]string{
			"a_actio/a_s_scientia/as_b_bibliotheca",
			"a_actio/a_s_scientia/as_s_studium",
			"a_actio/a_o_opus/ao_a_ars",
			"e_ego/e_m_mens",
		},
		nil,
	)
	tr, _ := Scan(root, ScanOpts{})
	return tr
}

func TestNavigate(t *testing.T) {
	tr := navTree(t)
	tests := []struct {
		keys string
		want string // codes of the matched chain, comma-joined
	}{
		{"a", "a"},
		{"as", "a,as"},
		{"asb", "a,as,asb"},
		{"ass", "a,as,ass"},
		{"aoa", "a,ao,aoa"},
		{"em", "e,em"},
		{"asz", "a,as"}, // 'z' matches nothing under scientia → stops
		{"zzz", ""},     // first char matches nothing
	}
	for _, tt := range tests {
		var codes []string
		for _, n := range tr.Navigate(tt.keys) {
			codes = append(codes, n.Code)
		}
		if got := strings.Join(codes, ","); got != tt.want {
			t.Errorf("Navigate(%q) = %q, want %q", tt.keys, got, tt.want)
		}
	}
}

func TestNavigateFromNode(t *testing.T) {
	tr := navTree(t)
	scientia := tr.Find("as")
	chain := scientia.Navigate("b")
	if len(chain) != 1 || chain[0].Code != "asb" {
		t.Fatalf("from scientia, 'b' = %+v", chain)
	}
}

func TestBreadcrumb(t *testing.T) {
	tr := navTree(t)
	chain := tr.Navigate("asb")
	if got := Breadcrumb("vol_f", chain); got != "vol_f ➜ actio ➜ scientia ➜ bibliotheca" {
		t.Errorf("breadcrumb = %q", got)
	}
}

func TestNavigateDensityBonus(t *testing.T) {
	// Two siblings both contain 'o'; the one where 'o' recurs (density bonus)
	// wins even though its first 'o' is later.
	root := build(t, []string{
		"a_actio/a_a_ports", // 'o' at index 3, once  → score 300
		"a_actio/a_b_oxooo", // 'o' at index 0, repeats → score -50
	}, nil)
	tr, _ := Scan(root, ScanOpts{})
	chain := tr.Find("a").Navigate("o")
	if len(chain) != 1 || chain[0].Name != "oxooo" {
		t.Fatalf("density bonus pick = %+v", chain)
	}
}
