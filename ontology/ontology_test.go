package ontology

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/LinusNyman/pantheon/prefix"
)

func loadExample(t *testing.T) *Table {
	t.Helper()
	tbl, err := Load("testdata/example.tsv")
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func TestShape(t *testing.T) {
	tbl := loadExample(t)
	if got := len(tbl.All()); got != 6 {
		t.Errorf("len(All()) = %d, want 6", got)
	}
	r := tbl.Roots()
	if len(r) != 2 || r[0].Latin != "Aqua" || r[1].Latin != "Terra" {
		t.Fatalf("Roots() = %v", r)
	}
	if n := tbl.Find("am"); n == nil || n.Latin != "Mare" || n.Deity != "Poseidon" {
		t.Errorf("Find(am) = %+v", n)
	}
	if n := tbl.Find("af"); n == nil || n.Deity != "" {
		t.Errorf("Find(af) = %+v, want empty deity", n)
	}
	if tbl.Find("zzz") != nil {
		t.Error("Find(zzz) should be nil")
	}
}

func TestLineage(t *testing.T) {
	tbl := loadExample(t)
	var latins []string
	for _, a := range tbl.Find("an").Lineage() {
		latins = append(latins, a.Latin)
	}
	if got := strings.Join(latins, ">"); got != "Aqua>Fons" {
		t.Errorf("lineage = %s", got)
	}
}

// Every code in the example equals its parent's code plus a discriminator
// allocated by the SPEC §6 algorithm from the Latin name, given the earlier
// siblings — "an" Fons exercises the collision fallback (f taken by Flumen →
// consonant n).
func TestCodesFollowAllocationAlgorithm(t *testing.T) {
	tbl := loadExample(t)
	check := func(parentCode string, siblings []*Node) {
		var taken []string
		for _, s := range siblings {
			name, err := prefix.Sanitize(s.Latin, prefix.Opts{})
			if err != nil {
				t.Fatalf("%s: %v", s.Latin, err)
			}
			disc, err := prefix.NextLetter(name, taken, prefix.Opts{})
			if err != nil {
				t.Fatalf("%s: %v", s.Latin, err)
			}
			if want := parentCode + disc; s.Code != want {
				t.Errorf("%s: code %s, algorithm yields %s", s.Latin, s.Code, want)
			}
			taken = append(taken, disc)
		}
	}
	check("", tbl.Roots())
	for _, n := range tbl.All() {
		if len(n.Children) > 0 {
			check(n.Code, n.Children)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"too few fields": "a\t\tAqua\tHydor\n",
		"duplicate code": "a\t\tAqua\tHydor\tα\na\t\tAlter\tAllos\tβ\n",
		"unknown parent": "ax\ta\tAqua\tHydor\tα\n",
		"empty code":     "\t\tAqua\tHydor\tα\n",
	}
	for name, in := range cases {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("%s: Parse accepted %q", name, in)
		}
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("testdata/no_such.tsv")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file err = %v, want fs.ErrNotExist", err)
	}
}
