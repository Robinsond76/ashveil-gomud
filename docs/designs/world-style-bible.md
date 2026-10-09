# World Style Bible

Status: draft, in progress with Robinson (last updated 2026-10-07). The working
copy is the Claude Doc
[Ashveil World Style Bible](https://claude.ai/code/artifact/fcde5707-6b43-41c1-8771-49ed38763a79);
this file is a snapshot of it for builder threads. The game's name and setting
are not final, so "Ashveil" here is a working name.

The current world is stock GoMud and will be replaced. Builders writing new
world content follow this bible; items still listed under
[Open questions](#open-questions) are not settled yet.

## Tone

Ashveil is a harsh, adult world: survival is earned, kindness is rare and costs something, and nobody is safe for long. Players lead a mercenary company through it, so the writing is a sellsword's view: practical, wary, unimpressed by banners.

Four pillars hold every line we write:

1. **The land is the enemy.** Cold, hunger, rot, distance and dark kill more people than monsters do. Descriptions name what the place does to a body.
2. **Everyone wants something.** Lords tax, priests tithe, guards take bribes, villagers lie to strangers. No NPC is there to help the player for free.
3. **Violence has weight.** Fights are short and ugly, wounds linger, the dead stay dead and somebody grieves them. Nothing is glorious up close.
4. **Small mercies matter.** A dry tent, a fire, a companion who shares bread. The world is cruel so these moments land; it is never cruel for its own sake. The wilds can be as bleak as Pillars of Eternity; towns always keep small signs of happiness, so returning to one feels like relief.

What this is not:

- **Not grimdark parody.** No piling on gore, swearing or misery to sound tough. Restraint reads harsher than excess.
- **Not edgy for the reader.** The narrator never winks, mocks the player or lectures.
- **Not high fantasy.** Magic is rare, costly and distrusted. No elves on white horses, no chosen ones.

Touchstones for the feel: Ogre Battle's war-torn kingdoms, The Black Company's mercenary voice, Darkest Dungeon's dread, Outward's survival, and the Witcher's grey villages. Use them for mood, never for names or plots.

## World premise

A fallen kingdom with a frontier at its edge (decided by Robinson, 2026-10-07; the game's name and the ash idea stay open).

- **The heartland.** The old kingdom collapsed a generation ago. Its heartland is a patchwork of feuding lords, toll barons and cults. This is where companies take sides, sell their swords and get paid in coin or land.
- **The frontier.** While the kingdom stood, a **pact or ward** held the wild borderland shut: a binding, part holy and part magical, that the crown kept. When the kingdom fell, the binding broke. Now settlers, deserters and treasure hunters push into land nobody living has walked, full of old ruins and whatever the ward kept out.
- **Why magic is feared.** People believe magic brought the kingdom down: the crown's mages reached for more power and the ward broke under them. Whatever the truth, the lesson everyone repeats is that magic is unfit for humanity's thirst for power. Casters are distrusted, watched, and in some towns hunted. A company that travels with one pays for it in standing.
- **The open mystery.** Nobody agrees what really broke the ward. The mages? A traitor inside the crown? Something on the far side that pushed back? Every faction tells it differently, and the answer lies somewhere in the frontier.

**What the ward kept out** (decided 2026-10-07: a mix of two things):

- **A hostile wild.** The frontier was always worse than the heartland: harder winters, blight, beasts that grew strange behind the ward. Survival is the first enemy out there.
- **The old owners.** A people the kingdom drove out and sealed beyond the ward long ago. Some survived and changed in the wild, some died and did not rest. They remember what was done to them, so the frontier's enemies have grievances, not just teeth. The kingdom's history books leave this out. They are their own race, not human (decided 2026-10-07): the world's first non-human people, so the kingdom's crime was one people erasing another. More races may be added later.

How it shapes the game:

- **Level is geography.** The heartland holds the lower-level zones; danger rises the farther past the old border a company goes. Zone border warnings become the remains of the ward: broken boundary stones, empty watchtowers, warning cairns.
- **Two kinds of work.** Heartland contracts (feuds, sieges, escorts, bounties) and frontier contracts (clearing roads, guarding settlers, recovering relics from ruins).
- **Relics and legends** come mostly from the frontier and the old kingdom's ruins, which ties relic hunting to the lore.

## Voice rules

Write plain, concrete English in short sentences, present tense, second person for the player. The battle narration already follows this voice (phase 29c: "dark and physical, no exclamation marks, no ALL-CAPS"); the world joins it.

- **Tense and person:** present tense. Rooms and status speak to "you"; NPCs and companions speak in their own voices.
- **Sentences:** mostly under 20 words. One long sentence per paragraph at most, for rhythm.
- **Nouns over adjectives:** "a gibbet with one boot still hanging from it" beats "a grim, foreboding gibbet". At most one adjective per noun.
- **The body, not the camera:** describe what the player smells, hears, feels underfoot and in the lungs, then what they see.
- **No exclamation marks** outside NPC dialogue, and rarely there. No ALL-CAPS.
- **Swearing:** allowed in dialogue when it fits the speaker; never in narration. Coin it in-world where we can ("ash-take you", "by the Pale") so it doesn't read modern.
- **No modern words or idioms:** no "okay", "stuff", "teleport", "level up", "loot" in world text. Mechanics words live in help pages and panels, not in rooms.

Banned in world text:

- "You see...", "You notice...", "There is a..." as openers
- "Suddenly", "very", "really", "somehow", "eerie", "ominous", "mysterious" (show it instead)
- "Ye olde" spelling, faux-archaic "thee" and "thou" (except one faith that uses it on purpose)
- Telling the player how to feel ("You feel uneasy")

## Room descriptions

A room is 3 to 6 sentences (40 to 90 words) and carries five things, in roughly this order:

1. **The strongest sense first.** Smell, sound, cold or footing, before geography.
2. **What the place is for, or was for.** A tannery, a toll bridge, a burned chapel.
3. **One sign of people:** their work, their neglect or their violence.
4. **One hook:** a detail that hints at a story or something to examine ("examine gibbet" must answer).
5. **Exits woven in** where they matter ("the road drops east toward the ferry"), the exit line handles the rest.

Rules:

- Every noun a player might type (`examine`, `look at`) gets its own short description. If you write it, it answers.
- Outdoor rooms get a short **night** variant (one or two sentences replacing the light-based lines) and a **bad weather** line hooked to the existing weather system. Towns get a **market day / empty** variant if they have a market.
- Never describe the player's feelings, other players, or mobs that may not be there.
- Paths between places are routes, not filler: give road rooms one distinct landmark each, or merge them.
- Each zone has a one-line **danger note** at its border that grows harsher as the zone's level rises above the company's (sign, warning from a guard, bones on the road). This is how the difficulty rule is felt in the fiction.

Worked example (a toll bridge, day):

> The river stinks of tannery runoff and wet ash. A timber bridge spans it, its rails hung with tin charms that clack in the wind. A toll-keeper's hut squats at the near end, door barred, a tally of notches cut deep into its frame. On the far bank the north road climbs into pine. Someone has nailed a hand to the toll post.

Night variant: *The charms clack in the dark. A lantern burns behind the hut's shutters, and someone inside is counting coins aloud.*

## People and towns

Towns should feel busy without anyone repeating themselves. The engine now limits idle chatter (PR #139): an NPC volunteers at most one line or conversation every `MobChatterCooldownRounds` (60 rounds, about 4 minutes), never repeats a line to a player who heard it within `MobChatterMemoryRounds` (900 rounds, about an hour), and stays quiet in an empty room. See `internal/mobs/AGENTS.md` ("Ashveil: idle chatter limits").

Rules for the new world:

- **Rare by default.** An NPC volunteers a line at most once every few real minutes, not per round. Most of the time they just work.
- **Never the same line twice to the same player** within a long window (about an hour). When an NPC runs out of fresh lines, it falls silent or does a silent action.
- **Mostly actions, few words.** A smith quenching a blade, a beggar counting coppers. Spoken lines are the minority.
- **Lines react to something.** Time of day, weather, the company's standing, a fresh battle, a wounded companion, the season. An NPC who comments on what just happened feels alive; one who recites doesn't.
- **Talk on request.** NPCs say real things when asked (`ask [npc] about [topic]`), so the volunteered lines can stay few. Townsfolk can each carry one short story (a memory, a grudge) told once per player, as Pillars of Eternity does; after that they offer a line about the present, or nothing.
- **Town ambience lives in the room, not the NPCs.** A fire cracking, a cart going past, bells. Room lines fire rarely and never repeat back to back (the engine already avoids an immediate repeat).

NPC voice:

- Each named NPC has a want, a fear and a way of speaking (clipped, flattering, pious, drunk). Write those three in their file before any lines.
- Common folk distrust armed strangers. Warmth is earned through standing.
- Merchants haggle and grumble; guards threaten before they help; priests ask for coin.
- Quests come as things in the world (a bounty notice, a sealed letter, a widow's request) with directions given in landmarks, not map markers.

## Status in words

Every condition a player tracks gets a short phrase in the text view; the panels keep their numbers and bars. A phrase changes only when the condition crosses a band, so it never spams.

| Condition | Bands, best to worst |
| --- | --- |
| Health | unhurt · scratched · bloodied · badly hurt · barely standing |
| Wounds | none · a shallow cut · a deep wound · a wound gone bad |
| Fatigue | fresh · tiring · worn down · spent · dead on your feet |
| Hunger | fed · peckish · hungry · gnawing hunger · starving |
| Thirst | slaked · dry-mouthed · thirsty · parched · dying of thirst |
| Cold and heat | comfortable · chilled · cold to the bone · freezing (and warm · sweltering · heat-sick) |
| Morale | steady · uneasy · shaken · breaking |
| Load | light · laden · heavy · staggering |

- **Companions** are described the same way, by name: "Ysolde is worn down and favours her left leg."
- **Travel** says distance in words: "a short walk", "half a day's march", "a hard climb" (the real minutes stay in the panel).
- **Battle prose** carries severity in words; the "(5 damage)" suffix stays on by default and hiding it is a player setting.
- Wording is checked against the code: a band name is a promise about the number behind it, so help pages list the bands with their ranges.

## Lore method

Lore reaches players as things people in the world wrote or said, never as an encyclopedia. This is the method that worked best on Geas, where players voted a legend-backed ruin the best area and the main city the worst.

- **Named, biased narrators.** Every lore text has an author in the world: a court scribe, a deserter's letter, a priest's sermon, a tavern song. Each one gets something wrong or leaves something out on purpose.
- **A short, dated timeline** kept by us (not shown whole to players) so every text agrees on what happened when. Few events, each concrete: a siege, a famine, a betrayal.
- **Every lore text ends on an open thread,** and every thread leads to a real place, boss or relic in the world. Relics and bosses still waiting for world placement get their legends this way.
- **Factions disagree.** The same war reads differently in a lord's chronicle and a burned village's song. Players piece it together.
- **Build the first region around one ruin with a legend,** not around the hub town. The hub exists to send people there.
- **Lore is found, not dumped:** books on shelves, inscriptions, NPCs who talk when asked, companion banter at camp. No lore walls in the tutorial.

The first open thread is the one in the World premise: what really broke the ward. Every faction's lore should carry its own wrong or partial answer.

## Adult content lines

Harsh means the world's cruelty is real and has consequences, shown with restraint. Confirmed by Robinson, 2026-10-07.

| Topic | On the page | Off the page or never |
| --- | --- | --- |
| Violence and death | Wounds, blood, corpses, gibbets, executions, aftermath | Lingering torture scenes written for their own sake |
| Cruelty | Raids, burned villages, hangings, slavery and debt bondage as facts of the world | Any of it played for laughs or as a reward |
| Sexual content | Brothels, desire and relationships named plainly in dialogue | Explicit scenes; sexual violence is never depicted, only its aftermath referenced rarely and seriously |
| Children | Orphans, hunger, child labour as part of a hard world | Children harmed on the page; children as combat targets |
| Language | Swearing in dialogue, mostly in-world oaths | Slurs based on real-world groups |
| Bigotry | In-world prejudice between peoples, faiths and classes, with consequences | Real-world groups as the targets |
| Drugs and drink | Drunkenness, ash-poppy, addiction as a cost | Glamourised use |
| Body horror | Rot, disease, ailments, the dead that walk | Gore piled on for shock |

The test for any line: does the harshness tell the player something about the world or the cost of their choices? If it only shocks, cut it.

## Sample passages

These show the voice across the kinds of text the game prints. Names are placeholders.

**A town room, market day**

> Smoke from a dozen cookfires hangs low over the square. Hawkers shout over each other, and a pig squeals somewhere behind the well. The reeve's men lean on their halberds by the tithe barn, watching purses more than faces. A fresh notice is pinned to the barn door. The west lane runs down to the stockyards; the north gate stands open.

**The same room, a cold night**

> The square is empty and frozen into ruts. One brazier burns at the tithe barn, where a guard stamps his feet and does not look up.

**Ambient line (rare, reacts to weather)**

> Rain drives the hawkers under the eaves. Someone curses the reeve, quietly.

**A merchant, after haggling**

> The chandler counts your coin twice. "Robbing a widower with a sick daughter. Hope it keeps you warm."

**A guard, low standing**

> "Your lot stays outside the walls tonight. Argue and you'll do it from the stocks."

**A bounty notice**

> TEN SILVER for the head of the man called Corran Vey, deserter, thief of the reeve's grey mare. Last seen on the ferry road past the hanged oak. Bring the head, not the story. *By order of the Reeve of Hollin.*

**Zone border warning**

> The road past the hanged oak is not kept. Three cairns stand where it forks, each with a broken spear driven into it. Nobody builds cairns for travellers who made it.

**Status line**

> You are worn down and hungry. Brann is badly hurt; his wound has gone bad.

**Lore fragment**

> *"The ash fell for forty days, and on the forty-first the Margrave declared it a blessing. He was the first to eat his horses."* From a chronicle kept by Brother Aldous of the Hollin chapter, its later pages torn out.

## Open questions

These need Robinson's call before builders rely on them.

- [ ] **World name and the ash.** The game's name, and whether the Burning or ash has a place in the premise. The ash in the samples above is provisional.
- [ ] **Faiths.** Tabled 2026-10-07: Robinson may add new races later, and the faiths should grow out of the peoples who hold them. Revisit once the races are settled.
- [ ] **The old owners' race.** Decided that they are their own race; their name, look and culture are still to design.

Decided 2026-10-07: the premise (above); bleakness (Pillars-bleak wilds, towns with small signs of happiness); the adult content table as written; magic rare and feared; the NPC chatter engine fix (shipped in PR #139); battle numbers on by default.
