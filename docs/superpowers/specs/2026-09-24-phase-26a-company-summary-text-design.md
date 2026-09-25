# Phase 26a: Company Summary and Text Surfaces

Implements the text half of the
[player information surfaces spec](2026-09-23-player-information-surfaces-design.md),
fifth on the [onboarding roadmap](2026-09-23-company-life-onboarding-roadmap.md).

**Status:** Decisions confirmed by the owner on 2026-09-25 ("Ok go"), after
revising B and E: a richer default prompt and rearranged panels (see
Decisions).

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

## Decisions

1. **The read model** is a new package, `internal/companyview`, with no
   state of its own. `companyview.For(user)` builds a `Summary` on the game
   loop by calling the existing seams:
   - the leader: archetype, level, HP and MP, alignment, needs, warmth
     (exposure), light where they stand, and rest tier;
   - each companion, keyed by member ID: `present`, `awaiting`, or `dead`,
     with name, archetype, level, HP when present, needs, formation cell,
     and rescue time left when dead;
   - the company: alive and dead counts, load and its band, activity
     (travelling, stopped on the road, resting at camp, or at an inn, with
     progress or time left), and checkpoint church.

   A missing provider leaves its field unknown (`Known=false`). It is never
   filled with a healthy default.
2. **New seams, only where none exists:**
   - `company.MemberViewProvider` (`CompanyMembers`) lists members with
     their state;
   - `expedition.ProgressProvider` (`Progress`) and `camping.RestProvider`
     (`RestActivity`, `RestTier`) are optional interfaces on the providers
     those modules already register. They read state under the module's own
     mutex, with no side effects;
   - `companyview.RegisterBuffGroup` lets the camping and exposure modules
     name their buff IDs (rest and exposure) for `conditions`.
3. **Labels come from one place.** Band labels and short forms live in
   `companyview` and serve both 26a and 26b:
   - needs use survival's `HungerLabel`, `ThirstLabel`, and `FatigueLabel`;
   - warmth uses the exposure buff names (Chilled … Freezing to Death;
     Overheated … Heatstroke);
   - light is Dark or Dim;
   - load is Unburdened, Burdened, or Overloaded, from the configured band's
     effect and the load ratio. The band table has no names.
4. **Rearranged panels** (owner revision of E). The panel code stays in
   separate Ashveil files; each engine command calls into them, so the
   upstream diff stays small.
   - **`status`** is Ashveil's character sheet. The layout file
     `panel-layouts/character/status.yaml` gains panels, and Go fills
     whichever panels the layout defines (a new
     `templates.PanelLayout.HasPanel`), so the upstream `empty` world keeps
     its old sheet.
     - **Character:** area, race, archetype, level, experience, alignment
       (−100..100 and its band).
     - **Vitals:** health, mana, armor, hunger, thirst, fatigue, warmth,
       light.
     - **Attributes**, **Wealth**, and **Training** stay as they are.
     - **Company:** members ("3 alive, 1 fallen"), load, what the company is
       doing, the rest tier, and where the player wakes if they fall.
     - A footer line names `company status`, `formation`, `conditions`, and
       `survival`.

     Stat training and `status bonuses` are unchanged.
   - **`conditions`** is fully grouped: Rest (Rested or Well Rested, real
     time left), Weapon edges (strikes left), Survival (need warnings and
     warmth), Company (chemistry: tier and band, never an expiring buff),
     and Other effects (every other visible buff, with time left). An empty
     group is left out. A player with none of these sees "None", as today.
   - **`inventory`** leads with company load: personal plus cargo against
     capacity, with the cargo portion separate, and a note that the engine's
     item-count limit (shown on the Carrying line) is a separate thing.
     Then the equipment and items as today. Filters are unchanged.
   - **`experience`** keeps its line and adds the most recent level lost to
     death (decision 7), when there is one.
5. **The prompt** (owner revision of B). There is a token for each value,
   in plain text so no colour is needed to read it:

   | Tokens | Meaning |
   |---|---|
   | `{hunger}` `{thirst}` `{fatigue}` | The need's band label |
   | `{hungerv}` `{thirstv}` `{fatiguev}` | The need's value |
   | `{light}` | Dark, Dim, or empty |
   | `{warmth}` | The exposure label, or empty |
   | `{load}` | The load label |
   | `{company}` | "3" alive, or "3, 1 dead" |
   | `{activity}` | Travelling 42%, Stopped, Camped (a pitched camp, not resting), Resting 12m, At inn 5m, or empty |
   | `{warn}` | A compact cluster, described below |

   `{warn}` shows each need at Hungry/Thirsty/Tired or worse, then Dark or
   Dim, the warmth band, and Overloaded. It has a fixed order and at most
   four words. Each word leads with a space, so an empty cluster adds
   nothing.

   The shipped default prompt becomes
   `…MP:{mp}/{MP}]{warn}{activity}{h}:`. A fed, rested, warm player in the
   light who is doing nothing sees exactly today's prompt. Players can
   still build their own prompt with `prompt` or `fprompt`, and `help
   prompt` lists the new tokens.
6. **The prompt is built from a cache.** The engine builds prompts on
   connection goroutines as well as on the game loop, and outside the world
   lock. The company module has no mutex, so a prompt must never read game
   state directly. `companyview` works out each online player's token
   values on the game loop, on every `NewRound` and after every command
   (one deferred call at the end of `usercommands.TryCommand`). It stores
   them in a mutex-guarded cache. `OnBuildPrompt` only reads the cache.
   Values can lag by at most one round.
7. **Level-loss note.** `Respawn` records the most recent level lost to
   death in `MiscData["death-last-loss"]` (the levels before and after), in
   the user file.

## Constraints

- Read only. No command or prompt path writes a store or advances the
  world clock.
- The prompt is built often and off the game loop. Its handler reads only
  the cache (decision 6), and nothing on that path flattens plugin config
  (the Phase 24 lesson).
- Member IDs are the keys everywhere, so two companions with the same name
  stay distinct.
- No new locks. The summary is built on the game loop; module seams take
  their own mutex inside the world lock, as today.

## Acceptance criteria

- Pure: every label function is table-tested. A missing provider gives an
  unknown field, never a default. `{warn}` keeps its order and its
  four-word cap, and a quiet player's default prompt matches today's.
- Concurrency: the prompt handler under `-race` reads the cache while the
  game loop refreshes it.
- The `empty` world's status layout (no new panels) renders with no errors.
- Wiring, through `plugins.Load` with the real modules: a leader with two
  companions is followed through recruitment, a journey (activity), a camp
  rest (the rest tier), one companion's death (counts, `status`, the prompt
  `{company}`), and its resurrection. Each step checks the output of
  `status`, `experience`, `inventory`, `conditions`, and the prompt through
  `usercommands.TryCommand` and `GetCommandPrompt`. The engine output of
  each command is still there.
- Restart: the summary after a save and reload matches the one before.
- `go test -race ./...`, `make generate`, and `make validate` pass.
