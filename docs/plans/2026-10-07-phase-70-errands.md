# Phase 70: Errands

A companion sitting out the formation is sent from an inn on an escort, a hunt or a scouting job of a chosen real-time length. It leaves the map, saved with a return time, and comes home with modest gold, a small find, a lair rumour or a wound, plus a chronicle line. Spec: [Pillars phases](2026-10-07-pillars-phases.md) §70. The current world is temporary stock GoMud, so only the rules and the test coverage ship, not new world content: any room tagged `inn` in any zone works.

## Shape

- **Rules** in `internal/errands` (pure, no GoMud types): `Kind` (escort, hunt, scout), `Length` (short 30 min, medium 2 h, long 8 h), the saved `Errand`, `Pay`, `WoundRisk`, `Resolve`. Real time only (Unix seconds); nothing reads or advances the world clock.
- **State** on the companion record (`Companion.Errand`, in `internal/company`): kind, length, zone, start and return time, and the values the outcome is computed from, frozen at the send (level, the zone's band, maximum health, a seed). `Companion.Away()` is "off the map" (separated or on an errand) and replaces `Separated()` at every site that means "not present": roster and survival (`Away`), members and GMCP status (new `MemberErrand`), gear and carry, banter, bonds, creature repair, class and training refusals, conditions, `restoreForLeader`.
- **Module** `modules/company/errands.go`: `errands` and `errand send|recall`, `startErrand`, `tickErrands` (once a round, from `onNewRound` beside `tickSeparations`), `finishErrands`, the `Company.Errands` panel (`company.ErrandsOf`, the same seam shape as bonds), and a small `errandWorld` seam for unit tests.
- **Client:** `Company.Errands` GMCP extra and an Errands sub-tab in the Company window (a card per companion with job and length selectors, Send, Call back, a countdown the client runs from the server's clock, the latest deeds); `errand` joins the status vocabulary in the Status, Vitals, Combat and Battle windows.
- **Help:** `help errands` (hub links from `adventure`, `company`, `chronicle`, `webclient`; keywords and aliases), a tutorial hint in the Departure lesson, the chronicle page lists `errands`.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| Sending needs an inn: a room tagged `inn` (the tag the inn rest uses), with the leader out of any fight, journey or camp rest. | "From a town": the world has no town flag, and the inn is the settlement room players already return to. Reuses the rule players know. |
| Any companion who is alive, with the leader and not fighting can go. It leaves the formation as it goes, the cell it held saved with the errand, and takes that cell again on return when it is still free (as a revived companion does). | Review change: the build required "not in the formation" (the spec's words), but an unplaced companion still fights (`internal/hooks/combat_engagement.go` counts every companion in the room), and the company auto-places all four companions, so the gate only added a `formation clear` step and its help wrongly said unplaced members sit out. Sending anyone already takes a fighter away, which is the real cost. |
| The errand's zone is the zone the leader stands in; its level band (37b's `Encounters.Band`) scales the pay and the wound risk. A zone with no band uses the companion's own level for both. | The band is the game's measure of a zone's level (difficulty rule). No destination picker: it would need a world map of towns the temporary world doesn't have. |
| Real-time expiry saved on the record. It returns only when the leader is online and free (the same test a separated companion uses), so a due errand waits for the next login; a long errand can be sent before logging out. | Spec: "saved expiry"; MUDs are played in sessions. Waiting for the leader keeps the return out of fights, journeys and camp rests. |
| The outcome is decided at the send, from a seed saved with the errand, and applied once at the return. | Restart, copyover or a retried tick cannot reroll it, and a failed save retries the same result. |
| Pay is 3 gold per level in the band's middle plus 1 per companion level, times the kind (escort 100%, hunt 140%, scout 60%) and length (short 100%, medium 250%, long 600%) shares. Level 8 in a 7-9 band: 32 short escort, 44 short hunt, 19 short scout, 192 long escort. | Modest: a long errand pays 6 times a short one for 16 times the time, so it is for gaps between sessions, not a wage. Four companions all paid in gold for a long hunt at level 8 would earn at most 4 x 264 = 1,056 gold in eight hours, and about 85% of hunts come back with pay (gold, or a find plus the rest in gold). Numbers from arithmetic, not sims. |
| Outcome weights (not wounded): escort gold 60 / item 25 / rumour 15; hunt 45 / 40 / 15; scout 20 / 15 / 65. Wound risk: escort 0, hunt 15, scout 8, plus 8 per level under the band (cap 60), halved above it. A wound is a lasting wound of 20% of the companion's maximum health (`wounds.Beaten`), saved in the same save that brings it home. | Escorts are safe for a companion who matches the zone, hunts pay best and bring finds, scouts bring rumours; underlevelled sends are risky, which is the difficulty rule applied to errands. |
| An item comes from `ErrandItemIds` in the company config (four small gathered goods) and only when its value is no more than the gold the same errand would have paid; the rest of that gold is paid in coin (review change: the build gave the item instead of the gold, so a find worth 3-20 replaced 44-264 gold and a hunt earned less than an escort). If none fits, it pays the gold. | Economy rule: gathered or bought goods never resell for profit, and an errand must not become a loop. The find plus its coin is never worth more than the gold (merchants pay a quarter of an item's value, so selling it nets less), and the pool is gathered goods only. Relics and rare drops are not on the list. |
| A rumour names a lair master (a boss composition in the zone's tables) the company has not slain, else any; it says "is said to hold court nearby". No lair in the zone, no rumour: it pays gold. | Spec: "a lair rumour". The chronicle's boss deeds say which are slain. No new data. |
| No experience, no loyalty change, no bond or opinion effect. | Keeps errands a small convenience; those systems have their own sources and cooldowns. |
| `errand recall` brings a companion home at once with nothing; it needs the same freedom as a return. No chronicle line for a recalled errand. | A leader who needs the companion now should not wait 8 hours; recalling must never be a way to collect a reward early. |
| Dismissing a companion while away is allowed: the errand goes with the record and the gear returns to the cargo as in any dismissal. | The dismissal path already works on a record with no live mob (as for a separated companion). |
| Formation `move`/`swap` of an away companion is refused (`ErrMemberAway`); `clear` is allowed. The `formation` view lists it as away. | Placing someone who is not on the map would put an absent member in the battle line. |
| A new chronicle kind `errand` ("came back from a hunt at Zone with 32 gold"), `Ref` `errand:<kind>`. | Spec: "a chronicle line". Phases 68 and 73 can read it by kind. |
| An away companion's `Roster` entry is `Away`, so hunger, thirst and fatigue are neither spent nor recovered, and it takes no inn bed. | Same rule as a separated companion; nothing about survival loops through errands. |

## Not done, on purpose

- No destination choice, no multi-companion parties, no errands from camps or on the road.
- No errand gear: the companion keeps whatever it wears and carries and returns with it.
- No reward from a recalled or dismissed errand; no experience.
- A crash between the company save (errand cleared, wound saved) and the leader's gold/item save loses that reward. It never duplicates one.

## Acceptance (from the spec)

- Errand survives restart: `TestErrandThroughTheRealCommandsAndRoundTick` (the registry is read again mid-errand; a login does not return it) and `TestOutcomeIsTheSameAfterARestart`.
- Outcomes within band: `internal/errands` tests (pay by band, kind and length; wound risk by level; the spread of 4000 seeds per kind), `TestItemOutcomeNeverWorthMoreThanTheGold`, `TestItemWithNothingThatFitsPaysGold`.
- The away companion is absent everywhere it should be: `TestAwayCompanionIsAbsentEverywhere` (roster, members, formation, relog, separation, status, conditions) and the wiring test (load, status, formation command).
- `help errands`: `TestErrandsHelp`; the tutorial pointer test passes.
