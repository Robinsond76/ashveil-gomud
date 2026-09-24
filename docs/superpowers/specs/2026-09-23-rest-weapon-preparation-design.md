# Rested Bonuses and Company Whetstones

**Status:** Design direction approved 2026-09-23; written spec awaiting owner
review. Part of the [roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Existing behavior and goal

Phase 7 camp rest is a durable 60-second real-time session with a lit-fire
requirement and fatigue recovery. Phase 16 inn rest grants the module *Well
Rested* buff, with +2 speed, +2 perception, and half walking strain for about
30 minutes. An older upstream inn script also creates a differently defined
buff named *Well Rested*. Players should receive a weaker camp *Rested*
benefit and prepare the entire company's blades with one whetstone.

## Rest tiers

- A completed camp rest grants every present living company member *Rested*:
  25% less walking strain for 15 real minutes. A completed inn stay grants
  the existing module *Well Rested*: 50% less walking strain, +2 speed, and
  +2 perception for 30 real minutes. Durations and percentages are config.
- The tiers are exclusive. Well Rested replaces Rested; a later camp rest
  while Well Rested is active refreshes fatigue but does not downgrade it.
  Once Well Rested expires, an older Rested effect does not reappear.
- Reconcile the older upstream inn script so player-facing conditions have
  one unambiguous *Well Rested* tier. Existing saved buffs load safely and
  normalize to at most one active tier.
- Grant on the game loop using the existing pending-rest pattern. A
  companion absent at completion receives its pending grant on restoration
  only while the tier would still be active. Each rest grants once.

## Whetstone behavior

**Owner amendment (2026-09-24), superseding the bullets below where they
conflict:**

- A whetstone can be used **on demand at any time**, not only during a
  camp rest.
- A whetstone has **10 uses**. Each company member whose blade(s) it
  sharpens spends **one** use, so a company of four spends 4 of a new
  stone's 10, and a leader with no companions can sharpen up to 10 times.
  (Phase 23b's design decides the details: a member with two blades
  still spends one use; members with no eligible or no unsharpened blade
  spend none; what happens when a stone runs out partway through a
  company.)
- The camp-rest auto setting may remain as a convenience, but it follows
  the same per-member use rule.

- Shipped settlement shops sell a consumable whetstone with one use. It is
  carried by the leader; company cargo counts only if the item is available
  at the camp through the existing cargo rules.
- One use covers **all** currently equipped bladed weapons held by present,
  living company members, including the leader. Bladed means the weapon's
  damage subtype is slashing, cleaving, or stabbing; unarmed attacks, claws,
  bows, and blunt weapons do not qualify. Each equipped weapon is considered
  independently, including two weapons on one member.
- `camp sharpen` previews eligible weapons outside a rest and performs
  preparation during an active camp rest. `camp sharpen auto on|off` stores a
  leader preference. Auto mode is initially off to avoid consuming a
  purchased item without an explicit choice; when on, completion of the rest
  performs the same operation without a separate command. A manual use
  earlier in that rest satisfies the operation, so auto completion does not
  spend another stone. `camp status` shows the setting, eligible blades, and
  whether a whetstone will be consumed.
- A single stone is consumed only if at least one weapon gains the effect.
  Already sharpened weapons are skipped, and the same stone still covers
  every other eligible blade. No stone or no eligible blade is a harmless
  no-op in auto mode and an explanatory response to a manual request.
- *Sharpened* is per durable weapon instance: +1 physical damage on its next
  20 successful melee strikes. The bonus and strike count are config. A
  second sharpening does not stack or reset remaining strikes on an already
  sharpened weapon. Transferring the weapon transfers its remaining edge.
  When strikes reach zero, the effect ends. Inventory and inspection show
  remaining strikes; `conditions` shows a member-level summary.

## Recovery and integration

Camp rest and sharpening must use an operation ID tied to that rest. Persist
the pending intent and per-weapon completion before clearing it; retry after
restart/copyover must finish incomplete work without consuming a second stone
or sharpening the same blade twice. The item and companion equipment stores
must expose enough durable state to reconcile partial writes. No action
advances the global clock, and sharpening does not change rest recovery.

## Acceptance criteria

- Camp and inn bonuses apply to the correct living members once, remain
  exclusive, expire at the configured time, and recover after restart.
- One stone sharpens two or more eligible weapons across different company
  members and is consumed exactly once; non-bladed and already sharpened
  weapons remain unchanged.
- Manual and auto paths use the same eligibility and recovery logic. Turning
  auto off saves stones. Interrupted saves and copyover cannot duplicate an
  edge or consume a second stone.
- Combat applies and decrements each sharpened weapon's bonus only on its
  own successful melee strikes, including companion attacks.
