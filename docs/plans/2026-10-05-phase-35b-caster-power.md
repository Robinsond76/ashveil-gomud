# Phase 35b — caster power, mana and recovery

Implements section 2 and sections 4.2–4.5 of the owner-approved
[level impact and class power design](../designs/2026-10-05-level-impact-class-power-design.md),
and asserts its zone-band targets. Branch and worktree:
`phase-35b-caster-power`, from master after 35a merges.
Read the [phase 35 handoff](2026-10-05-phase-35-handoff.md) first.

## Goal

- Spells a character owns never fizzle in battle.
- Spells and abilities grow with level and Mysticism.
- Casters have large mana pools that refill **only** through a camp rest, an
  inn stay or mana draughts.
- Healers patch the company up after battle with that mana.
- Passive HP recovery trickles only up to 50% of max HP.
- Easy fights leave fewer wounds.
- The zone-band harness cells from 35a pass the design's targets.

Out of scope:
- the Witch and new hexes (38a);
- new martial abilities such as Shield Bash, Feint and Pinning Shot (specified
  with talents and routes in 38b);
- converting Withering Hex to damage over time (38a);
- Scribe (36a).

## Code context (master after 35a)

- **Fizzle.**
  - Players: `internal/hooks/NewRound_DoCombat.go` ~L235 checks
    `roll >= successChance` with
    `successChance = Character.GetBaseCastSuccessChance(spellId)`
    (`internal/characters/character.go` ~L198:
    100 − difficulty + proficiency + Mysticism/5 + statmods).
  - Mobs (companions and enemies): ~L859 checks
    `util.RollDice(1,100) >= successChance`.
  - Both fail on 100 even at a 100% chance.
- **Spell formulas live in JS:**
  - `heal.js`: 2d3 + level;
  - `healall.js`: 2d3 + level×0.5;
  - `mm.js`: 1d6+2;
  - `sparks.js`: 1d3+1 per target;
  - `hex.js`: 2d6+2;
  - `tend.js`: 2d3 wound tending.
  - The script actor API has `GetLevel()`, `GetStat(name)`, `AddMana`,
    `GetManaMax`.
- **Go duplicates of the heal formula:** `internal/wounds/wounds.go`
  `DefaultRules` (HealDice 2d3, HealCost 3) and `Healer.HealBonus` (level),
  used by `heal wounds` in `modules/company/wounds.go` (~L428).
- **Chant breaks:** `internal/interrupt.BreakChance(damage, maxHP, heavy)`,
  40–90%, 100 for heavy force; called from `internal/hooks/combat_interrupt.go`.
  `interrupt.Refund(cost)` returns half the mana of a broken chant.
- **Mana:**
  - `Character.RecalculateStats` sets
    `ManaMax.Mods = ManaBase + statmod + level×ManaPerLevel + Mysticism×ManaPerMysticism`.
  - Archetype HP resolves through `HealthGainPerLevel` / `HPArchetype`; mana
    has no per-archetype rate.
- **Regeneration:** `internal/hooks/NewRound_AutoHeal.go`, every 3rd round
  out of battle:
  - players call `Heal(HealthPerRound(), ManaPerRound())`;
  - companions use `regenCompanionVitals`;
  - enemies use `regenEnemyVitals` (33i2).
  - `ManaPerRound` and `HealthPerRound` are `1 + statmod`.
- **Mana refills:**
  - `Character.LevelUp` sets `c.Mana = c.ManaMax.Value`;
  - an inn stay calls `restoreVitals` (`modules/camping/tiers.go`, Well
    Rested tier only), which raises HP to the wound limit and fills mana;
  - **a camp rest restores neither today.**
- **Mana potion:** item 30014 "small blue potion" (buff 27,
  `27-minor_potion_mana_recovery`: 1d5 mana ×3), sold by Moilyn in Frostfang
  (`mobs/frostfang/50-moilyn_the_wizard.yaml`). `drink` and `use` already
  refuse in battle (`InBattle`).
