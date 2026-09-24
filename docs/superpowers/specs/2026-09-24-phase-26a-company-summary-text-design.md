# Phase 26a: Company Summary and Text Surfaces

Implements the text half of the
[player information surfaces spec](2026-09-23-player-information-surfaces-design.md),
fifth on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).

**Status:** Draft for owner review (2026-09-24). The open decisions below
carry recommendations and are **not yet confirmed**. Implementation starts
once they are.

**Split.** The parent spec has three parts that change different code. They
are split the same way as 21a/21b and 25a/25b:

- **26a (this document):** a read-only company summary built from the
  owning providers. The `status`, `experience`, `inventory`, and
  `conditions` commands get Ashveil sections, and the prompt gets new
  tokens.
- **26b (next):** the browser side: a `Company` GMCP payload (a full
  snapshot plus vitals updates) and a Company section in the Party window.
  It reads the same summary, so Telnet and the browser can't disagree.

## Prior-art check

- **Engine panels** (`internal/usercommands`): `status.panels.go`,
  `inventory.panels.go`, and `conditions.panels.go` build their panels in Go;
  `experience` renders `templates/character/experience`. Each command has
  engine behaviour that must stay: status training, `experience chart`, and
  the inventory filters.
- **Prompt** (`internal/users/userrecord.prompt.go`): `OnBuildPrompt` lets a
  module fill tokens after the built-in ones. An unknown token stays
  literal.
- **Providers already in place**, each behind an `internal/` seam:
  - company: roster, formation, instances, archetype, alignment, chemistry
    standing, and (25b) the dead with their time left;
  - survival: `CompanyNeeds` (living members, with `HungerLabel`,
    `ThirstLabel`, and `FatigueLabel`);
  - encumbrance: `CurrentLoad` (personal, cargo, capacity) and
    `CurrentBand`;
  - climate: `ExposureOf`;
  - expedition and camping: `MovementBlocked` (blocked plus a message), but
    no structured activity query;
  - archetypes: `PlayerArchetype`;
  - death: the checkpoint in `MiscData`.
- **Rest tiers and edges** are buffs (Rested and Well Rested, IDs from the
  camping config) and item fields (`sharpbonus` and `sharpstrikes`).
- **Browser Party window** (`modules/gmcp/gmcp.Party.go`) is GoMud's human
  party only. That is 26b's concern.

## Proposed design

1. **The read model** is a new package, `internal/companyview`, with no
   state of its own. `companyview.For(userID)` builds a `Summary` on the
   game loop by calling the existing seams:
   - the leader: archetype, level, HP and MP, alignment, needs, rest tier,
     and exposure;
   - each companion, keyed by member ID: `present`, `awaiting`, or `dead`,
     with name, archetype, level, HP when present, needs, formation cell,
     chemistry tier, and rescue time left when dead;
   - the company: alive and dead counts, load and band, activity
     (travelling, camping, resting, at an inn, or none), and checkpoint
     church.

   A missing provider leaves its field unknown (`ok=false`). It is never
   filled with a healthy default.
2. **One new seam each** where there isn't one yet:
   - a `company.MemberViewProvider` (`CompanyMembers`) listing members with
     their state;
   - an `ActivityProvider` on the expedition and camping seams: a
     structured activity plus the label the travel and camp commands
     already use.
3. **Labels come from one place.** Band labels and short forms ("Weary",
   "Heavy", "Resting 12m") are functions in `companyview`, used by both 26a
   and 26b.
4. **Text commands.** Each keeps its engine output and adds an Ashveil
   block. Every line of that block names the command that shows more:
   - `status`: archetype, alignment, the leader's needs in short form, the
     company "3 alive, 1 fallen", the load band, the activity, the rest
     tier, and pointers to `company status` and `formation`;
   - `experience`: level, experience to the next level, training and stat
     points, and the most recent level lost to death (a new durable
     `MiscData["death-last-loss"]` written by `Respawn`);
   - `inventory`: company load (personal plus cargo) against capacity, with
     cargo shown separately, and whetstone uses. A note explains that the
     engine's item-count limit is a separate thing;
   - `conditions`: buffs grouped as ordinary, rest (Rested or Well Rested,
     with the real time left), edges (strikes left per weapon),
     survival/exposure, and chemistry (its tier, never shown as an expiring
     buff).
5. **Prompt tokens** through `OnBuildPrompt`: `{fatigue}` (band), `{fatiguev}`
   (value), `{company}` ("3/1": alive/dead), `{load}` (band), and
   `{activity}` (a short word or empty). They are plain text, with no colour
   needed to read them.

## Open decisions (owner to confirm)

- **A. Split** 26a (text) / 26b (browser), as above. *Recommended.*
- **B. Default prompt.** Leave the shipped default prompt alone and document
  the new tokens, or add one survival warning and one activity word to it.
  *Recommend adding them*, as the parent spec asks, shown only when there
  is something to warn about, so a quiet prompt stays as short as today.
- **C. Where the read model lives.** A new `internal/companyview` package,
  or methods on `modules/company`. *Recommend `internal/companyview`*: the
  engine commands can import it, and it reads other modules only through
  their seams.
- **D. Level-loss note.** Record the most recent level lost to death for
  `experience`. *Recommended*, as one `MiscData` entry in the user file.
- **E. Scope of `conditions`.** Regroup the whole engine panel, or append
  Ashveil groups beneath it. *Recommend appending*, to keep the upstream
  panel and its tests intact.

## Constraints

- Read only. No command or prompt path writes a store or advances the
  world clock.
- The prompt is built often. Token handlers do constant work per member and
  never flatten plugin config on each call (the Phase 24 lesson).
- Member IDs are the keys everywhere, so two companions with the same name
  stay distinct.
- No new locks. The summary is built on the game loop; module seams take
  their own mutex inside the world lock, as today.

## Acceptance criteria

- Pure: every label function is table-tested. A missing provider gives an
  unknown field, never a default.
- Wiring, through `plugins.Load` with the real modules: a leader with two
  companions is followed through recruitment, a journey (activity), a camp
  rest (the rest tier), one companion's death (counts, `status`, the prompt
  `{company}`), and its resurrection. Each step checks the output of
  `status`, `experience`, `inventory`, `conditions`, and the prompt through
  `usercommands.TryCommand` and `GetCommandPrompt`. The engine output of
  each command is still there.
- Restart: the summary after a save and reload matches the one before.
- `go test -race ./...`, `make generate`, and `make validate` pass.
