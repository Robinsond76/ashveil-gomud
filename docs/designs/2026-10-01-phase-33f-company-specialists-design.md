# Phase 33f: Company Specialists and Expedition Skills

Status: owner-approved direction, 2026-10-01. The original future design
(specialist assignments, group stealth, solo scouting, medic and
quartermaster roles) was reviewed with the owner and replaced by the
decisions below. The first proposal is in git history.

See the [company gameplay roadmap](2026-10-01-company-gameplay-roadmap.md)
for sequencing and the decision register.

## Owner decisions (2026-10-01)

The owner reviewed every skill shipped from GoMud against a game about
leading a persistent company on expeditions, and decided:

1. **Retire the skills built for a lone player.** Search, Track (the
   stock command), Peep, and Sneak are weak in a company game; the owner
   also confirmed that a separate medic role adds nothing to `heal wounds`.
2. **Retire the skills that clash with Ashveil:** Portal (it bypasses
   travel, terrain, fatigue, and interruptions), Pickpocket and Bump
   (theft from other players), Tame, Change Form, Scribe, and Pray.
3. **Remove charm functionality** as a player-facing mechanic: taming,
   hired mercenaries, and mobs that befriend a player. Company companions
   keep the engine's internal charm link (it is how a companion follows its
   leader); only the ways for a player to gain other followers go.
4. **Build all of the suggested expedition skills,** and add more where
   they fit. A dedicated cleanup phase is acceptable.
5. **No group stealth, no solo scouting, no medic, no quartermaster
   assignment.** Cooking is wanted but needs care; it ships last.

## Delivery

| Slice | Scope |
|---|---|
| 33f1 | Skill and charm retirement |
| 33f2 | Expedition specialists: the specialist resolver, `company specialists`, Read the Trail (replaces `track`), Keen Eye (replaces `search`), Pathfinder, Weather Sense, Haggle |
| 33f3 | Camp specialists: camp raids and Camp Watch, Field Smith, Vigil, Forage, camp cooking |

`search` and `track` are retired in 33f2, together with their
replacements, so secret exits never become unfindable between slices.

## Shared rules (every slice)

- **Who can act.** A specialist is the leader or one of the leader's own
  living, present company companions (by `company.LeaderAndKeyForInstance`
  and `IsCharmed(leader)`), never a pet, another player's company, an ally,
  the dead, a fled or withdrawn member, or one separated in another room.
  This is the 33e side list, reused.
- **Best one acts, no stacking.** The member with the highest level in the
  capability does the job (ties: the leader, then the lowest companion id,
  the 17b `BestMember` rule). Other members add nothing.
- **Levels.** A player's level is their level in the mapped skill, only if
  their archetype performs the capability (17b `PlayerUtilityLevel`). A
  companion's comes from its archetype and character level (17b
  `CompanionLevels`: levels 1/10/20/30 give 1–4).
- **Attribution.** Results name the member who did the work.
- **No battles, no items in a fight.** No specialist action runs inside the
  actor's own battle (33a), and none uses an item in a fight (owner rule).
- **Ownership and multiplayer.** Specialist effects reach only the
  leader's own company. Nothing affects another company's battle, and
  information (tracks, forecasts, hidden exits) is shown only to the leader.
- **Time and persistence.** Nothing advances shared world time. New durable
  state has yaml defaults; existing saves load unchanged.
- **Toggles.** Passive capabilities obey the existing 17b
  `autoskill <utility> on|off` switch.

## 33f1: Skill and charm retirement

**Removed player commands:** `peep`, `portal`, `tame`, `changeform`,
`scribe`, `sneak`, `bump`, `pickpocket`, `pray`, and `backstab` (it needed
the hidden state only `sneak` gave; in a battle 33e's automatic Opening
Strike already plays the rogue's opening, so `help backstab` opens
`help skulduggery`). Typing one gets the
ordinary unknown-command answer. Mob-side `sneak`, `portal` (scripted room
portals), and engine portal exits stay: they are world behavior, not skills.

**Removed skills (data):** `peep`, `portal`, `tame`, `changeform`, `scribe`
are deleted from `skills/`. Skulduggery keeps picklock, traps, and the
automatic Opening Strike (its levels still raise trap and utility checks,
so no refund); Protection keeps aid and rank and is capped at level 3 (its
level 4 granted only pray), refunding level 4's 4 points. Their
descriptions are updated.

