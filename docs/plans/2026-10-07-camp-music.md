# Camp music and inn gigs (phase spec, 2026-10-07)

Robinson's idea (thread "Camp music", 2026-10-07): companions learn to play
instruments and play at camp for buffs; the more different instruments the
company plays, the better the rest; instruments come in tiers and are
bought, crafted or looted. Agreed in the thread:

- Playing is a short **song before the rest**, not a rest duty. Musicians
  still sleep and can still take a duty as normal.
- Music works **at camp only**, never in battle (battles stay hands-off).
- Musicians **can earn coin** at inns, through timed gigs the company must
  turn up for, with a time limit so it can't be farmed.

The phase number is assigned when the coordinator slots this into
`docs/plans/2026-10-06-remaining-roadmap.md`. Standing rules apply: shared
world time is never advanced, state survives restart and copyover,
difficulty comes only from zone level, and bought or crafted goods never
resell for profit (looted rare goods may).

## Design

### 1. Instrument families and the Music skill

- Four families: **strings**, **winds**, **drums** and **voice**. Voice
  needs no instrument and weighs nothing; the other three need one carried
  in a member's pack or the company cargo.
- **Music** is a per-member skill, levels 0 to 4, tied to one family. The
  player character and every companion can learn it; it is not tied to an
  archetype.
- **Learning:** a music teacher (one in Alderbrook for the test world)
  teaches level 1 of a family for gold. Levels 2, 3 and 4 come from
  practice: each camp song a member plays counts, at 10, 25 and 50 songs.
  Changing family drops the member back to level 1 of the new family
  (paid again). Why: practice makes the skill something a company grows
  together; a teacher keeps level 1 a deliberate choice.
- An untrained member can still sing along. It prints, but adds nothing.

### 2. Instruments and tiers

| Tier | Strength | How you get it | Resale |
| --- | --- | --- | --- |
| Crude | 1 | Crafted from gathered goods (bone flute, gut-strung lyre, hide drum) | Never above what it cost |
| Common | 2 | Bought at markets (lute, reed pipes, frame drum) | Never above what it cost |
| Fine | 3 | Crafted from a found recipe page (phase 56), or a dear shop | Never above what it cost |
| Masterwork | 4 | Looted only: named, one quirk each, from lairs and relic bosses | May sell or salvage for profit |

- Instruments have real weight and count against company load (drums
  heaviest, winds lightest).
- Each masterwork has one small quirk, for example a fiddle whose strings
  don't dull in rain, or a drum that carries less far.
- Item ids are picked well clear of the highest shipped id in each range.
- The existing arcane flute (item 101) stays as it is; it is not a winds
  instrument for this system.

### 3. The camp song

- When `camp rest` starts, every member with Music 1+ and an instrument of
  their family plays automatically (voice members sing). `camp music`
  shows who plays what and the song's effect; `camp music off` and
  `camp music on` silence or restore the song (persisted on the camp).
- **Family strength** = the best player's Music level + their instrument
  tier (voice: Music level + 1). Range 2 to 8. A second player in the same
  family adds nothing to the effect but still earns practice.
- The song prints a few short lines at rest start, visible to everyone in
  the room (the "towns and camps feel alive" lesson from the Geas research).
- Effects are granted when the rest finishes, to every member present
  (sleepers and members on a duty alike). A spoiled rest grants none, as
  with Rested and duties (phase 51, decision 9). Effects last as long as
  that member's rest buff.

| Family | Effect at strength s |
| --- | --- |
| Strings | Rested and Well Rested last 5% longer per point (max +40%) |
| Winds | The rest restores 1 more Fatigue per point (max +8 on the base 20) |
| Drums | +1 speed per 2 points in the first battle after the rest (max +4) |
| Voice | Ailments fade 5% faster per point (max +40%) |

- **Ensemble:** three or more families each at strength 4+ upgrade the
  sleepers' Rested to Well Rested, the same upgrade as the large pavilion
  tent (phase 52), and they don't stack. All four families at strength 4+
  also raise every family effect by a quarter.
- **Costs:** music carries. Each playing family raises the raid and thief
  chance by 10% (drums 20%), multiplied with the tent's scaling in the same
  single roll at rest start (`planRaidLocked`), so a restart never
  re-rolls. Rain or snow halves strings and winds strength unless a tent is
  pitched.
- Numbers are starting values. Tune them so a full ensemble is worth about
  as much as a good tent, never enough to make a higher zone safe.

### 4. Inn gigs

- Each inn posts a gig on its notice board for one window of world time
  each game evening (for example 19:00 to 21:00). The window reads shared
  world time and never advances it.
- The company leader runs `inn gig` in the inn during the window, with at
  least two families able to play. The gig takes about a real minute, like
  a camp rest; the company can't leave the inn until it ends, and everyone
  in the room sees the performance lines.
- **Pay** scales with the inn's zone band and the sum of family strengths,
  with a full ensemble paying well. Tune it so a top gig at a band is worth
  about an hour's hunting there. A weak gig pays little, and the crowd
  says so.
- **Time limit:** one paid gig per company per inn per game evening, and
  at most one paid gig per company every three real hours. Both are
  persisted. Gig pay is earned for a performance, not a resale, so it
  keeps to the economy rule; the limits keep it from becoming a farm.
- Gigs give practice like a camp song.

### 5. Companions and banter

- Banter (phase 49) gains lines for the song: some personalities ask for a
  tune, some grumble at the noise. Fired through the camping banter hook
  (`onDuties` is still unused; add an `onSong` beside it).
- Later, with the Pillars companion opinions work, a member who loves music
  may warm to a skilled musician. Out of scope here; the hook is enough.

