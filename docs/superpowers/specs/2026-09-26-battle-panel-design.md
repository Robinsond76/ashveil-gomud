# Potential Phase 29: Browser Battle Panel

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: proposed by the owner (2026-09-26); needs a design pass and plan.

## Goal

During a fight, the web client shows:
- the enemy party's 3×3 formation and the company's 3×3 formation, facing
  each other;
- who each fighter is targeting.

## Prior-art check (2026-09-26)

- **Company data is already sent:** the `Company` GMCP package (26b,
  `modules/gmcp/gmcp.Company.go`) carries every member's formation cell
  (`cell {row, col}`), status, and health. It goes on change, only to the
  leader, and is built from `internal/companyview` on the game loop.
- **Enemy data is already computed every round:** `hooks/combat_formation.go`
  resolves the live enemy party from 11a's `mobparty.Assemble`, which gives
  each enemy a formation cell.
- **Targets:** each mob's and player's current target is
  `Character.Aggro`.
- **Web client:** windows live in
  `_datafiles/html/public/static/js/windows/`. `window-party.js` holds the
  Company section and `window-tutorial.js` is the newest window. Strings go
  through `textContent`.

## Scope

1. **Server:** a new `Company.Battle` GMCP message, sent only to the leader,
   only while the company is fighting an enemy party, and only on change.
   - **Enemies:** name (with 28c ordinals), cell, a rough health band
     (unhurt, wounded, badly wounded, down) rather than exact numbers, and a
     fallen mark.
   - **Targets:** one arrow per fighter on both sides, from attacker to
     target key.
   - **End of fight:** an empty payload clears the panel.
2. **Browser:** a Battle window with two grids, both front rows facing the
   middle.
   - Target lines are drawn between the grids.
   - Hovering or tapping a fighter highlights its target and everyone
     attacking it.
   - A plain text list underneath (`Garrick → first cutthroat`) serves
     screen readers and narrow screens.
   - A hidden live region announces only a new target on the viewer, or a
     fall.

## Mock

```
        BANDITS                              YOUR COMPANY
  ┌──────────┬──────────┬──────────┐   ┌──────────┬──────────┬──────────┐
  │ captain  │ cutthr.1 │ cutthr.2 │   │ Aria     │ Garrick  │ Tamsin   │  front
  │ wounded  │ unhurt   │ unhurt   │   │ 8/8      │ 7/12     │ ✝ fallen │
  ├──────────┼──────────┼──────────┤   ├──────────┼──────────┼──────────┤
  │ bruiser  │ slinger  │          │   │          │ Oswin    │          │  mid
  │ unhurt   │ unhurt   │          │   │          │ 6/6      │          │
  ├──────────┼──────────┼──────────┤   ├──────────┼──────────┼──────────┤
  │          │          │          │   │          │ Ysolde   │          │  back
  └──────────┴──────────┴──────────┘   └──────────┴──────────┴──────────┘
  Targets:  Aria → captain · Garrick → captain · cutthr.1 → Garrick · slinger → Tamsin …
```

## Open decisions

- **Enemy health:** a band or exact numbers. Exact numbers leak mob stats;
  23b-style "peep" skill gating is an option.
- **Window placement:** its own window, or a tab of the Party window.
- **Other viewers:** whether non-leader players in the room (GoMud human
  parties) get the panel.

## Acceptance criteria

- **GMCP wiring test:** a real fight through `hooks.DoCombat`:
  - sends `Company.Battle` with both grids and correct targets;
  - updates it when a target changes or someone falls;
  - clears it when the party is beaten;
  - sends nothing to a non-leader.
- **Browser check** in Chromium, as in 26b: the grids render, the lines
  follow targets, the text list matches, and there are no `innerHTML` sinks
  for names.