- **Abilities:**
  - `internal/strategy/abilities.go` defines Tackle (`TackleChance`, stat
    edge), Opening Strike and Aimed Shot (a forced crit via
    `characters.BackStab`), applied in `internal/hooks/combat_abilities.go`.
  - Guardian guards come from `internal/battle/guard.go` (`MaxGuards = 2`).
- **Companion spells:** archetype `CompanionSpells` (healall and sparks at
  level 5). Players get `GrantSpells` once at class choice.
- **Wounds:** `internal/combat/combat.go` ~L675:
  - a crit that gets through calls `wounds.FromCrit`, with `w.Light` set for
    enemies;
  - `wounds.Crushing` makes light wounds.
- **Tactics:** the company healing threshold (30c1, 10–90%) and the 33e mana
  reserve.

## Implementation decisions

1. **Spell power has one source of truth in Go.**
   - Add optional `power` fields to `spells.SpellData`:

     ```yaml
     power:
       base: 5
       dice: 1d6
       perlevel: 1.25
       mysticismdiv: 4
     ```

     These are validated on load.
   - New pure package `internal/spellpower`:
     - `Roll(spell, level, mysticismAdj, roll)` returns
       `base + dice + floor(perlevel × level) + floor(mysticism / mysticismdiv)`;
     - `Range(...)` returns the min and max for display.
   - Expose `SpellPower(actor, spellId)` to scripts (`internal/scripting`, plus
     the DTS file `api_v1_scripting_dts.go`).
   - `heal.js`, `healall.js`, `mm.js`, `sparks.js` and `hex.js` call it in
     place of their own dice. Spells without `power` keep their scripts.
   - `wounds.Rules` and `Healer` read the same helper, so `heal wounds`, the
     level-up report and the harness can't drift.
2. **Shipped values** (design section 2b; verify with the harness and adjust
   within ±25% if a band misses):

   | Spell | base | dice | perlevel | mysticismdiv | cost |
   |---|---|---|---|---|---|
   | mm | 5 | 1d6 | 1.25 | 4 | 6 |
   | sparks (per target) | 3 | 1d4 | 0.75 | 6 | 10 |
   | hex (Withering Hex, direct for now) | 6 | 2d4 | 1.5 | 4 | 8 |
   | heal | 8 | 2d4 | 1.5 | 3 | 3 |
   | healall (per patient) | 4 | 1d4 | 0.8 | 6 | 6 |

   Costs stay as shipped. `tend` is unchanged.
3. **No fizzle for owned spells in battle.**
   - **Player:** skip the roll when the caster is in a battle
     (`battle.Current(userId)`) and the spell is in their `SpellBook`.
   - **Mob** (companion or enemy): skip it when `battle.Engaged(instanceId)`
     and the spell is in its known list.
   - Everywhere else, keep the roll but change both checks to
     `roll > successChance`, so 100% never fails.
   - Keep `difficulty` and proficiency for out-of-battle casts and the
     spellbook display.
4. **Difficulty sets chant fragility.** Change the signature to
   `interrupt.BreakChance(damage, maxHP, heavy, difficulty)`. It adds
   `difficulty / 5` points (Magic Missile +15) before the 40–90 clamp. Heavy
   force stays 100. Update the 30d1b help numbers.
5. **Per-archetype mana.**
   - Add `ManaBase` and `ManaPerLevel` to the archetype overlay:
     wizard 40/10, cleric 36/9 (and witch 40/10 when 38a lands).
   - Others use the progression defaults (`ManaBase: 4`, `ManaPerLevel: 1`).
   - Resolve them the way `HealthGainPerLevel` resolves HP: the player's
     archetype registry, else the companion's `HPArchetype`, else the default.
     Rename the field to `ClassArchetype` only if it stays a small diff;
     otherwise reuse `HPArchetype` and document it.
   - Enemies keep the defaults unless a template sets `manabase` or
     `manaperlevel`.
   - Maxima clamp and never refill on load (30g4 rule).
6. **No passive mana for players or companions.**
   - In `AutoHeal`, players heal HP only, and `regenCompanionVitals` gives no
     mana. Enemies keep `regenEnemyVitals` unchanged.
   - `ManaRecovery` statmods on players and companions are ignored; document
     it in the item help.
   - `LevelUp` stops refilling mana and only clamps it. This is the owner's
     rule (mana only from rest or draughts).
   - The interrupt refund stays: it returns mana that was spent, it isn't
     regeneration.
