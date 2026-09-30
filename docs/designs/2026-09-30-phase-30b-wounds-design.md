# Phase 30b: Wounds, Treatment, and `heal wounds` — Design

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md);
refines the 2026-09-26 proposal (in git history at `3233a84`, removed once this shipped)
(handoff §36 item 8), built on 30a's statuses. Written 2026-09-30. The open
decisions were put to the owner the same day (answers below). No standing
"proceed with your recommendation" instruction was in force.

## Goal

Sustained damage costs something that healing can't erase at once. Some
hits leave a **wound**, which lowers how far healing can restore a fighter
(the *wound limit*). After a fight, one command, `heal wounds`, has whoever
and whatever can help tend the company.

## Owner decisions (2026-09-30)

1. **Wound rules: lasting and light.**
   - A damaging critical hit leaves a *lasting* wound. It holds back half
     the crit's damage, rounded up and at least 1.
   - A crushing blow leaves a *light* wound: a strike, other than a crit,
     that deals at least 25% of the target's max health. So does a bleed
     that runs its course.
   - Light wounds close when the fight ends.
   - The limit never falls below a quarter of max health.
   - Only players and company companions are wounded. Enemies are not.
2. **Rests:** a completed camp rest (Rested) **and** a completed inn stay
   (Well Rested) both close every wound of the members present.
   **Amended by the owner (2026-09-30, after merge):** a camp rest closes
   a lasting wound only by using up one linen bandage the company carries,
   one per wound; without bandages a camp heals no wounds. An inn stay
   still closes them all; otherwise it takes bandages or a physician.
   **Amended again the same day:** a broken bone needs a birch splint at
   camp, not a bandage; cuts and punctures take a bandage.
3. **Tend** is a new castable restoration spell. Clerics and cleric
   companions get it, and `heal wounds` uses it.
4. **Death clears wounds**: a player waking at a church, and a companion
   raised by resurrection, come back unwounded.

## Prior-art check

- **Healing choke points.** Health rises through:
  - `Character.Heal` (AutoHeal regeneration);
  - `Character.ApplyHealthChange` (scripts' `AddHealth`: the heal spells
    and potion and regeneration buffs);
  - `ScriptActor.SetHealth`;
  - the level-up refill (`Character.LevelUp`);
  - the companion respawn (`modules/company` `applyState`);
  - the church wake (`modules/death`).

  All but the admin `paz` get the limit. The limit caps healing only: it
  never takes health away.
- **Crits.** `combat.calculateCombat` already knows, strike by strike,
  whether a crit got through the armor and with which weapon subtype (30a's
  crit table). A wound is decided there and applied in the four `Attack*`
  functions, where the live health change already happens.
- **Bleeding.** `status.Tick` reports `Expired`. `combat_status.go`'s
  `statusPass` and `fightSides.end` are the fight-end and stray seams 30a
  built.
- **Durability.** A player's `Character` is saved in the user file (autosave,
  logout, copyover). A companion's `domain.MemberState` (22b) is snapshotted
  from the live mob at the existing seams and rebuilt by `applyState`.
  Companion health isn't saved (a restart refills it), so the respawn must
  cap at the limit.
- **Treatment sources.** 32f's `company meal` already takes provisions from
  the cargo, then the member's pack, then the leader's pack
  (`modules/company/provision.go`). Recruiters (22c) show how to put a
  service in a room through config. `user.StartPrompt` re-dispatches a
  module command with the answer (the `delete` pattern).
- **Healers.** Player clerics are granted `heal`. Cleric companions know
  `heal` and `healall` by level (32d, `archetypes.SpellsAtLevel`). The
  strategy healer reads `Ally{HP, MaxHP}`. It must read the limit, or it
  would keep casting heals that can't land.

## Scope

### Domain: `internal/wounds` (pure)

```go
type Kind string // cut, puncture, fracture, bruise
type Wound struct {
    Kind   Kind   `yaml:"kind"`
    Place  string `yaml:"place"`           // "arm", "scalp", ...: text only
    Points int    `yaml:"points"`          // health held back
    Light  bool   `yaml:"light,omitempty"` // closes at fight end
}
```

- `Limit(max int, ws []Wound) int` is `max − Σ points`, floored at
  `ceil(max/4)` and at least 1.
- `FromCrit(subtype, damage, roll)` gives a lasting wound of `ceil(damage/2)`
  points (at least 1). Its kind comes from the weapon subtype:

  | Weapon subtype | Kind |
  |---|---|
  | slashing, cleaving, whipping, claws, generic (natural attacks and fists) | cut |
  | stabbing, shooting | puncture |
  | bludgeoning | fracture |

  (Implementation note: fists have no subtype of their own, since
  GoMud's unarmed attack is `generic`, so fists leave a cut, not a
  fracture.)

  The place is picked from the kind's list.
