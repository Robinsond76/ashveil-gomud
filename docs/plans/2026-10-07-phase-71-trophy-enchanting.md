# Phase 71: Enchanting with hunted trophies

Spec: [Pillars phases](2026-10-07-pillars-phases.md) §71. Hunted creatures drop trophies (a heart, a hide, ash) by their kind. An enchanter works one into a weapon or piece of armor for a fee, and the piece then grants the trophy's small effect while worn. Build decisions below (full autonomy; each fork records its pick and why). The world is temporary: the trophies and enchanters shipped here exist to test the feature.

## What shipped

- **Trophies** are ordinary light items (`type: commodity`) with a `trophy:` block: `part` (heart, hide or ash), `races` (lowercase mob races that drop it), `chance` (percent per ordinary kill) and `effects` (gear-effect keys, the same ones a relic's signature uses). Five ship in the test world, ids 40001 to 40005 (item ids from 40000 sit in the items root folder, clear of the relic range): beast hide (+2% damage reduction), brute's heart (+1 damage on every landed blow), chitin plate (+4 Evasion), grave ash (+6% spell damage), hollow heart (+4 Attack).
- **Drops** (`loot.Trophies`, `loot.TrophyRoll`, called from `zoneDrops` in `internal/mobcommands/drops.go`): each company that fights makes its own roll when a creature falls. It picks one of the race's trophies and drops it on that trophy's chance; an elite doubles the chance, a boss always drops one. It needs no zone drop profile, skips the Training zone, and rides the existing corpse and spoils paths, so the spoils line names it.
- **Enchanting** is `imbue [item] with [trophy]` (`internal/usercommands/imbue.go`) at a mob whose character carries the `enchanter` adjective. The piece may be carried or worn. The fee is 25 gold per tier of the piece (rolled gear by its rolled tier); the trophy is spent; the enchanter is paid. One trophy to a piece, permanent. Refused in a battle (it is a management command in `actionpolicy`). Plain `imbue` lists the trophies you carry and what each gives. Two enchanters ship: one beside the armorer in Frostfang (mob 90311) and one in the test area's armory (mob 90310).
- **The enchant lives on the item instance** (`Item.Trophy`, the trophy's item id): it rides with the item into the pack, cargo and a companion's gear, saves with it, and never touches the item's spec, value or roll.
- **Effects** reach the wearer through the gear-effect path relics use: `items.GearEffects` adds `TrophyGear(worn)`; `Character.wornGear` keys its cache on the enchant too, so an enchant takes effect at once, on any worn piece, for the leader or a companion, with no relic worn.
- **Display:** `look` and inspect, GMCP `Char.Inventory` and the Company window's gear and cargo rows carry an "Enchanted with ... (while worn): ..." line through the existing relic lines (so the web gear tooltip shows it, in purple), lists tag the piece `(enchanted: brute's heart)` (`EdgeLabel`), and `why` names an enchant that raised a landed blow (`Enchanted with brute's heart: +1 damage on every landed blow`). The web quick menu has Services -> Enchant gear and an enchanter's room menu has `imbue`; the enchanter shows a purple `enchanter` tag.
- **Bestiary (66):** the habits tier says which trophies a kind may yield ("Hunted for trophies: grave ash (about 25 in 100 kills)"; a boss "always yields" one).
- **Help:** `help enchanting` (aliases `imbue`, `imbuing`, `enchanter`, `trophy enchanting`, `enchant gear`, ...), listed under the company category, linked from relics, goods, salvage and bestiary; a hint in the tutorial's gear lesson. The existing `enchant` skill page is untouched.

## Decisions

| Decision | Why |
| --- | --- |
| **Command `imbue`, not `enchant`.** | `enchant` is GoMud's caster skill (damage, defence, stat enchants from a scroll-less skill). A separate verb keeps both pages clear; `help enchanting` says what it is. |
| **Effects come from a trophy's own data, not from the item.** One trophy, one fixed small effect; hearts lean offence, hides defence, ash spirit. | Simple to read and to test, and the replacement world authors its own trophies by data. No new effect keys: trophies reuse the 36d gear effects, so every number already has a cap and text. |
| **Each trophy effect is at most a third of the effect's cap (at least 1), checked at load; one-shot effects cannot be enchanted in.** | The awakening rule (`maxAwakenedEffect`): the 36d caps are the balance, so no sim is needed. |
| **One trophy per piece, permanent; a second is refused.** | Keeps the choice real (which piece to commit) and avoids an un-enchant economy. The enchanter says so. |
| **All of one wearer's enchants together give at most half of an effect's cap (at least 1).** | Ten slots of enchant would otherwise stack past what a relic may give. Half the cap keeps enchants a sidegrade (`TestEnchantsAloneNeverReachAnEffectsCap`). |
| **A relic may be enchanted, but its signature, awakenings and the enchant never pass the cap; the enchant is trimmed.** | Plan: "stack sanely with awakenings". `TrophyEffects` subtracts the relic's own and awakened effects from the cap per effect, and the help says so. |
| **Resale: an enchant adds nothing to a piece's sale price.** The enchant is not a spec override, so `SaleBaseValue` and `GetSellPrice` read the plain spec. Fee and trophy are spent. | "Never worth more at a merchant than parts and fee": enchanted equals unenchanted (`TestAnEnchantedPieceSellsForNoMoreThanAPlainOne`), which is less than the parts plus the fee. Trophies themselves sell for a few coins (value 12) and are never auto-junk; they are drops, not gathered or bought goods, so the relic and rare-drop profit exception is not stretched. |
| **Fee: 25 gold per tier.** | A Common piece costs 25, a Masterwork 100: dearer than a camp rest, far below the cost of the piece. A gold sink with no path back to gold. |
| **Drop chances: 20 to 25% per ordinary kill, doubled for an elite, a boss always.** | A company with a few kills in a zone has a trophy to spend. A boss kill is a guaranteed reward; the plan names hunting as the point. Rolled per company like every drop (personal loot). |
| **Races are the trophy's key** (like relic slay awakenings), not a new family tag. | The only kind tag the engine has; the replacement world's trophies name its own races. A test holds each shipped trophy's races to ones that exist and spawn outside Training. |
| **Chronicle: no deed for enchanting.** | Not an event of the company's story, and the chronicle's 300-entry log and web filters stay as 63 left them. The enchant is on the item. |
| **A companion's piece must be in the leader's pack to enchant.** | Companion gear is already moved through the pack (`help companion-gear`); a second path to a member's slots is not worth its code. A piece enchanted in the pack carries the enchant when worn. |
| **Enchanters are mobs with an `enchanter` adjective**, not a new mob field. | The adjective already reaches the web client through GMCP room data, so no payload changed. |
| **Trophy item ids 40001-40005, mobs 90310-90311.** | `ItemFolder` ties 50000-59999 to the relic folder; 40000+ is the items root. Both well clear of the ids in use. |

## Balance

No sim: every number is bounded by the 36d gear caps (per trophy a third, per wearer half, with relics clamped to the cap), so a fully enchanted company gains at most half of each effect's cap, spread over several effects. Tests hold each rule.

## Not done

- Enchanting a companion's worn piece in place (it goes through the pack).
- Removing or replacing an enchant.
- A chronicle deed.
- Trophy drops for the creature templates that carry no race a trophy names (`kobold`, `human` and others): the replacement world decides which trophies exist.
