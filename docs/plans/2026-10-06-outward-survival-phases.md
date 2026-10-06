# Survival lessons from Outward: phases 50–56 (2026-10-06)

The owner asked which features of Outward, a survival-heavy RPG, Ashveil
could learn from (research thread, 2026-10-06), then asked for phases for
the recommended features. This plan scopes those seven phases. Each one
still gets its own execution plan in `docs/plans/` from its build thread,
and the usual review gate.

Ashveil already has most of Outward's survival layer (needs, strain,
temperature, weather, camps, cooking, carry limits, wounds, gathering).
These phases make that layer matter more in the core loop rather than add
another meter.

## Rules every phase keeps

- Battles play out on their own. The only mid-battle inputs stay retreat
  and company focus. Everything here acts **before** a battle (condition,
  sigils, rest) or **after** it (defeat).
- Difficulty comes only from a zone's level. Penalties here come from the
  company's own choices (going in hungry, skipping the watch), never from a
  zone property. Balance cells are measured with members at neutral
  condition so the 37b win rates still hold.
- Nothing gathered or bought resells for profit (meals, remedies, sigil
  reagents, tents: markets never buy them back above cost).
- Never advance global game time. Outward skips hours on defeat and rest;
  Ashveil uses real-time recovery instead.
- The world will be replaced. Each phase is a system with data hooks (YAML
  tables, tags on mobs and zones), with only enough test content to prove it.
- Ship indexed help and tutorial pointers, per `AGENTS.md`.

## Phases at a glance

| Phase | Scope | Size | Depends on |
|---|---|---|---|
| 50 | Condition carries into battle; cooked meals give buffs | S–M | 47 merged (shared `modules/company` files) |
| 51 | Rest duties: each member sleeps, watches, tends or works | M | 47 merged; coordinate with 49 if banter hooks the camp rest |
| 52 | Tents with trade-offs | S | 51 |
| 53 | Defeat scenarios in place of the church respawn | M–L | Companion equipment phase merged; 40a4 |
| 54 | Sigils: prepare the ground before a battle | M | — |
| 55 | Ailments and remedies | M | 50 |
| 56 | Recipe discovery | S–M | 50 |

**Build order:** 50 and 54 can start now (50 once 47 is merged). 51 after
47; 53 after the companion equipment phase merges. Then 52 (after 51), 55
and 56 (after 50). At most two of 50–56 building at once, per the
concurrency rule.

