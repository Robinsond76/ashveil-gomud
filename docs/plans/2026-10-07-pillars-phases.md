# Lessons from Pillars of Eternity: phases 60–77 (2026-10-07)

The owner asked what Ashveil could learn from Pillars of Eternity and its
sequel Deadfire (research thread, 2026-10-07), liked every idea, and asked
for them as phases. This plan scopes those phases. Each one still gets its
own execution plan in `docs/plans/` from its build thread, and the usual
review gate. The research write-up is in the project's shared files
(`research/2026-10-07-pillars-of-eternity-lessons.md`).

## Rules every phase keeps

- Battles play out on their own. The only mid-battle inputs stay retreat
  and company focus. Everything here acts before a battle (orders, stances,
  knowledge), after it (log, chronicle, rites) or outside it (events,
  errands, towns).
- Difficulty comes only from a zone's level. Bonuses here are small and
  earned by the company's own choices; balance cells are measured without
  them so the 37b win rates still hold.
- Nothing gathered or bought resells for profit. Relics and rare drops may
  pay (owner, 2026-10-06), so bounties and errands can, within the zone band.
- Never advance global game time. Errands and cooldowns run on saved real
  time, as sigils do.
- The world will be replaced. Each phase is a system with data hooks (YAML
  tables, tags on mobs, zones and NPCs), with only enough test content to
  prove it. Town lines and events written now are placeholders.
- Status in words: new knowledge, bonds, opinions and creeds are shown as
  short phrases, not numbers, wherever the player reads them.
- Ship indexed help and tutorial pointers, per `AGENTS.md`.

## Phases at a glance

| Phase | Scope | Size | Depends on |
|---|---|---|---|
| 60 | Story events: illustrated text pages with choices a company member's skill decides | L | — |
| 61 | Battle orders: up to three "when ... do ..." rules per member | M–L | — |
| 62 | Battle lines that explain themselves | M | — |
| 63 | Company chronicle: a durable log of deeds, read in prose | M | — |
| 64 | Companion opinions of the leader's choices | M | 63 |
| 65 | Bonds between companions | M–L | 64 |
| 66 | Bestiary earned by fighting | M | — |
| 67 | Relics that awaken through deeds | M | 63 |
| 68 | Towns that remember the company | M | 63 |
| 69 | Weapon stances set before battle | M | — |
| 70 | Errands for companions sitting out | M | 63 |
| 71 | Enchanting with hunted trophies | M | 66 |
| 72 | Backgrounds that unlock choices | S–M | 60 |
| 73 | Faith creeds | M | 63, 64 |
| 74 | Rites for the dead | S | 65 |
| 75 | Inn room tiers | S | — |
| 76 | Bounty boards | M | 63 |
| 77 | Hardcore (one life) and account blessings | M | 63 |

**Build order.** First wave (no dependencies, most value): 60, 61, 62 and
63. Then 64, 66, 67, 68 and 69; then 65, 70, 71, 75 and 76; last 72, 73, 74
and 77. At most two building at once, per the concurrency rule.

