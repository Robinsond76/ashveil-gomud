# Phase 51: rest duties (execution plan and decisions, 2026-10-06)

Scope is phase 51 of `docs/plans/2026-10-06-outward-survival-phases.md`.
Owner standing rules apply: full autonomy (decide, record why), the world is
temporary, difficulty comes only from zone level, nothing gathered or bought
resells for profit.

## Decisions

1. **Duties are extra hands on top of today's automatic specialists.** The
   automatic work (best watch, forage, vigil, kit healer, doll mending,
   flask brewing, auto-sharpen) is unchanged and free, so a player who never
   sets a duty sees no change (the spec's acceptance rule). A duty adds a
   worker and costs that member their sleep. Why: replacing the automatic
   work would make assigning a duty a trap, and "default reproduces today"
   would need a second hidden mode.
2. **Cost of a duty:** any member on a duty misses the Rested buff of that
   camp rest; a watcher's Fatigue also stops at 75 (Ready, never Rested), by
   a ceiling in survival's rest recovery (`ApplyCompanyRestRecoveryCapped`,
   never lowering anyone). Why: the spec says a watcher ends Ready, and a
   work duty with no cost would be strictly better than sleeping.
3. **Standing assignment, locked per rest.** `Camp.Duties` persists until
   changed; `RestSession.Duties` copies it, for the members at the camp, when
   the rest starts; the rest settles from the locked copy. Why: restarts
   never re-roll, and the player sets it once.
4. **Watch** stacks independently: each watcher adds
   `max(WatcherBasePct 20, own Camp Watch level x WatchPctPerLevel)`, combined
   as 1 - product of misses, with the specialist and bells as before. A
   specialist who is also a watcher counts once. New per-member lookup:
   `archetypes.MemberUtilityLevel`.
5. **Tend:** each tender has the packed surgeon's kit treat one more lasting
   wound (same `FieldSurgery` rules: a Tend healer with mana, one kit use),
   else sharpens one member's blades with one whetstone use (the sharpen
   rules, limited to one member), else reports nothing to tend.
6. **Forage:** each forager forages from the zone table with their own level,
   under the existing 15 minute forage cooldown and the same food table.
   Cooldown-blocked foragers get a "picked over" line. (51 review: the finds
   do sell, about 1.7 gold a find at a shopkeeper's 25%; kept, see the
   review in PROJECT_STATUS. The company's best forager on the duty forages
   once.)
7. **Cook:** each cook makes one dish with their own Cooking from the pack
   and cargo (`cookDish`, the camp cook path without the fire check: the
   rest was the fire). Dishes carry phase 50's meal buffs when eaten; duties
   add no meal logic of their own.
8. **Brew** is offered to Alchemists only. Alchemists already brew on their
   own at every rest end, so the duty's real effect is priority: brewers are
   served first when reagents are short. Help says so plainly.
9. **A spoiled rest settles no duties** (the raid broke it), matching Rested
   and the camp rewards.
10. **Banter hook:** `CampingModule.onDuties(leader, duties)` is called when a
    rest starts with duties locked. Nothing is wired to it yet; phase 49's
    lines can use it later without touching the duty code.

## Tasks

- [x] `internal/camping/duties.go`: duty model, `Camp.Duties`, `RestSession.Duties`.
- [x] Survival ceiling, per-member utility level seam.
- [x] `modules/camping/duties.go`: `camp duties`, locking, watch/tend/cook/brew/forage settling, status lines, Camp tab rows.
- [x] GMCP `Company.Camp` duties payload and the web Camp tab picker.
- [x] Help and tutorial: `help camp duties` (indexed, aliased), `help camp`, `help campwatch`, `help vigil`, `help forage`, `help cooking`, `help webclient`; tutorial camp lesson hint.
- [x] Tests: duty model, command and persistence, watch stacking, ceiling, Rested exclusion, tend, cook, forage cooldown, brew priority, hook, Camp tab rows, GMCP, help.