7. **Rest refills.** A completed camp rest (the Rested tier) now also calls
   `restoreVitals` for the leader and live companions: HP to the wound limit,
   full mana. The inn already does this. Camp rest wound rules (supplies)
   are unchanged. The rest marker is cleared exactly once, so a crash or
   copyover can't refill twice.
8. **The HP trickle.** Out of battle, `AutoHeal` and `regenCompanionVitals`
   heal HP only while `Health < 50% of HealthMax` (and below the wound
   limit), stopping at that line. Bleeding-out rules are unchanged. Enemies
   are unchanged.
9. **Mana draughts.**
   - Rename item 30014 to **minor mana draught** and add 30022 **lesser mana
     draught** and 30023 **greater mana draught**.
   - Each applies a one-trigger buff that restores 25%, 40% or 60% of the
     drinker's `ManaMax` (new buffs; buff 27 is replaced by the minor one).
   - Weights stay about 250 g.
   - Values (shop prices) are 150, 400 and 900 gold.
   - Moilyn sells the minor and lesser draughts. The greater draught is a
     loot-only item, tabled in 37.
   - The wizard starting kit keeps one minor draught. Use outside battle only
     (existing `InBattle` refusal).
10. **After-battle patching up.**
    - Battles end in `endBattle` (`internal/hooks/combat_battle.go`), called
      from `battlePass`; there is no event today. Queue a new
      `events.BattleEnded{UserId, Outcome}` there. The company module
      handles it only if the player is still out of battle when it runs:
      a waiting group may begin the next battle in the same pass, and then
      there is no patch. On that event, run the 30b `heal wounds` planner in
      a new **patch** mode for each company that took part:
      - heal spells only, no tend and no supplies;
      - each member is healed up to the company's healing threshold (30c1);
      - each healer stops at its 33e mana reserve.
    - Narrate with the existing heal lines, paced (29f).
    - Add `company patch` (alias `patch`) to run the same plan on demand out
      of battle; it is refused in battle, travel and rest.
    - Allied companies (33d) patch only their own members.
11. **Ability scaling.**
    - Opening Strike's forced crit adds `4 + level/2` damage.
    - Aimed Shot's adds `2 + level/3` (it is already a guaranteed crit, so the
      design's "+crit chance" is replaced by damage).
    - Tackle's chance stays on the stat edge, which smooth stats already raise
      each level. Its knockdown lasts 1 more round from level 20.
    - Guardian: `MaxGuards` becomes `2 + guardianLevel/10`, captured when the
      battle's guard state starts. The refill cadence is unchanged.
12. **Second option at level 3.**
    - Move `CompanionSpells` healall and sparks from level 5 to 3.
    - Add archetype `LevelSpells` for players (wizard sparks at 3, cleric
      healall at 3). These are granted when a level-up first crosses the
      level, idempotent through the spellbook (owning the spell is the
      marker), and never revoked by death.
    - Flip the 35a milestone entry for level 3 to shipped for casters. Martial
      classes' level-3 options stay "(coming)".
13. **Easy-fight wounds.** At the crit-wound site: if the attacker's level is
    3 or more below the target's, a crit leaves no wound when the target stays
    at or above 50% max HP. Otherwise it leaves a light wound (the same
    severity `wounds.Crushing` produces) instead of a lasting one. Enemies and
    the even and harder fights are unchanged.
14. **Harness assertions.** `TestBalanceZoneBands` (35a) now asserts the
    design's section 4 table:
    - band middle 2–3 foes: ≥97% wins, 0 fallen in ≥85% of fights, ≤30% HP
      lost, median 4–8 rounds;
    - band low 2–3 foes: ≥85% wins, ≤1 fallen;
    - band middle 4 foes: ≥90% wins, ≤1 fallen;
    - boss at band top: 70–85% wins;
    - company 3 under band low: 30–60% wins.
    Add `TestBalanceManaRun`: a full company runs 4 band-middle 3-foe fights
    in a row with after-battle patching and no rest between them. The casters
    end the third fight with mana above their reserve and are below 25% by
    the end of the fourth. Tune within the bounds in decision 2 and the mana
    rates. If targets can't be met inside those bounds, stop and report to
    the owner with the table.

