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
| `cmd/pan`  | CLI: `tree` · `resolve` · `doctor` · `mk` · `onto`          | — |

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

**v0.1.0 — the spine is implemented and tested** (prefix, tree, ontology,
`pan`). `v1.0.0` follows once pensum has migrated onto it. Cascading rename
(`pan mv`) and auto-placement are planned for v1.1.

The full grammar specification and design docs are kept locally alongside the
maintainer's volume; the package docs (`go doc`) document the public surface.
