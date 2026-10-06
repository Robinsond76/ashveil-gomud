# Phase 36b — tier 1–3 gear catalog, goods and an audit

Implements slice 2 of the owner-approved
[loot system design](../designs/2026-10-05-loot-system-design.md) on the
[equipment tiers design](../designs/2026-10-01-equipment-tiers-design.md):
the first real catalog (swords, axes, maces, spears, glaives, bows,
crossbows, staffs, four armor paths and shields), sellable trade goods, and
a tier for every shipped weapon and armor piece. Branch
`claude/36b-gear-catalog-dsqe7z`. Everything is built on the 36a item model
(quality, rarity, affixes), which already reads an item's `tier`.

## What ships

- **Item data.** Two optional spec fields join `tier` (36a): `family`
  (catalog family, shown on `look`; never inferred from a name) and `goods`
  (a goods category). `Validate` checks tier 0–6 and the goods category, and
  refuses goods on equipment. `look` and `inspect` show "Tier 2 (Steel)
  glaive" for a tiered base item and "Trade good (trophy): worth about 7
  gold a kg" for goods. The design's `basevalue` and `affixslots` overrides
  are not needed yet and are not added.
- **Weapons (26 new, ids `10101`–`10183`).** Id = `10100 + family × 10 +
  tier`. Every family has tiers 1–3 (the staff's tier 1 is the shipped ash
  quarterstaff, 10021). Each carries a `weaponclass` (the class rules of
  35a2 read it) and a `family`.

  | Family (id base) | Hands | Subtype / class | Reach | Damage by tier (1 / 2 / 3) |
  |---|---|---|---|---|
  | Sword (10101) | 1 | slashing / sword | no | 1d4+1 / 1d6+1 / 1d8+1 |
  | Axe (10111) | 1 | cleaving / axe | no | 1d6 / 1d8 / 1d10 |
  | Mace (10121) | 1 | bludgeoning / mace | no | 1d6 / 1d6+1 / 1d6+2 |
  | Short spear (10131) | 1 | stabbing / spear | no | 1d6 / 1d8 / 1d6+2 |
  | War spear (10141) | 2 | stabbing / spear | yes | 1d8+1 / 2d4+2 / 1d10+3 |
  | Glaive (10151) | 2 | slashing / glaive | yes | 1d10 / 2d6 / 1d12+2 |
  | Staff (10021, 10162–3) | 2 | bludgeoning / staff | no | 1d6 / 1d6+1 / 1d8+1, parry +5 / +6 / +7 |
  | Bow (10171) | 2 | shooting / bow | any rank | 1d6+1 / 1d8+1 / 1d10+1 |
  | Crossbow (10181) | 2 | shooting / crossbow | any rank | 1d6+2 / 1d8+2 / 1d10+2 |

  Budget: a one-handed weapon averages 3.5 / 4.5 / 5.5 by tier, a
  two-handed reach weapon 5.5 / 7 / 8.5 (the price of the shield), a bow one
  more than a one-handed weapon and a crossbow two more. The glaive ladder is
  the design's militia glaive, steel glaive and tempered war glaive.
- **Armor (60 new, ids `20100`–`20219`).** Four paths of five pieces
  (head, body, gloves, legs, feet) in three tiers. Id = `20100 + path × 30 +
  slot × 3 + tier − 1` (paths cloth, leather, medium, heavy; slots head, body,
  gloves, legs, feet). Protection is for a full set, weight is the same at
  every tier (a higher tier improves effectiveness at comparable weight):

  | Path | Bulk | Set protection (1 / 2 / 3) | Set weight |
  |---|---|---|---|
  | Cloth: padded, quilted, layered | light | 6 / 8 / 10 | 1.75 kg |
  | Leather: hide, cured, hardened | light | 12 / 18 / 24 | 4.7 kg |
  | Medium: padded jack, scale, brigandine | medium | 19 / 27 / 36 | 9.2 kg |
  | Heavy: iron, steel, tempered plate | heavy | 27 / 37 / 49 | 16.2 kg |

  Catalog armor carries no stat mods (affixes add them), so the best cloth
  stays below tier 1 medium. Bulk is set explicitly (`bulk:`) so a heavy
  helm is heavy even when light in grams.
- **Shields (7 new, ids `20300`–`20306`).** Steel and tempered bucklers,
  steel and tempered round shields (the shipped wooden shield is the tier 1
  round shield and the leather buckler the tier 1 buckler), and a kite
  shield at each tier (armor 7 / 10 / 13, heavier than a round shield). No
  new tower shields: the shipped one is audited tier 2.
