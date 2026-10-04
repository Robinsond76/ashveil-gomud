# Phase 30g6 — stat edges and tuning: verification

Branch `phase-30g6-stat-edges`, on `phase-30g6a-fixes`. Design: the
[30g6 amendment](../designs/2026-10-04-phase-30g6-amendment.md), decisions A,
B, E and F, with the acceptance list there. Full table:
[measurements](2026-10-04-phase-30g6-measurements.md). Supersedes the first
candidate (`phase-30g6-tuning`, kept on `origin` for its history).

## What changed

- **Stat edge (A):** every opposed chance reads `(a − d) / StatEdgeSpan`,
  held to ±1: hit and crit chance move from an even value toward their
  bounds; dodge, parry, bash, the crit multiplier and the Strength part of
  damage grow from their minimum with an advantage; block moves either
  way; tackle is 40% even, 20–80%, for automatic and manual tackles alike.
  Equipment ranking weighs stats per point of span.
- **Damage (B):** `DamageBonusMin + Strength × DamagePerStrength +
  advantage × DamageEdgeMax`, held to the bonus range.
- **HP (E):** class rates carry levels to `HPFullLevels`; after that each
  archetype gains `HPAfterFull × its rate ÷ DefaultHPPerLevel`.
- **Healing (F):** Minor Heal 2d3 + 1 per caster level; Minor Heal All
  2d3 + 1 per 2 caster levels to each patient.
- **Config, admin and help:** new keys `StatEdgeSpan`, `ToHitEven`,
  `CritChanceEven`, `DamagePerStrength`, `DamageEdgeMax`, in the admin
  config page and wizard; the progression editor's defaults. Help: new
  indexed `stat-edge` (aliases `hit-chance`, `to-hit`, `crit-chance`, ...),
  updated `speed`, `smarts`, `perception` (now indexed), `strength`,
  `defense`, `abilities`, `brawling`, `attack` (its stale hit formula),
  `progression`, `health`, `friendly-effects`, `combat`; the Practice Yard
  hint points to `help stat-edge`, `strength` and `progression`.

## Shipped values

| Key | Value | Key | Value |
|---|---|---|---|
| StatEdgeSpan | 40 | HPBase | 40 |
| ToHitEven (25–100) | 60 | HPPerVitality | 0.5 |
| CritChanceEven (5–30) | 15 | DefaultHPPerLevel | 2.5 |
| DamageBonusMin / Max | 8 / 20 | HPFullLevels | 10 |
| DamagePerStrength | 1.25 | HPAfterFull | 1.3 |
| DamageEdgeMax | 4 | Class rates | warrior 3, cleric and ranger 2.5, rogue 2, wizard 1.5 |
| TempoSpeedRef / Span | 2 / 20 | Heals | +1 / +½ per caster level |

HP at the harness's balanced investment, L10 → L60: default 65 → 136
(2.09×), warrior 70 → 154 (2.20×).

## Acceptance (all passed, 100 fights a cell)

| Level | Spread mirror median | Mirror wins | Kit wins | Focus wins (reported) |
|---|---|---|---|---|
| 1 | 12 | 54% | 43% | 69% |
| 5 | 13 | 43% | 51% | 69% |
| 10 | 14 | 44% | 60% | 77% |
| 30 | 14 | 53% | 61% | 85% |
| 60 | 13 | 56% | 45% | 83% |
| 100 (reported) | 17 | 56% | 58% | 86% |

- Enemy weakest and caster targeting cut a passive company's wins to
  8–33%, and remove significantly more of its health at every asserted
  level.
- Level 15 against level 10: 90 wins of 100 (median 11 rounds). Level 30
  against level 10: 100 of 100, 0.1 members lost per fight.
- Coordination (33i2) assertions pass.
- No stalls in any cell.

## Tuning notes

- A span of 10 left level 15 unbeatable by level 10: stats rise in lumps
  (level 15 has +2 Strength, +3 Smarts and +2 Perception over level 10,
  half of level 30's lead). Span 40 keeps each point modest; HP and
  Strength damage carry the level gap.
- Spanning the whole bonus range with the Strength advantage made a
  two-point lead worth a third of a blow: `DamageEdgeMax` (4) replaces it.
- Company focus wins more often at every level but not sooner; the owner
  decided it is reported, not asserted.
- Kit was a handicap until heals scaled (a cleric chanting gave up more
  than a flat 2d3 restored); with F it is level with the mirror.
- Limitations: starter kits at every level (no equipment catalog yet);
  balanced stat investment on both sides.

## Checks

After the review follow-up: `make generate`, `make validate`, `make js-lint`
and `go test -race ./...` passed (no Lua changed). The opt-in suite above ran
before the follow-up; the follow-up touched out-of-battle healing, help,
admin text and hit-chance rounding by float error only. The review outcome
is in `docs/PROJECT_STATUS.md`.