**Overlap check (2026-10-06):** 47 (PR #79) changes pack capacity in
`modules/company/members.go` and `runtime.go`. 50 and 51 must not change
capacity or load rules; they read needs and camp state only. Companion
equipment changes what a fallen member keeps, so 53 waits for it. Banter
(49) may speak at the camp fire; 51 leaves a hook for it rather than
changing its lines. The admin test area does not overlap.

## 50 Condition carries into battle

**Why.** `help survival` says Hunger and Thirst at 0 only change a label,
and Fatigue only blocks setting out. In an auto-battler, the state a
member enters battle in is exactly the preparation the game rewards.
Outward cuts stamina, regeneration and maximums below need thresholds, and
gives buffs from meals.

**Scope.**
- At battle start, each member's needs set a battle condition: neutral at
  Sated/Comfortable/Ready or better; a small penalty in the Hungry, Thirsty
  or Tired band (for example −5% tempo or −1 skill edge per need); a clear
  one in the Starving, Parched or Exhausted band (about double). Exact
  numbers are tuned in the phase and must leave a well-kept company exactly
  where 37b measured it.
- Cooked meals (hearth and `camp cook`) give a buff lasting a number of
  battles (not game time): for example seared meat +regen after battle,
  hunter's stew +HP for three battles, grilled fish +mana. One meal buff at
  a time; a new meal replaces it.
- Display: battle header and battle screen mark a hungry or exhausted
  member; `conditions`, `status`, Company > Status and GMCP show the
  battle condition and meal buff.
- Help: update `help survival`, `help cooking`, `help conditions`; tutorial
  hint in the lesson that covers eating.

**Acceptance.** A starving member fights measurably worse in a harness
cell; a fed one matches the 37b baseline; meal buffs persist across
restart; nothing changes carry capacity.

## 51 Rest duties

**Why.** Outward splits a rest between sleeping, guarding and repairing.
A company has several people around the fire, so the choice is richer
here. Today a camp rest is all-or-nothing and specialists act on their own.

**Scope.**
- `camp duties` shows and sets one duty per member for the next rest:
  **sleep** (default: full Fatigue, Rested), **watch** (adds to the raider
  and thief spot chance; every watcher counts, not only the best; a watcher
  ends Ready, not Rested), **tend** (treats one lasting wound per tender, or
  sharpens one blade, using the existing surgeon's kit and whetstone rules),
  **forage**, **cook**, **brew** (today's automatic camp specialist actions
  become duties).
- The default assignment reproduces today's behaviour, so a player who
  never sets duties sees no change.
- Duties are fixed when the rest begins, persisted with the camp, and
  settled at rest end, so a restart never re-rolls them.
- Display: Camp tab duty picker, `camp status` line per member, rest-end
  summary by duty. Leave a hook for 49's camp-fire banter.
- Help: new `help camp duties`; update `help camp`, `help campwatch`,
  `help vigil`; tutorial hint in the camp lesson.

**Acceptance.** Two watchers spot raiders more often than one; a watcher
is not Rested; tending clears a wound; default duties match today's
results in the existing camp tests.

## 52 Tents with trade-offs

**Why.** Outward's tents trade comfort for safety. Ashveil has one canvas
tent.

**Scope.** Three or four tents on the 40a3 camp gear plumbing: a fur tent
(no cold penalty during the rest), a camouflaged tent (lower raider and
thief chance, no Well Rested), a large tent (Well Rested for everyone, but
a higher raider and thief chance). Bought only; markets never buy them
back above cost. Help: update `help camp gear`.

**Acceptance.** Each tent's effect shows in `camp status` and the Camp
tab and is covered by a rest test.

## 53 Defeat scenarios

**Why.** Today a fallen player loses a level, may drop gear, and wakes at
the last city's church (`help death`). In Outward you never die: you wake
in a situation that fits what beat you. Ashveil's battles are decided by
preparation, so a loss should tell a story and send the player back to
fix it, not only take a level.

**Scope.**
- When the company is defeated (leader down with no one left standing, or
  the company wiped), roll a scenario from a table keyed by the enemy's
  kind (beast, humanoid, undead, …) and a zone tag:
  - **Rescued:** a traveller carries the company to the nearest settlement;
    members wake Exhausted and Hungry.
  - **Captured** (humanoid foes): the company wakes bound in a nearby room;
    pack and cargo are in a chest guarded by a weak group from the zone's
    lowest band; a share of gold is gone.
  - **Left for dead:** wake where you fell, foes gone, a lasting wound each.
  - **Robbed:** wake in place; some gold and loose goods gone (40a4's theft
    rules: equipped gear, the treasury and quest items are safe).
- No game time passes. The church respawn stays as the fallback where no
  scenario fits, and as the route for a later Hardcore option. Whether a
  scenario also costs the level is decided in the design (default: no level
  loss for scenarios; the recovery is the cost).
- Fallen companions follow today's resurrect window unchanged.
- Scenarios are YAML data with mob family and zone tags, so the replacement
  world can add its own. Test content: one capture room and one scenario of
  each kind.
- Persisted: a restart mid-scenario resumes it, never re-rolls it.
- Help: rewrite `help death`; new `help defeat`; tutorial hint in the
  combat lesson.

**Acceptance.** Each scenario fires from a real defeat in a test; the
capture room can be escaped and the chest recovered; no item is
duplicated or lost across restart.

## 54 Sigils

**Why.** Outward's magic rewards set-up: a caster lays a sigil and spells
cast inside it gain effects. Ashveil's battles reward preparation, and
`cast` is already the out-of-combat utility command.

**Scope.**
- `cast sigil of [kind]` lays a sigil in the room: costs mana plus a
  bought or gathered reagent, lasts some real minutes, one per company per
  room. Three or four kinds: fire (fire spells and Shower of Sparks burn),
  ward (front row starts with a small shield), stillness (enemy chants
  start one round slower), and one for healers.
- Any battle the company fights in that room uses the sigil, including
  raiders attacking a camp there. Enemies never use sigils in this phase.
- Display: `look` shows the sigil to everyone; battle header and battle
  screen show it; GMCP room data carries it.
- Help: new `help sigils`; update `help spell` and `help battlefield`;
  tutorial hint where casting is taught.

**Acceptance.** A sigil changes the battle it covers in a test; it expires
on time across restart; reagents never resell for profit.

## 55 Ailments and remedies

**Why.** Outward gives colds, indigestion and infection obvious causes
and cheap cures. Ashveil has the causes already: temperature, raw meat,
three wound types.

**Scope.** Three ailments: a **Chill** after resting or travelling while
Frozen; a **Gut-ache** from eating raw game meat; a **Fever** when a
puncture wound stays untreated through several battles. Each has one clear
penalty that uses 50's battle-condition plumbing and one remedy prepared at
camp from gathered herbs (43a's `camp prepare`). Remedies never resell.
Help: new `help ailments`; update `help survival` and `help wounds`.

**Acceptance.** Each ailment is caught from its real cause and cured by
its remedy in a test.

## 56 Recipe discovery

**Why.** Outward lets you combine ingredients and learn recipes by trying.
Ashveil's hearth makes the first matching recipe in a fixed order.

**Scope.** `cook [ingredient] [ingredient] …` at a hearth or lit camp fire
tries that combination; a match makes the dish and records it in a
per-character recipe book (`recipes`); a miss makes a plain meal and keeps
the ingredients' worth as Hunger only. Recipe books from trainers or loot
teach the rest. Recipes stay YAML data. `use hearth` and `camp cook` keep
working from known recipes. Help: update `help cooking`; new
`help recipes`.

**Acceptance.** A new combination is learned once and listed; a known
recipe cooks without listing ingredients; meal buffs from 50 apply.

## Considered and passed

Backpacks (company cargo already makes that choice), mana from a health
sacrifice (fights class balance), three breakthrough points (clashes with
promotions and talents), exclusive factions (world content; revisit with
the new world), corruption zones (a non-level difficulty source), legacy
chests, and landmark-only navigation.
