# Phase 66: Bestiary earned by fighting

Spec: [Pillars phases](2026-10-07-pillars-phases.md) §66. A leader's bestiary starts blank and fills in as the company beats each kind of creature: lore, then defences, then habits and weaknesses, in words. It feeds preparation (phase 61 orders) and, later, trophy enchanting (71, "which creature drops what").

## Shape

- **Package `internal/bestiary`** (pure rules plus reads of the creature templates): `Tier` (lore, defences, habits), `TierFor(kills, boss)`, `Build(spec, kills)` (an `Entry` with the lines each tier has earned), `Known`, `Find`, `NotesFor` (short habit phrases for captions), `FoeLine` (the `consider` line), `LearnedLine`. Every line is generated from the template (`mobs.GetMobSpec`), race, spells and wind-ups: nothing is written per creature, so the replacement world gets entries for free.
- **State: none new.** Knowledge is a function of the character's own kill tally (`Character.KD.Kills`, creature template id to count), which the company already credits to the leader for every kill the company makes (`mobcommands.Suicide`), skips for the Training zone and practice foes, and saves with the character. It therefore survives restart and copyover and can't be lost by a separate file failing to save. `bestiary.KillsOf` is the one place the source is read: switching to `internal/chronicle` (phase 63, under review when this was built) later means changing that function and nothing else.
- **Commands:** `bestiary` (every kind beaten, by zone, with its tier), `bestiary [name]` (an entry; only known kinds are searched, so it never confirms a creature the player has not beaten), `bestiary [zone]` (a filter). `consider` gains a **Bestiary** line for the foes the player can see: habits known, and which kinds are new. A kill that takes a kind to a new tier says so at once (`mobcommands.Suicide`).
- **Web client:** `Char.Bestiary` GMCP (asked for when the tab opens; refreshed when a battle ends, only for clients that asked), a **Bestiary** tab in the right-hand dock after Comm (`window-bestiary.js`, entries open on tap, open ones stay open across a refresh, text set as text), `known` habit notes on each Battle-view enemy (Combat tab "Bestiary:" line, battle-screen caption). Checked at desktop width and 360px in `scripts/browser/dock-windows-check.mjs`.
- **Help:** `help bestiary` (aliases `monsters`, `beast-lore`, `foe-lore`, `known-foes`, `foe-habits`), listed under combat, linked from `combat`, `consider`, `webclient` and `gmcp-char`; a hint in the Combat tutorial lesson.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| Tiers come from **kills** of a template: 1 (lore), 3 (defences), 6 (habits); a boss teaches at 1, 2, 3. | Kills are already tracked per template and per leader, credited to the whole company, and can't be farmed in Training or on practice foes. A boss is met rarely (a lair respawns slowly), so it would otherwise never be learned. |
| Kind = creature **template**, not family or race. | Races in the shipped world are stock (a wolf is a wolf, a goblin a goblin); a family would let a kill of a rat teach about a giant rat's habits. The replacement world can tag families later. Same-name templates in different zones stay separate entries, told apart by zone. |
| Elite and ordinary versions share their template's entry. | Elites are the same kind, rolled stronger (`IsElite` is a runtime flag); their kill is already counted in the same tally. |
| No new persistence; read the character's tally. | See above. The cost is that the tally also moves with anything else that edits `KD.Kills` (nothing does). |
| Defences are armor percent, evasion and poison/wound words, boss hex resistance, reach and sweep. No health numbers. | A spec's health is computed at spawn, not stored on the template, so a figure here could disagree with the live foe; armor is read from the template's gear, as combat reads it. |
| Habits include spells known, role (healer, caster, guardian), wind-ups with what breaks them, the target rule it re-aims by, attack training and the gear it carries. | These are the facts the engine already acts on, so the text claims only real effects. "Carries" names what the template wears and holds (the drop pool phase 71 will name), not drop chances. |
| The bestiary says nothing of a kind never beaten: the list, search, `consider`, battle notes and the GMCP feed all read known entries only. `consider` names a visible foe's kind as "new to you" without any fact about it. | The coordinator's rule: never reveal anything about foes the player can't see. Hidden foes are left out as `consider` already leaves them out. |
| Habit notes appear in battle only from the habits tier. | Lore and defences are for reading between fights; a caption has room for "heals its allies", not paragraphs. |
| The dock tab is on by default. | A bestiary nobody finds is not a feature, and it only costs one tab (Kills, which is off by default, stays a settings option). The GMCP refresh goes only to clients that asked for it; the web client asks when its windows are built at load (as every docked window does), so in practice every web client gets one small payload when a battle ends. Review accepted that cost. |
| No `orders` change; the page and the habit lines name the rule to set (`orders [who] add foe healer then break`). | Phase 61's conditions already cover healers, casters, chanters and bosses; the link is the text. |

## Not done

- Per-creature drop chances, and which creature drops which trophy: left to phase 71.
- Spell and ability names for foes that use class-style abilities beyond a spell book and wind-ups (the template has none).
- Live smoke run in review (passed).
