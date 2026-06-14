package prefix

import (
	"errors"
	"testing"
)

func TestParseDir(t *testing.T) {
	tests := []struct {
		name     string
		basename string
		parent   string
		wantErr  error
		want     DirEntry
		wantFull string // FullPrefix of the returned entry (also checked on best-effort entries)
	}{
		// depth 1
		{"depth1 letter", "a_actio", "", nil,
			DirEntry{Disc: Discriminator{Letter, "a", "a"}, Name: "actio"}, "a"},
		{"depth1 multiword name", "2024_tax_papers", "", nil,
			DirEntry{Disc: Discriminator{Number, "2024", "2024"}, Name: "tax_papers"}, "2024"},
		{"depth1 word disc", "vol_o", "", nil,
			DirEntry{Disc: Discriminator{Word, "vol", "vol"}, Name: "o"}, "vol"},
		{"depth1 nameless", "a", "", ErrMalformed, DirEntry{}, ""},
		{"depth1 plain word", "sort", "", ErrMalformed, DirEntry{}, ""},

		// depth ≥2
		{"depth2", "a_s_scientia", "a", nil,
			DirEntry{Inherited: "a", Disc: Discriminator{Letter, "s", "s"}, Name: "scientia"}, "as"},
		{"depth3", "as_b_bibliotheca", "as", nil,
			DirEntry{Inherited: "as", Disc: Discriminator{Letter, "b", "b"}, Name: "bibliotheca"}, "asb"},
		{"name with underscores", "aoa_p_pantheon_os", "aoa", nil,
			DirEntry{Inherited: "aoa", Disc: Discriminator{Letter, "p", "p"}, Name: "pantheon_os"}, "aoap"},
		{"nameless number", "assefqf_12", "assefqf", nil,
			DirEntry{Inherited: "assefqf", Disc: Discriminator{Number, "12", "12"}}, "assefqf12"},
		{"padded number", "assefqf_03", "assefqf", nil,
			DirEntry{Inherited: "assefqf", Disc: Discriminator{Number, "3", "03"}}, "assefqf3"},
		{"word disc with name", "assef_aar_marcus", "assef", nil,
			DirEntry{Inherited: "assef", Disc: Discriminator{Word, "aar", "aar"}, Name: "marcus"}, "assefaar"},
		{"nameless letter", "as_b", "as", nil,
			DirEntry{Inherited: "as", Disc: Discriminator{Letter, "b", "b"}}, "asb"},
		{"swedish letter tolerated", "as_å_årsbok", "as", nil,
			DirEntry{Inherited: "as", Disc: Discriminator{Letter, "å", "å"}, Name: "årsbok"}, "aså"},

		// meta
		{"meta", "asb__", "asb", nil,
			DirEntry{Inherited: "asb", Disc: Discriminator{Meta, "__", "__"}}, "asb"},
		{"meta wrong stem", "asx__", "asb", ErrBadMetaName,
			DirEntry{Inherited: "asx", Disc: Discriminator{Meta, "__", "__"}}, "asx"},
		{"meta at root", "asb__", "", ErrBadMetaName,
			DirEntry{Inherited: "asb", Disc: Discriminator{Meta, "__", "__"}}, "asb"},
		{"bare meta", "__", "asb", ErrMalformed, DirEntry{}, ""},

		// mismatch with best-effort entry (the restructuring case)
		{"mismatch 3seg", "asbo_p_pantheon", "aoap", ErrPrefixMismatch,
			DirEntry{Inherited: "asbo", Disc: Discriminator{Letter, "p", "p"}, Name: "pantheon"}, "asbop"},
		{"mismatch 2seg", "b_bibliotheca", "as", ErrPrefixMismatch,
			DirEntry{Disc: Discriminator{Letter, "b", "b"}, Name: "bibliotheca"}, "b"},
		{"mismatch plain word", "formula", "as", ErrMalformed, DirEntry{}, ""},

		// rejects
		{"empty", "", "as", ErrEmpty, DirEntry{}, ""},
		{"dotfile", ".git", "as", ErrDotfile, DirEntry{}, ""},
		{"uppercase", "Formula", "as", ErrCharset, DirEntry{}, ""},
		{"hyphen", "as-b-x", "as", ErrCharset, DirEntry{}, ""},
		{"internal double underscore", "as__b_x", "as", ErrMalformed, DirEntry{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDir(tt.basename, tt.parent)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseDir(%q, %q) err = %v, want %v", tt.basename, tt.parent, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseDir(%q, %q) = %+v, want %+v", tt.basename, tt.parent, got, tt.want)
			}
			if tt.wantFull != "" && got.FullPrefix() != tt.wantFull {
				t.Fatalf("FullPrefix() = %q, want %q", got.FullPrefix(), tt.wantFull)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name     string
		basename string
		prefix   string
		wantErr  error
		want     FileEntry
	}{
		{"own file", "asb.md", "asb", nil, FileEntry{Prefix: "asb", Ext: "md"}},
		{"descriptor", "asb_todo.md", "asb", nil, FileEntry{Prefix: "asb", Descriptor: "todo", Ext: "md"}},
		{"multiword descriptor", "asbop_suite_plan.md", "asbop", nil,
			FileEntry{Prefix: "asbop", Descriptor: "suite_plan", Ext: "md"}},
		{"double extension", "asb_data.tar.gz", "asb", nil,
			FileEntry{Prefix: "asb", Descriptor: "data", Ext: "tar.gz"}},
		{"extensionless", "asb_makefile", "asb", nil,
			FileEntry{Prefix: "asb", Descriptor: "makefile"}},

		// volume root exceptions
		{"root readme", "README.md", "", nil, FileEntry{Descriptor: "README", Ext: "md"}},
		{"root readme bare", "README", "", nil, FileEntry{Descriptor: "README"}},
		{"root meta doc", "_pan_v2_1.md", "", nil, FileEntry{Descriptor: "pan_v2_1", Ext: "md"}},
		{"root stray", "notes.md", "", ErrPrefixMismatch, FileEntry{}},

		// rejects
		{"stray file", "notes.md", "asb", ErrPrefixMismatch, FileEntry{}},
		{"wrong prefix", "asbx_todo.md", "asb", ErrPrefixMismatch, FileEntry{}},
		{"double underscore", "asb__notes.md", "asb", ErrMalformed, FileEntry{}},
		{"dotfile", ".DS_Store", "asb", ErrDotfile, FileEntry{}},
		{"uppercase stem", "Makefile", "asb", ErrCharset, FileEntry{}},
		{"empty", "", "asb", ErrEmpty, FileEntry{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFile(tt.basename, tt.prefix)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseFile(%q, %q) err = %v, want %v", tt.basename, tt.prefix, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseFile(%q, %q) = %+v, want %+v", tt.basename, tt.prefix, got, tt.want)
			}
		})
	}
}

