# Phase 67: Relics that awaken

Spec: [Pillars phases](2026-10-07-pillars-phases.md) §67. Each relic carries two or three awakenings: deeds done while it is worn that wake a further small power in it, shown on the item with progress.

## Shape

- **Data (`relic.awakenings` in an item file, `items.AwakeningSpec`):** `name`, `kind` (`slay`, `lair`, `place`), the kind's target (`races` and `count`; `mob`; `zone`), `count` (1 when omitted), `target` (the thing in words, for the item text) and `effects` (ordinary gear effects, `classes.GearEffect` keys). Validation (`validateAwakenings`) runs where the rest of the relic is validated, so a bad file fails to load and the shipped-data tests catch it.
- **Progress is saved on the item instance:** `Item.Awaken []int`, one count per awakening, replaced whole on each change (never edited in place, so item copies don't alias it). A relic nobody has touched saves nothing extra. It rides inside the item wherever it goes (pack, cargo, a companion's saved state), so a relic handed to a companion keeps what it earned.
- **Wiring the effects:** `items.GearEffects` adds each worn relic's woken effects to its own, so a woken power reaches every place a signature already does (damage, armor, evasion, auras, second wind) with no new combat path. The wearer's cached gear effects are keyed by relic id **and awakened mask**, so waking takes effect at once without re-wearing.
- **Sources (package `internal/awakening`):**
  - `slay`: `awakening.Slain(leader, race)`, called from `mobcommands.Suicide` beside the kill tally (so not in the Training zone, not for practice foes), once for each company that fought the foe.
  - `lair`: the chronicle's `Boss` deed. `chronicle.OnRecord` is a small new seam (observers called after each recorded deed); `awakening` watches it and reads the `mob:<id>` ref, so a lair master's fall counts from the same record the chronicle keeps, with no parallel counter. An executed boss is a boss deed, so it counts.
  - `place`: `hooks.RelicPlaceAwakening`, a `RoomChange` listener. It fires when a member crosses from one zone into another (`Room.Zone` differs from the room left; a step from no room, such as logging in, is not a crossing), once per crossing. The leader is credited by their own step (`awakening.Reached`) and each companion by its own (`awakening.CompanionReached`, while the leader is in that zone), because companions follow a moment after the leader.
  - Who counts: the leader and each living companion that is in the leader's room and not withdrawn from the fight (the same standing `enemyparty.CompanyEffect` uses). A relic in the pack or cargo does nothing.
- **Waking:** the leader is told ("Mara's Ogrebane awakens: Giant-Slayer. It now gives ... while worn."), the web client's gear window and company inventory are refreshed (`CompanyAssetsChanged`, which each progress step also queues), and a new chronicle deed kind `awakened` records it (member key of the bearer, `item:<id>`).
- **Display:** `Item.RelicLines()` now ends with one line per awakening, `Sleeping, Giant-Slayer (12 of 15): slay 15 ogres and goblins while it is worn, and it wakes with +1 damage on every landed blow.` or `Awakened, Giant-Slayer: +1 damage on every landed blow.`. That one list feeds `look`/inspect (`RelicDescription`), GMCP `Char.Inventory` and the Company window's gear and cargo rows (`relic` lines), and the gear tooltip, which colours woken lines gold and sleeping ones dim (`window-gear.js`). `inventory` lists each worn relic's `N of M awakened; next: <name> (x of y)`.
- **In battle:** an awakened power that raised a landed blow (a bonus on every blow, the wounded-foe, smite and shooting bonuses, each only when its condition held for that blow) is named on the strike's explained line, so `why` shows `Ogrebane woke Giant-Slayer: +1 damage on every landed blow`. Defence, health and aura powers work through the numbers they change, as relic signatures already do, and are read on the item.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| Three condition kinds: slay, lair, place. "Carried while devout" is **not built**. | Devotion needs the creeds of phase 73, which are tabled until the races and faiths are settled. The kind can be added then as one more `Matches` case. |
| Progress lives on the item; the chronicle is the source of the **lair** deed only. | The plan asks for progress saved on the item, and a deed that must happen *while wielded* cannot be read back from a company-wide log (the boss may have fallen before the relic was found). The chronicle holds the boss deeds, so they are read through `OnRecord`; it holds no per-kill deeds (a log of every kill would drown its 300 entries), so slay reads the kill the engine already credits. No counter is duplicated. |
| Slay matches a foe's **race name**. | The world is temporary and races are the only kind tag the engine has (`Smite` reads them the same way); the replacement world's data uses whatever races it has. |
| A `place` awakening is a **zone crossing** that counts once, and `count` over 1 is allowed by the spec but not used. | Walking back and forth must earn nothing. A test holds every shipped place awakening to one crossing. |
| Each awakening adds at most **a third of the effect's cap** (at least 1), and signature plus awakenings never pass the cap; one-shot effects (cap 1: second chance, divine shield) cannot be awakened. | "Awakened powers are modest and counted in 36d's relic balance": the 36d caps are the balance, and validating against them at load keeps every combination inside them with no sim needed. A relic fully awakened gains about a quarter to a third more than its signature, in small sidegrades. |
| No farmable loop: finite counts (15 to 30 kills, one boss, one zone crossing), each awakening wakes once and then stops counting, no gold, items or experience, and sale value is the spec's value whatever the progress (`SaleBaseValue` reads the spec; a test holds it). | The rule is that awakenings must not create a profit loop; relic sale stays as 36d left it. |
| Slay counts for each company that fought the foe, as drops and the kill tally already do. | Allied companies each roll and are credited for themselves; the same rule keeps it uniform. |
| A companion's relic counts only while the companion stands in the leader's room and can fight (for a zone crossing: its own step in, while the leader is in that zone). | Matches the rule used for company effects; a fallen or left-behind companion did not take part. (Review fix: the first build credited companions on the leader's step, before they had followed, so their place awakenings never woke.) |
| Every shipped relic has two or three awakenings, mixed kinds, tied to its own boss's family and zone (ogres and Hollowweb Deep for Ogrebane and the Ogre-hunter set, undead and Glassvault Depths for the lich's relics, woodland horrors and Stormcrown Heights for the ent's, and so on). | World is temporary; these name test-world zones and bosses and a test checks they exist (boss, race, zone with rooms). |

## Review notes

- Slay races must have ordinary foes that spawn, not only the lair's master (a test holds it). The ogre race is the forest ogre alone in the test world, so the ogre relics count ogres and goblins; Widow's Weave counts spiders and other insects.
- A full set's bonuses plus every piece's awakenings stay within each cap (a test holds it).
- Progress is stored by awakening position: keep the order when editing a relic's awakenings.

## Not done

- "Carried while devout": a follow-up for phase 73 creeds (one more `Matches` case once devotion exists).
- Awakening names for defence and aura powers on the strike line (they have no blow to name).
- A balance sim: the numbers are bounded by the 36d caps rather than measured; no new combat path was added.
