# Combat rebalance: second opinion (2026-10-06)

Status: analysis only, written 2026-10-06 at the owner's request after 35b
merged (`fb89b3f`). Its suggestions became the
[35d combat feel design](../designs/2026-10-06-phase-35d-combat-feel-design.md).

Read against master `fb89b3f` (35b merged): the 35a2 and 35b measurement docs,
`internal/combat/calculations.go` and `combat.go`, `internal/coordination`,
`internal/strategy/decide.go`, `internal/interrupt`, the shipped
`_datafiles/config.yaml` Combat block, and the harness in
`modules/company/balance_test.go` and `balance_mana_test.go`. Analysis only.

## What actually drives each miss

### 1. Fight length (equal 5v5 at 13 to 15 rounds, zone middle at 9 to 10)

Round count is arithmetic, not tuning. A landed blow is a fifth of a
fighter's HP (35a2 row 1), so a kill takes 5 landed blows. A blow lands when
the 75% to-hit passes and the one active defense fails (block 28 + shield
armor, or dodge/parry 8 plus modifiers), which the harness measures at 53 to
57%. Tempo at level 10 is about 0.8 turns per round. So one fighter lands
0.55 x 0.8 = 0.44 blows a round and needs about 11 rounds to kill its
opposite number; a 5v5 ends when the last pair resolves, so 14 to 15. At level
60 tempo is 1.07 and the same sum gives 8 to 9 rounds, which is what the table
shows (11). The 35b retune of chances already pushed landed-hit rate close to
its ceiling: raising ToHitEven from 75 to 80 "changed nothing more" because
block and armor, not the to-hit roll, take most of what remains.

Two things make the 8 to 12 target the wrong goal rather than a near miss:

- The level-impact design itself says "the 10 to 15 round mirror target from
  30g6 remains a stress check, not a goal". The shipped encounter contract is
  2 to 3 under-level foes, and those fights run 9 to 10 rounds (36 to 40
  seconds at RoundSeconds 4). That is a reasonable MUD fight; 4 to 8 rounds
  would be 16 to 32 seconds of text.
- Players experience lines and seconds, not rounds. Fewer rounds with more
  turns each produces the same text. If fights feel long, the lever is the
  combatpace window and the lines per round (mean 12, peak 30), not the
  round count.

### 2. Coordinated tier 3 at level 10 (47 to 63% with tactics, 32% without)

Tier 3 (Drilled) is 100% focus fire, casters first, spells at the focus, two
guards, heal below 60%. Tier 2 (Band) is 50% focus with a 15% noise floor.
Against 5-blow HP pools, concentration is decisive on its own: five foes
focusing one member drop it in about two rounds, before any heal lands.

The company's default is each member aiming at "the weakest it can reach",
with no shared aim, no guard, and a cleric that is the enemy's first target.
The measured cast counts show the cost: at level 10 the default cleric begins
5.0 heals and finishes 2.5 (2.0 broken), healing 13% of damage taken; with the
warrior guarding it, 7.8 begun, 5.9 finished, 32%. `BreakChance` is 40 + 2 per
percent of max HP taken, so any ordinary blow on a 55 HP cleric breaks the
chant about 80% of the time. The owner asked for "stronger healing" and the
numbers say healing is mostly being interrupted, not undersized.

Also note the test asks a level-10 company to beat the tier it meets at level
25. The 33% floor there is already generous. Tier 3 at 10 is a probe, not a
balance failure.

### 3. Mana after four fights (cleric 28%, wizard 34%, target under 25%)

The mana run shows HP, not mana, ends a run: wins fall 100 / 89 / 65 / 45%
across the four fights. The company patches only to the 50% healing threshold
(`DefaultHealing`), so it enters fights 3 and 4 at half health and loses
fight 4 more often than not. Mana is "slightly high" because the healer is
told to stop at 50%. This miss is a symptom of the patch threshold, not of
pool sizes.

### 4. Zone bands

Reading the cells in `TestBalanceZoneBands`:

- **Members fallen at bands 18+ (no one fallen 46 to 60%, target 85%).** The
  cells are level-flat (company at the band's middle against foes one level
  below or at the band), so what changes with band is the enemy's coordination
  tier, which `coordination.ForLevel` ties to level: Rabble below 10, Band at
  10, Drilled at 25. Targeting noise floors drop 30 -> 15 -> 5, so high-band
  enemies pick the weakest member reliably and the cleric dies. Nothing on the
  company side grows to match. A second, smaller effect: HP growth slows to
  0.2 a level after 20 while Strength damage and Speed tempo keep growing, so
  same-level fights get deadlier with level (the mirror shortens from 15 to 11
  rounds).