// Round-trip: FormatDir(ParseDir(x)) == x for every conforming name (SPEC §11.6).
func TestRoundTrip(t *testing.T) {
	dirs := []struct{ basename, parent string }{
		{"a_actio", ""},
		{"a_s_scientia", "a"},
		{"as_b_bibliotheca", "as"},
		{"aoa_p_pantheon_os", "aoa"},
		{"assefqf_12", "assefqf"},
		{"assefqf_03", "assefqf"}, // padding round-trips via Padded
		{"assef_aar_marcus", "assef"},
		{"asb__", "asb"},
	}
	for _, d := range dirs {
		e, err := ParseDir(d.basename, d.parent)
		if err != nil {
			t.Fatalf("ParseDir(%q, %q): %v", d.basename, d.parent, err)
		}
		if got := FormatDir(e); got != d.basename {
			t.Errorf("round-trip %q → %+v → %q", d.basename, e, got)
		}
	}

	files := []struct{ basename, prefix string }{
		{"asb.md", "asb"},
		{"asb_todo.md", "asb"},
		{"asb_data.tar.gz", "asb"},
		{"asb_makefile", "asb"},
	}
	for _, f := range files {
		e, err := ParseFile(f.basename, f.prefix)
		if err != nil {
			t.Fatalf("ParseFile(%q, %q): %v", f.basename, f.prefix, err)
		}
		if got := FormatFile(e.Prefix, e.Descriptor, e.Ext); got != f.basename {
			t.Errorf("round-trip %q → %+v → %q", f.basename, e, got)
		}
	}
}