**Archetypes and professions.** The rogue loses `peep`, the wizard
`portal`, the ranger `tame`. Professions (job titles) drop retired skills:
Arcane Scholar keeps Enchant and Inspect, Merchant becomes Trading and
Inspect, Treasure Hunter loses Peep; Explorer and Monster Hunter, left with
too little, are removed.

**Trainers and world content.** Trainer rooms stop offering retired
skills. The Whispering Wastes obelisk no longer teaches Portal and becomes
scenery. The long whip loses its `tame` stat bonus. Scripts and
conversations that teach or mention a retired skill are corrected.

**Charm removal.**
- The Tame skill, the `tameskill` spell, tame mastery (learning from kills,
  stored per character), and the charm cap from Tame are removed.
- Shops no longer hire mercenaries: shop entries with a `mobid` are hidden
  from `list` and refused by `buy`; the Bonecrafter's skeleton is removed.
- The mob `befriend` command is removed.
- The scripting API loses `CharmSet`, `CharmRemove`, `CharmExpire`,
  `GetCharmCount`, `GetMaxCharmCount`, `IsTameable`, `GetTameMastery`,
  `SetTameMastery`, and `GetChanceToTame`. Read-only `IsCharmed` and
  `GetCharmedUserId` stay (companions use them).
- **Recovery:** a charm is runtime state (`Character.Charmed` is not
  saved), and nothing can create a non-companion charm any more, so no
  tamed creature or mercenary survives the restart or copyover that
  deploys this change; companions are re-summoned by the company module as
  before. No sweep is needed.
- **Pets stay.** GoMud pets (`Character.Pet`, bought from shops) are not
  charmed mobs; the owner did not ask to remove them. They are flagged in
  the roadmap's pets/charms decision.

**Migration.** A character holding a retired skill is refunded the
training points it cost (1 + 2 + … + level) at their next spawn, and the
skill is removed from their sheet (Protection above level 3 is lowered,
refunding the difference), with a one-line notice. The refund and
the removal change the same user record and are saved together, so the
refund never repeats. Skill levels granted free (the
Portal obelisk's level 1) are refunded as well, a small generosity
accepted for simplicity.

**Help.** Help pages and keywords for retired commands and skills are
removed; pages that mention them (`guide`, `training-schools`, skill
lists, `help company`, `help webclient` if needed) are corrected.

## 33f2: Expedition specialists

**The specialist view.** `company specialists` (alias `specialists`)
lists each capability, who in the company would perform it now and at what
level, or why nobody can ("no tracker present"), and whether its
`autoskill` switch is on. It reuses 17b's utilities table.

**Capabilities and archetypes (17b utilities table):**

| Capability | Archetype | Player skill | What it does |
|---|---|---|---|
| traps (existing) | rogue | skulduggery | unchanged |
| light (existing) | wizard | cast | unchanged |
| trail (Read the Trail) | ranger | track | warns of enemy groups nearby and on routes |
| pathfinder | ranger | track | lowers the company's walking strain on rough terrain |
| keeneye (Keen Eye) | rogue | skulduggery | notices secret exits |
| weather (Weather Sense) | wizard | cast | forecasts the next weather |
| haggle | rogue | skulduggery | better market prices |

The `track` skill stays as the ranger's skill (it already unlocks Aimed
Shot in 33e); its description and help change to Read the Trail. The
`search` skill is retired (refunded as in 33f1).

**Read the Trail (ranger).** On each step, and on `track` with no
argument, the best tracker reads the exits of the room the company stands
in: enemy groups in adjacent rooms are reported per exit, once per room
visit. Level 1: "fresh tracks of something, north"; level 2 names the
group ("a band of ruffians"); level 3 adds how many; level 4 also reads
rooms two steps away. Only hostile groups count; players, companions,
and peaceful mobs leave no report; groups of one name seen the same way
are counted together. On an expedition route whose interruption can be an
ambush, a tracker present at departure warns the leader (the kind is still
rolled when the interruption fires, so the warning says "may"); when the
ambush fires, the tracker leads the company around it with a chance of 20%
per level: the saved interruption becomes ordinary tracks and nothing
spawns. Otherwise the ambush happens as before. No new durable state.

**Pathfinder (ranger).** The company's walking strain on terrain heavier
than road is reduced by 5% per level (20% at level 4). It does not touch
time, travel durations, or exertion on expedition routes.