- `Crushing(damage, max)` gives a light bruise of `ceil(damage/4)` points,
  but only when the damage is at least 25% of max health.
- `Bled(stacks)` gives a light cut of 1 point per stack.
- `Close(ws, points)` closes points from the worst lasting wound. `CloseAll`,
  `CloseLight`, and `Lasting` do what their names say.
- `Treat(item, w)` is for items. A bandage closes 3 points of a cut or a
  puncture. A splint closes 4 points of a fracture.
- The text for each kind: "a cut to the arm", "a broken arm", and so on.
- `Plan` is the pure `heal wounds` planner. It takes the patients, the
  healers, and the item counts, and returns the steps (below).

### Characters

- `Character.Wounds []wounds.Wound` (yaml `wounds,omitempty`).
- `HealthLimit()` gives the limit, and `Wounded()` whether any wound is
  held.
- `AddWound(w)` adds a wound.
- `CapHealing(old, new)` is `min(new, max(limit, old))`. It is used by
  `Heal`, a positive `ApplyHealthChange`, and the level-up refill.

### Combat

- **Hit resolution.** `calculateCombat` puts `AttackResult.WoundsToTarget`
  on a woundable target: a player, or a company companion
  (`company.LeaderAndKeyForInstance`).
  - A crit that got through the armor gives a lasting wound.
  - Any other strike gives a light wound if it was crushing.
  - A crit's line names it: `(critical hit, 6 damage, bleeding, wounded)`.
  - The four `Attack*` functions apply the wounds with the damage.
- **Bleeding out.** `tickStatuses`: a bleed that expires by its count (not
  one cleared at fight end) gives a woundable holder a light wound.
- **Fight end.** `fightSides.end` closes light wounds, next to
  `clearFightStatuses`.
- **Strays.** `statusPass` closes the light wounds of any holder with no
  fight left, as it does statuses. A restart can't carry light wounds.
- **Strategy.** `combat_strategy.go` passes the limit as the ally's
  `MaxHP`.

### Spells and scripting

- New `ScriptActor` methods:
  - `GetHealthLimit()`;
  - `WoundNote(rolled, healed)`, which gives `", wound limit 10 of 16"`
    when a heal was held back;
  - `TendWound(points)`, which closes points of the worst lasting wound
    and gives its text.
- `heal.js` and `healall.js` add the wound note.
- The new `tend` spell (restoration, cost 4, waitrounds 1) closes 2d3
  points of the target's worst lasting wound.
- The `cleric` archetype gains `tend` in `GrantSpells` (players get it at
  their next login, as 32d did) and in `CompanionSpells` at level 1.
  Battle strategy doesn't cast it (it has no strategy use), so companions
  tend only through `heal wounds`.

### `heal` and `heal wounds` (`modules/company/wounds.go`)

**`heal`** lists each member present with a wound or below their limit:
health, limit, wounds, and what would help. It changes nothing.

**`heal wounds`** is refused in a fight: if the leader or any present
companion has aggro, or the leader is in a battle. Otherwise:

1. **Healers.** Every living, present member who knows `tend` or `heal`
   and has mana, most mana first. A player needs the spell in their
   spellbook; a companion needs it in its archetype at its level.
2. **Patients.** Living, present members (the leader and companions out),
   most wound points first, then lowest health against the limit.
3. **Casting.** Each healer, patient by patient:
   - tends each lasting wound (4 mana, 2d3 points closed);
   - then casts Minor Heal (3 mana, 2d3) until the patient is at the limit;
   - and stops when out of mana.

   This is resolved at once. The spells' mana and dice are used, but not
   their chant rounds, and no clock moves.
