# Phase 32a2: Per-Player Recruit Rosters — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md)), answering 32a's open
question about recruits in the room.

## The owner's rule (2026-09-28)

> "Let it be listed in the room's notice board, but let it be specifically
> for the player and randomly generated every random amount of time. This
> is because each player can always have a list of characters to look at
> and potentially recruit, without other players coming and taking all
> the recruitable characters."

So:

1. **Per player.** Each player sees their own candidates on a recruiter's
   notice. What one player hires never changes another's list.
2. **Generated.** Candidates are made up (name, archetype, level,
   alignment, price), not only authored.
3. **They come and go.** Each candidate stays for a random while, then
   leaves and is replaced by a new one.

## Prior-art check

- **Recruiters (22c):** a configured room with a named notice and
  authored candidates by mob template (`Recruiters` in
  `modules/company/files/data-overlays/config.yaml`). `company recruit`
  lists them; `company recruit <id>` hires one. Free tutorial candidates
  are claimed once per account, keyed by template. **Hiring never uses a
  candidate up:** every player who hires Garrick gets their own Garrick.
- **A companion** (`internal/company.Companion`) is an id, a mob
  template, an archetype, a disposition (alignment, loyalty), and
  `MemberState` (level, experience, gear, gold). **Its name always comes
  from the template**; there is no name field, and custom names are
  deferred.
- **Alignment (21a):** a candidate further than `RecruitMaxGap` from the
  company average refuses (`company inspect` weighs it).
- **The clock:** rosters must read the round, never advance it.

## Decisions

### A. Rosters

- **A roster is per player, per recruiter room**, stored durably in the
  company module's registry, so it survives logout, restart, and
  copyover.
- **Size:** 3 generated candidates per recruiter (config
  `RosterSize`). **(recommendation applied)**
- **Coming and going:** each candidate has its own random stay, between
  half a game day and two game days (config `CandidateStayRounds`, a
  min and max in rounds). When it ends, that candidate leaves and a new
  one takes the slot. Stays are staggered, so a list changes a face at a
  time rather than all at once. **(recommendation applied)**
- **Lazy, not ticking:** a roster is brought up to date when it's read
  (`look`, `company recruit`, `company inspect`): lapsed candidates are
  replaced, from the stored round they arrived, and saved. Nothing runs
  per round, and the clock is only read.
- **Offline time counts:** the stay is in world rounds, so a player back
  after a week finds new faces.
- **Hiring** removes the candidate from the roster; the slot fills with a
  new candidate after a short random delay, so the notice doesn't
  instantly refill.

### B. A generated candidate

- **Archetype:** random among warrior, rogue, wizard, cleric, and
  ranger. A recruiter can weight or limit them in config (the Trappers'
  Post favours rangers).
- **Base template per archetype:** five new mob templates (a recruit
  warrior, rogue, and so on) wearing their archetype's starter gear, with
  `itemdropchance 0`, no loot, and no gold, per 22c's farming rule.
- **Name:** from name lists in config (given names plus an optional
  byname, "Hild Marrow", "Anselm of the Ford"). It's unique within the
  player's company and roster. The word to type is the given name, as
  with authored candidates.
- **Level:** the leader's level, −1 to +1, at least 1.
- **Alignment:** spread across the scale, so some candidates are marked
  "(won't join you)" and 21a's recruit gate matters.
- **Price:** by level and archetype (config `RecruitPriceBase`,
  `RecruitPricePerLevel`).
- **A line of character** from a small config list ("a scarred former
  caravan guard", "quiet, and quick with a sling"), shown on `look` and
  `company inspect`.

### C. Companions get their own names

- `Companion` gains **`Name`** and **`Description`**, set for generated
  recruits. The live mob takes them over the template's on every spawn
  and restore. Everything that names a member (`company`, formation,
  chemistry, combat text, GMCP, the battle summary) goes through the
  spawned mob or `MemberView`, so it follows. A test checks each display.
- **Several companions can share a base template** (two generated
  warriors). Any lookup that assumes one companion per template is fixed
  to go by companion id.
- Authored companions keep a blank `Name` and use their template's.

### D. Authored candidates

- **The tutorial's** Tamsin, Oswin (free, once) and Corvin (refused)
  stay authored and fixed: the lessons name them.
- **Settlement regulars** (Garrick at the Waymark Inn, Ysolde at the
  Trappers' Post) stay listed above the generated candidates, as
  well-known faces. **(recommendation applied; could be dropped)**

### E. On the notice (with 32a)

32a's room line lists the viewer's roster:

```
On the hiring slate by the hearth:
  Garrick Vane, warrior, level 3 (120 gold)
  Hild Marrow, ranger, level 2 (95 gold)
  Anselm of the Ford, cleric, level 3 (130 gold), won't join you
  Tobin Reyes, rogue, level 2 (90 gold)
Type company recruit <name>, or look <name>.
```

## Module

- `internal/company`: `Companion.Name`, `Description`; roster types and
  pure generation (seeded RNG in tests).
- `modules/company`: rosters in the registry, lazy refresh, `recruit`,
  `inspect`, `look <candidate>`, config parsing, name override on
  spawn/restore.
- Content: five recruit base templates; name, byname, and character-line
  lists in config.

## Invariants

- **The clock:** rosters read the round and never advance it.
- **Restart and copyover:** rosters and generated companions are durable
  on the company record; a restored generated companion keeps its name.
- **Locks:** roster reads and refreshes run on the game loop under the
  company module's own lock, never across a world call.
- **Isolation:** no roster state is shared between players.

## Acceptance criteria

- **Unit:** generation (name uniqueness, level range, price, archetype
  weights); staggered stays; lazy refresh across a long absence; hire and
  delayed refill.
- **Wiring** (shipped config, real commands):
  - two players at the same recruiter see different rosters; one hiring
    leaves the other's unchanged;
  - hire a generated recruit: it joins under its generated name, keeps it
    through logout, restart (`plugins.Load` reload), and copyover, and
    shows it in `company`, `look`, the formation, and GMCP;
  - two generated warriors in one company are both addressable;
  - rounds past a stay: `look` shows a new face;
  - a far-aligned candidate is refused by `company recruit`;
  - the tutorial's recruiters still list Tamsin, Oswin, and Corvin.
- **Player help:** `help company` (recruiting: rosters that change,
  per-player), `help recruit` alias; the Departure lesson mentions that
  settlements post their own recruits.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- Candidates as NPCs standing in the room.
- Renaming companions (the `Name` field makes it easy later).
- Pronouns for generated recruits: 29d.
- Recruiter reputation (standing changing the roster's quality).
