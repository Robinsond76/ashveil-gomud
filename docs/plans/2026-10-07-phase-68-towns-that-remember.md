# Phase 68: Towns that remember

Town NPCs tagged as talkers read the company's chronicle ([phase 63](2026-10-07-pillars-phases.md)) and say a fitting line once per deed per player, then fall back to a weather or time-of-day line, or silence. Spec: [Pillars phases](2026-10-07-pillars-phases.md) §68. The current world is temporary stock GoMud, so only one test talker and a few test lines ship; the replacement world brings its own mobs and lines.

## Shape

- **Rules** in `internal/townsfolk` (GoMud-free): the `Line` shape and validation, `Catalog.Choose`, the `NPC`/`Context` inputs and the `Provider` seam (`townsfolk.Speak`). **Module** in `modules/townsfolk`: what each player was told (saved), the line data, the `townsfolk`/`renown` command and the `Company.Townsfolk` GMCP message.
- **A talker** is a mob template with `townsfolk: [tag, ...]`. `internal/hooks` `HandleIdleMobs` asks `townsfolk.Speak` during the mob's idle turn; if it has a line, the NPC says it instead of its usual idle command (`sayto @<user id> ...` for a deed, `say ...` for a state line).
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
| The idle-chatter limits (cooldown, per-line memory) still apply; the hook asks only when `ChatterReady` and a listener is present, and confirms (records and saves) a deed only when the limits let its line through, before the queued line runs. A held-back line leaves the deed untold (review fix). | A town talker must be as rare as any idle NPC; a crash may drop a line but never repeats one. |
| A talker speaks in place of its usual idle command on a turn it has something to say. | No second chatter source. A talker with nothing to say behaves exactly as before. |
| Weather lines use the weather module's condition name for the NPC's zone; time lines use the shared clock (read-only). | The spec's fallback; no new weather state. |
| Marks are company flags (`storyevents.SetCompanyFlag`); a line's `sets` leaves one when told, and `flag`/`no_flag` read them, so a story event and a townsperson can build on each other. | This is the "company marks" hook phase 60 left. A failed mark never blocks the telling. |
| `member_tag` reads `storyevents.TagsFor` for the members named in the deed. A deed that names nobody (the game records boss kills, relic finds and mercy answers that way) is the leader's: `{who}` names the leader and `member_tag` reads the leader's tags (review fix). | The seam phase 72 backgrounds fill: a town can speak differently of a company with a given background with no change here. Nothing registers tags yet. |
| No price effects. | The brief only allows ones that respect the economy rule; the spec has none, and a price change for fame would be a profit lever. Left out on purpose. |
| Talk is not kept across a purge and is snapshotted by the admin test area. | `purge_coverage_test` and `userstate` rules. |
| Shipped content is test-only: one talker (mob 90301) in test room 90015 (Gossip Corner, southwest of the hub) and ten test lines. | The world is temporary. IDs are well clear of master's max (mob 261). |

## Acceptance

- A deed produces a line once per player (`TestADeedIsToldOncePerPlayerAndSurvivesARestart`, through the real idle turn in `TestATalkerMentionsADeedThroughTheRealIdleTurn`); no line repeats within the chatter window (the engine's limiter, exercised in the same test).
- `help townsfolk` (aliases `renown`, `town memory`, ...), listed under the road, linked from `adventure`, `chronicle`, `company` and `webclient`; Departure tutorial hint; `TestTownsfolkHelp` and `TestTutorialHelpPointersExist`.
- Web: the Chronicle tab shows "What the towns say of you" above the deeds (said, not yet spoken of, marks) over GMCP `Company.Townsfolk`; checked in `scripts/browser/dock-windows-check.mjs` at desktop and phone width.

## Review (Opus review thread, independent reviewer subagent plus lead checks)

Accepted and fixed:

1. Real boss, relic and mercy deeds name no member, so lines read "Word is The company put down ..." and `member_tag` could never match them. Such a deed is now the leader's (`TestACompanyDeedIsTheLeaders`; the real idle-turn test uses the real deed shape).
2. A deed was marked told (and its mark left) before the chatter limits decided, so a held-back line used it up; e.g. the same boss slain twice within the hour. Now recorded only on `Speech.Confirm` after the line is let through (held-back case in `TestATalkerMentionsADeedThroughTheRealIdleTurn`, `TestAnUnconfirmedTellingLeavesTheDeedUntold`).
3. Gossip Corner's exit back to the hub pointed southwest instead of northeast (`TestEachHubExitLeadsBack`).
4. `member_tag` compared case-sensitively while story events' tag requirements do not; now case-insensitive.
5. The talker targeted the player by name; now `@<user id>`, which is unambiguous.
6. A new deed did not refresh the web view's "not yet spoken of" list; `chronicle.OnRecord` now pushes it (`TestANewDeedRefreshesTheView`).

Rejected or recorded: the town block is hidden when the chronicle is empty (towns have nothing to say without deeds); each telling opens the Company window as each chronicle deed does (kept consistent); a chronicle load failure restarting seqs could hide new deeds for that session (the chronicle's own load-error path, out of scope); a chronicle query per listener on a ready idle turn (cheap at this scale). **Price effects:** the spec (§68) names none, and a fame discount or markup would be a profit lever against the economy rule, so leaving them out matches the spec. The phase 72 seam (`member_tag` via `storyevents.TagsFor` and the chronicle's member keys, which use the same `leader`/`companion:N` keys) works as described.

## Not done

- A library of real lines and talkers (the replacement world's job).
- Per-town memory (each town knowing only what happened near it), price or reputation effects, and tags from backgrounds (phase 72 registers them).