func TestFormatFile(t *testing.T) {
	if got := FormatFile("asb", "todo", ".md"); got != "asb_todo.md" {
		t.Errorf("leading-dot ext: got %q", got)
	}
	if got := FormatFile("asb", "", "md"); got != "asb.md" {
		t.Errorf("own file: got %q", got)
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		raw     string
		opts    Opts
		want    string
		wantErr error
	}{
		{"Hello World", Opts{}, "hello_world", nil},
		{"  Marcus Aurelius — Meditations!  ", Opts{}, "marcus_aurelius_meditations", nil},
		{"foo-bar--baz", Opts{}, "foo_bar_baz", nil},
		{"Crème Brûlée", Opts{}, "crme_brle", nil},
		{"År 2024", Opts{}, "r_2024", nil},
		{"År 2024", Opts{AllowSwedish: true}, "år_2024", nil},
		{"__a__b__", Opts{}, "a_b", nil},
		{"!!!", Opts{}, "", ErrEmpty},
		{"  -  ", Opts{}, "", ErrEmpty},
	}
	for _, tt := range tests {
		got, err := Sanitize(tt.raw, tt.opts)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("Sanitize(%q) err = %v, want %v", tt.raw, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNextLetter(t *testing.T) {
	tests := []struct {
		name    string
		taken   []string
		want    string
		wantErr error
	}{
		{"bibliotheca", nil, "b", nil},
		{"bibliotheca", []string{"b"}, "l", nil}, // subsequent consonant
		{"fons", []string{"f"}, "n", nil},        // collision → consonant (testdata example)
		{"petra", []string{"p"}, "t", nil},
		{"ego", []string{"e"}, "g", nil},
		{"aaa", []string{"a"}, "b", nil},                    // fallback to alphabet
		{"ae", []string{"a"}, "e", nil},                     // remaining-rune stage
		{"studium", []string{"s", "t", "d", "m"}, "u", nil}, // vowels after consonants
		{"x", allLetters(), "", ErrAlphabetExhausted},
	}
	for _, tt := range tests {
		got, err := NextLetter(tt.name, tt.taken, Opts{})
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("NextLetter(%q, %v) err = %v, want %v", tt.name, tt.taken, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("NextLetter(%q, %v) = %q, want %q", tt.name, tt.taken, got, tt.want)
		}
	}
}

func allLetters() []string {
	var out []string
	for _, r := range "abcdefghijklmnopqrstuvwxyz0123456789" {
		out = append(out, string(r))
	}
	return out
}

func TestValidateSiblings(t *testing.T) {
	ok := []Discriminator{
		{Letter, "a", "a"}, {Letter, "b", "b"}, {Number, "3", "03"}, {Meta, "__", "__"},
	}
	if err := ValidateSiblings(ok); err != nil {
		t.Errorf("unique siblings: %v", err)
	}
	dup := []Discriminator{{Letter, "3", "3"}, {Number, "3", "03"}}
	if err := ValidateSiblings(dup); !errors.Is(err, ErrDuplicateDiscriminator) {
		t.Errorf("padded/letter collision on value: err = %v", err)
	}
}

func TestSplitCode(t *testing.T) {
	cases := []struct {
		code, parent, own string
		err               error
	}{
		{"a", "", "a", nil},
		{"au", "a", "u", nil},
		{"auk", "au", "k", nil},
		{"ass", "as", "s", nil},
		{"a1", "a", "1", nil}, // digits admitted (portable core widening)
		{"", "", "", ErrEmpty},
		{"A", "", "", ErrCharset},  // uppercase rejected
		{"a_", "", "", ErrCharset}, // separators rejected
		{"aå", "", "", ErrCharset}, // åäö rejected for created codes
	}
	for _, c := range cases {
		p, o, err := SplitCode(c.code)
		if !errors.Is(err, c.err) {
			t.Errorf("SplitCode(%q) err = %v, want %v", c.code, err, c.err)
			continue
		}
		if c.err != nil {
			continue
		}
		if p != c.parent || o != c.own {
			t.Errorf("SplitCode(%q) = (%q, %q), want (%q, %q)", c.code, p, o, c.parent, c.own)
		}
	}
}
