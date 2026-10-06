# Phase 38a: The Witch

Source: [level impact design](../designs/2026-10-05-level-impact-class-power-design.md) §3.
A sixth starting class that takes enemy turns away instead of dealing damage.

## What shipped

- **Statuses** (`internal/status`): Asleep (1109), Paralyzed (1110), Blighted
  (1111). Asleep and Paralyzed lose every action while live; the first damage
  wakes a sleeper (`status.Wake`), and blows against a sleeper hit 25 points
  more often. Paralysis is not broken by damage and removes dodge and block.
  Blighted halves the healing a target receives.
- **Hexes** (`internal/hexes`, `_datafiles/world/default/spells`): Slumber L1,
  Earthbind L3, Leaden Curse L5, Miasma L7, Binding Hex L10, Curse of
  Frailty L12, Dread Whisper L15, Blight L18, in the `hexcraft` school.
  Reach is 1 foe, 2 at L8, 3 at L16, 4 at L24 and the whole group at L30.
- **Resist and immunity**: `hexes.LandChance` sets the Witch's Mysticism
  against the foe's (Vitality for Binding): 65% at equal stats, clamped
  25–90, a boss 25 lower and holding a status half as long. A runtime ledger
  keeps a foe from being re-hexed for 2 combat rounds after a status ends, so
  no foe is held more than half of a fight.
- **Controller role** (`internal/strategy`): the Witch's default role. It
  casts the first known, affordable hex that has a worthwhile foe, aimed by
  `strategy.HexTargets` (winding-up foe, then a chanting caster, then the
  nearest), falls back to Withering Hex, then swings.
- **Class data**: the `witch` archetype (HP 1 and 0.55 a level, Attack 0.7,
  Evasion 0.75, light armor, 40 mana and 10 a level, weather utility, kit),
  a Witch recruit (mob 94) in the company templates, and `ShippedFor`.
- **Help and tutorial**: `help witch`, `help hexes`, the statuses page, and
  every page the class made stale; tutorial hints in the Departure,
  Practice Yard and creation lessons.

## Decisions

- Poison damage can wake a sleeper; sleep never holds a foe more than 50%.
- A `Boss` flag on mob templates marks bosses for the resist; nothing sets
  it yet (a later phase's encounter data will).
- A manual `cast` of a multi-target hex is capped at the Witch's reach;
  row limiting applies to automatic casts only.

## Balance (timeboxed)

`TestPhase38aWitchAgainstWizard` (opt-in, `ASHVEIL_BALANCE=1`) swaps one
warrior of the default company for a wizard or a witch against an equal
five-foe group, 30 fights a cell:

| Level | Warrior wins | Wizard wins | Witch wins |
|---|---|---|---|
| 5 | 56% | 40% | 23% |
| 10 | 63% | 10% | 23% |
| 20 | 46% | 13% | 13% |

The Witch lands no worse than the Wizard (noisy at 30 fights). Both lose to the
warrior in this mirror because about half of a caster's chants are broken by
blows (Witch 4–5 of 7–14 begun). No retune: the mirror has no healer or
resist-poor foe, so it understates a control class, and chant breaking is a
shared caster rule (35b). Recorded, not chased.
