# Potential Phase 28c: Pronouns and Ordinals in Combat Text

Part of the [combat presentation roadmap](2026-09-26-combat-presentation-roadmap.md).
Status: owner-approved direction (2026-09-26); needs a design pass and plan.

## Problem

- **Pronouns:** Combat text says "swings their cudgel" for everyone, because
  mobs and companions have no stored pronoun. The owner wants pronouns for
  people and "it" for beasts.
- **Duplicate names:** Two enemies with the same name ("bandit cutthroat")
  read identically, so the log can't say which one struck.

## Scope

### Pronouns

- **Where they're stored:** an optional `pronouns` field on mob templates
  (`he`, `she`, `they`, `it`).
- **Defaults when unset:** from the race (a beast race defaults to `it`,
  a humanoid race to `they`).
  - Open: whether the race files already carry a usable beast/humanoid
    marker, or need one.
- **Players:** a player's pronouns come from the character. How they're
  chosen (a `start` step, or a setting) is an open decision. Until then,
  players are "they" in other people's text; your own lines use "you".
- **New message tokens:**
  - `{sourcehis}`, `{targethis}` (his/her/their/its);
  - `{sourcehim}`, `{targethim}`;
  - `{sourcehe}`, `{targethe}`, capitalised when a line starts with one.
  - Exact token names are settled in the design pass.
- **Content:** companion and recruit templates get pronouns from their
  descriptions: Tamsin, Ysolde, Sister Maren, and Old Wenna she; Oswin,
  Garrick, and Corvin he. Beasts in the shipped world are `it`.

### Ordinals

- **When they appear:** when two or more living or fallen members of the
  same enemy party share a name, combat text names them "the first
  cutthroat", "the second cutthroat", and so on. A name with no duplicate
  is unchanged.
- **Stable for the whole fight:** the order is set by instance id (spawn
  order) when the fight starts. If the first cutthroat dies, the survivor is
  still "the second cutthroat".
- **Short form:** the head noun of the name ("cutthroat" from "bandit
  cutthroat").
  - Open: whether names need an explicit short form in the template.
- **Scope:** only inside a fight with an enemy party (11a). Outside combat,
  `look` keeps its grouped listing.
- **Targeting:** whether players can type `attack second cutthroat` is an
  open decision; the existing `2.cutthroat`-style selection (if any) is
  checked in the design pass.
- **Battle panel:** the Phase 29 grid uses the same names ("cutthr. 1",
  "cutthr. 2").

## Acceptance criteria

- A fight with two same-named enemies:
  - names them "the first …" and "the second …" consistently in every line;
  - keeps the survivor's ordinal after one dies.
- A companion with `pronouns: she` renders "her" in the attacker and room
  lines. A beast renders "its". A mob with no pronoun on a humanoid race
  renders "their".
- Pronoun and ordinal state survives copyover for a fight in progress, or
  the design says why it needn't (for example, recomputed from the party on
  the next round).
