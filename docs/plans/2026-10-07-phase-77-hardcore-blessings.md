# Phase 77: Hardcore (Iron) and account blessings

Spec: [Pillars phases](2026-10-07-pillars-phases.md) §77. Built under the owner's full-autonomy instruction (2026-10-06). One owner steer shaped it: **"For now, remove the permadeath idea"** (owner, 2026-10-07 11:44), so Hardcore does not delete the character. The plan's "defeat routes to permanent death" became "defeat costs more", described below. Every other fork was decided here, each with its reason.

## Decisions

1. **Hardcore is called Iron in the game; the option is a harder defeat, not permanent death.** *Why:* the owner removed permadeath for now. The seam to add it later is small: `Suicide` already knows the character and the death module owns the route (see Follow-ups).
2. **Chosen once, at creation, and final.** The last question of `start` (after the looks and life story, before the tutorial question) is "Take the Iron option?" (`standard` / `Iron`), then a confirmation for Iron. Existing characters are never offered it, and an Iron character cannot put it down. *Why:* a mode people can switch at will is not a mode, and existing characters would be forced into an opt-in they never saw. A player who wants the other way makes a new character (`delete character`).
3. **The cost: two levels instead of one, and never a rescue.** An Iron defeat always takes the church route (the defeat scenarios' `ClaimDefeat` declines for Iron) and `Character.LoseLevels(2)` runs once, with `PeakLevel` kept so re-levelling grants nothing twice. At level 2 it stops at 1. *Why:* the scenarios are the soft landing (no level lost); the church path is already the "harsher fallback". Two levels is a clear, readable price that does not touch any fight's numbers. **Foes are never stronger** (the difficulty rule): Iron changes only what a defeat costs.
4. **Companions are unchanged.** A fallen companion keeps today's resurrect window. *Why:* the leader's defeat is the thing the option prices; changing companion death is a larger design (loyalty, errands, rites all key on it).
5. **The reward is the badge plus Iron blessings.** The badge shows on `online` (`(Iron)` after the name, which is what `who` lists), the `status` sheet (`Mode` row), `Char.Info` (`hardcore`) and the web Character window. Two blessings only an Iron character can earn (`iron-tested`, `iron-oath`). *Why:* the Pillars Trial of Iron is a badge people chase; the extra blessings are small perks, never combat power.
6. **Blessings are data, earned from the chronicle's lifetime counts.** `_datafiles/world/default/blessings.yaml`: a deed kind, a count, `iron` and one perk (a starting item, 1-3 of an existing camp supply, or a recruit discount of up to 10%). Nine ship: boss 1/5, joined 5/15, spared 3, relic 1, story 5, and the two Iron ones. *Why:* the chronicle's `Total` (a lifetime tally that never forgets) is exactly this question; nothing new is recorded, so this phase edits none of the chronicle code (phase 76 is changing it). The names and lines are placeholders for the replacement world; ids, deeds and perk shapes are the contract.
7. **Account-level, kept through `delete character`, dropped with the account.** `modules/blessings` saves, per user id, the blessings earned (id, time, the character whose deeds earned it). `UserPurged{KeepAccount: true}` (a deleted character) keeps them; only a purge that removes the account drops them. *Why:* an account perk that dies with the character it was earned by would be pointless. It is deliberately *not* a `userstate` contributor (the test area's snapshot would otherwise restore it); `purge_coverage_test` lists it with that reason.
8. **Checked on every deed and at login.** `chronicle.OnRecord` and `PlayerSpawn` run the check; a character that already has deeds from before this phase catches up at its next login. The player is told the moment a blessing is earned, and that it goes to their next character. *Why:* the seam existed (phase 67 added it) and costs nothing.
9. **Given once, at the end of creation, never to the character that earned it.** `usercommands.Start` applies the account's earned blessings to the new character (`blessings.Apply`): starting items into the pack, and the ids recorded on `Character.Blessings`. The recruit discount is read from the character's own list, not from the account. *Why:* "later characters" is the spec; reading from the character makes it impossible to grant twice (`Start` re-runs on every later prompt answer) and keeps a blessing earned mid-life for the next one. Characters made before this phase carry none, but what they earn reaches their successors.
10. **Perks are small and sell-proof.** Starting items are camp supplies the markets never buy back (fortifying broth 30040, warming draught 30041, cooling salve 30042, sigil chalk 30060). The recruit discount is capped at 10% in total (`MaxDiscount`, validated per blessing and enforced in the sum), a price never rounds below 1 gold, and a free (tutorial) recruit stays free. The saved roster keeps the full price: the discount is applied where the price is shown (listing, notice, `look`, `inspect`) and charged. *Why:* economy rule (nothing gathered or bought may resell for profit), and "a full company-wide edge is worth about one level at most" (a few gold off a recruit and a flask of broth are well inside that).
11. **UI.** The creation panel needed no code: the Iron question is an ordinary `Char.Creation` choice with option text. The Character window's Overview gets an Iron badge and a Blessings section (carried, waiting for the next character, still to earn with progress) from a new `Char.Blessings` message, set with `textContent` and kept in a local copy because a full `Char` snapshot replaces the namespace's children. It reuses the Overview's single column, so it fits a 360px phone.

## Numbers

| What | Value |
|---|---|
| Levels lost in an Iron defeat | 2 (standard 1, scenario 0), never below level 1 |
| Recruit discount, per blessing | 5% |
| Recruit discount, total cap | 10% |
| Starting items, per blessing | 1-3 camp supplies |
| Blessings shipped | 9 (2 Iron-only) |

## Help and tutorial (acceptance)

- New: `help hardcore` (aliases `iron`, `iron option`, `hardcore mode`, `trial of iron`), `help blessings` (aliases `blessing`, `account blessings`, `starting perks`). Both indexed under the road in `keywords.yaml` and listed in `help adventure`.
- Updated for the change: `help death`, `help defeat`, `help combat` (the cost of falling), `help delete` (blessings stay; Iron is asked again), `help lifestory` (the creation question), `help company` (the recruit discount), `help webclient` (the Iron badge and blessings).
- Tutorial: a Departure-lesson hint (`modules/tutorial/stages.go`); `TestTutorialHelpPointersExist` passes.
- `TestHardcoreAndBlessingsHelp` renders both through `help`, checks the aliases, the numbers against the code and the hub links.

## Tests (the integration points)

- `internal/usercommands/iron_test.go`: the real `start` flow: the question follows creation, Iron asks to confirm, declining at the confirmation is standard, the choice survives the `start` re-run, blessings are given once (not twice across the re-run), an unknown id is skipped, nothing given without any; the web panel's view is published and cleared.
- `modules/death/wiring_iron_test.go`: the real `suicide` command: a rolled rescue is not claimed for Iron, two levels are lost with the peak kept, the church is the destination, a standard defeat still gets the scenario, level 2 stops at 1.
- `modules/blessings/blessings_test.go`: deeds recorded through `chronicle.Record` earn blessings once and tell the player; Iron-only blessings; kept through a keep-account purge, dropped by an account purge, reloaded from disk; a failed save keeps the grant; an unreadable file is never overwritten; login catches up; the panel and command text; the module is the provider the creation step reads.
- `modules/company/recruit_blessings_test.go`: the discount shows in the listing, notice, `inspect` and the hire's charge for authored and generated candidates, decides affordability, and leaves the table's price alone.
- `internal/blessings`: file validation (bad ids, unknown deeds, over-cap discounts, too many items), the shipped file and its items, the cap, rounding.
- `status` Mode row, `online` badge, `Char.Info` flag. Existing creation tests answer the new question (`creationUser.standard`, or `Character.MarkIronOffered` where the test is about something else).

## Review (2026-10-07)

Checked against the owner's rules; each finding verified.

- **No permanent death** (accepted as built): Iron touches only `ClaimDefeat` and the levels `Respawn` takes; `ExtraLives` and the delete path are unchanged.
- **Foes never stronger** (accepted): nothing in combat, spawning or zone rating reads `IsIron`.
- **Counts can't be faked or carried over** (accepted): the chronicle drops a user's log on every `UserPurged`, including a deleted character's, so a new character starts at zero; a blessing is earned once per account, so repeating a deed gains nothing. Cheap-looking routes (recruit and dismiss for `joined`) cost real gold and only ever earn the 5% discount once.
- **Delete-and-recreate farming** (rejected as not worth code): each new character gets the starting items again, and a player could drop them before deleting. That is at most 10-13 camp supplies (value 6-12 gold each, never bought back) per full re-creation and tutorial, well inside the one-level cap. Binding the items would add a new item flag for no real gain.
- **Recruit discount** (accepted): applied at every shown and charged price, capped at 10%, nothing refunds a recruit, so no profit path.
- **UI fix:** the Blessings section sat between the stats and Experience/Gold, pushing gold out of view on a phone once several blessings are listed. It now sits last on the Overview (`order: 1`). Browser check added to `scripts/browser/dock-windows-check.mjs` (badge, sections, HTML escaping, survives a full `Char` snapshot, fits 360px).
- **UI fix:** the panel is pushed at spawn, before `start` gives the blessings, so a new character's window still listed them as waiting. `blessings.NotifyGiven` now refreshes it (`TestGivingBlessingsRefreshesThePanel`).

## Follow-ups

- **Permanent death** (owner removed it for now): a character-ending route would call the same `delete character` hand-off from `Suicide` for an Iron character, after recording a Hall of the Fallen entry on the account. The Iron option and badge already exist; only the route and a confirmation at creation would change.
- Iron could also change companion death (no resurrection) once its interaction with errands, rites and bonds is designed.
- The test area (admin `testarea`) restores the chronicle but not blessings; a trip's deeds could earn a real blessing for the admin's account. Acceptable for an admin tool; revisit if the test area gains non-admin use.
- Blessing names and flavour lines are placeholders for the replacement world.
