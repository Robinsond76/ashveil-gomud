# Phase 39e: The Beast Tamer

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md) §4.
The fifth neutral lineage: a handler who fights beside a living, bonded beast.
It reuses 39d's durable-record pattern (a record on the owner, a mob only for a
battle) but the beast, unlike the doll, takes its own turn.

## What shipped

- **Archetype and recruit**: `beasttamer` (HP 5 and 0.8 a level, Attack 0.8,
  Evasion 0.9, medium armor, no shield, whips, spears and daggers, perception 3 /
  vitality 3 / strength 2 / smarts 2, the Taming skill, a kit with the long whip),
  a recruitable Beast Tamer (mob 204) in the Dunmar templates, level-3 second
  class option. The long whip (item 10020) gained `weaponclass: whip`.
- **The beast** (`internal/beasts`, `internal/company/beasts.go`,
  `internal/characters/beast.go`): a record on the Tamer (`characters.BeastState`
  on a player's character, on `company.MemberState` for a companion: name, damage,
  wounded flag). When a battle opens `beastPass` stands each Tamer's whole beast
  as a charmed mob (mobs 200 wolf, 201 warhound, 202 war bear, 203 drake hatchling,
  race 26) registered as a member no company record holds; `company.FormationFor`
  lays it over the formation in front of its Tamer. It takes no company slot, and
  is dismissed (and synced back to the record) when the battle ends. It takes its
  **own turn** through the ordinary companion paths at 70% tempo, with
  teeth (1d8; the drake 1d6), 35% of a warrior's health and 90% Attack and Evasion
  rates. At 0 health it is **wounded**, out of the battle and of later ones until
  the company rests; it never dies.
- **Abilities** (`internal/hooks/combat_beast.go`):
  - **Sic** (rank 1): at the start of the Tamer's turn the beast is pointed at the
    Tamer's foe with +10 Attack on its strike this round. The Tamer still strikes
    its own blow, and a whip reaches like a polearm (`combat/reach.go`).
  - **Rally** (rank 3): the Tamer heals a hurt beast (below the company's healing
    threshold) for a Minor Heal's worth, no mana, twice a battle.
  - **Pack Sense** (rank 8): +5 Evasion on the Tamer while its beast stands.
- **Routes** (`internal/classes/routes_beast.go`), all open at any alignment:
  Houndmaster (warhound, 110% health; bites hobble a foe below half health with
  the shipped Hobbled status), Bearward (war bear, 120% health; guards the most
  hurt ally, the Tamer included, three then four times a battle through the 30c2
  guardian rule), Dragon Tamer (drake hatchling; every 3 rounds, in place of its
  bite, Breath: 1d6 plus half the Tamer's level to its foe and up to two more).
  The elites are `Planned`: 39i.
- **Talents**: Toughness, Thick Pelt, Strong Jaws, Steady Leash, Field Hand.
- **`beast` command** (`modules/company/beasts.go`): view the company's beasts,
  name your own. A camp rest or inn stay heals the beast and closes its wound
  (`modules/camping/tiers.go`, `restBeasts`).
- **UI**: `Company.Battle` `dolls` lists beasts too (with a `kind`); the Combat tab
  lists each beast as a fighter, the battle screen draws it as the nearest existing
  unit (wolf-timber, dog-junkyard, unknown-large, unknown-beast) and the Tamer
  has base map and battle art (`scripts/sprites/figures.py`).
- **Help and tutorial**: `help beasttamer`, `help beasttamer-routes`, `help beast`,
  and every page the class touched (classes, promotion, archetype, abilities,
  strategy, talents, health, growth, armor, shields, evasion, progression,
  combat, formation); a tutorial pointer in the creation lesson.

## Decisions

- The beast is a charmed mob with a registry (the summon pattern) plus a durable
  record (the doll pattern): it needs no company slot, roster, XP or loyalty
  machinery, and the ordinary companion paths give it a turn, a target and a cell.
- Sic is free: it doesn't cost the Tamer's own blow. The design reads "the Tamer's
  action sends the beast"; a Tamer that gives up its swing would be a doll with
  extra steps.
- A wounded beast recovers only with rest (camp or inn), with nothing to spend.
  The design says the beast eats a ration a day; feeding is **not built** (see
  follow-ups), because the company food path isn't a hook for a mob outside the
  roster.
- Beast health is `BaseHPPct` 35 (design: 60) and the Tamer's Attack 0.8 (design:
  0.9): at the design values the Tamer company won 100% at L10 with a third of
  the damage taken. With these, the Beast Tamer lands within the noise of the
  Warrior and Ranger at L5, L10 and L20 (below).
- Beast kinds on the battle screen reuse existing unit art; dedicated beast
  sprites are an art-pass follow-up.

## Balance

Five-foe groups, Beast Tamer in the third slot, 30 fights a cell, wins:
L5 80% (warrior 76, ranger 83); L10 96% (86, 90); L20 76% (90, 76). Routes at
30 fights: L15 houndmaster 83, bearward 76, dragon-tamer 83 (base 70); L25 66, 80,
76 (base 63). The routes lift the base class as intended.

## Follow-ups

- Feeding: a ration a day from cargo, then the Tamer's pack (32f order); an
  unfed beast should weaken rather than vanish.
- Dedicated beast sprites (wolf, warhound, bear, drake) and beast poses (S4).
- "Panic": the beast can be broken by the company's morale like a member; today
  it fights to the end of the battle.
- Elite routes (Packlord, Beastlord, Dragon Lord): 39i. Creature recruits that
  share the beast record: 38e.

## Tests

`internal/classes` (routes, ranks, talents), `internal/strategy`, `internal/beasts`
(gifts, record, rest), `modules/company/wiring_beast_test.go` (the real round: its
own turn, Sic, wounded beast, Rally, Pack Sense, drake Breath, bear guard,
warhound hobble), `modules/company/beasts_test.go` (the command),
`modules/camping/beasts_test.go` (rest), `modules/gmcp/gmcp_battle_beast_test.go`,
help render tests, the sprite tests, and the archetype and company suites updated
for the eleventh archetype.
