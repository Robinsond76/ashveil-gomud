# Phase 40e: structured combat-event messages

Status: **approved 2026-10-06 under the owner's delegation**; open questions are decided in the [remaining roadmap](../plans/2026-10-06-remaining-roadmap.md#decisions-on-open-questions) (handoff rule 20). Part
of the [visual client milestone](2026-10-05-visual-client-milestone-design.md).
Roadmap item 5a. No art. **No visible player change**; it is the data feed
for the battle screen (40f and 40g). It can be built in parallel with
40a–40d.

## Goal

Send the web client one structured message for each combat happening,
such as an attack, cast, heal, status or fall. Each message is released in
step with the paced narration line it belongs to, so animations match the
text the player reads.

## Prior art (code as of 2026-10-05)

- **`internal/combatstream`** (Phase 29b) already models every happening
  as an `Event`. Its fields include Seq, Kind, Round, FightID, Source and
  Target refs, Outcome, Defenses, Damage, Crit, WeaponType, SpellId,
  Amount, HeldBack, BuffId, Status and Rule. The kinds are:
  - `attack`;
  - `spell-hit`, `heal`;
  - `cast-start`, `cast-progress`, `cast-complete`;
  - `windup-start`, `windup-land`, `interrupt`;
  - `status-applied`, `status-tick`, `status-expired`;
  - `wound-change`;
  - `guard-used`, `guard-exhausted`;
  - `yield`, `flee`, `death`, `mercy`;
  - `ability`;
  - `fight-start`, `fight-end`;
  - `target-change`, `focus-change`.

  Consumers subscribe with `Subscribe(sink)`. **The stream only reports**:
  it never owns state, is never persisted, and never advances the clock.
- **`internal/combatpace`** (Phase 29f) holds each player's combat text and
  releases it with the player's pace (fast, normal, slow or off). It is
  wired in `internal/hooks/combat_pace.go`. Held lines are never dropped;
  they are flushed on a move, quit or copyover.
- **GMCP `Company.Battle`** (Phase 32g2) is a *snapshot*: cells, labels,
  health in scout's words, targets, darkness and narrow ground. It uses IDs
  `m:<instance>` for enemies and member keys for the company.
- **Information rules already in force:**
  - enemy health is shown in scout's words, never numbers;
  - every damaging hit's damage appears in the narration;
  - in darkness the enemy can't be made out;
  - another company's private conditions are never shown.

## Owner decisions

- An OB64-style battle screen; the narration stays (2026-10-05).

## Proposed for approval

### Message

Module `Company.Battle.Event`, sent to a player for happenings in **the
fight they take part in**. The player must be its company leader, an
allied party member in the same fight, or — for a companion's events —
the companion's leader.

```json
{"fight": 12, "round": 341, "events": [
  {"seq": 5501, "kind": "attack", "src": "me", "tgt": "m:88",
   "outcome": "hit", "damage": 6, "crit": true, "weapon": "slashing",
   "defenses": [], "status": "bleeding"},
  {"seq": 5502, "kind": "status-applied", "tgt": "m:88", "status": "bleeding"}
]}
```

