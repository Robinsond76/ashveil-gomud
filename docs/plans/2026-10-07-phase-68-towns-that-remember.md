# Phase 68: Towns that remember

Town NPCs tagged as talkers read the company's chronicle ([phase 63](2026-10-07-pillars-phases.md)) and say a fitting line once per deed per player, then fall back to a weather or time-of-day line, or silence. Spec: [Pillars phases](2026-10-07-pillars-phases.md) §68. The current world is temporary stock GoMud, so only one test talker and a few test lines ship; the replacement world brings its own mobs and lines.

## Shape

- **Rules** in `internal/townsfolk` (GoMud-free): the `Line` shape and validation, `Catalog.Choose`, the `NPC`/`Context` inputs and the `Provider` seam (`townsfolk.Speak`). **Module** in `modules/townsfolk`: what each player was told (saved), the line data, the `townsfolk`/`renown` command and the `Company.Townsfolk` GMCP message.
- **A talker** is a mob template with `townsfolk: [tag, ...]`. `internal/hooks` `HandleIdleMobs` asks `townsfolk.Speak` during the mob's idle turn; if it has a line, the NPC says it instead of its usual idle command (`sayto <player> ...` for a deed, `say ...` for a state line).
- **Lines** are YAML (`modules/townsfolk/lines/*.yaml` embedded, plus `<DataFiles>/townsfolk/*.yaml`, disk wins by file name), strict-parsed, each validated; a bad line is skipped with one warning.
  - Deed line: `kind` (a chronicle kind), optional `ref` (`mob:12`; beats a plain kind line for that deed), `days` (window, default 14, max 90), `tags` (which talkers), `zones`, `flag`/`no_flag` (company flags), `member_tag`, `sets` (a mark), `text` with `{who}`, `{subject}`, `{place}`.
  - State line: `weather: [names]` and/or `time: day|night`, plus `tags`.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| Town memory is a query over the chronicle plus a per-player "told" list of deed seqs; no new deed tally. | The brief says build on the chronicle. A deed's `Seq` is stable and unique within a company, and the log keeps 300, so the told list is capped at 300 too. |
| The player is the company: the company key is the leader's user id, so "once per deed per player" is "once per deed per company". | That is how the chronicle already keys deeds. |
| "Once per deed per player" means once in total, by whichever talker speaks first, not once per NPC. | The spec's accept line. Different towns telling the same deed would read as repetition, the thing Robinson dislikes. |
| A deed stays tellable for 14 real days (a line may set 1-90). Deeds older than that fade. | "Recent chronicle entries". Real time only; nothing here touches the world clock. |
| Newest unheard deed first; the first listener (in the room's order) who has a deed this NPC can tell hears it. | Predictable; the NPC says one thing per turn. |
| A deed line is said to the player (`sayto`); a state line is said to the room. | A deed is about one company; a remark on the weather is for everyone. |
| The idle-chatter limits (cooldown, per-line memory) still apply; the module marks a deed told only when `ChatterReady` and a listener is present, and saves it before the NPC speaks. | A town talker must be as rare as any idle NPC; a crash may drop a line but never repeats one. |
| A talker speaks in place of its usual idle command on a turn it has something to say. | No second chatter source. A talker with nothing to say behaves exactly as before. |
| Weather lines use the weather module's condition name for the NPC's zone; time lines use the shared clock (read-only). | The spec's fallback; no new weather state. |
| Marks are company flags (`storyevents.SetCompanyFlag`); a line's `sets` leaves one when told, and `flag`/`no_flag` read them, so a story event and a townsperson can build on each other. | This is the "company marks" hook phase 60 left. A failed mark never blocks the telling. |
| `member_tag` reads `storyevents.TagsFor` for the members named in the deed. | The seam phase 72 backgrounds fill: a town can speak differently of a company with a given background with no change here. Nothing registers tags yet. |
| No price effects. | The brief only allows ones that respect the economy rule; the spec has none, and a price change for fame would be a profit lever. Left out on purpose. |
| Talk is not kept across a purge and is snapshotted by the admin test area. | `purge_coverage_test` and `userstate` rules. |
| Shipped content is test-only: one talker (mob 90301) in test room 90015 (Gossip Corner, southwest of the hub) and ten test lines. | The world is temporary. IDs are well clear of master's max (mob 261). |

## Acceptance

- A deed produces a line once per player (`TestADeedIsToldOncePerPlayerAndSurvivesARestart`, through the real idle turn in `TestATalkerMentionsADeedThroughTheRealIdleTurn`); no line repeats within the chatter window (the engine's limiter, exercised in the same test).
- `help townsfolk` (aliases `renown`, `town memory`, ...), listed under the road, linked from `adventure`, `chronicle`, `company` and `webclient`; Departure tutorial hint; `TestTownsfolkHelp` and `TestTutorialHelpPointersExist`.
- Web: the Chronicle tab shows "What the towns say of you" above the deeds (said, not yet spoken of, marks) over GMCP `Company.Townsfolk`; checked in `scripts/browser/dock-windows-check.mjs` at desktop and phone width.

## Not done

- A library of real lines and talkers (the replacement world's job).
- Per-town memory (each town knowing only what happened near it), price or reputation effects, and tags from backgrounds (phase 72 registers them).