- **Goods (24 new, ids `200`–`242`, `other-0`).** Trophies, salvage,
  materials, valuables and curios, as commodity items (the garnet is a
  gemstone), each with a `goods` category, a weight and a value. The
  shipped wolf hide is tagged a trophy and the raw game meat a provision.
  Value density bands (gold a kg): trophy 4–20, salvage 1–4, material
  5–250, valuable 100–2500, provision 5–30, curio 200–1500. A test holds
  every good to its band.
- **Audit.** Every shipped weapon and armor piece (73) has a `tier`, and
  weapons also a `family`. Tiers follow protection and damage, not name:
  plain gear tier 1; the captain's broadsword, obsidian dagger, finely
  crafted shortsword, iron shield, tower shield, chain coif, circlet and
  similar tier 2; the shadowsteel breastplate, wolf pelt, spider
  exoskeleton, snow wolf mane, royal ring, ogre's great club, tree trunk,
  jeweled dagger and dancing needle tier 3; the glowing battleaxe, ancient
  royal scepter, ice armor and spider queen breastplate tier 4 (bosses and
  uniques). No stat changed.
- **Shops.** Frostfang Armory (Ivar) stocks tier 1 gear: iron shortsword,
  iron hand axe, iron mace, iron short spear, militia glaive, shortbow,
  hunting crossbow, kite shield, hide jerkin, padded jack and iron mail. The
  general trader (Brynja) stocks scrap iron, iron ore and tanned leather,
  which also makes her buy goods (a merchant buys the types it stocks).
  Starter kits are unchanged.
- **Help and tutorial.** New `equipmenttiers` and `goods` pages, indexed
  with aliases and linked from `equipment`, `armor`, `shields`, `loot` and
  `itemlevel`; a Departure hint.

## For the neutral classes (39a and 39h)

- **Halberdier (39a):** the militia glaive (10151) is the kit weapon. Glaives
  are class `glaive`, two-handed, `reach: true`, slashing. War spears are
  class `spear` (shared with short spears, which are one-handed): 39a can
  keep `glaive` and `spear` as its class list and tell war spears apart by
  hands and reach if it must. Ashen Reaper (tier 6) is a later legendary.
- **Arbalist (39h):** the hunting crossbow (10181), heavy crossbow and
  arbalest are class `crossbow`, shooting, two-handed. They do not reload:
  36b ships no waitrounds or ammunition, so a crossbow is a heavier,
  slightly harder-hitting bow. 39h adds the reload and re-tunes damage.

## Decisions and deviations

1. **No new mechanics.** Families differ by hands, reach, subtype, class,
   damage shape and weight, using fields that exist. Axes have no armor
   penetration, daggers no speed bonus, and the crossbow no reload.
2. **A family field, not name inference.** `family` is shown on `look` and
   used by the audit tests; the equipment design forbids inferring tier or
   family from a name.
3. **Staff tier 1 is the shipped quarterstaff;** a tier 1 tower shield, a
   tier 3 tower shield, and tier 1 buckler and round shield duplicates are
   not authored.
4. **Tier 4–6 stay unauthored** (delivery slice 5). The shipped uniques are
   audited tier 4 so the ladder has no gap above 3.
5. **Merchant selling of catalog gear works as for any item;** only rolled
   gear is refused by merchants (36a deferral). Goods sell where a merchant
   stocks their type; markets for goods and saturation are 36c.
6. **Balance.** The catalog is data against budgets, checked by tests
   (damage and protection rise by tier, two-handed out-damages one-handed,
   cloth stays below medium) rather than a harness run. The loot design's
   "harness cells in tier-appropriate gear" belong to Phase 37, when drops
   put tiers in front of players. The only new stock players can buy is
   tier 1.
7. **Deferred by 36a and left deferred:** merchants declining rolled gear,
   GMCP showing base names, and Legendary and Set effects.

## Tasks

- [x] `family`, `goods` and `Tier` validation, `TierName`, look lines.
- [x] Weapons, armor, shields and goods data; the tier audit; shop stock.
- [x] Tests: shipped data (ids, tiers, ladders, bands, shops, files),
  validation, descriptions, real `equip` with hands and reach, merchant
  pricing, class rules, generator rolls on every catalog item, help render.
- [x] Help pages, aliases, hub links and the Departure hint.
- [ ] Independent full-diff review; fix findings with regressions.
- [ ] Final checks: `make generate`, `make validate`, `go test -race ./...`,
  `make js-lint`. Project Status entry, PR.
