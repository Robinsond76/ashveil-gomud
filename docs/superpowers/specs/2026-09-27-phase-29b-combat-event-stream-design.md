# Phase 29b: Combat Event Stream and Battle Summary — Design

The second slice of the [combat roadmap](2026-09-26-combat-presentation-roadmap.md)
(build order decided 2026-09-27: 29a, then 29b). The owner-approved
direction is the [29b spec](2026-09-26-combat-event-stream-design.md); this
document settles the design for implementing it.

The owner asked to "continue with next phase" (2026-09-27). The one open
decision in the spec (copyover loses the summary) and the smaller ones
below were settled by applying this design's recommendations, as earlier
phases did under "carry on". They are marked **(recommendation applied)**
and listed for the owner in `docs/PROJECT_STATUS.md`.

## Prior-art check (2026-09-27)

- **Where an attack is resolved:** six call sites.
  - `internal/hooks/NewRound_DoCombat.go`: `AttackPlayerVsPlayer`,
    `AttackPlayerVsMob`, `AttackMobVsPlayer`, and `AttackMobVsMob`.
  - `internal/hooks/combat_formation.go`: the two interception
    resolvers, `resolveInterceptedMobAttack` (enemy → companion that
    shields the leader) and `resolveInterceptedAttackOnLeader`
    (enemy → leader that shields a companion).
  - Each gets a `combat.AttackResult` with `Hit`, `Crit`,
    `DamageToTarget` (the damage applied, before any clamp), and the
    `BuffSource`/`BuffTarget` ids the blow applies.
- **Spells:** `DoCombat` runs a cast's `onWait` rounds, the success roll
  (fizzle), and the script's `onMagic`. It measures harm spells' damage to
  mobs (`mobHealthBefore`) only to credit the player. Healing is never
  measured. Cast start is set in `characters.Character`, which knows no
  user or mob ids.
- **Death:** `handleAffected` at the end of `DoCombat` sends `suicide` to
  a mob under 1 HP and to a player at −10 or below, and queues
  `PlayerDrop` for a downed player. The actual death is processed after
  the round.
  - **Gap found:** the two interception resolvers never add the
    interceptor to the round's affected lists. So an interceptor brought
    under 1 HP by an intercepted blow isn't sent `suicide` that round,
    and can swing again on its own turn.
- **Target changes:** 29a's upkeep (`keepCompanyEngaged`,
  `keepPartyEngaged`/`aimPartyMember`) and the in-turn reassignment
  (`reassignPlayerTarget`, `reassignCompanionTarget`).
- **Fleeing:** only players flee (the `Flee` aggro in `DoCombat`). Mobs
  have no flee.
- **Fight identity:** nothing today. 29a's upkeep already finds, each
  round, which company is engaged with which enemy party
  (`companySide.engagedWith`), and 11a's party ids (`mobparty.Party.ID`)
  are derived from the group tag, so they are stable across rounds.
- **The events bus:** `internal/events` logs an error for every 20
  events of a type nobody listens to. 29b has no bus consumer yet.
- **Settings:** per-user toggles live in `user.GetConfigOption` and are
  flipped by `set <name>` (`auction`, `tinymap`).

## Decisions

1. **A new pure package, `internal/combatstream`.** It depends only on the
   standard library.
   - **`Event`:** a `Kind`, a sequence number, the round, the fight id,
     the enemy party id, the room, a `Source` and `Target` (`Ref`: user
     id or mob instance id, mob id, name, and company leader and member
     key when the actor is in a company), a `Previous` target (target
     changes), and kind-specific fields: `Outcome`, `Damage`, `Crit`,
     `WeaponType`, `SpellId`, `Amount`, `HeldBack`, `BuffId`, `Status`.
   - **Every kind in the spec is declared.** Those with a producer today
     are emitted now (decision 3). The rest (wind-ups, interrupts,
     status expiry, wounds, guards, yield, mercy, cast start) are declared
     for the phases that add them (30a–30e).
   - **The stream is a `Stream` value** with a package default. Tests
     make their own or reset the default.
