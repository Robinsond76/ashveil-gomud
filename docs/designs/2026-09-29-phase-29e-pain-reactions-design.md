# Phase 29e: Pain Reactions — Design

Builds the [owner-approved direction](2026-09-29-phase-29e-pain-reactions-design.md).

## Intent and decisions

Critical weapon strikes should make a surviving victim feel present in the
fight. This is narration only: damage, critical chance, buffs, combat events,
turn order, and the shared world clock do not change.

- A reaction follows each damaging critical strike that leaves the target above
  zero health. Damage from earlier strikes in the same attack is included when
  deciding whether this strike leaves the target standing. A lethal strike
  gets the existing death line instead. Normal hits, dodges, and fully blocked
  criticals get no reaction. Backstabs count when they actually deal damage.
- The victim gets a second-person line. The attacker and other witnesses get
  the same third-person line, using the target's stable battle name and
  combat pronouns. A player victim's name remains a player name without an
  article. Separate-room attacks send the third-person line to both rooms.
- An NPC template may override its race's reaction set. Otherwise the race
  set applies. Races without a set use a generic humanoid or creature set;
  races whose default pronouns are `it` use the creature fallback. Players
  and companions use the humanoid set unless an authored NPC override applies.
- Each set is an array of paired `toVictim` and `toRoom` lines in the race
  or NPC YAML. Pairs keep viewpoint variants together. This follows the
  existing race and mob loaders and needs no independent reload path.
  One tier is enough; reactions do not vary with damage percentage.
- Choose a variant deterministically from the victim's name, remaining
  health, and strike ordinal. Narration must not consume the combat RNG or
  change the next hit, dodge, or critical roll.
- Shipped beast races get distinct lines: canine, rodent, insect, reptilian,
  giant spider, lagomorph, and reptile. Other nonhuman races can use the
  creature fallback until they get authored sets. The default world's
  practice dummy uses the generic creature line.

## Integration

`internal/combat.calculateCombat` already decides each strike's critical flag
and final damage before appending its weapon lines. It appends the reaction
immediately after that strike's lines in each recipient's message list. The
four attack wrappers supply the live NPC victim where present, so an authored
mob override can win over its race. The existing `DoCombat` dispatch then
delivers the lines in order; no new event, persistent field, or timer is needed.

Reaction templates permit `{name}`, `{he}`, `{him}`, and `{his}` in room text.
Only `{name}` is substituted in the shared third-person line; the pronoun
tokens use the target's 29d combat pronouns. Data validation rejects a pair
with a missing viewpoint or unsupported token.

## Help and tutorial

Update `help narration` and `help combat` to explain when a pain line appears
and why a fatal critical instead has a death line. Keep `critical` and `crit`
as narration aliases. Add a pointer in the Combat lesson. The page and
tutorial pointers must render in the shipped world.

## Acceptance criteria

- A real attack round demonstrates exactly one correctly ordered reaction
  per damaging, non-lethal critical strike for every relevant viewer.
- Noncritical, dodged, fully blocked, and lethal strikes emit no reaction.
- A sequence of strikes uses cumulative damage to suppress a reaction on a
  later lethal strike; it never announces pain after a target has fallen.
- NPC override, race set, generic humanoid, and generic creature selection
  are covered by tests. Shipped beast sets load and validate.
- Existing combat mechanics and event output remain unchanged in a seeded
  before/after comparison.
- `help narration`, `help combat`, and the Combat tutorial lesson describe the
  feature and their help pointers resolve.
- Focused tests, `make generate`, `make validate`, `go test -race ./...`,
  `make js-lint`, and independent full-diff review pass before integration.
