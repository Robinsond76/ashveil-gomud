# Phase 85: the bestiary feeds the chronicle

Owner steer (Robinson, 2026-10-06, full autonomy): decide, record each
decision with its reason, no questions. The world is temporary stock GoMud
(Robinson, 2026-10-06), so this phase is designed against the mechanics and
ships no content beyond what tests need.

The roadmap note was "bestiary onto chronicle". It goes back to two open ends:
phase 66 left `bestiary.KillsOf` as the one seam to switch the bestiary's
source to the chronicle "if the chronicle gains per-kill counts", and phase 76
recorded "the bestiary still reads `KillsOf`, not the chronicle". This phase
settles that question the other way round: the bestiary keeps reading the kill
tally, and **mastering a kind of creature becomes a chronicle deed**, so the
company's study of its foes shows in its history and in what towns say.

## What the player sees

- **On the kill that teaches a kind's habits** (the 6th kill of a kind; the
  3rd of a boss kind; the moment `bestiary` already announces "you now know its
  habits and weaknesses"), the chronicle gains one line:
  "The company learned the habits of the big rat at Old King's Road."
  A proper name reads without the article ("... the habits of Rodric ...").
  The existing `Bestiary:` tier-up line is unchanged; nothing new is printed in
  the terminal.
- **`chronicle`** lists the deed with the others; `chronicle lore` (also
  `mastered`, `mastery`, `beasts`, `beast`) filters to them. The web
  Chronicle tab's filter gains a **Beast lore** entry with no client change
  (the tab builds its filter from `chronicle.Kinds`).
- **Towns** can speak of it: a townsfolk line may now name the kind
  `mastered`. One test line ships for the test talker in Gossip Corner (room
  90015), e.g. "Heard {who} has the measure of {subject} now." It claims no
  effect.
- Nothing else changes: no new command, no new panel, no stat or price effect.

## Shape

- **`internal/chronicle`:** a new kind `Mastered` ("mastered"), label
  "Beast lore", filter words `lore`, `mastered`, `mastery`, `beasts`, `beast`;
  its `Prose`; and a per-kind cap on kept entries, `KindCaps =
  map[Kind]int{Mastered: 30}`. `Log.Add` drops the oldest kept entry of a
  capped kind once that kind holds more than its cap, before the usual
  `MaxEntries` trim. `Tally` (and so `Total`) still counts every deed.
- **`internal/bestiary`:** `MasteredDeed(mobID int, place string)
  (chronicle.Entry, bool)` builds the deed from the creature template
  (`mobs.GetMobSpec`): `Subject` the template's name, `Ref` `mob:<id>`,
  `Zone` the template's zone, `Place` the room title, no `Members` (a company
  deed, which townsfolk already reads as the leader's). False for an unknown
  template.
- **Recording:** `mobcommands.Suicide`, inside the existing tier-up branch
  (Training zone already excluded there), records the deed for each contributor
  `uid` when the new tier is `bestiary.Habits`. The tier is read from the
  template, as the tier-up line already is, so an encounter's boss and an elite
  share their template's thresholds and entry.
- **Townsfolk:** accepts `kind: mastered` in lines (it validates kinds with
  `Kind.Valid`, so this should need no code); one test line added to
  `modules/townsfolk/lines/test-lines.yaml`.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| The bestiary keeps reading the kill tally (`Character.KD.Kills`); the chronicle does not become its source. | A deed per kill is one or more lines per fight, the pattern the phase 76 review rejected: it would push boss kills, relics, rites and story choices out of the 300-line log and the towns' newest-80 window. The tally is already durable, per company and excluded from Training. Phase 66's seam stays as it is. |
