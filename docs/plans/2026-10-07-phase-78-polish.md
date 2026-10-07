# Phase 78: polish for phases 60-72 (decisions, 2026-10-07)

A small pass over follow-ups the reviews of phases 60-72 left open. Each was
confirmed against master before it was touched. No balance changes, no new
mechanics.

## What shipped

| Follow-up | Real in master? | Fix |
| --- | --- | --- |
| Story scene window after a restart | Half. The terminal text and the server's page survived (`onPlayerSpawn` reshows it), but the `Event` GMCP message went out at spawn, before GMCP was accepted, so the web modal never opened. | The Event window asks for its page (`!!GMCP(Event)`) on the first `Char` message of a connection; `gmcp` queues `GMCPEventRequest`, which fires `storyevents.OnPushRequest`, and the module sends the modal message again with no terminal text. `event` still brings it up any time. |
| Orders menu shows duplicates | Yes: it offered an order the member already had, which the command refuses. | `Company.members[].order_cmds` carries each order as `orders add` reads it; the menu leaves those out. |
| Orders: "fall back a row" and forcing a class ability | Not bugs. `help orders` never claimed either (phase 61 left them out for formation and `abilityPass` reasons). | Nothing built. See decisions. |
| Round heading says "critical" when the crit was absorbed | Yes, when a round's critical strike was taken whole by armor but another strike got damage through. | The `Company.Battle.Event` crit flag now follows `combatstream.CritLanded`, as the text log and `why` already did. |
| Stance menu lists all four stances | Yes. | `Company.members[].stances_fit` (what the gear can use, `[]` for none, omitted when gear can't be read) filters the menu; with nothing to offer the Combat tab shows a line saying what is needed instead of a button to an empty menu. |
| Background town line greets every member | Yes: `{who}` named the whole deed. | A `member_tag` line says `{who}` as the first member who carries the tag. The deed itself is unchanged. |
| Duty picker can't tell same-named companions apart | Yes: the name matched either, and the first won. | `camp duties #2 watch` (the companion's number, as `strategy` and `orders` take it). Same-named members show as `Mira (#1)` and `Mira (#2)` in the picker and the view, and the picker sends `#id`. A bare name that fits two is refused with their numbers. This also helps `camp prepare`, which shares the resolver. |
| "nameless-..." in the Character panel | Yes, until a character is named at creation. | The panel shows no name for the placeholder. |

## Decisions

- **Ask for the page, don't delay the send.** Retrying the push on a timer
  would guess at when the web client is ready; the client knows. One request
  per page load is enough (a reconnect reloads the page).
- **Crit flag fixed on the server, not in the heading.** The heading only has
  the flag and the damage; the strikes are the truth. The battle screen's
  flash and "strikes hard" line use the same flag, so they are right too.
- **`order_cmds` rather than matching text on the client.** The words can be
  reworded; the command form is what the command compares.
- **Stances: fit, not ready.** The menu offers what the member could use now.
  A set stance that no longer fits stays visible as "idle" (phase 69).
- **A background line greets the first member with the tag.** Greeting all
  tagged members would restate the problem; one is enough and deterministic.
- **Left out, again, with reasons:** "fall back a row" (formation is durable
  state; a battle-only column touches every reach gate) and forcing a
  fighter's class ability (`abilityPass` gates and cooldowns). The help never
  promised either, so there is nothing to correct. They stay follow-ups.

## Tests

- `TestBattleEventCritFlagNeedsDamageThrough`, `TestCompanyStancesFitGear`,
  `TestCompanyPayloadBattleOrders` (order_cmds), `TestEventWebRequestReachesTheStoryEventHook`
  (gmcp); `TestTheWebClientCanAskForTheWaitingPageAgain` (storyevents);
  `TestABackgroundLineGreetsOnlyTheMemberWhoHasIt` (townsfolk);
  `TestDutiesTellApartCompanionsOfOneName` (camping).
- `scripts/browser/dock-windows-check.mjs`: the stance menu offers only fitting
  stances, a member with nothing that fits shows a line not a button, and the
  orders menu leaves out an order already set.

## Help

`help stances`, `help orders`, `help campduties`, `help events` and
`help townsfolk` say what changed.
