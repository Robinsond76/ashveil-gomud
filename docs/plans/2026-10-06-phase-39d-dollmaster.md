# Phase 39d: The Doll Master

Source: [neutral classes design](../designs/2026-10-05-neutral-classes-design.md) §3.
The fourth neutral lineage: a Master in the back row who fights through a
wooden doll.

## What shipped

- **Archetype and recruit**: `dollmaster` (HP 1 and 0.55 a level, Attack 0.7,
  Evasion 0.8, light armor, no shield, daggers, rods and staffs, smarts 4 /
  perception 3 / speed 2 / mysticism 1, the Puppetry skill, a kit with doll
  parts), a recruitable Doll Master (mob 160) in the Dunmar templates, level-3
  `ShippedFor`. Doll parts (item 70) are sold at Dunmar's market.
- **The doll** (`internal/dolls`, `internal/company/dolls.go`,
  `internal/characters/doll.go`): a record on the Master (`characters.DollState`
  on a player's character, on `company.MemberState` for a companion: name,
  damage, broken flag, gear). When a battle opens `dollPass` stands each Master's
  unbroken dolls as charmed mobs (mob 161, race 25) that register as members no
  company record holds, so the existing combat paths find them like a summon;
  `company.FormationFor` lays them over the formation in front of their Master.
  They take no company slot, have no aim and no turn of their own, and are
  dismissed (and synced back to the record) when the battle ends. A doll at
  0 health breaks: it is out of the battle and stays broken until mended.
- **Abilities** (`internal/hooks/combat_doll.go`):
  - **Puppet Strike**: the Master's turn is the doll's strike (`extraBlow`
    through the same hit, defense, armor and wound rules), at the Master's foe
    or the foe the front row puts in the way; with extra tempo turns, one
    strike per turn. A Master with no doll able to act swings itself.
  - **Guard String** (rank 5): the doll guards through the 30c2 guardian rule,
    twice a battle (three from rank 8), counted on the Master.
  - **Tangle** (rank 12): the doll's first landed blow pushes the foe's meter
    back 50 points (25 for a boss), cooldown 3, a foe immune for 2 rounds.
  - **Emergency Splice** (rank 18): once a battle, a doll that would break
    stands up at 25% and the Master loses its next turn.
- **Routes** (`internal/classes/routes_doll.go`), all open at any alignment:
  Puppeteer (two dolls at 50% health each, Attack, 60% health, armor),
  Golemancer (one golem, 130% health, +15 armor, no armor, 4 guards, damage,
  150% health, more armor), Marionettist (Tangle two foes and a round faster,
  Attack, a harder pull, damage). The elites are `Planned`: 39i.
- **Talents**: Toughness, Hardwood, Stout Strings, Fine Carving, Steady Hand.
- **`doll` command** (`modules/company/dolls.go`): view, wield, wear, remove,
  name, mend; camp rest end mends the company's dolls with the leader's parts
  (`modules/camping/tiers.go`).
- **Help and tutorial**: `help dollmaster`, `help dollmaster-routes`,
  `help doll`, and every page the class touched (classes, promotion, archetype,
  abilities, strategy, talents, health, growth, armor, shields, evasion,
  progression, combat, formation); a tutorial pointer in the creation lesson.

## Decisions

- Dolls are summon-like mobs plus a registry, not company records: avoids the
  roster, food, morale, XP and loyalty machinery and takes no company slot.
- The doll is spawned at battle start and removed at its end; only its record
  persists, so copyover and restart need nothing from the combat state.
- A Master with no doll able to strike (broken, stunned, down) swings its
  own dagger rather than losing the turn.
- Tangle uses the foe's tempo meter after the round's turns are allocated, so
  it takes effect on the next round.
- Companion Masters' dolls keep the starter cudgel; only the player's own dolls
  are dressed with the command (companion dressing is a follow-up).
- `doll mend` works anywhere out of battle, not only at camp: the parts are the
  cost, and the camp's automatic mend is the convenience.
- Doll parts are bought back unused at the market (economy rule: nothing
  gathered or bought resells for profit).

## Follow-ups

- Battle screen and `Company.Battle` GMCP: a doll sprite and cell (today a doll
  is a charmed mob with an unlisted key; the combat lines name it).
- Dress a companion Master's dolls; salvage doll parts from constructs.
- Doll sprite art (art pass); elite ranks (39i).

## Tests

`internal/classes` (routes, ranks, talents), `internal/strategy`,
`internal/dolls` (gifts, mending, parts, YAML), `modules/company/wiring_doll_test.go`
(the real round: strike in place, break and persistence, Splice, Tangle, Guard
String), `modules/company/dolls_test.go` (the command),
`modules/camping/dolls_test.go` (camp-rest mend), help render tests, and the
archetype and company suites updated for the ninth archetype.
