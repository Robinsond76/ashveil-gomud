# Phase 72: Backgrounds (decisions, 2026-10-07)

Phase 72a gave every character a life story (homeland, upbringing, trade;
the trade is the background). This phase fills the hooks phases 60 and 68
left: story-event choices and town lines can now require a life-story tag.

## What shipped

- `lifestory.Tags(picks)` names each stage pick as a member tag:
  `homeland-<id>`, `upbringing-<id>`, `trade-<id>` (for example
  `trade-soldier`, `upbringing-hunters-get`). Lowercase and dashed, so they
  satisfy both story events' `require: {tag: ...}` and town lines'
  `member_tag:`.
- `modules/storyevents/backgrounds.go` registers the tag source
  (`storyevents.RegisterTagSource`). Story events and `townsfolk` both read it
  through `storyevents.TagsFor`.
- Town lines: a deed line with `member_tag` set beats a plain line of the
  same kind while the listener has not heard it lately (within their last 12
  tellings); after that it joins the plain pool, so it is not said of every
  deed of its kind. A specific `ref` line still beats both.
- Test content only (the world is temporary): one background choice in each
  of the three test scenes (gorge: `upbringing-hunters-get`; deserter:
  `trade-soldier`; shrine: `trade-acolyte`), and two town lines in
  `modules/townsfolk/lines/test-lines.yaml` (`trade-soldier` on a boss deed,
  `homeland-hill-clans` on a relic).
- Help: `help lifestory` (alias `backgrounds`) gains "What it opens", and
  `help events`, `help townsfolk` and the tutorial's scenes and towns hints
  point at it.

## Decisions

1. **Leader-only.** Only the leader has a life story, so only the leader
   carries tags; companions return none. Reason: 72a gave companions no
   life story, and a generated one would be invented content the new world
   replaces. A background choice therefore always names the leader.
2. **Tag naming `stage-id`.** Reason: ids are unique only within a stage, and
   the stage prefix keeps `trade-soldier` and `upbringing-soldier` apart.
   Tags are lowercase, matched case-insensitively.
3. **No stat or combat effect.** 72a caps the life story at +3 stats; this
   phase adds no more power, only options and lines.
4. **`help backgrounds` stays an alias of `help lifestory`** (72a ruling): the
   trades are placeholder data, so a page listing what each opens would go
   stale. The page says what kinds of things a background opens; the choice
   itself names it in play (`closed: needs a leader who was a soldier`).
5. **The choice note names the life story (review).** An open choice a life
   story opened reads `(Aldous, life story: a soldier)` in the terminal and
   the web modal (`because` in the `Event` payload), so a player sees their
   past mattered; a closed one shows its hint. Only the leader's note says
   "life story". A tag with no hint reads `life story: a soldier`, not the
   raw tag. The Chronicle tab needs nothing: a background town line names
   the background in its own words.

## Acceptance

- At least one background choice in each test event: done, with real-trigger
  tests (`TestALifeStoryOpensBackgroundChoicesInTheRealScenes`,
  `TestABackgroundChoiceIsClosedWithoutTheBackground`).
- A town line that needs one: done (`TestAShippedLineSpeaksToTheLeadersBackground`,
  `TestALineForAMembersBackgroundBeatsAPlainOne`).
- `help backgrounds` explains what backgrounds open: done (help test).