## Tasks

- [ ] **Tests first:**
  - `spellpower` table tests;
  - YAML validation;
  - a 100% chance never fizzles, and an owned spell in a real battle never
    fizzles (player and companion) while out of battle it still can;
  - `BreakChance` difficulty boundaries;
  - archetype mana at levels 1, 10, 30;
  - no mana regeneration across many `AutoHeal` rounds (players and
    companions), and enemies still recover;
  - the HP trickle stops at 50%;
  - camp rest refills once, also across a crash or copyover marker;
  - level-up doesn't refill mana;
  - draught percentages, and refusal in battle;
  - patch mode respects threshold, reserve and ownership;
  - ability bonuses and the guard count by level;
  - `LevelSpells` grant once;
  - easy-fight wound rules.
- [ ] **`internal/spellpower`**, the `SpellData.power` schema, the script
  function, the spell YAML and JS rewrite, and `wounds.Rules` reading the
  helper. Run JS lint.
- [ ] **Fizzle and chant-break changes** in `NewRound_DoCombat.go`,
  `characters`, `interrupt` and `combat_interrupt.go`.
- [ ] **Mana:** archetype fields and resolution, `RecalculateStats`,
  `AutoHeal`, `LevelUp`, the camping rest refill, and save and load clamps.
- [ ] **Draught items and buffs,** Moilyn's stock, and the wizard kit.
- [ ] **Patch mode** in `internal/wounds` and `modules/company/wounds.go`, the
  battle-end hook, the `company patch` command, its pacing, and the GMCP
  vitals refresh.
- [ ] **Ability scaling,** guard count and `LevelSpells`; flip milestone 3 for
  casters.
- [ ] **Easy-fight wound rule.**
- [ ] **Level-up report spell lines:** for each owned scaling spell and
  ability, `name old range -> new range` (from `spellpower.Range`), appended
  to the 35a template.
- [ ] **Harness:** run the 35a table plus the assertions and
  `TestBalanceManaRun` at 100 fights a cell. Record them in
  `docs/plans/2026-10-05-phase-35b-measurements.md`, including fizzle-free
  cast counts and healing as a share of damage.
- [ ] **Help and tutorial:**
  - update `help mana`, `help spells`, `help spell`, `help cast`,
    `help heal`, `help health`, `help readiness`, `help camp`, `help inn`,
    `help interrupts`, `help abilities`, `help guardian`, `help wounds`,
    `help tactics` and `help progression` (spell scaling);
  - new `help patch`, plus a `help draughts` page with keywords and aliases
    (`mana potion`, `draught`), linked from `help combat` and `help mana`;
  - fix stale numbers (`grep -rn "2d3\|fizzle\|regain mana\|mana returns"`
    in the help folder);
  - add hints: Practice Yard (spells always cast in battle; watch mana),
    Campground (camp rest refills mana and HP), Departure (`company patch`,
    draughts);
  - render tests and `TestTutorialHelpPointersExist`.
- [ ] **Independent full-diff reviewer.** Verify and fix findings with
  regressions; record them in Project Status.
- [ ] **Final checks:** `make generate`, `make validate`, JS and Lua lint,
  `go test -race ./...`. Project Status entry, commit and merge.

## Acceptance

- Owned spells always resolve in battle. A 100% out-of-battle cast never
  fails.
- Magic Missile and Minor Heal match the table at levels 1, 10 and 30 (in the
  `spellpower` tests and live `heal wounds`).
- **Mana never rises for players or companions** except through a camp rest,
  an inn stay, a draught or a chant refund. Level-ups don't refill it.
- After a battle, healers heal to the threshold and stop at the reserve.
  `company patch` does the same on demand.
- HP recovers passively only to 50% of max.
- All zone-band assertions and the mana run pass, or the owner has the table
  and a decision.
- Help pages are accurate and indexed, and tutorial pointers resolve.
- No world-time change; save, copyover and allied ownership are intact.