2. **Consumers subscribe to the stream, not the bus (recommendation
   applied).** `combatstream.Subscribe(func(Event))` returns an
   unsubscribe func. A subscriber runs on the game loop, synchronously,
   and must be cheap.
   - **Why not the bus now:** 29b has no bus listener, and the bus logs
     an error every 20 events with none. A later phase that needs the bus
     (31's GMCP panel) adds a one-line bridge subscriber when it adds its
     listener.
   - **Why synchronous:** the per-fight totals and the summary need every
     event of the fight before the fight-end event, in order. A
     subscriber called in `Emit` gets exactly that.
3. **Producers, all in `internal/hooks`,** emit at the point where they
   now build text, and still send today's text. **No player-visible
   output changes except the summary** (and the death fix below).
   - **Attack:** one event per `AttackResult`, at all six call sites.
     `Outcome` is `hit`, `miss`, or `crit`; `Damage` is
     `DamageToTarget`; `WeaponType` is the attacker's weapon subtype
     (`stabbing`, `bludgeoning`, ...), or empty when unarmed.
   - **Status applied:** one event per buff id in `BuffTarget` and
     `BuffSource`, named from the buff spec.
   - **Casts:** `cast-progress` for each `onWait` round and
     `cast-complete` with `Outcome` `fizzled` or `cast`, for players and
     mobs.
     - **Spell damage and healing** are measured the same way for both:
       every target's health before and after `onMagic`. A loss is a
       `spell-hit` with `Damage`; a gain is a `heal` with `Amount`.
       Health is never changed by this; it is only read.
     - **Cast start** needs the actor's id where the cast is set (in
       `characters`); it is declared and left to 30d, which owns casting
       phases.
   - **Target change:** every reassignment (the upkeep's three, and the
     two in-turn reassignments), with `Previous`.
   - **Flee:** a player's successful flee.
   - **Death:** from `handleAffected`: a mob under 1 HP is `slain`; a
     player at −10 or below is `slain`, and one under 1 HP is
     `incapacitated`. The killer is the last actor in the fight to damage
     the victim, filled in by the stream.
   - **The interception gap is fixed (found during design).** The two
     resolvers add the interceptor to the round's affected lists, so a
     killing intercepted blow ends the interceptor that round, as a
     direct blow does, and its death is reported.
4. **Fight identity: one company against the enemies it fights in a
   room** (revised after review: the spec's "one enemy party" gave three
   ungrouped wolves three fights and three summaries, and let a
   re-forming group leave an enemy in two fights).
   - **Start:** 29a's upkeep already finds each engaged company/party
     pair at the top of the round. When the leader has no open fight in
     that room, the stream opens one (a new id, from 1, in memory) and
     emits `fight-start`. Every party the company engages there joins the
     same fight, and new members join it each round. Each event names its
     enemy actor's own party.
   - **Company side:** the leader, and each companion in the room when the
     fight opens or while it runs (a companion summoned mid-fight joins).
   - **Solo fights** (no companion present, the same rule as 29a's
     upkeep) and fights between two outsiders are not fights: their
     events carry fight id 0 and no summary is made.
   - **End,** checked twice per round:
     - **at the end of the round,** after deaths are reported: every
       company member is dead (`defeat`; a company that walked or fled
       away is not beaten), or every enemy of the fight is dead or gone
       from the room and no other mob there is fighting the company or
       would join (`victory`);
     - **at the top of the round,** after the upkeep: no living member of
       either side is attacking the other in the room (an enemy still
       beating a downed leader counts, as in the upkeep)
       (`broken-off`: a flee, a `break` that held, a walk out, a logout).
       This is checked after the upkeep so a party member that the
       upkeep draws in keeps the fight open.
   - The fight-end event carries the summary (decision 5).
