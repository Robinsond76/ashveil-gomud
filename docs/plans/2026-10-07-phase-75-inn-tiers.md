# Phase 75: inn room tiers (decisions, 2026-10-07, full autonomy)

Plan: [pillars phases, 75](2026-10-07-pillars-phases.md#75-inn-room-tiers).

## Decisions

- **Three rooms, set by room tags.** Every room tagged `inn` lets the common
  room (today's stay, unchanged: 5 gold a member, Well Rested for 30 minutes).
  A room also tagged `inn-private` lets private rooms (15 gold, 1 hour) and one
  tagged `inn-suite` a suite (40 gold, 2 hours). Tags, prices and lengths are
  config (`InnPrivateRoomTag`, `InnSuitePricePerMember`,
  `InnSuiteWellRestedDuration`, and so on). *Why:* the plan asks for two or
  three tiers an inn; tags let a small roadside inn sell only a bed and the
  big towns sell more, and they follow the `inn` tag the errands, gigs and
  looks features already use. In the default world the Waymark Inn and Frostfire
  Inn have all three; the Drovers' Rest, Cinder Arms and Highwatch Hearth have
  private rooms; the rest are common only (the world is temporary, this is
  only enough to test).
- **Length, never strength.** A dearer room changes only how long Well Rested
  lasts. The buff is the same buff (1030: walking strain halved, +2 speed,
  +2 perception), and every room gives the same fatigue, wounds knit, full
  health and mana. *Why:* Robinson's at-level fight tuning (easy fights, rest
  because health runs low) is being set right now; a bigger buff, or any health
  or mana difference between rooms, would move that. A longer Well Rested only
  keeps the same +2 speed up for a long expedition, which is a convenience, not
  a bigger company. No new buff, no sim: the combat effect per fight is the
  shipped Well Rested's, for longer. Recorded in fights: no change per fight;
  at about a minute a fight the suite's two hours is the same buff over a
  longer stretch, not a stronger one.
- **Gold is the only price.** The cost is per member, takes the settlement's
  standing markup (`help standing`) and refusal as it does today, and is paid
  from the leader's gold. The buff earns nothing, so the economy rule holds;
  a suite for five members is 200 gold, against about 46 gold an hour an
  errand brings in at level 8.
- **A short buff never cuts a long one.** `applyTier` skips a same-tier grant
  that would shorten the rounds a member already holds, so a common room after a
  suite leaves the suite's hours running (the same tier used to refresh to the
  new length). This also covers a camp Rested over a longer Rested.
- **Command.** `inn rest` stays the common room; `inn rest private` and `inn
  rest suite` (aliases `common`, `room`, `single`). `inn` lists every room this
  inn lets with its price for the whole company, markup included, and its Well
  Rested length. A room the inn lacks is refused with nothing charged.
- **Stay record.** `InnStay.Tier` (`common` is saved empty so earlier saves
  read the same; unknown tiers fail `Validate`). The length is read when the
  Well Rested is granted, on the game loop, so a stay that finishes across a
  restart or copyover, or while the leader is offline, still gives its room's
  length; companions absent at the grant are owed the same length.
- **Web client.** The Camp tab shows a button per room at an inn ("Suite room,
  80 gold") with the length in its tooltip, from `Company.Camp.inn_rooms`.
  The plain "Inn" button still shows status.
- **Not done, on purpose.** No per-tier opinions (companions already like or
  dislike a bed against a rough camp), no room quality effects on rests at
  camp, no room booking or reservation, no tier on gigs.

## Acceptance

- Each tier through a real stay: `TestEachRoomTierLeavesItsOwnWellRestedLength`
  (price paid, rounds granted to the leader and companion, owed length), plus
  `TestInnCommandPicksTheTier` through the command.
- Help: `help inn` rewritten with the three rooms and prices; alias topics
  `suite`, `private-room`, `inn-rooms`; the tutorial's rest lesson points at
  `inn rest suite`. Tested by `TestInnHelpDescribesTheRooms` and
  `TestTutorialHelpPointersExist`.
- World: `TestShippedInnsOfferTheirRooms`.
- Web: `TestCompanyCampPayloadCarriesTheInnRooms`; buttons in
  `window-company.js`.

## Review (2026-10-07)

Accepted:

- **No word when a cheaper room keeps a longer rest.** A common room after a
  suite granted nothing and said nothing, so the player paid without learning
  why. The grant now says the company is still Well Rested from an earlier,
  longer stay (`TestShorterRoomSaysTheLongerRestStillRuns`).
- **Restart coverage.** Added `TestSuiteStayKeepsItsLengthAcrossARestart`: a
  suite stay finished across a reload still grants two hours.
- **Camp tab browser check.** `scripts/browser/dock-windows-check.mjs` now
  checks the room buttons, their command, the hidden state while resting, and
  the phone fit (360 px). Screens: `75-camp-inn-rooms.png`,
  `75-camp-inn-rooms-phone.png` in the project's `screens/`.

Checked and kept: the buff strength is unchanged in every room (only
duration), so the at-level fight tuning is untouched and no gold returns from
a stay; a completed stay waits in `m.stays` until the grant, so the tier read
at the grant is the stay's own; `innRoomRows` takes `m.mu` only after
`CampStateOf` releases it.

Also fixed on the way: after merging master (phase 74), the browser check's
"bestiary: an open entry stays open across a refresh" failed every run on
master too. The check set an entry open and refreshed before the async
`toggle` event recorded it; it now waits for that event. Test-only.
