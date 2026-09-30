# Phase 32d: Automatic Combat by Strategy — Design

From the owner's play-test notes of 2026-09-28
([roadmap](2026-09-28-playtest-feedback-roadmap.md), row 32d, decisions 8
and 9). Once a battle starts it plays out on its own (32c made nothing
typed change it). This phase is what it plays out *by*: each character's
strategy, set before the battle, picks its target and decides whether it
swings, heals, or casts, and characters cast real spells with real mana.
It takes the casting slice of [30c](2026-09-29-phase-30c-company-tactics-design.md);
30c keeps the company-wide tactics settings (focus, healing threshold,
interrupts, rotation, mercy), guards, and enemy personalities.

## The owner's rules

From the roadmap (decisions 8 and 9) and 32c's design (rules 5 and 6):

- **Rule 5:** "think Ogre Battle: we can't do anything mid battle. When the
  battle starts, it all plays out based on company setup beforehand. Cast
  ends up being a utility skill for casting outside combat only. During
  combat, offensive, defensive and any spells are cast by characters on
  their own based on strategies predetermined."
- **Rule 6:** "based on each character's strategy is the target set, and
  if it can't reach them (for example, the weakest in group), then it
  automatically attacks one from the available list based on formation."

## Decisions (owner, 2026-09-28)

Asked in this session; the owner chose the recommendation each time
except where noted.

1. **Target rules:** eight, after Ogre Battle's Best/Strong/Weak/Leader
   and Final Fantasy XII's gambits (the owner asked for "the options from
   Ogre Battle or Final Fantasy 12"; the list below was then confirmed):
   `weakest`, `strongest`, `wounded`, `nearest`, `furthest`, `leader`,
   `assist`, `defend` (B).
2. **Roles:** every character, the player included, has a role, by
   default from their archetype: **fighter**, **healer**, or **caster**.
   The player is fully automatic, spells included (C).
3. **Mid-battle:** only `flee`. `break`, `formation` changes, `strategy`
   changes, `eat`, `drink` (potions too), `use`, `equip`, `remove`, and
   walking out are refused in a battle (E).
4. **Peaceful mobs:** only hostile mobs group by a shared tag. A
   non-hostile mob stands alone unless it came from a spawn group (F).
5. **Companion spells:** by archetype and level, from config; companions
   regain mana out of combat as players do, and spawn with full mana (D).
6. **Starting spells:** wizards start with Magic Missile and clerics with
   Minor Heal; existing wizards and clerics get it at their next login (D).
7. **The command:** `strategy` (alias `strategies`) (G).

## Prior-art check

