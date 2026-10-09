# Phase 72a: Looks and life story at character creation (phase spec, 2026-10-07)

Robinson's idea (thread "Character creation", 2026-10-07): a stronger
character creation, much like Mount & Blade Bannerlord. The player chooses
their character's looks, height, weight, age and so on, plus a background
story. Robinson approved the design and its defaults below ("go",
2026-10-07 01:40).

Today `start` (`internal/usercommands/start.go`) asks race, name and
archetype (`start_archetype.go`), then offers the tutorial. Nothing else
describes a player: `look` at another player prints the stock
"They seem thoroughly uninteresting." This phase adds two creation steps,
**looks** and **life story**, after the archetype step and before the
tutorial question.

Standing rules apply: shared world time is never advanced, state survives
restart and copyover, and difficulty comes only from zone level, so a life
story's effects stay small.

## Fit with phase 72 (Backgrounds)

[Phase 72](2026-10-07-pillars-phases.md#72-backgrounds) planned "a
background chosen at character creation ... offered to existing characters
once", with story events (60) and town lines (68) able to require one. This
phase builds the creation half: the **trade** stage of the life story *is*
the background (soldier, scholar, outlaw, acolyte, labourer, noble...), and
it is stored under the id phase 72 checks. Phase 72 shrinks to its event
and town hooks: background-gated choices in story events and town lines,
plus `help backgrounds` gaining the list of what each one opens. Phase 72
therefore depends on 72a, 60 and 68. 72a depends on nothing unbuilt.

## Decisions (Robinson approved the defaults, 2026-10-07)

| Question | Decision |
|---|---|
| Life-story stages | Three: homeland, upbringing, trade |
| Pronouns | Chosen at creation (he, she, they), used in the description |
| Changing looks later | Free at any inn; the life story stays locked |
| Existing characters | Asked once at next login |
| Age | Words only; nobody ages, no stat effect |

## Design

### 1. Looks

Each look is a pick from a short list of **bands** (not sliders, which
don't work in telnet). The bands are data, so the new world can rename or
add them.

| Trait | Example bands |
|---|---|
| Pronouns | he, she, they |
| Age | young (around 18-24), in their prime, middling years, past forty, greying, old |
| Height | short, below middling, middling, tall, very tall |
| Build | gaunt, lean, wiry, sturdy, broad, heavy |
| Face | sharp, weathered, plain, handsome, hard, pockmarked, long |
| Skin | five or six tones, each mapped to a sprite skin ramp (section 4) |
| Eyes | grey, brown, dark, green, pale blue, mismatched |
| Hair | colour (black, brown, auburn, fair, grey, white) and style (cropped, long, braided, shaven, matted, tied back) |
| Voice | low, rasping, soft, clipped, lilting |
| Marks | up to two: burn scar, old sword cut, brand, tattoo, missing finger, broken nose, none |

A race may limit or extend the lists in its own data (an elf's age bands
read differently, for example). Height and build are words only; no
centimetres or kilograms are shown, as the style bible's status-in-words
approach asks.

**Description.** The picks compose into one or two sentences of prose
written into `Character.Description`, the field `look` already shows. For
example: "A tall, rawboned woman past forty, grey-eyed, her hair cropped
close. A burn scar runs across the back of her left hand." The composer is
template-driven (phrases per band, joined by a few sentence patterns) so
the result reads naturally and the wording can be changed in data. The
player can add one free line of their own (up to 160 characters, filtered
like names) after the generated text, and can preview the result before
confirming.

Looks have no stat, combat or carry effect.

### 2. Life story

Three stages, each a numbered choice with a short paragraph of story and a
small effect. Five or six options per stage to start; all are data
(`_datafiles/world/default/lifestory.yaml` or a module data file) with ids,
names, text and effects, so lore can rename them without code.

1. **Homeland**: where you were born (placeholders: river farms, hill
   clans, a walled city, the ash marches, the coast, the forest roads).
2. **Upbringing**: how you grew up (placeholders: farmhand, temple
   foundling, street child, hunter's get, minor house, camp follower).
3. **Trade**: what you did before taking up the road. This is the
   **background** (soldier, scholar, outlaw, acolyte, labourer, noble, plus
   one or two more if the world needs them).

**Effects, kept small.** Each option names one or two stats; the player's
pick gives +1 to one of them (the option may fix which). Across all three
stages the total is +3, no stat above +2 from the life story. The trade
also gives one **keepsake** (a cheap, flavourful starting item: a soldier's
tin badge, a scholar's worn primer) and, where it fits, a level-1
familiarity in one non-combat skill the archetype does not already teach
(for example cooking, foraging or tracking, chosen from existing skills).
Keepsakes have no resale value above a merchant's floor price (economy
rule). No gold, no gear, no combat skill.

**Backstory.** The three paragraphs, with the character's name and
pronouns, form a short backstory shown by `lifestory` (and `lifestory
[member]` for another player at a glance), in the Character panel, and
recorded as the first entry in the company chronicle when phase 63 lands.

### 3. Flow, commands and existing characters

Order of `start`: race, name, archetype, **looks**, **life story**,
tutorial question. Each step uses the existing `prompt` machinery the
archetype step uses, so an answer can be rejected and asked again, and a
"go back" answer restarts the current step. A summary screen at the end
shows the description and backstory with **confirm** or **redo looks** /
**redo story**.

- `appearance` shows your description. `appearance edit` re-runs the looks
  step, only in a room tagged as an inn, free of charge. Pronouns are part
  of looks.
- `lifestory` shows your backstory and the effects it gave.
- **Existing characters** with no looks get the looks and life-story steps
  once, at their next login after the update, with a "skip for now"
  answer. Skipping leaves the stock description and marks the offer as
  made; `appearance edit` at an inn still works afterwards, and
  `lifestory choose` is allowed once while no life story is set.
- Pronouns are stored on the character for description text only.
  Rewriting every engine message to use pronouns is out of scope; later
  phases (status in words, story events) can read the field.

### 4. Web client

The web client gets a **creation panel** that opens while a creation step
is pending: lists of bands as buttons, a live preview of the composed
description, and the sprite preview (section 5). It sends the same
answers the telnet prompt takes, so the server stays the one source of
truth and both clients run the same steps. The server announces the
current step and its options over GMCP (`Char.Creation`, with step id,
options and current picks). The panel works at phone width.

Telnet gets the numbered lists and the same summary screen.

### 5. What shows on sprites

Art is code-generated (`scripts/sprites/`), one sheet per class. Redrawing
every class per height, build, face or mark is not realistic, so those
stay in words. What can show on the map and battle sprites:

- **Skin tone.** The generator already draws faces and bare hands in the
  `skin` ramp. The client recolours that ramp on the player's own sheets
  (draw once to an offscreen canvas per class and look, then cache) to the
  chosen tone's ramp.
- **Hair colour.** Hair is drawn today in shared ramps (`leather`,
  `charcoal`, `bone`) that gear also uses, so recolouring them would tint
  armour. The build gives player-class figures a dedicated hair ramp in
  the generator (adding three palette entries or reusing an unused ramp,
  with the palette test updated), then the client recolours it like skin.

GMCP `Char.Info` and `Party.Vitals` carry the two look keys beside the
existing `lineage` and `classid`, so other players' sprites show them too.
Companions keep their fixed art; giving recruited companions generated
looks is a follow-up (see Out of scope).

## Out of scope

- Face sliders, a 3D model or portrait art.
- Ageing over time (it would tie to global game time).
- Pronoun-aware rewriting of engine messages.
- Generated looks and backstories for recruited companions (a natural
  follow-up that reuses this composer; it pairs with the "companion
  personality display" follow-up).
- Background-gated story choices and town lines (phase 72).
- Final world names for homelands, upbringings and trades (the new world's
  lore replaces the placeholders in data).

## Acceptance

- A new character created through the real `start` command in telnet and
  through the web client panel goes race, name, archetype, looks, life
  story, tutorial; answers can be redone; the summary confirms.
- `look [player]` shows the composed description; the free line is
  filtered and length-capped.
- Life-story effects apply once: +3 stats in total with no stat above +2,
  the keepsake is in the pack, the skill familiarity is granted, and none
  apply twice on redo, relog, restart or copyover.
- The trade is stored as the background id phase 72 will read, with a
  test helper other phases can use.
- `appearance edit` works in an inn room and is refused elsewhere; the
  life story cannot be changed once set.
- An existing character is offered the steps once at login; skipping is
  remembered across restart.
- Skin and hair colour show on the player's map and battle sprites and on
  other players' sprites; `make sprites` stays byte-for-byte reproducible
  and the sprite tests pass.
- Looks, life stories and pronouns persist across restart and copyover.
- **Help:** `help appearance` and `help lifestory` (`backgrounds` is an
  alias of `lifestory`; phase 72 adds what a trade opens), indexed in
  `keywords.yaml` with aliases (`looks`, `description`, `backstory`,
  `background`), linked from `help adventure`; the first lesson of the tutorial points
  to `appearance` and `lifestory`; tests render each page through `help`
  and `TestTutorialHelpPointersExist` passes.

## Plan

1. **Data and storage.** Look bands and life-story options in data files;
   character fields for looks, pronouns, life-story picks and background id;
   load-time validation (every option has text, effects within limits).
2. **Description composer.** Phrase tables and sentence patterns; tests
   for every band combination producing a well-formed sentence.
3. **Creation steps.** Looks and life-story steps in `start`, summary and
   redo, effects applied once; the login offer for existing characters.
4. **Commands.** `appearance`, `appearance edit` (inn only), `lifestory`.
5. **GMCP and web panel.** `Char.Creation`; the creation panel with live
   preview at desktop and phone width; look keys in `Char.Info` and
   `Party.Vitals`.
6. **Sprites.** Dedicated hair ramp in the generator; client recolour of
   skin and hair with caching; `make sprites` and sprite tests.
7. **Help and tutorial.** The three pages, keywords, aliases, hub links and
   tutorial pointer, with render tests.
8. **Gates.** Integration tests through the real `start`, login and command
   entry points; full checks once before the PR.

Size: one build phase, medium (comparable to camp music). Steps 5 and 6
are the largest; if the build runs long, sprite recolouring (step 6) can
split into 72a2 without blocking the rest.

## Build decisions (2026-10-07, full autonomy)

Each is the builder's call with its reason.

1. **Stat bonus is derived, not stored.** The picks hold the stat each
   stage's +1 went to; `Character.StatMod` adds them on read. Reason: a
   stored bonus would double-apply on a reload and breaks the stat-point
   catch-up accounting (training-based bonuses were tried and rejected).
2. **Caps:** +2 per stat, +3 in all (one per stage). Reason: a start, never a
   lasting edge; the player picks which of an option's two stats gets it.
3. **`backgrounds` is an alias of `lifestory`, not a page.** Reason: the
   trades are placeholder data that the replacement world renames; a page
   listing them would go stale, and `lifestory` already names each pick.
4. **Hub link is `help adventure`.** Reason: `help character` is the
   `character` command's page (re-create or switch characters).
5. **Tutorial pointer is a hint in the first lesson ("Your character").**
   Reason: it already covers `status`, `inventory`, and what you chose.
6. **Hair is its own palette ramp** (the palette grows from 64 to 67
   colours, the spec's cap is raised to match). Reason: skin and hair need
   separate recolour targets and no existing ramp was unused.
7. **Recolour is client-side** (`sprite-tint.js`, cached per look), so the
   PNGs stay one reproducible set and `make sprites` is byte for byte.
   Reason: per-look sprite sets would multiply the art by every colour.
8. **The panel answers with telnet's own input** (option number, `back`,
   `skip`, a typed line). Reason: the server stays the one source of truth
   and both clients share one state machine and one set of tests.
9. **The legacy offer follows phase 56:** a MiscData key
   (`creation-offered`), no date cutoff, asked once whether written or
   skipped. `creation` (and `lifestory choose`) ask again on request.
10. **Placeholder world data:** six homelands, six upbringings, six trades,
    in `lifestory.yaml`; looks in `looks.yaml`; keepsakes are items
    30400-30405. All easily renamed; the new world replaces them.
