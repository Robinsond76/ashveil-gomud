# Play-Test Feedback (2026-09-28) — Spec Roadmap

The owner's notes from play-testing the tutorial and the first areas after
Phase 29b2, what the code does today for each, the owner's decisions, and
how the work is split into phases. Each phase gets its own design doc
before any code, per `CLAUDE.md`.

## The notes, and what the code does today

Researched on `4cb2faf` (2026-09-28).

| Note | Today |
|---|---|
| Company load is 200 kg with no one in the company | Flat `CapacityKg` 200 per company plus one mount's capacity; not tied to company size (Known issues, Phase 28). |
| Load should come from horses, saddles, backpacks | Only one mount per company (Phase 10). Nothing else raises capacity. |
| How does XP work for a company? | Only players earn XP. A kill's XP goes to the players who damaged it; GoMud *player* parties split it evenly (`internal/mobcommands/suicide.go`). **Companions never gain XP**: their saved experience is only set at spawn or reset (`modules/company/runtime.go`), so they stay at their starting level. |
| `exp` should show the company | Shows only the player. |
| Formation names in the web UI are hard to read | The grid is `0.68em` in dim cells (`window-party.js`, `.company-formation`). |
| See every member's equipment together | `company gear` shows one member at a time; `cargo` is separate. |
| `♥friend` on `look` | GoMud's `charmed` adjective (`internal/characters/formattedname.go`); companions are charmed mobs, so they carry it. |
| Companions' "enters from the west" lines | Each companion walks as its own mob (`internal/mobcommands/go.go`), so each arrival and departure prints a line, to the leader and to everyone else in the room. |
| The drink message | The Hydrated buff's `onStart` (`buffs/34-hydrated.js`). `drink` already appends a thirst status line. |
| A company inventory with cargo on the same screen | No such screen. |
| One command for everyone to eat and drink | `eat`/`drink` feed the player or one named companion, from the player's own pack only; cargo is never used. |
| Cargo UI with hover actions like the gear tab | The gear window (`window-gear.js`) has tabs, tooltips, and a click menu; there is no cargo view. |
| Show the camp and campfire in the room | Camps and lit fires exist (`modules/camping`, the fire warms the room) but `look` shows neither. |
| "a pack of 4 creatures": how do I attack it? | Mixed groups are labelled "a pack of N creatures" (`internal/rooms/mobparty_display.go`); `attack` matches a member's name only; nothing tells the player that. No way to see an enemy's formation. |
| A wizard doesn't cast in combat | Players swing automatically; spells only by `cast`. Planned in 30c, with the player's own role left open. |
| How do I delete a character? | No player path. `users.DeleteUser` exists, is never called, and leaves every module's state behind. |
| Tutorial NPCs aren't obvious | Recruit candidates aren't mobs in the room at all: they exist only in `company recruit`'s list (`Recruiters` config), so nothing shows them. |
| A tabbed Company/Comm/Combat panel under the map | The Company section sits inside the Party window (26b); the battle panel (31) is a separate proposal. |
| Replay the tutorial without new characters | Finishing or skipping is final (`modules/tutorial/tutorial.go`); recruits and the graduation reward are once only. |

## Owner decisions (2026-09-28)

1. **Company XP:** every member present gets the **full** award, not a
   split.
2. **Load:** one **weight** limit only (no item-count limit). Horses,
   saddles, and backpacks raise it.
3. **Tutorial replay:** a fresh level-1 character with nothing runs the
   course; on leaving, the player is back **exactly** where they were and
   keeps **nothing** from the replay.
4. **Targeting:** only a **group** can be attacked, by the group's name.
   Naming a member is not a way to start a fight.
5. **Order:** as recommended (below).
6. **Recruits:** listed on the room's notice, per player, randomly
   generated and changing over time, so every player always has
   candidates of their own (32a2).
7. **Replay character:** copies the real character (name, race,
   archetype) at level 1 with nothing, and goes straight into the course.

## Phases

| Phase | Scope | Spec |
|---|---|---|
| 32a | Company polish: no `♥friend` on companions; one arrival/departure line for a company; no drink flourish; camp and fire in `look`; recruiters listed in the room; readable formation grid | [32a design](2026-09-28-phase-32a-company-polish-design.md) |
| 32a2 | Per-player recruit rosters: generated candidates (name, archetype, level, alignment, price) that come and go on each player's own notice; companions get their own names | [32a2 design](2026-09-28-phase-32a2-recruit-rosters-design.md) |
| 32b | Tutorial replay: a throwaway level-1 character, back to the real one on leaving | [32b design](2026-09-28-phase-32b-tutorial-replay-design.md) |
| 32c | Enemy groups: named and described groups, `attack <group>` only, `scout` to see a group's formation before a fight | to write |
| 32d | Automatic combat for the player and companions: act by archetype role, casters cast with real mana (the casting slice of 30c, pulled forward) | to write |
| 32e | Company experience: every member present earns the full award; `experience` lists the company | to write |
| 32f | Company logistics: capacity from members, backpacks, horses and saddles; `company inventory` (gear, packs, cargo); `company eat`/`drink` | to write |
| 32g | Web company dock: a tabbed section under the map (Company: Cargo, Status, Camp; Comm; Combat), cargo hover actions | to write |
| 32h | Character deletion: confirmed by password, purging every module's state | to write |

### Build order (recommended, accepted 2026-09-28)

1. **32a → 32a2 → 32b** — 32a and 32b are small and make play-testing
   easier; 32a2 builds on 32a's notice listing.
2. **32c → 32d** — combat you can start and watch play out is worth more
   than polishing its text, so these go before 29c. 32d takes 30c's
   casting and role-driven action; 30c keeps tactics settings, guards,
   and enemy personalities.
3. **32e** — small; pairs with 32d, since companions who fight should
   grow.
4. **32f → 32g** — the dock's buttons send 32f's commands, so 32f comes
   first. 32g's Combat tab replaces Phase 31's separate window: 31's
   grids move into the tab.
5. **32h** — reuses 32b's per-module purge.
6. Then the combat roadmap resumes at **29c**, minus what 32d and 32g took
   over.

## Cross-cutting

- **A per-user purge.** 32b (throwaway characters) and 32h (deletion) both
  need every module to drop a user's state. 32b adds it once (see its
  design), and 32h reuses it.
- **Player help** for every phase, per `AGENTS.md`; each design lists its
  pages.
- **The review gate** in `CLAUDE.md` applies to each phase.