**Not a phase:** the deep, many-floored dungeon with a story told floor by
floor (Pillars' Endless Paths) is content for the replacement world; it is
recorded here for the world design and uses 60, 63 and 66 when built.

**Style bible:** the townsfolk vignettes, narrator voice and "harsh, not
grim for its own sake" lessons go to the style bible thread, not a phase.

## 60 Story events

**Why.** Pillars stops at a cliff, a stranger or a ruined shrine and shows
a page: a picture, a paragraph and choices, some gated by a skill and
resolved by the party member who has it ("Eder climbs down"). It makes the
whole company matter outside battle and gives the harsh world consequences
without more fighting. It is text, which a MUD already is.

**Scope.** A data-driven event engine: YAML pages with text, an optional
picture, and choices. A choice may require a member's skill, class,
personality, alignment, level or a carried item; the engine picks the best
qualified member present and names them. Outcomes: wound, ailment, need
change, supply or item lost or gained, gold, loyalty, a battle with a named
group, a room or exit opened, a follow-on page. Triggers: entering a tagged
room, a travel leg, a camp rest, a lair door. Once per company per event
(or a saved real-time cooldown). Web client shows a modal page; telnet shows
numbered choices (`choose 2`). Pending pages survive restart.

**Accept.** Three test events (a climb, a stranger, a shrine) reachable in
the test world; each outcome kind tested through the real trigger;
`help events`.

## 61 Battle orders

**Why.** Deadfire's rule editor ("if an ally is below half health, heal
them") is the Ogre Battle model done well: the plan is the play.

**Scope.** Up to three rules per member, each from fixed menus: a
condition (ally below a health share, foe chanting, I am below a health
share, boss present, first round, foe of a kind) and an action (heal that
ally first, break the chant, fall back a row, use my strongest ability,
guard that ally, hold my mana). Rules are checked in order before the
member's 32d role and target rule. Class presets. The battle log credits a
fired rule ("as ordered"). Stored with strategies (`modules/strategy`),
refused in battle.

**Accept.** Each condition and action fires in a real round; presets
loaded; `help orders`; Company panel editor at desktop and phone width.

### Phase 61 build decisions (2026-10-07, full autonomy)

Where the scope left the menus open, the build picked these defaults.

- **Conditions (six):** `ally [n]` (the most hurt ally who is not the member, below n% health; n a multiple of 5 from 10 to 90), `self [n]`, `chanting`, `boss`, `first` (the battle's first round), `foe [kind]` with kinds `caster` and `healer`. *Why:* each reads a fact the combat round already has (`strategy.Foe` flags, battle round, ally health), so no new state; kinds follow the existing foe flags, and race kinds (beast, undead) wait for the replacement world's races.
- **Actions (five):** `heal` (the ally or self the condition named; the heavy heal below 40%), `break` (turn on the foe the condition named; for a chanter, break its chant: a fighter strikes if it reaches it, a caster or one out of reach casts an attack spell), `guard` (the member guards the named ally this round, through the real guardian path and its guard counts), `strongest` (cast the heaviest attack spell it can pay for, ignoring its mana reserve), `hold` (no attack spell this round; heals go on). *Why:* these are the Deadfire rules that change the outcome and that the engine can already carry out. Legal pairs are one rule in `orders.Order.Validate` (heal needs ally/self, guard needs ally, break needs chanting/boss/foe).
- **Left out: "fall back a row" and "use my strongest ability" for fighters.** Formation is durable state: moving a member mid-battle would either persist or need a battle-only column in every reach gate. Class abilities fire through `abilityPass` with their own cooldowns and gates; forcing one is a larger change. `strongest` covers casters; a fighter's ability still fires by its own rule. Both can follow as a later phase if players want them.
- **Reading:** each round, in `strategyPass`, before the role and target rule. The first order whose condition holds and whose action the member can carry out (a heal needs a known, paid-for heal spell; a break needs a foe in reach or a spell) fires and takes the turn; `guard` and `hold` modify the turn instead (the member then goes on by its role). An order that keeps holding (guard, hold) is noted when it starts, not every round. A member chanting, stunned, surprised or stood down reads no orders, as it casts nothing.
- **Log:** a text line to the player and the room ("Brother Oswin tends you, as ordered.") and an `order-fired` combat event (member, subject, action) that the web battle log narrates in the same words. Not shared with allied companies' watchers.
- **Storage and rules:** up to 3 per member, stored with strategies in `modules/strategy` (own map, rolled back on a failed save, pruned with the member, dropped on purge, own test-area snapshot entry). Set with `orders [who] add|remove|up|preset|clear`; refused in a battle; reading is always allowed. `strategy [who] default` leaves orders alone. Presets by archetype are only loaded on request (`orders [who] preset`), never automatic, so existing companies fight exactly as before and no balance changed.
- **Web client:** the Combat tab's Setup lists each member's orders under their row, with an Orders button (add from 14 offered pairs, remove, move up, class preset, clear); in a battle the list shows without the button.

## 62 Battle lines that explain themselves

**Why.** When a player can't steer a fight, understanding why it went wrong
is how they prepare the next one. Pillars lets you hover any log line for
the full roll.

**Scope.** The combat event stream carries each roll's parts (chance to
land and what made it, the defence it met, quality, damage before and after
armour, named modifiers). The Combat tab expands a line into a short plain
breakdown; an after-battle summary in words (who took the most, who never
reached their target, which rule or sigil mattered) for every client.

**Accept.** Breakdown matches the engine's own numbers in tests; summary on
telnet and web; `help battlelog`.

## 63 Company chronicle

**Why.** Pillars ends with slides of what your choices did. A MUD can keep
that running. It is also the shared record that 64, 67, 68, 70, 73, 76 and
77 read.

**Scope.** A durable, capped log of notable deeds per company: joins,
deaths, departures, bosses and lairs, mercy and executions, relics found,
promotions, defeat scenarios. Each entry has a kind, the members involved,
place and real time. `chronicle` reads it in prose; a Chronicle view in the
Company panel. A small query seam for other modules.

**Accept.** Each deed kind recorded from its real source; survives restart
and copyover; purged with the user; `help chronicle`.

### Phase 63 build decisions (2026-10-07, full autonomy)

Where the scope left the menus open, the build picked these defaults.

- **Shape: a seam package plus a module.** `internal/chronicle` is GoMud-free: entry shapes, kinds, prose, `Filter`, the capped `Log`, and package functions `Record`, `Query`, `Count`, `Has`, `Total`. `modules/chronicle` installs the provider, saves the logs, and owns the command and the web message. Other code records and reads through the package and never imports the module (the pattern `internal/strategy` and `internal/storyevents` use). `Record` does nothing without a provider, so no caller checks. *Why:* 64, 67, 68, 70, 73, 76 and 77 are queries over the log, and a seam keeps them out of the module's state.
- **An entry** is `{seq, at, kind, members[], keys[], subject, detail, place, ref}`: names for reading, `keys` (the members' company keys, `leader` or `companion:<id>`, matched with `Filter.Key`; review addition) for telling apart members who share a name, and `ref` (`mob:<id>`, `item:<id>`, `event:<id>`, `class:<id>`, `scenario:<id>`) for matching, so a renamed boss still matches. Time is real time; the world clock is never touched. A company is its leader's user id.
- **Thirteen kinds,** each written from the real source of the deed, not from a view: `joined` (`enlist`, so `company summon`, `recruit` and generated hires), `dismissed` (one deed for `dismiss all`, naming everyone), `deserted` (loyalty or morale ran out; `removeCompanion` has only those callers), `fell` (a companion's `MobDeath` in the company module; the leader's death in `usercommands/suicide.go`, with the killer's name, before any defeat it leads to; not in the Training zone), `raised` (`ResurrectCompanion`), `lost` (the allowance ran out), `defeated` (the scenario applied when the company wakes, not when it is claimed; placed where it fell), `boss` (`Suicide`, per contributing company, an executed boss included, not in the Training zone), `relic` (the company's relic roll), `spared` and `executed` (the mercy answer, once), `promoted` (`class promote ... confirm`), `story` (a scene's last answer, with the choice taken and who took it; a page that only leads on is not a deed). Lairs are boss kills: there is no separate lair record. Not recorded: level-ups, camps, crafting and gathering (noise; they carry no choice). New kinds are one entry in `Kinds` and `Prose`.
- **Capped at 300 deeds, with a lifetime tally.** The log drops the oldest; `Tally` counts every deed per kind forever. `Has` and `Count` see the kept deeds; `Total` is lifetime. *Why:* durable and bounded, while Hardcore and creeds can still ask "how many ever".
- **Persistence:** plugin file `chronicle`, saved at once on each deed (a failed save keeps the deed in memory for the next save; a deed that happened is never refused), and on the plugin save. Not saved while a load error is outstanding, so an unreadable file is never overwritten. Purged with the user; a `userstate` contributor for the test area.
- **Reading:** `chronicle [kind] [page]` and `chronicle all`, twelve a page, newest first, relative real time ("3 days ago"), and an "All told" tally line. Allowed in a fight (read-only). The web client's Company window has a Chronicle tab with a filter by kind, fed by the `Company.Chronicle` GMCP message (newest 60 deeds, the tally, labelled kinds), pushed on every deed and at login and copyover. It is plain text set with `textContent`, one column, so it fits a 360px phone.
- **No numbers move.** The chronicle only records and reads: no gold, XP or balance.
- **Help and tutorial:** `help chronicle` (aliases `chronicles`, `deeds`, `company chronicle`), listed under the road, linked from `help adventure`, `help company`, `help events` and `help webclient`; a hint in the Departure lesson.

## 64 Companion opinions

**Why.** Deadfire companions approve or disapprove of the player's choices,
out loud, and the relationship moves.

**Scope.** Each personality likes and dislikes a few kinds of choice
(mercy, execution, sharing food, inn or rough camp, selling relics, story
event choices from 60). A reaction is one short line and a loyalty nudge,
at most once per choice per companion. Opinions shown in words in
`company inspect`. Builds on 21a loyalty and desertion; no new meter
beyond what loyalty already holds unless the plan shows one is needed.

**Accept.** Every personality reacts to at least two kinds; reactions
change loyalty through the real choice; `help opinions`.

### Phase 64 build decisions (2026-10-07, full autonomy)

Where the scope left the menus open, the build picked these defaults.

- **Shape: a pure package, a seam in `internal/company`, and the company module.** `internal/opinions` is GoMud-free: the kinds, each personality's likes and dislikes, the authored lines, the nudge and its bounds, and an observer seam. Sources report a `opinions.Choice` through `company.Opinion(leader, choice)`, which the company module implements (`modules/company/opinions.go`); no source imports the module. *Why:* mercy, camps, sales and scenes live in four packages, and a seam keeps them out of the company module (the pattern `internal/chronicle` set).
- **Twelve kinds.** Six real-source choices: `spare` and `execute` (the mercy answer, in `hooks.resolveMercy`), `inn` (an inn stay that started, in `modules/camping/inn.go`), `rough` (a camp rest that started), `share` (a meal in which the leader's own pack, not the company cargo, fed a companion; `modules/company/provision.go`), and `sell-relic` (`usercommands.Sell` of a relic). Six story stances an event author tags on a choice (`stance:` on a Phase 60 choice, validated at load): `kindness`, `greed`, `courage`, `prudence`, `reverence`, `cunning`. The test scenes are tagged. *Why:* each is a choice the player really makes; stances let any scene (including the replacement world's) carry opinions without code. Not wired: salvaging a relic (the smith refuses relics, so it never happens), buying gear, or level choices.
- **Who has views.** The six banter personalities, each liking four or five and disliking three or four of the twelve kinds (every kind has a liker). The table is in `internal/opinions/opinions.go`; the page `help opinions` summarises two personalities. Alignment is not read for the verdict: it already moves loyalty on mercy (Phase 27), so a companion that alignment moved on a mercy choice stays quiet (one reaction per choice per companion), and temperament answers the rest. *Why:* no parallel mood system, and no double count.
- **The reaction.** One line in the companion's own voice with banter's verb (`Maren murmurs, "Mercy is a prayer said with the hands." (loyalty +2)`), shown with the choice's own text. A companion who is dead, away, a creature or a construct does not weigh in. Witnesses are who the source names (mercy: the surrendering fight's witnesses; scenes: members present), else everyone walking with the leader.
- **Loyalty moves ±2 through the real record, bounded.** Approval never lifts loyalty above 80, disapproval never lowers it below 30 (above battle nerve's 25, so opinions alone never make a companion hesitate or flee, and never desert one). *Why:* "reactions change loyalty" without a loyalty farm or a surprise exit; the old alignment and morale rules remain the only way out.
- **No farming: a cooldown per companion per kind.** Real time, kept on the companion: 10 minutes for mercy, executions and scene stances, 30 minutes for a relic sale, 2 hours for inns, rough camps and shared food. Inside the cooldown the companion is silent and nothing moves. A choice is reported once (`Op`, remembered with the record's applied operations), so a retried mercy answer changes nothing. Nothing here gives gold, experience or power, and relics' sale price is untouched.
- **Memory and display, no new meter.** A companion keeps its last six reactions and when it last spoke per kind (`Companion.Opinions`). `opinions [member]` and `company opinions` read each companion's temperament, mood word (devoted, loyal, steady, wavering, close to leaving), likes and dislikes, its recent reactions, and the newest chronicle deeds that name it (matched by the company key, not the name); the company's lifetime "spared N, executed M" is the chronicle's tally. `company inspect <member>` ends with the same view in one line.
- **Web client: an Opinions tab** in the Company window, fed by `Company.Opinions` (an extra the company feed keeps current and sends only on change, so loyalty drifting moves it too). One column of cards, plain text set with `textContent`; checked at desktop and 360px width in `scripts/browser/dock-windows-check.mjs`.
- **A seam for Phase 65 (bonds).** `opinions.Observe` registers a function that hears every batch of reactions (leader, choice, who liked and who disliked it). Bonds can raise a pair that agreed and lower one that clashed without this phase knowing them.
- **Help and tutorial.** `help opinions` (under the road, aliases `opinion`, `company opinions`, `companion opinions`, `loyalty opinions`), linked from `adventure`, `company`, `events`, `webclient`, `mercy`, `banter`, `camp`, `inn`, `relics` and `company-meal`; a hint in the Departure lesson after the chronicle's.

- **Review (2026-10-07).** Observers get the resolved witnesses in `Choice.Witnesses` (silent ones included); `sell junk` counts as a relic sale, once per command; deserting companions have no say. Findings and rejections are in `docs/PROJECT_STATUS.md`.

## 65 Bonds between companions

**Why.** Deadfire companions become friends or rivals with each other. In
Ashveil, that makes who stands next to whom a human choice as well as a
tactical one.

**Scope.** A saved value per pair, raised by shared camps, battles side by
side and one saving the other, lowered by clashing personalities or
alignments and by banter clashes. Shown in words ("trusts Ysolde", "can't
stand Merek"). Banter picks lines from the bond. Small battle effects:
friends in adjacent cells guard each other more readily; a rival won't
guard the other. A rivalry pushed to its end warns, then one leaves.

**Accept.** Bond changes from each real source; adjacency effect measured
in a balance cell (small, no win-rate shift beyond noise); `help bonds`.

### Phase 65 build decisions (2026-10-07, full autonomy)

Where the scope left the menus open, the build picked these defaults.

- **Shape: a pure package, a saved pair, and the company module.** `internal/bonds` is GoMud-free (range, words, sources and cooldowns, limits, camp affinity, battle numbers, the warning rule). The saved value is `Record.Bonds` (`internal/company/bonds.go`): one entry per pair of companions by id (`A < B`), pruned by `Registry.Put` when either leaves. `modules/company/bonds.go` is the single writer (`applyBonds`); combat reads through `company.BondValue` and reports through `company.BondEvent`, as chemistry and opinions do. *Why:* the same seam pattern as 24, 63 and 64, and combat never imports the module.
- **Companion pairs only; the leader has loyalty.** A bond is between two living, non-creature companions. The leader's side of a companion is its loyalty and alignment (21a, 64), so there is no leader bond and no second meter. A creature, construct, fallen or fled companion has no bond, and bonds go with a companion that leaves. *Why:* "who stands next to whom" is a company choice; loyalty already measures the leader.
- **One number, words for it.** -100..100, shown as "trusts" (25), "is close to" (50), "is like kin to" (75), "is wary of" (-25), "can't stand" (-50) and "cannot bear" (-85), and "still getting to know" between. `bonds` and `company bonds` read them; one line ends `company inspect`. *Why:* the plan asked for words ("trusts Ysolde", "can't stand Merek"); the number is shown beside them (as loyalty is) so a player can tell how close a pair is to a line.
- **Sources, each with a per-pair real-time cooldown.** A camp rest (+2 suited, +1, -1 clash, one less across an alignment gap of 70; half an hour), a won battle (+1 unless the pair clashes; ten minutes), banter (a friend line +1, a rival line -1; half an hour), a shared opinion (Phase 64's seam: both liked or both disliked +1, one each way -1; ten minutes), stepping in (+3; an hour) and a refused guard (-2; an hour). Temperament affinity is a small table (six suited pairs, four clashes); alignment's gap is the rest of "clashing alignments". *Why:* each is a real event, time-gated, so none loops.
- **Time together stops at 50; only real acts go further.** Camp, battle, banter and opinions raise a bond to +50 at most and lower one to -50 at most; stepping in goes on to +100; a refused guard goes to -100, and an opinion clash goes below -50 only for a pair already at "can't stand". *Why:* a company that merely travels together settles into trust or wariness and never into a feud or into kin; the deeper tiers (and the warning) take something the leader and the company did.
- **Battle effects, automatic, small, logged.** A friend (25+) in reach of a friend at 40% health or less steps in once a battle, twice for kin (75+), whatever its role, and the company makes two such steps a battle at most ("steps in front of X for a friend. (bond guard, 1 left)", a `guard-used` event with status `bond`, so the web feed says "for a friend"). A guardian *set* to guard a rival (50 below) refuses, says so once a battle ("lets the blow fall on X. (no guard: they can't stand each other)", a `guard-refused` event), and the refusal sours the bond; a guardian with no ward set simply passes over a rival when it picks the most hurt. The ordinary guard that one companion makes for another counts as a rescue. Nothing else changes: no stat, hit or damage bonus. *Why:* "friends in adjacent cells guard each other more readily" is a step in without needing the guardian role, bounded by half health and the count; "a rival won't guard the other" is the refusal. Both go through `guardianFor`, so the guard's own interception rules, the narrow-ground rule and the grounded/stunned checks apply unchanged.
- **A rivalry pushed to its end warns, then one leaves.** At -85 the leader is told the two can hardly stand to share a company; at -100 *after* the warning, the less loyal one (the newer one, on a tie) goes through the existing `MoraleDesert` desertion, never mid-fight (the same wait-for-the-fight rule as mercy); a pair that recovers to -60 or better loses its warning. A warning always comes first, even for a hand-edited file at -99. *Why:* "warns, then one leaves" without a new departure path; a rivalry reaches -85 only by repeated refusals and clashes, which a leader can see coming in `bonds` and avoid by choosing wards.
- **Banter picks lines from the bond.** Two new banter contexts, `friend` and `rival` (24 lines each, four a personality, each naming the other with `{other}`). When the pair drawn for an exchange are friends or rivals, half the time they talk to each other about it, as a pair; the `Said` line carries its context so `bondsFromTalk` can move the bond. No leader setting changes: `set banter off` silences this too, though camp rests still count whether or not banter is on.
- **Web client: a Bonds tab** in the Company window (`Company.Bonds`): a card per pair with its phrase, number, a centred bar and what it does in battle, then one line for each companion's view; plain text set with `textContent`; checked at desktop and 360px in `scripts/browser/dock-windows-check.mjs`. The battle feed names a bond guard and a refusal.
- **Help and tutorial.** `help bonds` (under the road, aliases `bond`, `company bonds`, `companion bonds`, `friendship(s)`, `rivalry`, `rivalries`, `rivals`, `friends`), linked from `adventure`, `company`, `opinions`, `banter`, `camp`, `guardian` and `webclient`; a hint in the Departure lesson after the opinions one.
- **Balance (timeboxed; `TestPhase65BondsInTheMirror`, opt-in, 80 fights a cell, five even foes, about ±8-11 points of noise).** A first cut (a friend steps in at 50% health, once each, no company cap) lifted an all-friends company from 76% to 93% wins at L5, so it was cut to 40% health and two bond steps a battle for the whole company. Settled, wins at L5/L15: no bonds 75/71; all friends 81/82; all kin 80/78; one friend pair 80/78 (modest, near the noise); all rivals 57/51; one rival pair (the front warrior and the healer it guards by default) 55/53. Fair three-foe fights stay at 100% in every cell (the first run; HP lost falls a little for friends and doubles for rivals). Recorded: a rivalry between a guardian and its ward costs about 18-20 points in the hard cell because it removes that guardian's service, which is what "a rival won't guard the other" means; the leader sees it in `bonds` and can set a different ward.
- **Not done, on purpose.** No chronicle deed (the chronicle names deeds, and a bond is a feeling; 70 and 74 can read bonds directly); no leader-set bonds, gifts or "mediate" command (a rivalry mends by changing wards, camping and agreeing); no bond effect outside combat (no formation bonus, no camp bonus).

## 66 Bestiary

**Why.** Pillars' bestiary starts blank and fills in as you kill a kind of
creature. It rewards experience and feeds preparation.

**Scope.** Per leader knowledge per enemy kind (template or family),
earned by battles against it: lore, then defences in words, then
weaknesses and habits (chants early, targets the weak). `consider`, the
Battle view and a `bestiary` command show what is known. Feeds 61 (you
learn a foe chants, you set a rule).

**Accept.** Knowledge tiers unlock through real battles; survives restart;
`help bestiary`.

Built: [phase 66 plan and decisions](2026-10-07-phase-66-bestiary.md).

## 67 Relics that awaken

**Why.** Pillars' soulbound weapons unlock powers as their bearer meets
conditions. Ashveil has 16 relics.

**Scope.** Each relic gets two or three awakenings with conditions (foes of
a kind slain while wielded, a lair cleared, carried while devout, a place
visited) shown on the item in words with progress. Progress is saved on the
item instance. Awakened powers are modest and counted in 36d's relic
balance.

**Accept.** Every relic has awakenings; each condition type advances
through its real source; relic sale value unchanged by awakening (no new
profit path); help pages for relics updated.

## 68 Towns that remember

**Why.** Robinson wants towns that feel alive and dislikes repeated lines.
Deadfire's townsfolk mention what you did.

**Scope.** Town NPCs tagged as talkers read recent chronicle entries (63)
and say a fitting line once per deed per player ("You're the ones who
cleared the Hollowweb nest"), then fall back to a state line (weather,
time of day) or silence. Lines are YAML by deed kind and NPC tag, so the
replacement world brings its own.

**Accept.** A deed produces a line once per player; no line repeats within
its window; `help townsfolk` (or folded into an existing page).

## 69 Weapon stances

**Why.** Deadfire's weapon modals trade one strength for another. Another
lever set before the fight.

**Scope.** One stance per weapon family, set per member out of battle:
great weapons trade accuracy for damage, shields trade attacks for
defence, bows fire faster but lighter, daggers favour crits. Saved with the
member; shown in the battle view.

**Accept.** Each stance measured in a balance cell (a trade, not a free
win); `help stances`.

## 70 Errands

**Why.** Pillars' stronghold sends idle companions on adventures; Deadfire
does it with bounties. Benched companions have nothing to do, and a MUD is
played in sessions.

**Scope.** From a town, send a companion not in the formation on an errand
(escort, hunt, scout) of a chosen real-time length. It returns with modest
gold, an item, a lair rumour, or a wound, scaled by its level and the
zone's band, and a chronicle line. Saved expiry; the companion is away
(not in battles, not in camp) until it returns.

**Accept.** Errand survives restart; outcomes within band; the away
companion is absent everywhere it should be; `help errands`.

## 71 Enchanting with trophies

**Why.** Pillars' enchanting needs creature parts, which ties hunting to
gear.

**Scope.** Mobs drop trophies (heart, hide, ash) by family; an enchanter
NPC upgrades an item for trophies plus gold. Upgraded items are never
worth more at a merchant than their parts and fee cost. Bestiary (66) names
which creature drops what.

**Accept.** Upgrade path tested end to end; resale check; `help enchanting`.

## 72 Backgrounds

**Why.** Pillars' backgrounds open dialogue options.

**Scope.** Choosing the background at creation moved to
[phase 72a](2026-10-07-phase-72a-character-creation.md) (2026-10-07): its
life story's trade stage is the background. Phase 72 keeps the hooks:
story events (60) and town lines (68) can require a background, and
`help backgrounds` lists what each one opens. Depends on 72a, 60, 68.

**Accept.** At least one background choice in each test event; a town
line that needs one; `help backgrounds` lists what each background opens.

## 73 Faith creeds

**Why.** Pillars' priests and paladins draw strength from acting by their
creed.

**Scope.** Each faith route (38b) has a creed of favoured and forbidden
deeds read from the chronicle (63). Keeping it gives a small bonus to the
faithful member; breaking it lifts the bonus for a while, and devout
companions say so (64).

**Accept.** Bonus on and off through real deeds; `help creeds`.

## 74 Rites for the dead

**Why.** Deadfire's crew deaths carry a funeral choice. Loss should cost
something besides gold.

**Scope.** When a companion dies for good or leaves after long service,
the next camp offers rites. Holding them gives a chronicle line and steadies
those close (65); skipping costs their loyalty.

**Accept.** Rites through a real camp; skip penalty; `help rites`.

**Check folded in:** whether a member knocked out in a won fight always
takes a lasting wound (Pillars' injury rule); record the finding.

## 75 Inn room tiers

**Why.** Pillars inns sell rooms with different rest bonuses; gold gets a
use beyond gear.

**Scope.** Two or three room tiers per inn (common, private, suite) with
different Well Rested strength or length. Inn standing (21b) still
applies.

**Accept.** Each tier through a real inn night; `help inn` updated.

## 76 Bounty boards

**Why.** Deadfire's bounties name a target, its lair and escort, and pay on
proof. Ashveil has lairs and wants treasure hunting to pay.

**Scope.** A board in each town lists bounties for lair bosses and named
groups in nearby bands, with a reward paid on proof (the kill recorded in
the chronicle). Bounties refresh on a real-time rotation.

**Accept.** Post, kill, claim through real commands; reward within band;
`help bounties`.

## 77 Hardcore and account blessings

**Why.** Pillars' Trial of Iron is a badge players chase; Deadfire's
Berath's Blessings reward finishing big things on later playthroughs.

**Scope.** A Hardcore option at creation (the 53 follow-up): defeat routes
to permanent death, shown on `who` and the character sheet. Account
blessings: milestones in the chronicle unlock small perks for the
account's later characters (a starting item, a recruit discount), never
combat power beyond starting gear.

**Accept.** Hardcore death through the real defeat path; a blessing
granted to a second character; help pages.