### 6. Web client

- Camp tab: a Music row listing each player, their family and strength,
  the families covered (for example 3 of 4) and the effect the next rest
  will give, plus the music on/off toggle.
- Company panel: each member's Music family and level, and practice
  toward the next level.
- Inn room: the gig notice with its window and the company's eligibility.

## Acceptance criteria

- With no musicians, or music off, a camp rest is exactly as today.
- Each family's effect fires and is visible on the member, at the stated
  numbers; the ensemble upgrade to Well Rested fires at three families at
  strength 4+ and does not stack with the large tent.
- A spoiled rest grants no song effects; raid and thief chance rises with
  playing families in the single rest-start roll.
- Practice levels a member at 10, 25 and 50 songs; a family change drops
  them to level 1. Skill, practice, music on/off and gig limits survive
  restart and copyover.
- Crude, common and fine instruments never sell or salvage above their
  cost; masterwork may.
- A gig works only in the window with two or more families, pays by band
  and strength, and respects both time limits. Nothing advances world time.
- **Help pages:** new `help music` (skill, families, effects, ensemble,
  costs), `help instruments` (tiers, where to get them, weights) and
  `help gigs`; updated `help camp`, `help camp duties` (music is not a
  duty), `help inn`, `help tents` (no stacking with the pavilion) and the
  crafting page. All indexed in `keywords.yaml` with aliases (`song`,
  `bard`, `lute`, `busk`), linked from `help camp`.
- **Tutorial:** a hint in the camp lesson pointing to `help music`.

## Tasks

- [x] Music skill model: family, level, practice, persisted on players and
      companions; teacher NPC and `learn` flow.
- [x] Instrument items for each family and tier; crude recipes in crafting,
      fine via recipe pages; masterworks in lair loot tables; no-profit
      resale for the first three tiers.
- [x] Camp song: `camp music`, strength, effects, ensemble upgrade, raid
      and weather costs, practice, room lines.
- [x] Inn gigs: notice board window, `inn gig`, pay, both limits.
- [x] Banter `onSong` hook and lines.
- [x] GMCP payloads and the Camp tab, Company panel and inn views.
- [x] Help and tutorial: the pages and hint above.
- [x] Tests: each family effect, ensemble threshold and tent non-stacking,
      raid scaling in the single roll, spoiled rest, practice levels,
      persistence across restart, resale caps, gig window and limits,
      help renders, `TestTutorialHelpPointersExist`.
- [x] Short tuning run: full ensemble against the large tent; gig pay
      against an hour's hunting at two bands. Timeboxed.

## Build decisions (2026-10-07, full autonomy)

Each is the builder's call, with a one-line reason.

- Skills live in the camping registry by leader and member key, not on
  characters: companions are not characters between sessions, and it keeps
  one persisted store (with purge and the test-area snapshot).
- Instruments are company-wide: one carried instrument of a family serves,
  the best tier is played. Reason: no per-member inventory bookkeeping.
- Winds' Fatigue is added at recovery time, so a spoiled rest gets none.
- Practice counts at the grant, once per rest (and once per paid gig), so a
  spoiled rest is no practice and a restart cannot double-count.
- Voice shortens each ailment's remaining battles by ceil(left x pct), at
  least one, so small percentages still matter.
- Drumbeat (buffs 9401-9405) ends with the first `BattleEnded` or expires
  with the rest buff.
- Masterworks are sold, never salvaged: instruments have no salvage line.
- Gig pay = (zone band top + 2) x summed family strengths x 3 / 2 (review;
  the build had / 2), x1.25 for an
  ensemble, x1.5 for a full one; zones with no band pay as band 5 (Dunmar).
- Crafting happens at the leader's camp, from packs and cargo.
- A gig that never finishes spends neither the evening nor the cooldown; the
  pay is credited once on the game loop and saved as settled first.
- Teacher is the room tag `music-teacher` (Alderbrook green); the shop is
  Dunmar's market, supply only. Fine recipe pages lie in three fen and down
  rooms (the stock world is temporary).
- Masterwork lairs: Wren's rainfiddle (boss 85), hollow king's horn (lich
  14), ent-heart drum (ent 34), 25% each per company kill.

## Tuning note

Build (analytic, timeboxed): a full ensemble of fine instruments (strengths
7, 7, 7, 5) at band 15 earned 331 gold a gig, about a third of an hour's
hunting there.

Review (2026-10-07, analytic, timeboxed; `TestGigPayTuning` pins it):

- **Gig pay** raised x3 to meet the spec's "a top gig is worth about an
  hour's hunting". Hunting income was taken as encounter caches
  (`loot.CacheGold`, about 5 gold a level each) at roughly 15 fights an
  hour. A top gig (masterworks at Music 4: 8, 8, 8, 5) now pays 456 at band
  5, 1,108 at band 15 and 2,088 at band 30, 14 to 18 caches; fine
  instruments at band 15 pay 994. A beginner pair at band 5 pays 42. At
  most one gig per 3 real hours, so musicians add about a third to a
  company's hourly coin at best: worth playing, not a farm.
- **Buffs** kept. A full ensemble is a little better than the pavilion
  (Well Rested, plus at most +50% rest length, +10 Fatigue, +5 speed for one
  battle, ailments 50% shorter) and needs three or four trained members and
  instruments; nothing in it changes how a higher zone's fights go beyond
  one battle's speed.
- **Raid chance** kept: +10% a family, +20% for drums, so +30% to +50% for
  an ensemble (the build note's +70% was wrong: the cap is +50%), against
  the pavilion's +50%. Old Kings Road's 15% raid chance becomes at most
  22.5%.
