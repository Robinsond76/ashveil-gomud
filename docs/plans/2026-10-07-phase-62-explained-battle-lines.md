# Phase 62: Battle lines that explain themselves

Spec: [Pillars phases](2026-10-07-pillars-phases.md) §62. A battle plays out on its own, so understanding why a blow missed, glanced or was turned aside is how a player prepares the next fight. Every weapon strike now keeps the engine's own roll, and the player reads it back in plain words.

## Shape

- **Engine** (`internal/combat`): `calculateCombatPower` records one `combatstream.Strike` per weapon strike as it resolves it, never recomputed afterwards: the chance rolled against and what moved it off the skill-and-speed base (darkness or a second weapon, company chemistry), the roll, the defence met and its chance, quality, crit, the dice before and after quality and named modifiers, the armor rating and what armor and wards took, and notes for each named modifier that acted (sharpened edge, readied strike, Iaijutsu, wind-up multiplier, divine shield, ward, aura, ward-of-life, coup de grace, class, weather or fare effects). `AttackResult.Strikes` carries them; `activeDefenseRoll` and `hitRollDetail` report the chances and roll the old functions used.
- **Stream** (`internal/combatstream`): `Event.Strikes`; `Strike.Explain` turns a strike into plain lines; the summary gains **Damage taken**, **Never landed** (per member: missed, turned aside, stopped by armor), **Moves** (class abilities used) and **Sigil** (the sigil held over the battle, set by the hooks). A per-leader `RollLog` (runtime only, 40 rounds, cleared when the leader's next fight opens) keeps rounds with strikes.
- **Text clients**: `why` (last three rounds), `why list`, `why [number]`; works in or out of a fight; free and changes nothing. The summary lines reach every client through the existing summary text.
- **Web**: `Company.Battle.Event` attacks carry `explain` (the same plain lines, from the same function) for the company's own rounds in either direction; the Combat tab keeps a **Last rounds** list (12, newest first, kept after the battle) under the battle view and Setup view, each a one-line heading opening into the breakdown (`battle-rounds.js`, pure, Node-tested).
- **Help**: `help battlelog` (aliases `why`, `explain`, `rolls`, `battle-log`), linked from `combat`, `battle-summary`, `narration` and `webclient`; the summary page documents its new lines; a hint in the Combat tutorial lesson.

## Decisions (best judgment, full autonomy)

| Decision | Reason |
| --- | --- |
| Record the roll's parts inside the strike loop, not rebuild them afterwards. | A rebuild could disagree with what happened (the roll, the defence, ward absorption); recording is the engine's own numbers, and tests check the strikes sum to the round's damage. |
| One command, `why`, not a setting that adds brackets to every line. | Robinson keeps the `(5 damage)` numbers on; extra text on every line would clutter the log. The breakdown is there when asked. |
| The log is per fight leader, runtime only, cleared at the next fight, 40 rounds. | Like the stream: reporting only, never saved, so a copyover costs the explanations and not the fight. It outlives the fight so `why` works after the summary. |
| Only weapon rounds are explained, not spells or abilities. | Spell and ability lines already say what they did; strikes are where the hidden roll lives. Left for a later phase if wanted. |
| The web breakdown is built server-side and sent as lines. | One wording for telnet and web; the client only draws it, with `textContent`. |
| Explanations go only to the leader, for the company's own rounds against a foe they can make out; never to a watching ally or for a masked ("?") foe. | The feed's existing privacy rules; an unseen foe's armor and chances would give it away. |
| Enemy armor and the foe's chances to hit and defend are shown. | These are the player's own rolls (Pillars shows them on hover); enemy health is still never sent. |
| "Which rule or sigil mattered": the summary names the sigil in force and counts class abilities used; focus rules and phase 61 orders are not repeated there. | The summary may claim only real, recorded effects. A sigil's own effect is not tracked per blow, so the line says what held, not what it did. Phase 61's "as ordered" log lines already name the rule. |
| "Who never reached their target" counts members who threw strikes and landed none (missed, turned aside, or all stopped by armor). | It answers the question without per-target bookkeeping, and says why. |
