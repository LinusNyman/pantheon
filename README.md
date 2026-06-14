# pantheon

`github.com/LinusNyman/pantheon` — the shared spine for the pantheon suite of
personal CLI tools (pensum, pinax, principium, studium, speculum, and the
atrium dashboard). It owns the **prefix naming grammar**, the **filesystem tree
scan with deviation detection**, and the **ontology data** — and nothing else.
No todo, note, course, or habit type ever belongs here.

The pantheon convention in three lines: every directory's name carries its full
address (`as_b_bibliotheca` under `a_s_scientia` → code `asb`), files carry
their directory's code (`asb_todo.md`), and a node's metadata lives in
`<code>__/`. The filesystem is the database.

## Packages

| Package    | Responsibility                                              | I/O |
|------------|-------------------------------------------------------------|-----|
| `prefix`   | parse / format / sanitize names; allocate discriminators    | none (pure) |
| `tree`     | scan a root into a tree; resolve codes; report issues; atomic write | filesystem |
| `ontology` | load *your* life-domain table from `<root>/_ontology.tsv`   | read only |
| `cmd/pan`  | CLI: `tree` · `cd` · `resolve` · `doctor` · `mk` · `mv` · `onto` | — |

## The `pan` CLI

```
pan tree [code]            render the (sub)tree
pan cd <keys>              path of the node reached by a typeahead jump
pan resolve <code|path>    code → path, or path → code
pan doctor [code]          list grammar deviations (exit 1 if any)
pan mk <parent> <name>     create a conforming child  [--kind|--disc|--range A-B|--meta]
pan mv <code>              rename a node, cascading to its whole subtree
                           [--disc x] [--name n] [--reroot] [--dry-run]
pan onto [code]            your ontology table / one domain's lineage
```

`pan mv` is a prefix-aware cascading rename: changing a node's discriminator,
name, or inherited prefix (`--reroot`, to repair a `PrefixMismatch`) ripples
through every descendant directory and file. It plans deepest-first, refuses to
overwrite, and rolls back on failure. Preview with `--dry-run`.

### Shell navigation shim

`pan cd` prints a path (breadcrumb to stderr); wrap it so your shell actually
changes directory — one function replaces a per-volume `f`/`o`/`p`:

```sh
f() { cd "$(pan cd "$1" --root ~/vol_f)"; }
o() { cd "$(pan cd "$1" --root "$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/vol_o")"; }
```

## Your ontology

The meaning behind the codes is personal: every user keeps their own table at
`<root>/_ontology.tsv` (tab-separated; `#` comments; parents before children):

```
# code  parent  latin   greek     symbol  deity(optional)
a               Aqua    Hydor     α
am      a       Mare    Thalassa  θ       Poseidon
```

`pan onto` renders it; nothing in this module ships or publishes a table.

## Install

```
go install github.com/LinusNyman/pantheon/cmd/pan@latest
```

## Status

**v0.2.0 — the spine is implemented and tested** (prefix, tree, ontology,
`pan`), now including cascading rename (`pan mv`) and typeahead navigation
(`pan cd`). `v1.0.0` follows once pensum has migrated onto it. Auto-placement
(`pan place`) is still planned.

The full grammar specification and design docs are kept locally alongside the
maintainer's volume; the package docs (`go doc`) document the public surface.
