# Potential Phase 28d: Pain Reactions on Critical Hits

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.

## Rule

- **When:** after a critical hit that leaves the victim standing, one
  reaction line from the victim follows the hit line. A critical hit that
  kills is followed by the death line instead.
- **Not on normal hits.**
- **Who sees what:** the victim sees a second-person line ("Pain tears
  through your side…"); everyone else sees a third-person line.

## Content model

- **Beasts:** each beast race has its own set: wolf, spider, rat, and so on.
- **Overrides:** a mob template can override its race's set, for a unique
  boss or a named NPC.
- **Fallback:** a generic humanoid set, and a generic beast set using "it",
  cover anything without a set of its own.
- **Players and companions:** use the humanoid set, with 28c pronouns.
- **Where it lives:** probably a `pain-reactions/` data directory keyed by
  race and optional mob id, loaded like `combat-messages`.
  - Open: whether it lives there or on the race files.
- **Varied by damage?** Open: whether reactions vary with the fraction of
  health lost (a graze vs. a near-fatal wound). The owner's direction is
  critical hits only, so one tier is enough to start.

## Reference lines (approved mock, 2026-09-26)

```
The bandit captain reels, blood running from under the helm, and bellows through clenched teeth.
The bandit cutthroat drops to one knee, a hand pressed to his skull, blood seeping between his fingers.
Garrick Vane spits blood, grins through red teeth, and keeps his feet.
Pain tears through your side, white and total. Your knees nearly give.
The first timber wolf yelps and drags its hind legs, snapping at the air.
The large spider shrieks, legs curling as ichor spills from the wound.
Hot pain floods your shoulder. The wolf's breath is on your face, rank and wet.
```

## Acceptance criteria

- A forced non-lethal critical hit through the real round emits exactly
  one reaction line after the hit, from the victim's race set (or its
  override, or the fallback).
- A lethal critical hit emits no reaction.
- A normal hit never emits one.
- Every shipped beast race in the default world has a set, or the fallback
  is used and logged once at load.