- **4 foes (1.4 to 2.3 fallen, 43 to 53% HP lost; target 1 fallen, 45%).** The
  cell puts 4 foes at the band's top minus one against a company at the band's
  middle: that is a near-even 4v5, two of the four are shield warriors, and
  the enemies focus the weakest. Losing one or two members in an even fight is
  the system working. The contract says 4 foes are "occasional"; they should
  be the zone's low-level foes, not its top.
- **Bosses (38 to 74% wins, 27 to 33 rounds; target 70 to 85%, 10 to 15).**
  The boss cell is a full 5v5 at the band's top where one foe has 2.5x HP and
  +2 levels, and the whole group runs at the band's live coordination tier.
  Two problems. (a) The owner's encounter contract records "bosses of up to 5
  with no strategy" (PROJECT_STATUS 2026-10-05); the harness gives the boss
  group the band's tier, which is the opposite. At 18+ that is tier 2 or 3
  focus fire, which is where the win rate collapses (38% and 46% against 60%
  and 74% at low bands). (b) A boss alone needs about 13 landed blows, so the
  fight cannot be 10 to 15 rounds with 4 full-strength escorts beside it.
- **Under-levelled company (93 to 98% wins; target 30 to 60%).** Three levels
  is 3/14 of a skill edge: to-hit moves from 75 to about 61 against them and
  theirs to about 79. Five members against 2 or 3 foes still win on numbers.
  The row conflicts with the skill curve as shipped: the 10-level gap gives a
  wall (L10 vs 2xL20: 20% wins) and a 3-level gap gives a nudge. Making 3
  levels bite would need a span near 8, and 35b already found span 12 drives
  the 10-level row to 4%.

## Suggestions, in priority order

1. **Patch to a higher threshold out of battle than in battle** (35b-sized
   follow-up, or fold into 36b or 37). Keep the in-battle heal threshold at
   50% and add an after-battle patch threshold defaulting to 75 to 80%
   (`company tactics patch 80`). Fixes: fight 3 and 4 win rates in the mana
   run, and mana lands under 25% by fight 4 without touching pools or costs.
   Risks: the run-length between rests shortens if pools are not raised; the
   doc and help pages for `patch` and `tactics` change.

2. **Make the healer survivable by default** (combat engine; fits a small 35d
   tuning slice before 37, since 37 measures against it). Two options, pick
   one or both: (a) a warrior guards the healer by default when the company
   has one (the "tactics" mode of the harness becomes the default company), or
   (b) lower chant fragility for 1-round chants: `BreakChanceMin` 40 -> 25 and
   the per-percent term 2 -> 1.5, keeping heavy force at 100. Fixes: tier 2/3
   win rates without tactics (the 32% cell), healing as a share of damage,
   and part of the band-18+ fallen rows. Risks: (a) spends the warrior's
   guards on the cleric so it absorbs less for the leader; (b) also helps
   enemy healers, which lengthens coordinated fights. Both are measurable in
   the existing harness within the 10-minute timebox.

3. **Give the company a coordination ladder that matches the enemy's** (37,
   or 38b with routes). The enemy gains focus, guards and casters-first by
   level; the company gets nothing. Mirror it: a company led at level 10 gets
   shared-aim focus by default, at 25 its guard count rises (already partly
   there via `MaxGuardsFor`), casters-first targeting as a tactic. Fixes: the
   band-18+ fallen rows and the tier-3 probe without lowering enemy tiers.
   Risks: overlaps the 38b talents and routes; shrinks the value of the manual
   `tactics` command unless the ladder is framed as unlocking options.

4. **Run bosses at the contract's "no strategy"** (37, where boss rolls are
   set; a harness-only change can land earlier). Set the boss group's
   coordination to Rabble (or explicit none) in the boss cell and in 37's boss
   spawns, and give the boss 2 to 3 escorts at the band's low level rather
   than 4 at its top. Keep 2.5x HP and +2 levels. Fixes: boss win rate into
   70 to 85% and rounds toward 15 to 20. Risks: a boss that does not focus
   fire is less frightening; compensate with its own abilities (a windup, a
   stagger crit) rather than numbers, which is what 38b abilities are for.

5. **Retarget the 4-foe cell** (37). Four foes come from the band's low level
   (or one below it), not its top. Fixes: fallen and HP-lost rows. Risk: none
   to code; the design table's row should be reworded "4 foes at band low".

6. **Replace the round targets with time-and-text targets** (docs, 37's
   measurements). Record median seconds (rounds x RoundSeconds) and lines per
   fight; set the zone-middle target at about 30 to 45 seconds and the mirror
   as a stress check only, as the design already says. Fixes: stops the
   harness asserting a number the design's own row 1 forbids. Risk: none.
   If shorter fights are still wanted after that, the one lever that does not
   break the fifth-of-HP rule is tempo (TempoSpeedRef 2 -> 0, about +10% turns
   a round); it shortens every fight proportionally and leaves the skill edge
   alone, at the cost of more lines a round.

