# Phase 60: Story events

Short scripted scenes with choices (spec: [Pillars phases](2026-10-07-pillars-phases.md) §60). A page of text opens when a trigger fires; choices may need a member's skill, class, personality, alignment, level or an item; the best qualified member takes a gated choice and is named; outcomes change wounds, ailments, needs, supplies, gold, loyalty and flags, start a named fight, or carry the company to another room. Phases 68 (towns that remember) and 72 (backgrounds) hook in through tags and flags. Tone follows [the style bible](../designs/world-style-bible.md); the three shipped scenes are test content only (the world is temporary).

## Shape

- **Rules** in `internal/storyevents` (types, YAML parse, validation, `Best`, risk, catalog, trigger lookup, movement seam, tag-source seam). **Module** in `modules/storyevents` (saved state, triggers, `event` and `choose` commands, effects on the world, GMCP `Event`). Data: `modules/storyevents/events/*.yaml` (embedded) plus an optional `<DataFiles>/events/*.yaml` in the world (disk wins by file name).
- Triggers: `room` (an exact room), `tag` (any room with the tag; a lair door is a tagged room), `arrival` (a journey ends in a room or zone), `camp` (a camp rest finishes). Each has `chance`, a level range, once-per-company or a cooldown, and required or forbidden flags.
- Outcomes: `wound`, `ailment`, `need`, `item`, `lose_item`, `gold`, `loyalty`, `battle`, `move`, `flag`; targets `actor`, `leader`, `all`, `random`.
- Web: a modal (`window-event.js`) with number keys and a Continue button on ended scenes. Telnet: `event` rereads the page, `choose N` answers.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| Four triggers: room, tag, arrival, camp. | They need no change to journeys. `arrival` is a journey's end; mid-journey interruptions would need expedition changes, so legs are never interrupted. |
| "Room or exit opened" is a per-company `move` plus company `flag`s, not a global exit change. | A shared exit change would affect other companies; flags let later phases (68) gate content per company. |
| Commit before apply: the answer is saved, then outcomes run. Idempotent outcomes carry op ids `story:<event>:<ops>:<n>`. | A crash loses an outcome and never doubles one, the same stance as the camp grant. |
| Free choices fall to the leader; gated choices go to the best qualified member (highest rank, then level, ties to the earlier member). | Predictable, and the player sees who acted. |
| Risk is shown in words ("a small risk", "chancy", "a long shot"), never numbers. | Status-in-words rule of the style bible. |
| A page whose room the company left (death, teleport) is dropped, not marked done. | The moment passed; the scene can open again later. |
| The company is held in place (`go`, `walkto` refuse) while a page waits. | Otherwise a company walks away mid-scene; the refusal tells the player how to answer. |
| An encounter, expedition or camp block holds a scene back until the company is free. | No scene in the middle of a fight or journey. |
| Gold is capped at ±500, items at 10, wound at 50%, need at 60, loyalty at 25, risk at 95%, six choices, battle 2-5 foes. | Keeps data from breaking the economy; validation drops events that exceed them. |
| Every page needs a free choice; the page graph is acyclic and fully reachable; risks need fail text or fail outcomes; a battle ends the event. | A scene can never trap a company or loop; one outcome path stays simple. |
| Picture keys match `^[a-z0-9][a-z0-9-]{0,47}$`, drawn from `static/images/events/<key>.png`. | Art can follow without code changes. |
| Tag sources (`storyevents.RegisterTagSource`) and company flags are the hooks. | Phase 72 backgrounds add tags per member; phase 68 reads and sets flags. |
| Broken events are skipped with one warning each, not fatal. | A bad data file must never stop the server. |

## Acceptance

- Three test scenes in the Test Area (rooms 90011-90014, reached from the hub 90001): a climb (`gorge-descent`: skill-gated rope, a fall wound, a move to the ledge), a stranger (`stranger-at-the-fire`: share food, a healer, robbery that starts a fight), a shrine (`burned-shrine`: devout and alignment gates, an ailment on failure, a 30-minute cooldown).
- Each outcome kind is tested through the real trigger in `modules/storyevents/wiring_test.go` (a real `go` step through the loaded world); `storyevents_test.go` covers trigger rules, holds, restart, stale pages, save-before-apply, bad answers, purge and userstate.
- Help: `help events` (aliases `event`, `scene`, `choose`, …), linked from `adventure`, `travel` and `camp`; Departure tutorial hint; `TestEventsHelp` and `TestTutorialHelpPointersExist`.

## Not done

- Authoring a real library of scenes (the replacement world's job).
- Interrupting journey legs; global exit changes; art for the picture keys.
