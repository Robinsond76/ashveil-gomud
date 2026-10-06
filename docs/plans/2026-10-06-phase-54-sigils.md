# Phase 54: sigils (execution plan, 2026-10-06)

Scope and acceptance come from the
[survival phase plan](2026-10-06-outward-survival-phases.md#54-sigils). This
page records how it was built and the decisions the owner delegated.

## Design

- **Rules** live in `internal/sigils` (pure): four kinds, their mana cost,
  numbers, and the `Laid` record (`Kind`, `RoomId`, Unix `Expires`).
- **State:** a company's sigil is saved with its leader's character
  (`Character.Sigil`). Expiry is a wall-clock Unix second, so it survives
  restart and copyover and never moves game time. Nobody ticks it.
- **Laying:** `cast sigil of [kind]` (in `Cast`, before spell lookup). Needs
  the Cast skill, mana and one *sigil chalk* (item 30060) from the player's
  own pack; refused in battle. One sigil to a room (a second in the same room
  is refused at no cost); one at a time per company (laying elsewhere lets
  the old one fade).
- **Battle:** `beginBattle` calls `startSigil`, which records the sigil on the
  battle (`battle.Sigil`, fixed for that battle) and acts at once for the two
  start-of-battle kinds. The other two change spells for the battle's length.
- **Enemies never lay sigils.** Fire and mending read the battle leader of the
  caster, which is 0 for an enemy mob.

| Kind | Mana | Effect |
|---|---|---|
| fire | 12 | spells with `element: fire` (Shower of Sparks, Fire Flask) deal 25% more (`SpellFactor`) and leave the foe Burning (`fireBurn`, both player and companion casts) |
| ward | 15 | each standing front-row member gets a ward: 1 blow, up to 4 + level/3 (about half a Priest's Ward) |
| stillness | 15 | every foe gets Windchilled for 3 rounds: chants and sling shots one round slower |
| mending | 12 | heals land 25% stronger (`HealFactor`) |

## Decisions (owner delegated; each with its reason)

1. **Stillness reuses Windchilled** (the Shaman's status) rather than a new
   one: the effect is the same, it already shows in `conditions` and the
   battle UI, and it saves a status id. A Shaman's Fog replaces it, which is
   accepted.
2. **A battle keeps the sigil it began with**, even if it fades mid-fight, so
   a long fight is never changed halfway and the header stays stable.
3. **Sigil data is on the leader's character**, not a room record: it needs no
   new store, saves with the player, and `look` finds it by scanning online
   companies (`users.SigilsIn`). A sigil whose leader is offline is not shown
   and serves no battle.
4. **Chalk is market-only** (both markets, 6 / 8 gold, `SupplyOnly`), so
   nothing resells for profit. A gathered source is left to a later phase.
5. **Fixed 15 real minutes**, no per-level scaling: the phase is about
   preparation, and the cost is the chalk and mana.
6. **Ward is deliberately small** (one blow, about half a Priest's Ward).

## Display

`look` (everyone), Room.Info `sigils` and a Room-panel badge, Company.Battle
`sigil`, the battle screen header banner, the Combat tab note, and a battle
opening line. `cast sigil` lists the kinds and what the company has laid.

## Help and tutorial

`help sigils` (combat category, aliases `sigil`, `chalk`, ...), links from
`help cast`, `spells`, `spell`, `battlefield`, `combat`; a tutorial hint in
the combat lesson.

## Tests

`internal/sigils`; `modules/company/wiring_sigil_test.go` (the command, each
kind through a real fight, faded and far-away sigils, a companion's burn);
`modules/gmcp` (battle feed and Room.Info); `help_sigils_test.go`; market
`SupplyOnly` test.