7. **Leave the under-levelled row to 37's band assignment, reworded** (37).
   Either change the row to "5 levels under band low" (where the edge is 0.36
   and the data would likely land near 50 to 70%), or let an under-levelled
   company meet the band's 4-foe groups more often. Do not shrink
   `SkillEdgeSpan` for this: it makes adjacent-level fights lopsided and the
   +2 boss far worse. Fixes: an honest acceptance row. Risk: none.

8. **Mana pools: leave them.** With suggestion 1 the fourth-fight numbers
   land under 25% on their own. If not, raise Minor Heal's cost 3 -> 4 before
   touching `ManaBase` or `ManaPerLevel`.

## What I would not do

- Push ToHitEven or cut defenses further to shorten fights. The data shows
  the ceiling is reached, and the shield's block is what makes the warrior the
  tank (35b's 0.70x row).
- Raise damage per blow. It breaks 35a2 row 1 and makes the fallen rows worse.
- Lower enemy tiers to pass the band-18+ rows. The owner asked for strategy
  rising with enemy level; the company should rise to meet it.

## Addendum: what I think is good gameplay here (asked 2026-10-06)

Robinson asked for the gameplay view, with settled decisions open to question.

**The frame.** Combat is Ogre Battle style: the player sets up and then
watches. In a watched fight the things that matter are (1) that every line
means something, (2) that the setup decisions visibly paid off or did not, and
(3) that the fight ends before the player's attention does. Round counts and
win percentages are proxies for those three.

**Decisions I would keep.** Skill over HP (35a2): a level should make a
fighter better, not thicker. Mana only from rest, inns and draughts, with
healers patching afterwards: a clear resource clock. No player input
mid-battle. Strategy rising with enemy level.

**Decisions I would question.**

1. *Skill expressed almost entirely as hit-or-miss.* With a 75% to-hit and a
   block or dodge after it, about 45% of swings produce nothing, and in a
   watched fight that is 45% of the text being noise. The design's own row 1
   then forces 5 landed blows per kill, so length is locked in. Better: let
   nearly every swing land (to-hit 85 to 90, defenses mostly as damage
   reduction) and move the skill edge into how hard it lands: a glancing blow
   for a tenth of HP when the edge is against you, a solid one for a fifth
   when even, a telling one for a third when it is with you. Same expected
   damage, same skill wins, fewer dead lines, and fights of 8 to 10 rounds
   fall out naturally because fewer swings are wasted. Shields keep their
   block as the one true negate so the warrior stays the tank. This is a
   35a2-sized change to `calculations.go` and the harness, not a retune.

2. *Heals as coin flips.* A cleric whose chant breaks 80% of the time it is
   touched is not a healer, it is a dice roll the player cannot influence
   except by guarding it. In a no-input game, an action a whole class is built
   around should resolve. I would make a 1-round heal unbreakable by ordinary
   blows (heavy force still breaks it), keep attack spells breakable, and cut
   the enemy healer's heal to match. Then healing is a real part of the
   setup: do you bring a cleric, and whom does it stand behind.

3. *The three-fights-then-camp loop.* Patching to 50% and trickling to 50%
   means fight 4 is lost half the time, so the real loop is three fights, then
   camp. That makes rest a wall, not a decision. Patch to 80% by default and
   size mana so a company gets 5 to 6 fights before the healer is dry: then
   mana is the clock the owner wanted, and pushing on with a dry healer is a
   real gamble instead of the only option.

4. *The boss as a hit-point sack.* 2.5x HP plus four full escorts is thirty
   rounds of attrition. A good boss fight is short and legible: 1.5 to 1.75x
   HP, two or three escorts, and one thing the boss does that the setup can
   answer (a telegraphed wind-up the guardian can absorb, a stagger that
   punishes the front line). That is where 38b's abilities earn their place.

5. *The under-levelled row.* Three levels under should not be a 50% fight,
   and the shipped curve agrees. The danger for an under-levelled company
   should come from where it walks: zones spawn by band, so a band two steps
   up is lethal and the map tells the story. Drop the row.

6. *Flat HP after level 20.* Damage and tempo keep growing with stats while HP
   grows 0.2 a level, so same-level fights at 30 are deadlier than at 10, which
   is why members fall more often at high bands. Either hold Strength damage
   after HPFullLevels or raise HPAfterFull toward 0.4. Small, measurable.

**The one big idea.** Make the company's growth show up in the setup layer,
not only in dice: shared aim at level 10, a second guard slot and
casters-first targeting at 25, and so on, matching the enemy's tiers. In
Ogre Battle the army got smarter, not just stronger; this fork should too.