4. **Items.** Once no healer can act:
   - splints on each fracture, and bandages on each cut or puncture,
     one after another until the wound closes or they run out (review:
     kept, and the help says so);
   - then a bandage on a patient under half their limit, healing 3.

   Items come from the cargo, then the patient's pack, then the leader's
   pack (32f's order), then another companion's pack. Each is spent
   first and only then applied.
5. **The physician.** In a physician room with lasting wounds left, the
   physician offers to close them all, for the room's price per wound.
   `Pay N gold? [yes/no]`:
   - the price is checked again on `yes`;
   - gold is taken, then the wounds close, and the user is saved.
6. **Report.** `Still hurt: Tamsin Reed 13/16 (wounded, limit 13), you
   12/14.`, plus a hint: a camp rest or an inn will do the rest.

The player never names a healer or a patient.

### Rest (`modules/camping`)

`grantPendingTiers`, after a saved grant (camp Rested or inn Well Rested),
closes the wounds of the leader and the live companions. An inn stay
closes them all. A camp rest (owner amendment) closes a lasting wound
only for the item `company.SpendSupply` uses up (a splint for a broken
bone, a bandage for a cut or a puncture; the cargo, then the packs, as
`heal wounds`), one each; the rest stay, with a line naming what ran
out. Each healed wound gets a line:
`Tamsin Reed's broken arm has knit. (wound healed; a splint used)`.

### Death

- `modules/death` clears the wounds before the church wake's health is set.
- A companion's death drops `State.Wounds` in the same save as its kept
  gear, so resurrection spawns it unwounded.

### Content

- Items:
  - **bandage** (id 36, 40 g, value 6);
  - **splint** (id 37, 250 g, value 14).

  Both are sold in the Dunmar and Old Kings Road markets (a little dearer
  on the road).
- A physician at the Waymark Inn (2003), configured in the company module
  under `Physicians: [{RoomId, Name, Description, PricePerWound: 15}]`.

### Surfaces

- `status`: the health line reads `13/16 (limit 13)` while wounded.
  (Implementation note: "wound limit" wrapped the sheet's Vitals panel,
  so it is shortened here.)
- `companyview.Member` and `company.MemberView` gain `HPLimit`. The GMCP
  `Company.Vitals` gains `hp_limit` (set only when below max).
- The web company dock's vitals strip shows `limit 13` in the member's HP
  text.
- `heal` lists wounds.

## Durable model and invariants

- **The clock.** Nothing advances it. Wounds come from combat rounds that
  already run. `heal wounds` is instant. Rests heal at the grant that 23a
  already schedules.
- **Players.** Wounds live in the user file, so they survive logout,
  restart, and copyover like the rest of the character.
- **Companions.** Wounds live in `MemberState` and ride the 22b snapshot
  seams: autosave, copyover, shutdown, logout, and item changes. A crash
  can lose a wound taken since the last save; losing a wound is harmless.
  They respawn capped at the limit.
- **Light wounds** can't outlive a fight: fight end, the stray pass, and
  login (`HandleJoin`) close them.
- **Healing is saved with the next seam.** A rest's or `heal wounds`'s
  healing is in memory until the next user save or company snapshot
  (the physician saves at once, as it takes gold). A crash in between
  brings the wounds back, as a rest's buff grant already can. Accepted
  at review.
- **Locks.** No new locks. Combat and commands run on the game loop, and
  camping heals outside `m.mu` (as it grants buffs).

## Constraints and deferrals

- **Enemies** take no wounds, and monster abilities that wound are deferred
  (no ability system yet).
- **Absent companions.** A companion not out when a rest completes keeps
  its wounds.
- **Displays.** Wounds aren't shown on the live battle-view grid (32g2) or
  in the battle summary. No combat stream event for a wound.
- **Pets.** A pet's strikes don't wound.
- **Burns.** No burn kind. Burning's only source is a weapon override.
- **Physician price** is flat per wound, with no standing markup.

## Acceptance criteria

- **The wound limit:**
  - regeneration, a heal spell, a potion, and a level-up stop at the limit;
  - a held-back heal spell says `(N healed, wound limit L of M)`;
  - health above the limit is never reduced.
- **Causes, through the real round:**
  - a forced crit on a player and on a companion leaves a lasting wound
    and names it `wounded`;
  - a crushing blow leaves a light wound;
  - a bleed that runs out leaves a light wound;
  - enemies are never wounded.
- **Light wounds** close at fight end and as strays.
- **Durability:** a lasting wound survives a user save/load and a companion
  snapshot, save, load, and respawn. The respawn is capped at the limit.
- **`heal wounds`** through the real command:
  - it is refused in a fight;
  - it uses a cleric's tend and heals first, then splints and bandages,
    then offers the physician, whose `yes` takes gold and closes the
    wounds;
  - it never advances the clock.
- **`tend`** through a real cast closes wound points.
- **Rests:** an inn stay closes every wound; a camp rest one per splint
  (broken bones) or bandage (cuts, punctures) it uses up, and none
  without them.
- **Death:** a church wake, and a companion's death, clear wounds.
- **Surfaces:** `status`, the companyview, and GMCP show the limit, and
  the strategy healer doesn't heal a member who is at their limit.
- **Player help:**
  - a new `help wounds` page (aliases `wound`, `wound-limit`,
    `heal-wounds`, `bandage`, `splint`, `physician`, `tend`);
  - `help heal` rewritten from its stale GoMud skill page into the `heal`
    command's page;
  - `statuses`, `camp`, `inn`, `death`, and `health` updated;
  - linked from `help combat`, and listed in `keywords.yaml`;
  - the Camp lesson points to `help wounds`;
  - a render test passes, and `TestTutorialHelpPointersExist` passes.