5. **Battle summary.** At a fight's end the stream builds a `Summary`
   from the fight's events, and the leader is sent its rendering, if
   they are online and their `battlesummary` setting is on.
   - **Contents, from events only:** damage dealt by each side (weapon
     and spell), healing and what a wound held back, damage by company
     member (most first), the highest single hit, interrupts, guards,
     effects applied (by name), kills by company member, how each enemy
     ended (slain, beaten for a practice foe, fled, left, or still
     standing; spared and yielded come with 30e), and the company's
     health at the end (a member slain in the fight reads as fallen).
   - **Company health at the end** is read from the live world when the
     fight ends and passed to the stream. It is never computed from
     events: the stream is never the source of truth for health.
   - **A line with nothing to report is left out** (healing, interrupts,
     guards, effects, kills). Damage, the enemies, and the company are
     always shown.
   - **Layout (the spec's, with the heading by outcome):**

     ```
     ── The fighting is over ──
     Damage dealt   Company 96 · Enemies 38
     Most damage    You 23 · Garrick Vane 21 · Ysolde 18
     Highest hit    Garrick Vane 11 on bandit captain (critical)
     Effects        dazed 1
     Kills          Garrick Vane 2 · You 1 · Ysolde 2
     Enemies        bandit captain slain · bandit cutthroat slain · ...
     Company        You 12/14 · Garrick Vane 6/15 · Tamsin Reed fallen
     ```

     The heading is `The fighting is over` (victory), `The company is
     beaten` (defeat), or `The fight breaks off` (broken off). The
     leader reads their own name as `You`. 29c restyles the words; 29d
     adds ordinals to repeated enemy names.
   - **The setting:** `set battlesummary` toggles it; it is on when unset.
     `set` with no arguments lists it.
6. **Totals are in memory only (recommendation applied).** A copyover or
   restart mid-fight loses the fight's summary and its open fight, not
   the fight itself: `Aggro` is not persisted either, and 29a's upkeep
   resumes a fight from live state. On the first round after, the fight
   reopens with a new id and its summary covers the rounds after the
   restart. Riding on the company record would write every blow to disk
   for a summary.
7. **Bounded memory.** A closed fight is dropped once its summary is
   built. An open fight's events are folded into totals as they arrive,
   not kept. Unended fights can't leak: the top-of-round check closes
   any fight whose leader is gone.

## Module

- **`internal/combatstream`** (new, pure): `event.go` (kinds, `Ref`,
  `Event`), `stream.go` (`Stream`, `Emit`, `Subscribe`, fight open,
  lookup, and end), `summary.go` (`Summary`, the fold, `Render`).
- **`internal/hooks`:**
  - `combat_stream.go` (new): refs from users and mobs, the emit helpers
    each producer calls, fight tracking at the top and end of the round,
    and the summary delivery;
  - `NewRound_DoCombat.go`, `combat_formation.go`, and
    `combat_engagement.go` call the helpers.
- **`internal/usercommands/set.go`:** the `battlesummary` toggle, and
  `help set`.

No persistence changes.

## Invariants

- **The clock:** the stream reads `NewRound.RoundNumber`; it never
  advances the world clock or the round count.
- **Restart and copyover:** nothing is persisted (decision 6).
- **Locks:** the stream has one mutex. `Emit` copies its subscribers
  under the lock and calls them after releasing it, so a subscriber may
  emit or read the stream without deadlock. Hooks never hold a stream
  lock across a world call.
- **Never the source of truth:** the stream only reports. No health,
  aggro, or state is read back from it.

## Acceptance criteria

- **Unit tests** in `internal/combatstream`: sequencing; fight open,
  merge, lookup by either side, and end; kill credit to the last
  damager; the summary fold (totals, most damage, highest hit, effects,
  kills, enemy endings); the rendering with lines left out; subscribers.
- **Wiring through the real round** (`modules/company`, extending 29a's
  5v5 `brawl`): with a subscriber on the default stream,
  - the event sequence is `fight-start`, attacks, deaths, and one
    `fight-end`, all with the fight's id, and every attack between the
    two sides carries it;
  - the summary's damage totals equal the sum of the attack and spell
    events for each side, and its kills equal the death events;
  - the leader receives the summary text once, at the fight's end;
  - `set battlesummary` off: no summary text, the events still flow;
  - a solo fight produces events with fight id 0 and no summary.
- **Death fix:** a leader dropped by an intercepted blow is reported
  down in that same round (wiring test; the interception needs a live
  world). Over a whole 5v5, nobody strikes after their death.
- **Also through the round:** a real `flee` (broken off), ungrouped
  enemies (one fight), and Minor Heal and Magic Missile (cast progress,
  cast complete, heal, spell hit, counted in the summary).
- **Player help** (added at the owner's request during the phase): help
  pages for the summary and every combat system before it, indexed and
  linked from `help combat`, pointed to from the tutorial, and tested.
- **Existing combat text is unchanged:** the 27c practice-fight and 29a
  wiring tests still pass untouched.
- `go test -race ./...`, `make generate`, and `make validate` pass. The
  independent review is recorded.

## Deferred

- **Narration from events:** 29c.
- **Kinds without producers:** cast start and interrupts (30d), status
  expiry and critical effects (30a), wounds and held-back healing (30b),
  guards (30c), yield and mercy (30e).
- **A bus bridge:** with its first bus listener (31).
- **Each company player's summary:** companies have one player today.
