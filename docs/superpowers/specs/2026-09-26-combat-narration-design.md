# Potential Phase 28b: Combat Narration Voice

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.

## Prior-art check (2026-09-26)

- **Weapon attack text:** eight files in
  `_datafiles/world/default/combat-messages/`: bludgeoning, claws, cleaving,
  generic, shooting, slashing, stabbing, and whipping, about 1,200 lines in
  all.
  - Each file has `prepare`, `wait`, `miss`, `weak`, `normal`, `heavy`, and
    `critical` pools. Each pool has `toattacker`, `todefender`, and `toroom`
    variants, with `{source}`, `{target}`, `{itemname}`, and `{damage}`
    tokens.
  - About 300 of those lines end in `!`.
  - These files come from upstream GoMud and cover every fight in the game.
- **Which pool is used:** `internal/items/attack_messages.go`
  `GetAttackMessage` picks a pool by damage as a percent of the roll's
  maximum. `critical` means over 100%, which only a critical hit's bonus
  damage reaches.
- **How critical hits render today:** `internal/combat/combat.go` (around
  line 400) wraps whatever line was chosen in `***` when `isCrit` is set. A
  critical hit can therefore draw a `heavy` line in stars, and the
  `critical` pool is ALL-CAPS ("CRITICALLY LACERATES").
- **Damage numbers:** only the attacker's and defender's own lines carry
  `{damage}`. Lines seen by the rest of the room have no number.
- **"prepares to fight":** hardcoded in `internal/usercommands/attack.go`
  (around lines 196 and 253) and `internal/mobcommands/attack.go` (around
  lines 126, 129, and 150). Each companion that joins prints one; so does
  each enemy that picks a target.
- **"(charmed)":** the `charmed` name adjective
  (`internal/characters/formattedname.go`). The shipped build renders it as
  `(♥friend)`.
  - `characters.OnGetFormattedName` is an existing hook that fires on every
    formatted name, and modules can change the name there.
  - Companions are charmed mobs (`Character.IsCharmed(leaderUserId)`).
- **Death lines:** `X has died.` comes from the core. The company notice
  (`X has fallen. You have 3h 0m of your own time…`) comes from
  `modules/company` (25b).
- **Formation interception** (11c) reuses the ordinary attack text. It
  prints nothing of its own.

## Scope

1. **Rewrite the eight message files** in a dark, physical, story voice:
   - no `!`, no ALL-CAPS;
   - keep the pool structure and tokens;
   - keep `wait` and `miss` lines (the owner wants every miss shown).
2. **Damage on every hit.**
   - Pools no longer put `{damage}` inside the sentence. The combat code
     appends ` (N damage)` to every hit line in all three variants
     (attacker, defender, room), so the room sees numbers too.
   - Misses get no suffix.
3. **Critical hits.**
   - When `isCrit`, always draw from the `critical` pool, whatever the
     damage percent.
   - Drop the `***` wrapping. Append ` (critical hit, N damage)` in code.
   - The top non-critical damage tier is re-voiced so it never reads as a
     critical.
   - Open: whether a non-critical roll over 100% (for example a sharpened
     edge at 23b's top roll) can still reach the `critical` pool. The
     current code already guards the edge case.
4. **No company tag.** A module hook on `OnGetFormattedName` removes the
   `charmed` adjective when the viewer leads the company the named mob
   belongs to. Other charmed mobs and pets keep it; it may be simplest to
   drop it for any company companion, whoever the viewer is.
5. **Engagement lines.**
   - Remove every "prepares to fight" line.
   - When a fight starts (the first engagement between the company and an
     enemy party), print one opener to the room. It's generic, with optional
     per-`groups` text (for example bandits: "The bandits fan out across the
     road, steel catching the moonlight, and close in.").
   - When a fighter takes a new target mid-fight, print one short line:
     `The bandit cutthroat turns toward you.`
6. **Fight end.** When the last standing member of an enemy party falls,
   print one closing line (generic, optional per group): `The last of the
   bandits lies still.`
7. **Death notices.** The core death line is re-voiced to be physical and
   final. The company's fallen notice is indented four spaces under it,
   with the time as `3 hours`.
8. **Articles.** A mob's name gets "The"/"the" in combat text ("The bandit
   captain slices…"). Proper-named mobs (companions, named NPCs) don't.
   - Open: how "proper name" is known. It could be a template flag, or any
     name starting with a capital.

## Constraints

- Keep the message files' schema and tokens. Only the text and the
  selection and suffix logic change.
- Screen-reader and plain-telnet output must still read correctly with the
  suffixes; there are no symbols-only markers.
- The rewrite changes shipped content for every fight, including the Phase
  27c practice fight. Its tests assert on patterns that must be updated
  deliberately, not loosened.

## Acceptance criteria

- No line in the eight files contains `!` or a word in ALL-CAPS.
- A wiring test through `hooks.DoCombat`:
  - every hit line ends in `(N damage)`;
  - a forced critical hit ends in `(critical hit, N damage)` and comes from
    the `critical` pool;
  - there is no `***`;
  - no company member's name carries the charmed tag for their leader;
  - no "prepares to fight" line appears;
  - exactly one opener per fight and one closing line when the party is
    beaten.
- Before and after transcripts of the same seeded fight are attached to the
  review.

## Reference text (approved mock, 2026-09-26)

The target voice. It uses 28c's pronouns and ordinals and 28d's pain lines,
which land in later phases. Damage numbers are illustrative.

```
> attack bandit captain
You draw your broadsword and go for the bandit captain. Your company moves with you.
The bandits fan out across the road, steel catching the moonlight, and close in.

Your swing at the bandit captain cuts only air.
Ysolde fits a stone to her sling and sights on the bandit captain.
Brother Oswin swings his cudgel at the bandit captain and hits nothing.
Garrick Vane's broadsword rings off the bandit captain's guard.

Your swing at the bandit captain goes wide.
The first cutthroat lunges at Garrick Vane and stabs empty air.
The second cutthroat's dagger slips under Garrick Vane's guard and sinks deep into his gut. (5 damage)
The bandit bruiser's cudgel whistles past Garrick Vane's head.
The bandit slinger sets a stone in her sling and sights on Tamsin Reed.
Garrick Vane nicks the bandit captain's forearm and draws a thin line of blood. (1 damage)

The bandit slinger's stone smashes into Tamsin Reed's chest with a crack of breaking ribs. (3 damage)
The bandit captain brings his sword down on Garrick Vane's shoulder, and it bites to the bone. (4 damage)
Ysolde draws a slow breath and waits for a clear shot.
Garrick Vane sinks to his knees in the mud and topples forward. He does not rise.
Tamsin Reed folds over her broken ribs, coughs blood, and lies still.
    Garrick Vane has fallen. You have 3 hours of your own time to reach a church or a village shaman.
    Tamsin Reed has fallen. You have 3 hours of your own time to reach a church or a village shaman.
The first cutthroat turns toward you.

Garrick Vane steps inside the bandit captain's guard and opens him from collarbone to hip. (critical hit, 9 damage)
The bandit captain folds around the wound and dies face down in the mud.
You gain 86 experience.

The cutthroat finds the gap beneath your arm, drives his dagger in to the hilt, and twists. (critical hit, 6 damage)
Pain tears through your side, white and total. Your knees nearly give.

The last of the bandits lies still. Rain begins to fall on the road.
```

A blank line between rounds is part of the reference. Whether the engine
can emit it cleanly per viewer is an open decision for the design pass.
