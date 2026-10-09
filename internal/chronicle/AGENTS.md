# Chronicle Package Guide

Phase 63. GoMud-free: entry shapes, kinds, prose, `Filter`, the capped `Log`, and the package-level seam (`Record`, `Query`, `Count`, `Has`, `Total`). `modules/chronicle` installs the `Provider`; no other code imports the module.

- A company is its leader's user id. Names go in `Members`/`Subject`; later phases match on `Ref` (`mob:<id>`, `item:<id>`, `event:<id>`, `class:<id>`, `scenario:<id>`) and on member `Keys` (`leader`, `companion:<id>`; `Filter.Key`), never on names, which can repeat.
- A boss deed is every boss kill, an executed boss included (it also gets an `executed` deed).
- The log keeps `MaxEntries` deeds; `Tally` counts every deed ever (use `Total` for lifetime questions, `Has`/`Count` for recent ones).
- Add a kind in `Kinds` (with its filter words), `Prose`, and the help page `chronicle`. Record it from the real source of the deed, never from a display path.
- `Record` is a no-op without a provider, so callers never check.
