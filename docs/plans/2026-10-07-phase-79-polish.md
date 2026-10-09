# Phase 79: wider polish pass (decisions, 2026-10-07)

A hunt for rough edges across commands, help text and the web client,
starting past the phase 78 list. Found by playing scripted telnet sessions
against a private server (recruits at the Waymark Inn, test-area fights,
every phase 60-77 command as a fresh solo character and with a company),
live web-client screenshots at 1280 px and 390 px, and an audit of the
phase 60-77 help pages against the code. No balance or economy changes.

Paused on 2026-10-07 at Robinson's request while the combat overhaul (82a-82d)
went through, and resumed on 2026-10-08 on top of it: master was merged in,
and every web-facing fix was re-checked on the docked battle screen (82a)
and the turn-order strip (82d) at 1280 px and 390 px against a live server.

## What shipped

| Finding | Where | Fix |
| --- | --- | --- |
| Web login eats a GMCP request: the creation window sends `!!GMCP(Char.Creation)` 2.5 s after the page loads, and over a websocket the text-prefix handler was only installed after login, so the request was taken as the username ("try again", "Invalid login"). | `main.go` (`HandleWebSocketConnection`) | The prefix handler now sits before the login prompts, as it does for telnet, and is no longer added a second time after login. |
| `why` named rounds by the server's counter ("Round 1314418"). | `internal/combatstream/rolllog.go`, `internal/hooks/combat_stream.go` | Rolls carry a `Turn`: the fight's own round number, the same "Round N" the text opens each round with since 82d (set from the fight's first round; the log counts for itself when a roll arrives unset). Test `TestRollLogNumbersRoundsWithinTheFight`; `help battlelog` says so. |
| A fallen (raisable) companion left a corpse that "crumbles to dust" mid-fight. | `internal/mobcommands/suicide.go` | A companion leaves no corpse on any of the three death paths (claimed loot, plain corpse, perma-gear); the flag is read before the charm is removed. The resurrect wiring test asserts no corpse. |
| Web Room panel tagged companions "charmed" (the terminal already hides it). | `modules/gmcp/gmcp.Room.go`, `quickmenu.js` | They are tagged `companion`; the quick menu treats it as not a foe. Test `TestRoomPanelTagsCompanionsNotCharmed`. |
| Blessings: "a later character will recruits cost 5% less". | `internal/blessings` `PerkText` | "pay 5% less for recruits". |
| Mob pronouns: "The first skeleton is back on their feet" / "closes their guard". | `internal/status`, `modules/walking` | "is up again" / "'s guard closes"; the walking recovery line reads "is up again, though still exhausted" (the review fixed it: the build had left "finds their feet"). |
| Tab completion offered help topics that are not commands (`events`, `stances`, `relics`, `awakenings`, `enchanting`, `battlelog`, `hardcore`). | `internal/usercommands` `GetCmdSuggestions` | Only registered commands are completed. |
| `company` with no verb printed one 600-character line. | `modules/company` `companyUsage` | Grouped on six lines (hiring, the band, gear, needs, growth). |

| Web Bonds tab with two companions said the same thing three times (the pair's card, then a line per member). | `window-company.js` | The per-member lines show from two pairs up, where they summarise. |
| Help pages that no longer matched the code (an audit of the phase 60-77 pages). | `templates/help/` | `rites` (a garbled sentence on `[member]`); `bonds` (a far-apart pair that suits still gains the battle +1, per `bonds.BattleGain`); `battlelog` (a solid blow is never printed); `relics` (grim, boastful and wry *approve* of a relic sold; only devout minds); `lifestory` (the "+3 so" sentence); `bestiary` (a fragment, and an unknown name is answered); `errands` and `bounties` ("3 gold times the band's middle level", not "for each level in the middle"); `chronicle` (the kind is `groups`, and mercy links `help mercy`); the `company` hub now links `help events` and names awakenings, errands, bounties and rites in the chronicle paragraph; the `combat` hub links `help bestiary`. |

## Checked and left

- Admin `teleport` while a journey is paused leaves travel "Stopped" and
  blocks the exits until `travel resume` or `travel return`. Admin-only, and
  ending the journey on an out-of-route room change would touch the
  expedition session's state machine for no player benefit. Left.
- The `who` table and other wide stock tables wrap in the web client's
  middle column at 1280 px with both docks. Stock GoMud output; the world and
  its tables are temporary.
- `help` topics that are not commands (`events`, `stances`, `relics`...) stay
  as topics; only tab completion stopped offering them as commands.
- Over-long lines in `errands`, `bonds`, `battlelog` and `inn` render fine
  through the `.template` renderer; not reflowed.

## Tests

- `TestRollLogNumbersRoundsWithinTheFight` (combatstream), `TestRoomPanelTagsCompanionsNotCharmed`
  (gmcp), the no-corpse assertion in `TestCompanionDeathAndResurrectionThroughPluginsLoad`
  (company), help tests updated for the new wording (`TestBountiesHelp`, `TestCreationHelp`).
- Live smoke: a new step logs in over the `/ws` socket after sending
  `!!GMCP(Char.Creation)` at the username prompt, as the web client does
  (`dialWeb` in `live_smoke_test.go`; the smoke server now opens an HTTP port).

## Decisions

- **Fix the server, not the client, for the login request.** Other windows
  ask early too (Event, Tutorial); any `!!GMCP(` before login must be
  swallowed, as telnet already does.
- **Round numbers count from the first round the log saw.** The log is
  cleared when a fight begins, so that is the fight's first round; two rolls
  in one round share a number.
- **No corpse for a companion.** Nothing looted it (their gear is kept by
  the company registry), and the decay line read as a second death.

## Review (2026-10-08)

Two small fixes (the walking line above, and the Bonds tab's comment said
three pairs where the code says two). Help claims, the round numbering and
the handler order were verified against the code; the web login, grouped
`company` usage, the Room panel's `companion` tag and the quick menu were
checked on the real page against a live server at 1280 px and 360 px.
