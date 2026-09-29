# Phase 29d: Pronouns and Stable Enemy Labels

Status: owner approved the detailed design and plan in the current task;
implementation in progress. Detailed design drafted 2026-09-28. The presentation direction was approved 2026-09-26. The
owner requested the specification and plan together, with GPT-5.6 Terra at
medium reasoning for implementation. Player pronoun selection is deferred
by the owner's explicit decision in this task.

Part of the [combat roadmap](2026-09-26-combat-presentation-roadmap.md).
Implementation: [29d plan](../plans/2026-09-28-phase-29d-pronouns-ordinals.md).
The owner has chosen to work on 29d before the still-pending 32g dock.

## Intent and scope

Combat should identify who acts without making the reader guess which of
several identical enemies the text means. An authored companion uses the
pronouns in their description; beasts use it. Two bandit cutthroats read
as “the first cutthroat” and “the second cutthroat,” and the latter keeps
that name after the first falls.

This phase changes presentation only. Attack selection remains group-only
under 32c. Players remain they in third-person text and you in their own
lines. No creation step, player setting, NPC dialogue rewrite, battle dock,
pain reactions, or tactical changes are included.

## Prior art and actual integration points

- `internal/combat/combat.go` owns `buildCombatMessages`, weapon wait
  messages, and all four attack entry points. It already applies articles
  and sentence capitalization. Combat calculation receives character copies.
- `internal/items/itemspec.go` defines the message tokens. All eight shipped
  weapon message pools are in `_datafiles/world/default/combat-messages/`.
  Some claws text hard-codes its; that is wrong for humanoids with claws.
- `internal/races/races.go` has no pronoun or reliable beast marker.
  `groupnoun`, size, and `tameable` are unsuitable proxies (the reptilian
  humanoid and reptile animal races are distinct).
- Mob templates embed `characters.Character` under `character:`. Adding
  `character.pronouns` follows the existing YAML shape without duplicating
  identity fields on `mobs.Mob`.
- `modules/company/runtime.go` restores authored companions from templates,
  then applies generated names/descriptions from durable company identity.
  Generated names have no authored pronoun metadata; template gender must
  not leak into an unrelated recruit.
- `internal/battle` retains enemy instance IDs for the battle, including
  fallen enemies. `beginBattle` and growth in `internal/hooks/combat_battle.go`
  precede attack resolution and event-stream creation/growth.
- `ScriptActor.GetCombatName` in `internal/scripting/actor_func.go` is the
  existing name surface used by shipped combat spells. Ordinary
  `GetCharacterName` and room look use the original name.
- `internal/hooks/combat_narration.go`, engagement/formation hooks, and
  `internal/mobcommands/suicide.go` construct additional enemy name lines.
  `combatstream.Ref.Name` supplies events and summary names.
- Battle and combatstream state are explicitly runtime-only. Mob instances
  and aggro are rebuilt at restart/copyover; `main.go` registers no mob or
  battle copyover contributor. Do not describe those restarts as retaining
  the same fight.

## Approach

Use explicit character pronouns with a data-driven race fallback, and
freeze enemy naming snapshots in the existing battle registry.

Alternatives considered: recalculating labels from living mobs is smaller
but renumbers survivors; adding a separate persisted narration registry
would duplicate battle ownership and preserve IDs whose mobs no longer
exist after restart. Neither fits the existing lifecycle.

## Pronouns

Add `Character.Pronouns string` with `yaml:"pronouns,omitempty"` and
`Race.DefaultPronouns string` with `yaml:"defaultpronouns,omitempty"`.
Allowed values are he, she, they, and it. Trim whitespace and normalize case.
An empty character value inherits its current effective race's default;
an empty/missing race default resolves to they. Unknown character values
fall back safely through the race; invalid nonempty race defaults are
reported by race validation. Do not rewrite old character saves merely to
materialize defaults. Rendering does not mutate character state.

A user actor always resolves to they in this phase, even when polymorphed
into an animal. Only mob actors use character/race pronoun selection.
Explicit NPC identity wins over a temporary race change.

| Value | Subject | Object | Possessive adjective |
|---|---|---|---|
| he | he | him | his |
| she | she | her | her |
| they | they | them | their |
| it | it | it | its |