| The bestiary feeds the chronicle instead, with one deed when a kind's **habits** tier is reached. | It is the one bestiary moment that is earned (6 kills, 3 for a boss) and happens once per kind per company: the tally only grows, so the tier crossing fires exactly once and needs no dedupe check. Lore (1st kill) would write a line for every new creature seen, mostly noise. Defences is a middle step with nothing to tell. |
| Kept mastery deeds are capped at 30 of the 300 lines, oldest dropped first; lifetime count kept in `Tally`. | Lifetime mastery is bounded by the number of creature kinds (106 templates in the stock world, many of them peaceful), which could still be a third of the log over a long career and arrive in bursts when a company enters a new zone. The cap guarantees at least 270 lines for other deeds. Readers that care (towns, 14-day window, newest 80) only read recent deeds, so dropping old mastery lines loses nothing they use. |
| No backfill for kinds already mastered before this phase. | Writing dozens of deeds at login would flood the log and stamp them with a false time. Old mastery stays visible in `bestiary`. |
| No deed for elite kills, named-foe kills or first sightings. | Elites are a runtime roll on ordinary spawns and turn up every few fights, so a deed each would be the per-fight pattern again. Bosses already write a `boss` deed on every kill. |
| No `why` (battle explanation) change. | `why` explains rolls that happened; bestiary knowledge does not change a roll. Adding a bestiary line there would claim an effect that does not exist (the project rule). |
| No opinion, bond, blessing or awakening effect. | Opinions match member keys (a mastery deed names none); awakenings read `boss` deeds only; blessings evaluate on every deed but none counts `mastered`. Adding a perk would be a new feature, out of scope. Text claims nothing beyond what is written. |
| The prose names the kind with "the" unless the name starts with a capital letter. | Template names carry no article ("big rat") but some are proper names ("Rodric"). |
| The deed refreshes the web Chronicle tab the same way other deeds do. | Consistency with every other kind (the phase 68 review kept this). It happens at a kill, as a boss deed does. |

## Must not

- Write a deed for any kill that does not cross into the habits tier, or more
  than once per kind per company.
- Name or describe a kind the company has not beaten: the deed is written only
  after the kill that teaches it, from the template the bestiary already shows.
- Claim an effect in prose, help or town lines ("folk sleep easier", a
  discount, a bonus in battle): the deed is a record only.
- Advance world time or touch shared state beyond the chronicle log.

## Acceptance (for the build)

- `internal/chronicle`: prose for both name shapes; `KindByWord` for each
  filter word; `Log.Add` keeps at most 30 `mastered` entries, drops the oldest
  first, leaves other kinds alone, and `Tally` keeps counting
  (`TestMasteryDeedsAreCappedWithinTheLog`).
- Real entry point: a kill through `mobcommands.Suicide` that takes a kind from
  5 to 6 kills records exactly one `mastered` deed with `Ref` `mob:<id>`; the
  5th and 7th kills record none; a boss template records at its 3rd kill; a
  Training-zone kill records none (`chronicle.Memory` provider).
- Townsfolk: the test line is told through the real idle turn for a mastery
  deed (extend the existing `TestATalkerMentionsADeedThroughTheRealIdleTurn`
  pattern or add a case).
- Help: `help chronicle` lists beast lore and its filter word; `help bestiary`
  says mastering a kind is written in the chronicle; the Combat tutorial's
  bestiary hint (`modules/tutorial/stages.go`) mentions it in a clause. Both
  pages render through `help` and `TestTutorialHelpPointersExist` passes.
- Gates: `make generate`, `make validate`, `go test -race ./...`, and a
  browser check that the Chronicle tab's filter shows Beast lore.

## Not done

- Switching the bestiary to read the chronicle (rejected above).
- Per-town memory of which beasts a company hunts near them, prices or
  reputation by mastery.
- A blessing for mastering many kinds (a later phase may add one; `Total`
  already counts them).

## Build notes (2026-10-08)

Built as designed; no design change. Small decisions:

| Decision | Reason |
| --- | --- |
| The cap lives in `Log.dropOldestOver`, run before the `MaxEntries` trim, and counts only kept entries of the kind. | Keeps `Add` simple and leaves other kinds untouched; the tally is bumped after, so it still counts every deed. |
| Prose adds "the" unless the name starts with a capital letter or already begins with an article (`theName`). | Covers proper names and templates such as "the Pale Wolf" without "the the". |
| The deed is recorded inside the existing tier-up branch only when the new tier is `Habits`. | The tally only grows, so the crossing fires once; Training is already excluded there. |
| The web filter needed no change: the Chronicle tab builds its buttons from the labels on the deeds present, so Beast lore appears once the first deed is kept. | Checked in `window-company.js` (`chronicleKinds`); the design said it builds from `chronicle.Kinds`, but the label arrives on each entry. |
| The shipped test line reads "Heard {who} has the measure of {subject} now." and a common name reads without an article. | Test content only (temporary world); the replacement world writes its own. |
| Help: lore line and 30-line note in `help chronicle`; a paragraph in `help bestiary`; a clause in the Combat tutorial's bestiary hint. | Per the design's acceptance list. |
