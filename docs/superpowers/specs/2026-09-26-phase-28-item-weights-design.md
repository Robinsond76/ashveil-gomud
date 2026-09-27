# Phase 28: Item Weights and the Company's Whole Load

The data pass Phase 9 deferred. Encumbrance has been wired since Phase 16:
a company's load band scales route travel time and walking strain, and
26a/26b show the load. But only 4 of the 120 shipped items have a weight,
so almost every company reads "Light" at a few hundred grams, and the
tutorial's `cargo` lesson (27b) teaches a number that doesn't move.

The roadmaps before this (Phases 13–21 and the company-life specs, 22–27)
are complete. The owner said "carry on" (2026-09-26), and this phase takes
the clearest open deferral in `docs/PROJECT_STATUS.md`. Its decisions
apply this design's recommendations and are recorded here.

## Prior-art check

- **`ItemSpec.Weight`** is in grams; zero means unweighted
  (`internal/items/itemspec.go`).
- **Company load** (`modules/encumbrance.CurrentLoad`) is the leader's
  carried and worn items plus the company cargo, against a flat
  `CapacityKg` (200) plus any mount's bonus (Phase 10). Bands start at
  75% of capacity (travel +10%, strain +8%) and reach +50% and +30% at
  full capacity.
- **Companions' gear** (22b) is durable: each companion's worn and carried
  items are on the company record (`MemberState`) and on the live mob when
  it is out. None of it counts toward the load today.
- **Readers of the load:**
  - expedition departure (locked for the journey);
  - walking strain per step;
  - `cargo`, `inventory`, `status`, and the GMCP `Company` load.

  All of them read `Load.TotalGrams()`.
- **Starter kits** (22a) are 5–7 items per archetype; every kit has a
  waterskin.

## Decisions

1. **Every shipped item gets a realistic weight** in grams, set in its
   YAML file:
   - weapons from a 100 g needle to a 20 kg tree trunk;
   - armour by coverage and material (a cotton shirt 250 g, an iron
     shield 6 kg, a steel breastplate 9 kg);
   - food and drink by portion (a full waterskin 1.5 kg);
   - trinkets, keys, and papers from 5 to 80 g;
   - books around 1 kg;
   - gear such as rope, a bedroll, and a lantern at 1–2 kg.

   The four authored weights (wild thyme, wolf hide, raw game meat,
   whetstone) are kept.
2. **Only a service is weightless.** Room rental (item 102) is a service,
   not a thing carried. A test pins that every other shipped item weighs
   something, within a sane range for its type (a ring under 100 g, a
   weapon at most 25 kg, and so on).
3. **The company's whole load counts.** Living companions' worn and
   carried gear joins the load:
   - from the live mob when it is out, since that is what it's carrying
     now;
   - otherwise from its record.

   A fallen companion's gear stays with the body (25b), so it doesn't
   count. A new `company.CompanionGearGrams(leader)` seam, implemented by
   `modules/company`, supplies it. `Load` gains `CompanionGrams`, and
   `TotalGrams` includes it, so every reader above picks it up unchanged.
   The `cargo` status line shows the split.
4. **Capacity and bands stay as they are.** With these weights:
   - a starter kit is about 4–8 kg;
   - a company of five in ordinary gear is around 25–40 kg (Light);
   - loads only start to tell with cargo or heavy armour.

   A test pins that every archetype's starter kit stays well within the
   lightest band on its own. Re-tuning capacity per member is a separate
   question, left for play-testing.
5. **No weight on the dead or on mobs' own inventories:** only a
   company's load is weighed, as before.

## Constraints

- Never advances the world clock. A load is read when a journey starts or a
  step is taken, as before.
- No persisted state changes shape: weights are item data, and a
  companion's gear is already durable.
- Game-loop only: the new seam is read where the load is read today.

## Acceptance criteria

- **Content test:** every shipped item except the service has a positive
  weight within its type's range; the four authored weights are unchanged.
- **Kit test:** each archetype's starter kit weighs 1–12 kg, below the
  first band on its own.
- **Unit tests:**
  - `Load.TotalGrams` includes companions;
  - the company module's `CompanionGearGrams` counts live mobs' and
    recorded gear for living companions only;
  - the encumbrance module adds it;
  - the `cargo` line shows it.
- **Wiring:** through `plugins.Load` with the company and encumbrance
  modules: recruiting a companion in gear raises the company load by that
  gear's weight, and the load band follows `cargo put` of heavy goods.
- `go test -race ./...`, `make generate`, `make validate` pass.

## Review amendments (2026-09-26)

- **Weigh from base data.** `Item.Weight()` reads the item's base spec, so
  an item holding a spec copy (from cargo, an enchantment, a rename, or a
  save from before this phase) is weighed correctly. It is used wherever a
  load is summed.
- **Companions:**
  - a live companion is weighed in place (`runtime.GearGrams`), with no
    copy of its gear;
  - one charmed away by another player carries nothing for this company;
  - one with no record yet weighs its template's gear;
  - an unreadable company weighs nothing.
- **GMCP** `load` carries `companion_g`.
- **The upstream `empty` world's** 14 items are weighed and tested too.
- **The band's response to load** is a unit test (injected gear), not a
  `plugins.Load` wiring test; the wiring test checks that recruits' gear
  reaches the real load.