Expose a small `characters.PronounForms` value,
`characters.PronounFormsFor(value string) PronounForms` (unknown values
return they), and `(*Character).CombatPronouns() PronounForms` for mob
resolution. Callers that know an actor is a player use PronounFormsFor("they").

Add six item tokens: `{sourcehe}`, `{sourcehim}`, `{sourcehis}` and the
matching `{targethe}`, `{targethim}`, `{targethis}`. These names designate
subject, object, and possessive forms, irrespective of the selected set.
Preserve you/your in first- and second-person variants. Use the existing
sentence capitalization after token replacement. Do not introduce subject
pronouns before a conjugated verb that only fits one set (he swings versus
they swing); retain named subjects where that avoids new verb machinery.

Expose `ScriptActor.GetCombatPronoun(form string) string`, with forms
subject, object, and possessive; an unknown form returns an empty string.
It returns they forms for users and character/race forms for mobs.
Update hard-coded actor pronouns in the combat spell scripts, including
Minor Heal's chanting and self-heal room lines. Leave literal pronouns
referring to wounds, light, or a whole group untouched.

Shipped race defaults: it for ghostly spirit (0), undead (6), insect (7),
eldritch horror (9), rodent (10), canine (11), fungus (12), tree (13),
giant spider (14), golem (16), monkey (17), lagomorph (18), dummy (19),
orb (20), and reptile (21). Human (1), elf (2), troll (4), goblin (5),
reptilian (8), and faerie (15) retain the they fallback. Individual
person-like undead or spirits can use an explicit template override.

Authored overrides: Tamsin (61), Ysolde (64), Sister Maren (65), and Old
Wenna (66) use she; Brother Oswin (62), Garrick (63), and Corvin (69) use
he. Generated companions (nonempty saved identity.Name) are explicitly
set to they whenever spawned, overriding the authored template's pronouns.
This is deterministic after login, resurrection, restart, and copyover;
no new company persistence field or guessed gender from names is needed.

## Stable enemy labels

Add optional `Character.CombatNoun string` (`yaml:"combatnoun,omitempty"`)
for authored short forms. Otherwise strip a leading a/an/the and use the
last whitespace-delimited word of the plain name (bandit cutthroat becomes
cutthroat). Empty names fall back to creature; empty optional nouns use
the derived form. Names are plain text; never infer nouns from ANSI markup.

Extend `battle.Battle` with `EnemyNames map[int]EnemyName`. Each immutable
snapshot records InstanceId, BaseName, Noun, and DisplayName. Clone this map
when returning a battle so callers cannot edit registry state.

At `beginBattle`, collect the full initial party's names, assign snapshots
before opening its event-stream fight, and sort each duplicate cohort by
ascending instance ID. Duplicate cohorts are names equal after trimming
and case folding, not every enemy with the same last word. A unique name
stays exactly as authored. A duplicate is first/second/etc. plus its noun;
the ordinary article helper produces “the first cutthroat.” Use words
first through tenth, then numeric ordinals (11th, 12th, 13th, 21st, etc.).
The existing mobparty.Ordinal helper has a numeric -th fallback for every
number; do not accidentally use 21th here. Keep the correct suffix helper
local to battle names rather than altering group naming in this phase.

If different base-name cohorts would yield the same noun in one battle,
use their stripped full base names as nouns from the start (first bandit
cutthroat versus first goblin cutthroat). An authored combatnoun can
supply a better distinction. Case variants use the lowest-ID member's
spelling for their shared noun.

Growth appends snapshots without changing any existing DisplayName.
If a previously unique cutthroat gains a same-named newcomer, the original
keeps bandit cutthroat and the newcomer is second cutthroat. The name
snapshot also reserves every fallen member's ordinal. Newcomers must not
collide with an already issued label; extend the newcomer to its full base
noun when a late-arriving different cohort creates a short-noun collision.
No alive-only recalculation and no reuse of ordinals within a battle.

When another player begins or grows a battle against the same enemies in
the same room, inherit the existing snapshots, including fallen members,
from the overlapping active battle. Merge additions into every overlapping
battle under the existing battle mutex so all viewers see the same labels.
Never perform mob, room, company, or stream lookups while holding that mutex:
collect input snapshots before entering it. Enemy IDs are runtime identity;
company membership, formation, group matching, and aggro keep their IDs.

Proposed battle interfaces in `internal/battle/names.go`:

