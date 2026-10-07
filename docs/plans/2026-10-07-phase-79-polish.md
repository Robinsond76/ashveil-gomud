# Phase 79: wider polish pass (decisions, 2026-10-07)

A hunt for rough edges across commands, help text and the web client,
starting past the phase 78 list. Found by playing scripted telnet sessions
against a private server (recruits at the Waymark Inn, test-area fights,
every phase 60-77 command as a fresh solo character and with a company),
live web-client screenshots at 1280 px and 390 px, and an audit of the
phase 60-77 help pages against the code. No balance or economy changes.

**PAUSED 2026-10-07 14:03 at Robinson's request** ("pause all other tasks
and let just the combat overhaul proceed"). Work in progress is committed
on `claude/phase-79-fable-polish-ufqqzc`; no PR yet. The sections below
say what is done and what is still open.

## Fixed so far (code on the branch)

| Finding | Where | Fix |
| --- | --- | --- |
| Web login eats a GMCP request: the creation window sends `!!GMCP(Char.Creation)` 2.5 s after the page loads, and over a websocket the text-prefix handler was only installed after login, so the request was taken as the username ("try again", "Invalid login"). | `main.go` (`HandleWebSocketConnection`) | The prefix handler now sits before the login prompts, as it does for telnet, and is no longer added a second time after login. |
| `why` named rounds by the server's counter ("Round 1314418"). | `internal/combatstream/rolllog.go` | Rolls get a `Turn` within the fight as they are logged; `Describe` prints it. Test `TestRollLogNumbersRoundsWithinTheFight`. |
| A fallen (raisable) companion left a corpse that "crumbles to dust" mid-fight. | `internal/mobcommands/suicide.go` | A companion leaves no corpse; the company keeps them until raised or lost. (Test still to add: the resurrect wiring test should assert no corpse.) |
| Web Room panel tagged companions "charmed" (the terminal already hides it). | `modules/gmcp/gmcp.Room.go`, `quickmenu.js` | They are tagged `companion`; the quick menu treats it as not a foe. Test `TestRoomPanelTagsCompanionsNotCharmed`. |
| Blessings: "a later character will recruits cost 5% less". | `internal/blessings` `PerkText` | "pay 5% less for recruits". |
| Mob pronouns: "The first skeleton is back on their feet" / "closes their guard". | `internal/status`, `modules/walking` | "is up again" / "'s guard closes"; the walking line avoids the pronoun too. |
| Tab completion offered help topics that are not commands (`events`, `stances`, `relics`, `awakenings`, `enchanting`, `battlelog`, `hardcore`). | `internal/usercommands` `GetCmdSuggestions` | Only registered commands are completed. |
| `company` with no verb printed one 600-character line. | `modules/company` `companyUsage` | Grouped on six lines (hiring, the band, gear, needs, growth). |

## Still open (not started when paused)

Help pages, from the audit (each verified against code):

- `rites`: the garbled "[member] is a name, or the number shown by rites as #4; all names everyone waiting..." sentence.
- `bonds`: a far-apart pair that suits still gains the battle +1 (`bonds.BattleGain`); only clashing pairs and ordinary far-apart pairs get nothing.
- `battlelog`: "solid" is never printed (`strike.go`); say glancing or telling, solid goes unsaid.
- `relics`: grim, boastful and wry *like* a relic sold; only devout dislikes it.
- `lifestory`: the "+3 so" sentence; three answers give +3 in all, no stat more than +2.
- `bestiary`: a sentence fragment ("...when it falls; whoever lands the last blow"), and an unknown name *is* answered ("Nothing called ... is in your bestiary").
- `errands` and `bounties`: "3 gold for each level in the middle of the band" means 3 gold times the band's middle level.
- `chronicle`: list the kind as `groups` (the label is "Groups"; `bands` is an alias) and link `help mercy` for mercy and executions.
- `company` hub: link `help events`, and extend the chronicle paragraph with awakenings, errands, bounties and rites. `combat` hub: link `help bestiary`.
- Cosmetic: over-long lines in `errands`, `bonds`, `battlelog`, `inn`; `bestiary`'s See also styling.

Other findings:

- Admin `teleport` while a journey is paused leaves travel "Stopped" and blocks the exits until `travel resume` or `travel return`. Admin-only; left as is unless cheap (the expedition module would need to end the session on an out-of-route room change).
- Web Bonds tab repeats each symmetric pair three times (the pair line, then one line per member). Consider dropping the per-member lines when both read the same.
- A regression test for the web login fix: a websocket login step in the live smoke (`/ws`), sending `!!GMCP(Char.Creation)` before the username.
- Help for `why` (`battlelog`) should say rounds are numbered within the fight.
- Project status entry and roadmap row for phase 79 (not yet written).

## Decisions

- **Fix the server, not the client, for the login request.** Other windows
  ask early too (Event, Tutorial); any `!!GMCP(` before login must be
  swallowed, as telnet already does.
- **Round numbers count from the first round the log saw.** The log is
  cleared when a fight begins, so that is the fight's first round; two rolls
  in one round share a number.
- **No corpse for a companion.** Nothing looted it (their gear is kept by
  the company registry), and the decay line read as a second death.
