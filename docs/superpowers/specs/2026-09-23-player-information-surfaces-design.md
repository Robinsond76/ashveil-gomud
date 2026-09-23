# Ashveil Commands, Prompt, and Browser Company View

**Status:** Design direction approved 2026-09-23; written spec awaiting owner
review. Part of the [roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Goal and existing seams

Players need a reliable overview of Ashveil's company and survival state in
Telnet and the browser. `status`, `experience` (`exp`), `inventory`, and
`conditions` currently render engine-specific panels. Separate Ashveil
commands already expose detailed company, formation, survival, weather,
temperature, strain, cargo, mount, camp, and inn information. Prompt
processing has an `OnBuildPrompt` hook for module tokens. The browser Party
window consumes GoMud `Party` GMCP data about human-player parties; it does
not represent the durable Ashveil company.

## Shared read model

Provide a read-only company/player summary assembled from owning providers,
not a second mutable state store. It contains stable member IDs, live/dead
status, archetype, level, HP, hunger, thirst, fatigue, formation cell,
alignment when Phase 21 exists, chemistry tier, sharpened weapons, rest tier,
load band, travel/camp state, and remaining companion rescue allowance.
Missing providers report `unknown` or omit a field; they never invent a
healthy or rested value. The summary uses member IDs as keys, not display
names, so two companions can share a name safely. Read providers may be
registered through narrow interfaces to avoid import cycles between engine
commands and gameplay modules.

## Text commands

| Surface | Required additions |
|---|---|
| `status` | Archetype, alignment, compact hunger/thirst/fatigue bands, company alive/dead count, load band, travel/camp/inn state, current rest tier, and a pointer to `company status`/`formation` for detail. Keep the existing stat training flow. |
| `exp` / `experience` | Current level, XP to next level, training/stat points, and the most recent level loss if death changed progression. Keep `experience chart` behavior. |
| `inventory` | Company load (carried items plus cargo) versus company weight capacity, with the cargo portion separate, food/water supply cues, and whetstone uses. Keep current equipment and filter output. Explain that engine item-count carry capacity is different from Ashveil encumbrance. |
| `conditions` | Group ordinary buffs, Rested/Well Rested, sharpened weapon edges, survival/exposure penalties, and chemistry. Show remaining real duration, strikes, or bond progress in the relevant units. Do not represent permanent chemistry as an expiring buff. |

Detailed commands retain ownership of the full underlying values. Every
summary line names the command that reveals more. Labels and warning bands
come from the same provider data as browser output, so the two clients do
not disagree about whether someone is exhausted or dead.

## Prompt

Add configurable tokens for fatigue band/value, company alive/dead count,
load band, and travel/rest state through `OnBuildPrompt`; unknown tokens
remain literal as today. The default prompt retains HP/MP and adds at most
one compact survival warning and one activity indicator, with plain text
alternatives for screen readers and narrow terminals. A player can opt into
more fields with the existing `prompt`/`fprompt` configuration flow.
Rebuild when the underlying state changes; avoid polling every browser frame.

## Browser client

- Add a `Company` GMCP payload with a full snapshot and lightweight vitals
  updates. Send it on login, recruitment/dismissal, formation change,
  member death/resurrection, needs/health change, rest change, and reconnect.
  A full snapshot replaces stale records after copyover. Never leak another
  player's private company state.
- The current Party window gains a **Company** section for the leader and
  companions, and keeps its separate **Players** section for GoMud human
  parties. The Company section shows a readable 3×3 formation and member
  cards with health, needs, archetype, chemistry, and dead rescue time. A
  compact layout works at narrow viewport widths; an accessible text list
  conveys the same information without relying on cell colour.
- Build DOM text safely from names and other player-controlled strings;
  do not interpolate them as HTML. Reconnect, missing optional fields, and
  partial vitals payloads must not erase a known roster or mislabel a
  companion as a human party member.

## Acceptance criteria

- Text commands, prompt, and browser panel agree on a leader and two
  companions through recruitment, fatigue, camp, combat, death, and revival.
- Human Party and Ashveil Company are visibly distinct even when both are
  active. A company is shown when no human Party exists.
- Inventory reports both item-count and weight limits accurately; the
  tutorial can use the same labels and commands.
- Browser inspection covers desktop, narrow viewport, keyboard, and screen
  reader use; Telnet remains fully usable. GMCP updates survive reconnect
  and copyover without stale members.
