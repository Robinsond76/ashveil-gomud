# Phase 56: recipe discovery (2026-10-06)

Spec: phase 56 in [the Outward phases](2026-10-06-outward-survival-phases.md).
Built under Robinson's full-autonomy rule; each decision has its reason.

## What shipped

- `cook [ingredient]...` at a hearth or the leader's own lit campfire (also
  `camp cook [ingredient]...`) cooks exactly the named mix from the pack and
  company cargo (names: short name, any word of the name, or its start;
  `2 meat` counts; at most 4 ingredients).
- A mix that matches a recipe cooks it, and the first time **learns** it into
  the leader's recipe book (`recipes` lists it). A mix that matches nothing
  makes a **makeshift meal** (item 30062, 25 Hunger, no buff) and spends the
  ingredients. A mix whose dish needs more Cooking than the cook has is refused
  with nothing spent and nothing learned.
- Bare `camp cook`, `use hearth` and the rest's cook duty cook only **learned**
  dishes. Dishes needing Cooking 1 or less (seared game meat, grilled fish) are
  common knowledge, so a new player still cooks on the first night.
- Recipe pages (`recipe: <dish>` on a usable item; 30063 thyme-roasted game,
  30064 hunter's stew) teach a dish when used, and are used up.
- `look hearth`, the manual cooking capability and the Camp tab list only the
  dishes you know.

## Decisions and reasons

- **The book is the leader's (character MiscData `recipebook`), not the
  camp's.** It follows the character through restarts and is read in
  `internal/usercommands` (`use`, `look`) and in the camping module without a
  new seam. Companions cooking at camp use the leader's book: the book is the
  company's knowledge, and the best cook's rank still gates the dish.
- **Recipes stay YAML.** Hearth containers and `CampRecipes` are unchanged;
  `internal/cookbook` adds only the book, name matching and the "common
  knowledge" rule (`MinLevel <= 1`), so no per-recipe flag is needed and the
  Waymark hearth's data is untouched.
- **A miss is plain food, never a profit.** It is a single fixed meal (25 Hunger,
  below seared meat's 40) and is never an ingredient again, so an experiment
  costs more than it returns. Merchants never buy it (`IsSpecialForSale`), nor
  a recipe page.
- **Only raw game, fish and herbs go in the pot** (`cookbook.IsIngredient`:
  botanicals, provision goods, plain nutritious food). Cooked meals, drinks,
  tools and weapons are refused before anything is spent.
- **A learned dish is recorded only when the dish was actually made.** If the
  save fails or the load check refuses, nothing is spent and nothing learned.
- **No world distribution of recipe pages yet.** The world is temporary
  (Robinson, 2026-10-06), so the pages exist as shipped items and tests, and
  the replacement world places them (trainers, loot).
- **Remedies (phase 55) share the book.** Phase 55 merged while this was
  built, so remedies are discovered the same way rather than by a second system:
  the book holds `r:<ailment>` entries beside dish ids. Thyme tea (chill) is
  common knowledge (`Common` on its ailment spec); gut-ache and fever are
  learned with `camp prepare remedy with [herb]...`, which tries exactly that
  mix on the ailing company. The right mix for someone who is ill cures them
  and teaches the remedy; any other mix, or the right one when nobody needs it,
  spends its herbs (a guess always costs, so the mixes cannot be probed for
  free). A plain `camp prepare remedy` makes only known remedies and names the
  ones it skipped; `camp supplies`, `recipes` and the Camp tab list known ones.

## Help

New `help recipes` (aliases recipe, recipe book, recipe pages, experiment),
indexed under road beside cooking, linked from `help cooking`, `help camp` and
`help camp duties`; `help cooking` and the camp lesson in the tutorial point at it.