**Keen Eye (rogue).** On entering a room with a secret exit the company
has not yet noticed, the best spotter rolls `level × 20 + Perception/4`
plus d100 against 100 (a company without a rogue rolls the leader's
Perception/4 only; switching `keeneye` off stops both). On success the
leader learns the exit, remembered on the character
(`KnownSecretExits`, saved with the user) and shown in that room's text
and GMCP exits from then on. One roll per room per visit; no command.

**Weather Sense (wizard).** Each zone's weather pre-rolls its next
condition when the current one is set (saved; old saves roll it on load),
so a forecast is the truth, not a guess. `weather` adds a forecast line
when a weather-sense member is present: level 1 says whether it turns
wetter, drier, colder, or warmer; level 2 names the next condition; level
3 adds about how long until it changes (game hours); level 4 forecasts the
neighbouring zones the room's exits lead into.

**Haggle (rogue).** At a market, the best haggler present improves the
price by 2% per level (8% at level 4), after standing: cheaper buying
(rounded up), dearer selling (rounded down). A sale never pays more than
one gold below what buying the same good back would then cost (haggled),
so a buy-and-sell loop can't make money. `market` lists the haggled
prices. Inn and shop prices are unchanged.

**Retirement of search, track, and trading.** The `search` command and
skill go (refunded as in 33f1), and so does the `trading` skill (review
found it granted nothing in Ashveil: its auction and stock commands never
existed here, and haggling is now the rogue's); the Merchant profession
goes with it; the stock `track` command goes and the archetype
module's bare `track` reads the trail (once a round, even with the
automatic reading off). Help (`help search` opens Keen Eye, `help track`
Read the Trail), keywords, the Frostfang trainer, and the professions
change to match.

**Who counts** (all 33f2 capabilities): the 17b member resolver: the
leader and their own spawned companions standing in the room (or, on a
step, the room just left), not downed; each capability's `autoskill`
switch gates it, and none acts in the leader's own battle.

## 33f3: Camp specialists

**Camp raids.** A camp rest in a room configured for raids (a per-room or
per-zone `CampRaids` table: mob template, chance) rolls once, at rest
start, whether raiders come and when (saved on the camp, so restart and
copyover keep it). Shared world time never moves; the rest is the existing
real-time rest.

**Camp Watch (warrior).** When raiders come, the best watch (warrior
level) spots them with a chance of 25% per level. Spotted: the company is
roused, fights, and the rest resumes afterwards, so Rested is still earned.
Unspotted: the raid catches the company asleep: the rest is broken (no
Rested tier) and the fight begins. Raiders spawn in the camp room as an
encounter of that company only (the expedition ambush spawner).

**Field Smith (warrior).** When a field smith is present, each whetstone
use gives an edge of 20 + 5 per level strikes (40 at level 4). Uses spent
are unchanged; no free whetstones.

**Vigil (cleric).** A completed camp rest with a cleric present raises
each present companion's loyalty by 1 per cleric level, up to a loyalty
of 60, once per rest. Steadier loyalty feeds 30e nerve and 21a desertion.

**Forage (ranger).** A completed camp rest in a room whose zone has a
forage table yields food and water by ranger level (0–2 + level/2 items,
table-weighted), into company cargo under 32f capacity (what does not fit
is left on the ground and named). Once per rest; no ranger, no forage.

**Camp cooking.** A lit camp fire is a hearth for cooking: `camp cook`
uses the existing recipe selection (18b), ingredients from the company's
cargo, and the cooking level of the best cook present. Cooking remains a
trade skill the leader trains; companions do not cook in this release.
Exact recipes and buffs are settled in 33f3's plan with the owner's
cargo-phase rules (no personal inventory, cargo-held supplies).

## Acceptance criteria and verification

Each slice ships its plan, integration tests through real entry points
(commands, the walking step, travel departure/interruption, camp rest
completion, market trades, load/copyover recovery), indexed help, a
tutorial pointer, and an independent review whose findings are recorded in
Project Status.

**Help pages.**
- 33f1: removed pages and keywords; corrected `guide`,
  `training-schools`, skill descriptions, and any page naming a retired
  command. `TestTutorialHelpPointersExist` passes.
- 33f2: `help specialists`, `help trail` (replaces `help track`),
  `help keeneye`, `help pathfinder`, `help forecast`, `help haggle`;
  `help company`, `help weather`, `help market`, `help autoskill` updated.
  Tutorial Departure points to `help specialists`.
- 33f3: `help campwatch`, `help fieldsmith`, `help vigil`, `help forage`;
  `help camp`, `help cooking`, `help sharpen` updated; the Camp lesson
  points to them.
