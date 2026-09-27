# Potential Phase 30b: Wounds, Treatment, and `heal wounds`

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.
This is handoff §36 item 8, formerly in deferred Phase 11d.

## Goal

Sustained damage has a cost that healing can't erase at once. Some hits
leave a **wound**, which lowers how far normal healing can restore a
fighter (the *wound limit*). After a fight, one command, `heal wounds`,
has whoever and whatever can help tend the company.

## Rules

- **Causes:** crits (especially stabbing and heavy weapons), bleeding that
  runs its course, crushing blows, and some monster abilities.
- **The wound limit:** while wounded, a fighter's health can be healed only
  up to the limit.
  - A heal that would go over reports what was held back:
    `(5 healed, wound limit 10 of 16)`.
  - Their true maximum health is unchanged.
- **After the fight:** light wounds close when the fight ends. Wounds from
  critical hits persist as injuries until treated.
  - These are durable: they survive logout, restart, and copyover, on the
    player's character and on the companion's member state (22b).
- **Treatment:**

  | Treatment | Effect |
  |---|---|
  | a cleric's tending (in `heal wounds`) | raises the limit partway |
  | a splint or bandage, from the packs | raises the limit partway, and is used up |
  | an inn physician, for gold | heals the wound fully |
  | a full camp rest (Rested, 23a) | heals remaining wounds when the rest completes |

## `heal wounds` (after a fight only)

- **In a fight:** it's refused. Clerics heal on their own during a fight,
  by the company's tactics (30c).
- **Out of a fight**, the command goes through these steps in order:
  1. Pick healers. Every living company cleric with mana, in order of
     mana.
  2. Pick patients. Most hurt first: wounds before plain damage, and lower
     health first.
  3. For each patient, the cleric tends the wound (raising the limit),
     then casts heals up to the limit, until mana runs out.
  4. When no cleric is left, use the company's splints and bandages on the
     remaining wounds and damage.
  5. In an inn with a physician, offer the physician for what's still
     wounded, with a price and a yes/no prompt.
  6. Report who is still hurt, and what would help: a camp rest or an inn.
- **Nothing to name:** the player never names a healer or a patient.

## Content

- **Items:** a bandage and a splint, sold in the Dunmar and Old Kings Road
  markets.
- **Physicians:** at the Waymark Inn, and wherever else suits. They are
  configured like recruiters (22c): room, name, and price per wound.
- **Cleric tending:** a small `tend` restoration ability, given to the
  cleric archetype (17a).

## Reference text

```
> heal wounds
Brother Oswin kneels beside Tamsin Reed first, the worst hurt.
Brother Oswin sets Tamsin Reed's arm with a wet click and binds it to a splint of birch. She doesn't scream, quite. (wound treated, limit 13 of 16)
Brother Oswin lays glowing hands on Garrick Vane's scalp, and the gash closes. (5 healed)
Brother Oswin rests a hand on Ysolde's calf. (2 healed)
Brother Oswin sways, pale and spent. (mana 1 of 12)
    Still hurt: Tamsin Reed 13/16 (wounded, limit 13), you 12/14.

> heal wounds
There's no one in the company who can heal. You open the packs instead.
You bind Tamsin Reed's arm with a splint and linen. (wound treated, limit 13 of 16; 1 splint left)
You wrap Garrick Vane's scalp. (3 healed; 2 bandages left)
    A camp rest or an inn will do the rest.

> heal wounds
The physician, a stooped man with ink-stained cuffs, looks Tamsin Reed's arm over and whistles. "Four silver, and she'll lift a shield by morning."
Pay 4 silver? [yes/no]: yes
The physician resets the bone properly and packs it in a poultice that reeks of comfrey. (wound healed)

Your camp rest ends. The company is Rested.
Tamsin Reed flexes her arm and winces, but the break has knit. (wound healed)
```

## Acceptance criteria

- **Wound limit:** a heal on a wounded fighter stops at the limit and
  reports the held-back amount.
- **Durability:** a critical-hit wound survives a save/load and a copyover,
  for both a player and a companion.
- **`heal wounds`** through the real command:
  - uses the cleric first, then items, then offers the physician;
  - is refused in a fight;
  - never advances the clock.
- A completed camp rest heals remaining wounds.
- The 26a/26b surfaces (`status`, the Company panel) show a wounded
  member's limit.