- **Targeting today** (29a, 32c): a company member's aim is kept while it
  is legal; one with no target, a fallen one, or one it can't reach is
  turned (`upkeepEngagements` → `retarget` → `chooseFromParty`) onto the
  weakest living member it can reach (`engagement.AssignTarget`,
  `Weakest`, with 11c's `formationcombat.Legal`). A player alone is turned
  by `turnAlone` (32c) with `enemyparty.FirstAim`, which is the same rule.
  `attack <group>` sets the player's first aim with `FirstAim`, and every
  companion joins on that same foe (`attack #<id>` from `attack.go`).
  Enemies aim at the weakest company member they can reach (29a); that
  stays until 30c's personalities.
- **Enemy formations** (`mobparty.buildParty`): toughest first, front row
  first. `Party.Members[0]` is the toughest: Ogre Battle's leader.
- **Casting:** `cast` (`usercommands/skill.cast.go`) checks the spell
  book and mana, builds `characters.SpellAggroInfo`, runs the spell's
  `onCast` script (29c's chant line), takes the mana, and `SetCast`s the
  chant. `DoCombat` counts the chant rounds down (`onWait`), rolls
  `GetBaseCastSuccessChance`, runs `onMagic`, and clears the caster's
  Aggro. 32c refuses `cast` in a battle. Mobs cast through
  `mobcommands.Cast`, which takes mana the same way; the mob round runs
  their chant and spell the same way, and keeps a harmful spell to a
  player's battle (29b2's `holdMobSpell`).
- **Mana:** players regain `ManaPerRound` every third round out of combat
  (`NewRound_AutoHeal`). **Mobs never regain mana**; they spawn full
  (`mobs.go`), and a companion is refilled only when it spawns
  (`applyState`).
- **Spells:** Minor Heal (`heal`, help-single, 3 mana, 2 rounds),
  Minor Heal All (`healall`, help-multi, 6), Magic Missile (`mm`,
  harm-single, 6, difficulty 75), Shower of Sparks (`sparks`, harm-multi,
  10, difficulty 50), Cure Poison, Illuminate, Floating Light, Polymorph,
  and two skill-only spells (`aidskill`, `tameskill`).
- **Archetypes** (17a): wizard claims illusion and conjuration and grants
  only Floating Light; cleric claims restoration and grants no spell.
  `applyGrants` runs on every login and is idempotent, so a new grant
  reaches existing characters at their next login with no migration.
- **Companions** have an archetype on the company record (17a, 22c, 32a2),
  but their mob templates have no spell book and no `combatcommands`.
- **Mid-battle today:** 32c refuses `attack`, `cast`, `backstab`, `shoot`,
  `tackle`, and `disarm`. Still allowed: `flee` (rolled, companions
  follow), `break` (stands the player down; with no Aggro, `go` then
  walks out with no roll), `formation move`, `eat`, `drink`, `use`,
  `equip`, `remove`. `go` refuses only while the player has an Aggro,
  which a finished cast clears for a moment.
- **Grouping** (`mobparty.groupKey`): a spawn group, else the first
  `groups` tag, else alone, for hostile and peaceful mobs alike. The
  tutorial's straw squad (27c) is non-hostile and grouped only by its tag.

## A. Strategies

A **strategy** is a character's **role** and **target rule**. Every
character has one: the player (`me`) and each companion. It is set before
a battle and read at every decision during one.

- **Defaults:** the role from the archetype (cleric → healer, wizard →
  caster, everyone else → fighter); the target rule `weakest` (today's
  rule), for everyone. A character with no archetype is a fighter.
- **Durable:** stored per player, per member (the player, or a companion
  by its id on the company record), in the new `strategy` module's plugin
  file. It survives restart and copyover; only settings that differ from
  the default are stored, so `strategy <who> default` removes one.
- **Purged** with the player (32b's `UserPurged`). A dismissed or lost
  companion's entry is dropped the next time the player uses `strategy`
  (companion ids are never reused, so a stale entry changes nothing).

## B. Target rules

A rule picks among the foes of the character's battle's group that are
alive and visible.

| Rule | Picks | After |
|---|---|---|
| `weakest` | the least health left | Ogre Battle's Weak; today's rule |
| `strongest` | the most health left | Ogre Battle's Strong |
| `wounded` | the lowest health *fraction*: finish the hurt one | FF12's "HP lowest" |
| `nearest` | the front-most, left to right | FF12's "nearest" |
| `furthest` | the back-most, left to right | FF12's "furthest" |
| `leader` | the group's leader: its toughest, placed first (a new leader steps up when it falls) | Ogre Battle's Leader |
| `assist` | the player's own target (companions only) | FF12's "party leader's target" |
| `defend` | the foe striking whichever of us is most hurt (by fraction) | FF12's "foe targeting ally" |

**Reach (rule 6).** For a blow, a rule chooses only among foes the
character can reach from its place (11c's `Legal`: the front of each
column; a reach weapon one rank deeper; a bow anyone); a character not in
the formation may reach anyone. When the rule's own choice can't be made
among those (the leader, the player's target, or a defended ally's
attacker is out of reach, or nobody is striking us), the character takes
the **nearest** foe it can reach. With no one in reach at all, it takes the
front-most foe (a blow at the back is caught by the front, 11c). **A spell
reaches anyone**, so a caster's spell takes the rule's choice among every
visible foe.

**Sticky aims.** A target is kept until it falls, is hidden, or can't be
reached; then the rule picks again (as today: no one hops between foes
every round, and the "turns toward" lines stay rare). Two rules are
re-read every round because they follow something else: `assist` turns
with the player, and `defend` turns to whoever now strikes our most hurt
member. Both turn only when the new choice is in reach.

**Where aims are set:** the first aim at `attack <group>` (the player's
own, and each companion's by its own strategy: companions no longer all
join on the player's foe), the upkeep's re-aim (29a's `retarget`), and a
player alone (`turnAlone`). Enemies keep 29a's `weakest` (30c).

## C. Roles

Decided at the start of each combat round, after the battle pass and the
upkeep and before any blow, for every living character in a battle that
is not already chanting and can fight (not `no-combat`):

- **Fighter:** swings at its aim. Never casts.
- **Healer:** when a company member here (the player or a companion) is
  below **half** health, it heals: the group heal it knows when two or
  more are below half, else its single heal on the most hurt (by
  fraction). With no one that hurt, no heal known, or too little mana,
  it swings like a fighter. (The half threshold becomes 30c's setting.)
  A player who is down and bleeding out counts as the most hurt of all
  (review fix); a companion at 0 is dead and is left to `resurrect`.
- **Caster:** casts its attack spell while it has the mana: the area
  spell it knows when two or more foes of its battle stand, else its
  single-target spell at its rule's choice (any visible foe: spells reach
  anyone). Out of mana, or knowing no attack spell, it swings.

**Which spells** (config, `modules/strategy`): each automatic spell is
listed with its use: `heal` (single heal), `healall` (group heal), `mm`
(single attack), `sparks` (area attack). Others (Polymorph, Cure Poison,
Illuminate, Floating Light, the skill-only spells) are never cast
automatically. The first listed spell of a use that the character knows
and can pay for is the one cast.

**Casting in a battle** goes through the same steps as `cast`: the spell's
`onCast` chant line (29c), the mana taken, the chant rounds (`waitrounds`),
the success roll, the spell's own effect, and 29b's events (a new
`cast-start` event with the rest). A player needs the `cast` skill and the
spell in their book; a companion knows its archetype's spells for its
level (D). The chant starts before the round's blows, so an automatic cast
takes as many rounds as a typed one did.

**After a spell** the caster turns back to the aim it had before (if that
foe still stands, is here, and can be reached), with no "turns toward"
line; otherwise the upkeep turns it by its rule. This also closes a gap:
between a finished spell and the next round a caster had no Aggro, so
`go` walked out of the fight and the out-of-combat mana regain ran.

## D. Spells and mana

1. **Companions know spells by archetype and level** (config on each
   archetype in `modules/archetype`, `CompanionSpells`): cleric Minor
   Heal from level 1 and Minor Heal All from 5; wizard Magic Missile from
   1 and Shower of Sparks from 5. A spell must be in the archetype's
   schools (checked at load, like `GrantSpells`). Nothing is written to
   the companion's mob; the list is read when it acts.
2. **Companions regain mana** as a player does: `ManaPerRound` every
   third round while out of combat, only while their leader is online.
   They spawn full, as they spawn at full health (neither is saved: a
   restart refills both; accepted).
3. **Starting spells:** the wizard archetype grants Magic Missile and the
   cleric Minor Heal (with Floating Light and nothing, today). The grant
   runs at every login, so existing wizards and clerics get it once, at
   their next login, with no migration.
4. **Casting odds** stay GoMud's: Magic Missile's difficulty 75 gives a
   new wizard about a one-in-three chance, which rises with practice and
   Mysticism. Retuning spells is not 32d's; it is reported for the owner
   (Known issues).

## E. What a player may do in a battle

"In a battle" is 32c's test: the player has a battle, or aims at a mob.

- **Allowed:** `flee` (the one way out: its roll, only the battle's group
  blocks it, companions follow), and everything that changes nothing:
  `look`, `scout`, `strategy` (to read), `status`, `inventory`, `cargo`,
  `formation` (to read), `conditions`, `say` and the other talk commands,
  `help`, `quit`.
- **Refused, with 32c's line** ("The battle is under way: it plays out as
  you set it up."): `formation` changes (`move`, `swap`, `clear`:
  anything but reading it); `strategy` changes; `eat`, `drink` (potions
  too), `use`; `equip`, `remove`. `break` and walking out (`go` and bare
  exit names) add "Only flee takes you out of it."
- **A flight that gets away ends the battle at once** (implementation),
  so the player isn't held to it for another round.
- **Not in a battle,** all of these work as today, including `break` for
  a player-versus-player fight.
- 32f's `company eat`/`drink` (another session) gets the same check when it
  merges; this is recorded for that session.

32c's "a bare attack after break rejoins the battle" can no longer happen
in a battle (break is refused); the code stays, since `break` still holds
outside one.

## F. Peaceful mobs

A mob groups by its first `groups` tag only when it is **hostile** (its
template's `hostile: true`). A non-hostile mob is a group of one unless it
came from a spawn group. So a room of townsfolk sharing a tag lists each
by its name, `scout` shows no "band of townsfolk", and `attack
<townsperson>` fights that one. If its friends join in (GoMud makes their
tag hostile to the attacker), each is its own group and waits its turn
(29b2). The tutorial's straw squad is given a spawn group when it is
raised, so it stays one group.

## G. `strategy`

```
> strategy
How your company fights (set before a battle; it plays out by these):
  You            wizard   caster   weakest   casts Magic Missile (6 mana)
  Dain           warrior  fighter  nearest
  Brother Oswin  cleric   healer   weakest   heals below half: Minor Heal (3 mana)
Change one with strategy [who] [role], or strategy [who] target [rule].

> strategy dain target leader
Dain will go for their leader, else the nearest foe he can reach.
> strategy oswin caster
Brother Oswin knows no attack spell yet; he'll swing until he learns one.
Brother Oswin will cast his attack spell while his mana lasts, else swing.
> strategy me assist
Only a companion can assist you.
> strategy oswin default
Brother Oswin goes back to his default: healer, weakest.
```

- `strategy` lists everyone; `strategy <who>` explains one (role, rule,
  the spells it would cast, its mana).
- `<who>` is `me` (or `self`, `you`), or a companion as `company` commands
  name one (the first word of its name).
- Roles: `fighter`, `healer`, `caster`. Rules: `target <rule>`, or the
  rule alone (`strategy dain leader`).
- A role the character can't perform yet is allowed with a warning, and
  it fights until it can.
- Refused in a battle (E). Without a company, `strategy` covers the
  player alone.

## Module

- **`internal/strategy`** (new, GoMud-free): `Role`, `Rule`, `Strategy`,
  parsing and descriptions, defaults by archetype, and `Pick(rule, foes,
  ...)` over plain `Foe` values (health, max health, row, column,
  reachable, leader, the health fraction of the ally it strikes). A
  provider seam (`For(userId, key)`) set by the module, with the default
  when none is set.
- **`modules/strategy`** (new): the durable registry (plugin file), the
  `strategy` command, config of automatic spells, `UserPurged`.
- **`internal/enemyparty`:** `FirstAim` becomes `Aim`, taking the
  attacker's strategy (and the player's target for `assist`); builds
  `Foe` values from the group.
- **`internal/hooks`:** the upkeep and `turnAlone` re-aim by rule
  (`assist`/`defend` re-read each round); a strategy pass for healers and
  casters (`combat_strategy.go`); the aim restored after a cast;
  companion mana regain (`NewRound_AutoHeal.go`).
- **`internal/usercommands`:** `attack` aims each companion by its own
  strategy; `break`, `eat`, `drink`, `use`, `equip`, `remove`, `go`
  refused in a battle (E).
- **`modules/company`:** `formation` changes refused in a battle (only
  `formation.go`; nothing in inventory or provisions, which 32f is
  changing).
- **`internal/archetypes`, `modules/archetype`:** `CompanionSpells`;
  config grants.
- **`internal/mobparty`, `internal/rooms`:** `Hostile` on the summary;
  the tag rule (F). **`modules/tutorial`:** the squad's spawn group; hints.
- **Help and content:** below.

## Invariants

- **The clock:** nothing advances or fast-forwards time. Mana regain
  follows the existing every-third-round rule.
- **Restart and copyover:** strategies are durable (the plugin file,
  `SetOnSave`/`SetOnLoad`). Battles, aims, and the aims to restore after a
  spell are runtime only, as 29b2's battles are.
- **Game loop:** the strategy pass, aims, and the restore map run on the
  game loop and read mobs as combat does; the registry has its own mutex,
  never held while calling the engine. `internal/battle`'s mutex is not
  held across any of it.
- **Narration:** 29c's voice: the spells' own chant and landing lines, "turns
  toward" for a change of aim, mechanics in lowercase parentheses.

## Acceptance criteria

- **Unit (`internal/strategy`):** each rule's pick, ties, and fallback
  (unreachable choice → nearest reachable; none reachable → front-most);
  `assist` rejected for the player; parsing and defaults by archetype;
  the role decisions (heal below half, group heal at two, the spell
  order, mana, an unknown spell never cast).
- **Unit (elsewhere):** the registry's load/save/purge; `CompanionSpells`
  by level and its school check; the tag rule for hostile and peaceful
  mobs.
- **Wiring** (shipped config, real commands, `DoCombat` rounds):
  - `strategy` sets a companion's rule and role; they survive a module
    reload (save, fresh module, load);
  - `attack <group>`: the player and each companion start on their own
    rule's choice (`strongest`, `leader`, `nearest` with a reach limit);
  - a companion on `assist` follows the player's target; one on `defend`
    goes for the foe striking the most hurt member;
  - a cleric companion heals the player below half health through the
    real round, spending mana and chant rounds, and swings otherwise;
  - a wizard player on `caster` casts Magic Missile with no command,
    mana spent, and turns back to their aim after the spell; out of mana,
    swings;
  - companion mana comes back out of combat, not in it;
  - in a battle: `break`, `formation move`, `strategy dain leader`, `eat`,
    `drink`, `use`, `equip`, `remove`, and an exit name are refused and
    change nothing; `flee` still works, companions follow;
  - out of a battle, those commands work as before;
  - a new wizard and cleric get their spell, and an existing one gets it
    at login;
  - peaceful tag-mates are separate groups; the tutorial's straw squad is
    one group, and the Combat lesson passes as before;
  - 29a, 29b, 29b2, 29c, and 32c's fight tests pass.
- **Player help:** a new `help strategy` (aliases `strategies`,
  `gambits`, `roles`); `help targeting` (targets by strategy; stepping out
  is only `flee`), `help combat` (the hub links `strategy`), `help cast`
  (casting in a battle is automatic), `help flee` and `help break` (in a
  battle), `help formation` (set before a battle), `help mana` (companions
  regain it), `help attack` (companions start on their own aims), and the
  wizard and cleric pages if they list starting spells; `strategy` in
  `keywords.yaml`; the Combat lesson points to `strategy`;
  `TestTutorialHelpPointersExist` passes.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- 30c: company-wide tactics (focus, healing threshold, interrupts,
  rotation, mercy), guards, enemy personalities, and a retreat order.
- Openers (a rogue's backstab from hiding) and per-spell gambit lists.
- Spell balance (Magic Missile's odds) and saving companion mana and
  health.
- `strategy` in the web client (32g's dock).