- **IDs** match `Company.Battle` and `Company`:
  - `m:<instance>` for enemies;
  - the member key for company members (the leader, companions);
  - `me` for the receiving player;
  - `o:<n>` for an outsider shown in `Company.Battle.others`;
  - `a:<leaderUserId>:<memberKey>` for an allied company's member in the
    same encounter (for 40f's reserve formations).
- **Fields** are passed through from the stream event: kind, outcome,
  damage, amount, held-back, crit, weapon type, spell ID, status name,
  defenses and rule. They are renamed to short JSON keys and omitted when
  zero.
- **Allied happenings** in the same encounter are included, so 40f can
  animate allies. They obey the same limits: no ally health numbers and no
  private ally conditions. An ally's status appears only when the
  narration shows it.
- **Never sent:**
  - enemy health or maximums;
  - any unseen enemy's identity;
  - another company's private statuses;
  - any number the narration does not already show.
- **In darkness** (`Company.Battle.dark`), enemy refs become `?` unless the
  enemy is already labelled in the narration. The client then draws a
  silhouette.
- `fight-start` carries the opening cells. `fight-end` carries the outcome
  (victory, defeat or broken off).

### Pacing

- Each event is **queued in the player's pacer queue** in emission order,
  next to the text lines held in that round. It is released together with
  the next text line that follows it, or at the round's drain if none
  follows. A new "data" entry in `combatpace` takes no beat of its own.
- With pace `off`, events go out immediately.
- Flushes (move, quit, copyover) send the held events with the held text,
  so the client never loses one silently. After a copyover the fight is
  gone (combatstream rule). The client then resyncs from the next
  `Company.Battle` snapshot.
- **Batching:** the events released at one moment go in one message.

### Opt-in

The feed is sent only to clients that support the
`Company.Battle.Event` GMCP module. The web client opts in; text and
Mudlet clients see no change.

## State and persistence

There is no new state. Events are transient, and the pacer's existing
copyover contributor already handles held text. Held data entries follow
the same "never dropped, flushed" rule; after a copyover they are simply
not restored, like the fight.

## Integration points

| Area | Change |
|---|---|
| `internal/combatpace` | data entries in a player's queue: hold, release order, flush |
| `internal/hooks/combat_pace.go` | send released data as GMCP |
| `modules/gmcp` | a new `gmcp.CompanyBattleEvent.go`: stream sink, recipient filter, payload builder, dark masking, ID mapping shared with `Company.Battle` |
| Web client | registers support. The handler stub logs events until 40f |

## Player help and tutorial

None. There is no visible change. 40f's help documents the battle screen
that uses it.

## Acceptance tests

1. A practice fight sends a `Company.Battle.Event` stream to the leader,
   covering attack, crit, status, cast, heal, death and fight end. Its IDs
   match those in `Company.Battle`.
2. A companion's attack reaches its leader. A different company's fight in
   the same room sends **nothing** to this player.
3. Enemy health never appears in any payload. In darkness, unlabelled
   enemy refs are `?`.
4. **Ordering:** with pace `normal`, each event is released no later than
   the text line that narrates it, and no earlier than the line before it
   (asserted on the pacer). With pace `off`, events go out at once.
5. A room change mid-round flushes held events along with held text.
6. A client without the module receives nothing. Text output is
   byte-for-byte unchanged against the existing narration tests.
7. The game clock and the combat round count are unaffected.

## Built (2026-10-06)

Built as designed, with these decisions (best judgment, delegated):

- **Own fight only.** A fight has one leader and each allied company has its
  own fight, so the feed carries the leader's fight and no `a:` refs yet;
  40f's reserve formations can add an allied relay. Outsiders are `u:<id>` or
  `m:<instance>`, as in `Company.Battle.others`, not `o:<n>`.
- **Fight-start carries the roster** (`company`, `enemies` refs), not cells:
  `Company.Battle` is the one source of cells and would otherwise be read a
  second time at an unsettled moment.
- **Masking.** In the dark, or for an enemy that is hidden (or was when last
  seen), its refs are `?` and statuses on it are not sent. The design's
  "unless already labelled in the narration" would need narration tracking;
  masking is the safe subset.
- **Opt-in.** The web client takes the feed by being the web client (it sends
  no `Core.Supports.Set`); any other client must list `Company.Battle.Event`.
- **Pacing.** `events.CombatData` is queued as the event is emitted, so it
  dispatches in order with the round's `Message`s. `combatpace` data entries
  take no beat; an event goes out with the next text line after it, so an
  event emitted after its own narration line (spell results) goes out with
  the following line or the round's last.
- **Client.** `Client.onBattleEvents(fn)` in `webclient-core.js` delivers each
  message; events never enter `GMCPStructs`. 40f adds the listener.

## Open questions

1. Should spectators (players in the room but not in the fight) receive
   events? The proposal is no, matching `Company.Battle`.
