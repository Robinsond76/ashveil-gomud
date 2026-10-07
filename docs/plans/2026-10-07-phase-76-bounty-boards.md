# Phase 76: Bounty boards

A board in a town lists bounties on lair masters and named groups from the zones near it. A company takes one, kills the mark, and claims gold at any board; the proof is the chronicle. Spec: [Pillars phases](2026-10-07-pillars-phases.md) §76. The current world is temporary stock GoMud, so only the rules and test coverage ship, plus two boards to try them: Alderbrook green (room 2114) and the Frostfire Inn (room 61), both tagged `bounty-board`.

## Shape

- **Rules** in `internal/bounty` (pure): `Post` (a board's list for a window), `Reward`, `Nearby`, `State` (held bounties and settled postings), `Take`, `Settle`, `Expired`. Real time only.
- **Chronicle** (`internal/chronicle`): a new `group` kind (a named group broken) and `bounty` kind (a bounty claimed); `Entry.Zone`, set on boss and group deeds; `Filter.Zone` and `Filter.AfterSeq`.
- **Group deeds** are written in `modules/encounters` (`noteGroupBrokenLocked`) when a tracked ordinary group's last foe is down. `Composition.Name` (optional) names a group for players; its id, spaced and capitalised, when empty.
- **Module** `modules/bounties`: the `bounty` / `bounties` command, saved holdings (`bounties` plugin file), the `Company.Bounties` GMCP message, the userstate contributor and the purge listener.
- **Client:** a Bounties sub-tab in the Company window (held bounties with progress and Claim / Drop; the board with Take), checked at desktop and 360px in `scripts/browser/dock-windows-check.mjs`.
- **Help:** `help bounties` (aliases `bounty`, `bounty board`, `notice board`, `wanted marks`, ...), linked from `adventure`, `company`, `chronicle`, `encounters` and `webclient`; a hint in the Departure lesson. The two board rooms say there is a board.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| A board is a room tagged `bounty-board`; the room itself seeds its list, the zone's encounter band is its band. | The inn tag is the pattern; the world has no town flag. A tag a new world can set anywhere. |
| Postings are derived, not stored: the list is a seeded shuffle of the nearby targets for the board and the six-hour window. | Restart and copyover keep the board for free, nothing to purge or migrate, and tests can name the list. |
| Targets are what zones already have: a lair is a boss composition's first member, a named group is an ordinary composition. A bounty never scales a foe. | Difficulty rule: it comes only from entering a zone above the company's level. The bounty shows the zone's band and how it rates for the company. |
| "Nearby" is a band test: the target's band reaches no more than 2 levels below the board's band, and starts no more than 3 above it. A board in a zone with no band posts targets whose band starts at 6 or lower. | The world has no map distance; the band is the game's measure. The Frostfang inn sits in a bandless zone, so it posts the gentle zones. |
| Up to five postings a window, at most two lair masters (more only if no groups fill the board); lair masters lead. A new list every six real hours. | Mostly bands (frequent, small) with a lair or two; six hours is a session or a day's pass. |
| A band bounty asks for three groups of that composition; a lair bounty one kill. | One random group is a single fight; three makes it a short hunt. |
| Pay is gold only: 20 gold for each level at the middle of the band for a lair, 6 per level for each group (three). A level 8 zone (7-9) pays 160 / 144. | Reward within band, no power. Bosses reset a lair for half an hour per company and bands are random, so the board is bounded by real fights; bounties add gold to what the company fights anyway (errands pay at most ~1,056 gold in eight hours). Relics keep their own profit rule; bounty gold is fine under the economy rule's exception. |
| Proof is the chronicle: a `boss` deed (`mob:<id>`) or `group` deed (`group:<id>`) in the target's zone, numbered after the bounty was taken (`AfterSeq`) and not before it was taken in time (`Since`). | Spec: "the kill recorded in the chronicle". The seq floor stops a kill within the same second as the take from reading as earlier. |
| A boss deed now carries `Zone` (the room's zone), so the same master in another zone does not count; the Boss deed itself is unchanged. | Bounties name a lair in a zone. |
| Group deed: written once from the encounter record when every foe of an ordinary table group is down, to the group's owner (the leader). Not for boss groups (the mob's own death writes the Boss deed), for story-event groups, or for groups cleared away with survivors (fled). | The only real source of "a group was broken"; foes cannot be removed by the module with nobody standing, so all-down means won. |
| Take and claim only at a board; reading what you hold works anywhere. Not in a battle. | Deadfire's boards are places. Reading is free. |
| At most three held, one per mark; a held bounty lasts 24 real hours, then lapses (saved, checked on every read). Dropping is free. A claimed posting cannot be taken again from the same list. | Keeps the board a small errand list, not an ever-growing pile; one posting pays once. |
| Claim writes the saved state first (settled), then pays gold, then writes a `bounty` deed. A failed save refuses the claim. | A crash between save and pay loses the reward and never duplicates it (the errands rule). |
| The company is the leader, as everywhere. A follower's own chronicle gets its own boss deeds, so a bounty works for any party member whose log has the kill; group deeds go to the group's owner. | The chronicle is per leader id. |
| World content: two boards (Alderbrook green, Frostfire Inn) and room text that says so; no new mobs, items or compositions. | The world is temporary; boards need only exist to test the feature. |

## Not done, on purpose

- No escort, "bring it back alive" or item-proof bounties; no named-person targets (no such mobs outside lairs).
- No board-specific reputation or prices; no reward items (relics and drops keep their own profit rules).
- No party-shared bounties: a bounty is the leader's.
- The bestiary still reads `KillsOf`, not the chronicle.

## Acceptance (from the spec)

- Post, kill, claim through real commands: `TestTakeKillAndClaimAGroupBounty`, `TestABossBountyNeedsTheLairMasterInItsZone`, `TestTheCommandOnTheShippedBoardPaysGoldToTheCharacter` (shipped world, real room tag, real gold); the kills come from the real deed writers (`TestAWonGroupIsWrittenIntoTheChronicleOnce`, `TestBossKillAndRelicAreWrittenInTheChronicle`).
- Reward within band: `internal/bounty` tests, and the help page pins the numbers.
- Real-time rotation and restarts: `TestTheBoardIsTheSameAcrossARestartAndRotates`, `TestHoldingsSurviveARestartAndLapseOnTheRealClock`.
- `help bounties`: `TestBountiesHelp`; `TestTutorialHelpPointersExist` passes.