```go
type EnemyName struct {
  InstanceId int
  BaseName string
  Noun string
  DisplayName string
}
func AssignEnemyNames(userId int, members []EnemyName)
func EnemyDisplayName(instanceId int, fallback string) string
```

`AssignEnemyNames` is called after Begin and each Grow, before the
corresponding stream Open/Grow. `EnemyDisplayName` returns a stored label
or the supplied original name, including when no battle exists. End,
Forget, Retain, and Reset remove labels with their battles. If another
player still fights the group, that player's snapshots remain available.

## Rendering and event flow

- Weapon attack wrappers use local character copies whose Name is the
  battle label for mob actors. Do not change the live Character.Name;
  health changes, equipment edges, XP credit, and sounds still act on the
  original objects. Wait/aim calls in DoCombat use the same local copies.
- In scripting, change only GetCombatName to use the battle label for a
  mob. GetCharacterName, room look, corpse/loot names, and searches retain
  their current names and behavior.
- Use the label in target changes, interception/shield-break messages,
  blocked flee notices, fizzle lines, in-round death/beaten notices, and
  suicide fallback notices while the battle is active.
- `mobRef` uses the label, so FightInfo, events, and the summary agree with
  the transcript. A vanished actor's event ref may retrieve a retained
  snapshot rather than losing its already assigned name. Keep Name plain;
  ANSI and articles belong to rendering. Do not change Ref.Key identity.
- Finish death text and summary construction before discarding the last
  battle's naming snapshots. Stream enemy snapshots must never be replaced
  by an unlabelled live ref during growth.
- No GMCP battle panel is introduced. Later 32g can consume FightInfo/Refs
  rather than deriving its own numbers.

## Persistence and multiplayer constraints

Never advance global game time. No new goroutines, timers, or database.
Pronoun and noun configuration survives through existing character/template
serialization; authored companion restoration reloads it, and generated
restoration reapplies the neutral override. Naming snapshots are runtime
only because battles, mob instances, and aggro themselves are runtime only.
After restart/copyover, newly rebuilt battles get new labels; this is not
promised to retain the prior battle's ordering or summary. Test this reset
contract, alongside durable data round-trips. Do not add a contributor for
labels detached from their mobs.

## Player help and tutorial

Update `help narration` with pronoun examples and the fixed enemy-label
rule, including restart starting a fresh battle. Add help aliases pronouns
and ordinals to narration; keep narration indexed under combat. Update
`help combat`, `help targeting`, and `help battle-summary` to explain the
labels where appropriate and reinforce group-only attack selection.
The Combat tutorial hint explains “first/second straw footman” and points
to help narration. No new help page or command is necessary because this
extends the existing narration mechanic; there is no player pronoun setting.

## Acceptance criteria

1. All six tokens resolve in same-room and separate-room variants, with
   he/she/they/it correct and the user's own you/your unchanged.
2. The authored NPC overrides and every listed race default load from the
   shipped world; invalid/missing pronouns fail safely. Player third-person
   remains they, including polymorph.
3. A generated hire uses they before and after snapshot/save/restore and
   resurrection; an authored Tamsin uses her in real attack/spell output.
4. Two same-named enemies receive ordered labels before the first attack.
   The survivor remains second after first dies or is removed; late arrivals
   reserve old ordinals and cannot change old labels.
5. Distinct name cohorts, explicit nouns, case variants, missing names,
   ordinals beyond ten, and overlapping multiplayer battles obey the above
   naming rules. One player's exit does not erase another's labels.
6. Real DoCombat integration covers player-to-mob, mob-to-player, and
   companion-to-enemy attacks, wait/aim, target changes, death/beaten,
   shield/interception, flee blocking, scripted casting, events, and summary.
   An untracked/PvP fight and look/group selection remain unchanged.
7. Existing battle reset followed by a fresh battle recomputes labels;
   no stale label crosses instance lifetimes. Character/race YAML and
   generated companion restoration prove the durable pronoun contract.
8. Help narration, combat, targeting, and battle-summary render through
   help; the pronouns/ordinals aliases resolve and
   TestTutorialHelpPointersExist passes. No help claims player selection.
9. A seeded before/after fight preserves damage, crits, target IDs, mana,
   rewards, and battle outcome, changing only narration/Ref.Name.
10. The lead independently reviews the full diff, verifies review findings,
    runs make generate, make validate, and go test -race ./... once after
    fixes, and records the outcome before any merge.
