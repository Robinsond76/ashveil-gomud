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

**Scope.** A background chosen at character creation (soldier, scholar,
outlaw, acolyte, labourer, noble), offered to existing characters once.
Story events (60) and town lines (68) can require one.

**Accept.** Creation flow; at least one background choice in each test
event; `help backgrounds`.

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
