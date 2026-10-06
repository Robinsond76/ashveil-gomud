# Browser artwork quality roadmap

Status: planned; implementation has not started. Owner requested a phased plan
after authorizing higher-quality browser artwork. Work stays on feature
branches; **do not merge to master until the owner explicitly authorizes it**.

## Outcome

The playable browser map should preserve the approved Direction D proof's
character detail, grounded proportions, muted colors, layered terrain, and
composition. The quality benchmark is the running client at normal desktop
and phone sizes, not an enlarged contact sheet or a generated illustration.

The existing 32x32, 64-color pilot is a compatibility experiment, not the
quality ceiling. Retain the original source renders. Do not upscale the
reduced pilot and call that higher-resolution art.

Visual reference: [approved style proof](../../scripts/sprites/authored/sources/approved-proof.png).
Retained original character/camp renders and crop metadata are in
[`scripts/sprites/authored/sources/`](../../scripts/sprites/authored/sources/).
This roadmap supersedes the old fixed-resolution/palette approach as the
visual target; existing assets remain compatibility fallbacks during rollout.

## Current constraints we will replace

- `scripts/sprites/authored.py` requires procedural dimensions, hard alpha,
  and the master palette. Detailed assets need their own validated contract.
- `sprites.js` already reads source frame sizes from the manifest, but
  `window-map.js` multiplies them directly into display sizes. Source pixels,
  display footprint, world cells, and anchors must become independent.
- The map canvas uses CSS-sized backing dimensions without device-pixel-ratio
  support. Its camera and hit testing currently share those dimensions.
- Companions render at 75% scale; rendering and framing must preserve detail
  without obscuring routes, resources, or other members.
- Terrain and UI remain separate work: better character sheets alone cannot
  reproduce the complete scene's appearance.

## Phase 1 — Prove the quality in the actual browser

Build one representative playable clearing using Warrior, Ranger, Witch,
tent, and animated fire, plus a small set of matching ground/road/tree assets.
Use the existing map window and sprite loader, driven through the existing
browser harness and then the real client. Do not build a separate showcase
renderer that the game cannot use.

- Preserve detailed source colors and transparency. Compare candidate
  exports (initially 64, 96, and 128 pixels per character frame) at intended
  screen sizes and device pixel ratios 1 and 2; add a phone DPR 3 check.
  These are experiments, not a preselected new limit.
- Implement the minimum manifest/display-size/anchor support and high-DPI
  canvas handling needed to run that comparison. Keep coordinates and input
  in CSS pixels; backing pixels are an implementation detail.
- Compare 48, 64, and 96 CSS-pixel character footprints alongside how much
  surrounding world fits. Pick desktop and phone defaults from the result.
- Compare nearest-neighbor and filtered downsampling on the actual exports;
  choose per asset type, preserving deliberate pixel clusters. Do not force
  soft illustration and pixel art through the same sampling rule.
- Capture actual browser stills and movement recordings, including a full
  company, a camp, a road intersection, resources, and day/night.

**Exit:** a viewable client scene and comparison evidence demonstrate that
detail survives delivery, drawing, and normal viewing size. Record chosen
source sizes, displayed sizes, sampling, and framing. Owner visual review is
the checkpoint before scaling asset production; functional tests alone
cannot establish a match to the proof.

## Phase 2 — Make the rendering and asset pipeline production-ready

Harden the Phase 1 implementation in `scripts/sprites/`, `sprites.js`, and
`window-map.js` without changing game rules or map topology.

- Document manifest fields for source frame rectangles, logical display
  dimensions, foot/pivot anchors, animation timing, sampling, and art version.
  Legacy assets retain compatible defaults and loading/failure fallbacks.
- Separate artist-authored masters from reproducible browser exports. Remove
  the old palette and binary-alpha restrictions for the detailed asset tier;
  retain checks for dimensions, frame bounds, transparent edges, anchors,
  animation coverage, and deterministic output.
- Complete DPR-aware resize, browser zoom/monitor changes, pan, camera easing,
  mirroring, hit testing, selection rings, labels, and overlays. Avoid canvas
  resets on every animation frame.
- Replace arbitrary companion shrinkage with the placement and display rules
  established in Phase 1. Preserve party visibility and correct feet ordering.
- Measure transfer bytes, decoded image memory, load time, and frame time.
  Use lazy loading, appropriate atlas grouping, caching/versioning, viewport
  culling, and idle redraw control where measurements justify them.

**Exit:** detailed and legacy assets coexist; regeneration is reproducible;
missing art does not break the map; resize/DPR/input regression tests pass.
Record numerical performance budgets using the Phase 1 test scene and named
test devices. Initial targets are p95 draw work below 16.7 ms on desktop and
33.3 ms on the selected phone, with no ongoing offscreen animation work.
Browser emulation is not a substitute for measuring phone hardware.

## Phase 3 — Build the matching map environment

Expand the clearing into a coherent representative region, using Alderbrook
where its existing roads, water, woodland, buildings, and landmarks fit.

- Produce ground variants, connected paths, shorelines, water, rocks, trees,
  buildings, and camp props in the approved art direction.
- Support larger props independently of their world-cell footprint, with
  defined anchors, depth ordering, and occlusion that keeps occupants legible.
- Implement terrain transitions and deterministic decoration placement;
  eliminate obvious repeated squares and discontinuous paths.
- Establish restrained shadows, firelight, day/night tint, and weather
  treatment that preserve colors and contrast. Decoration must not imply a
  traversable exit or reveal unexplored rooms.

**Exit:** actual browser travel through the region looks coherent at normal
zoom, with no seams, clipping, incorrect layering, or blocked interaction.
Verify fog, boundaries, indoor/outdoor shading, resources, paths, and camps.

## Phase 4 — Expand the character and animation catalogue

Use the approved browser scene and production contract for every batch.

- Inventory current classes, promoted classes, companions, enemies, mounts,
  battle assets, resources, and camp states. Record coverage and fallback for
  each; do not silently treat the three-class pilot as complete.
- Produce base classes first, then promoted classes and other unit families;
  carry the style into combat with its own appropriate screen-size contract.
- Author consistent directional poses, equipment silhouettes, idle/walk
  cycles, and other states required by the existing client. Review frame
  alignment, foot sliding, silhouette changes, and directional equipment.
- Review each batch in motion in both a sparse and crowded real scene before
  proceeding. Re-export from masters whenever sizes change.

**Exit:** the coverage inventory is complete for shipped content; all mapped
states resolve to intentional art and no class unexpectedly drops to the old
style. Atlas, animation, anchor, and visual regression checks pass.

## Phase 5 — Bring the map and game UI into the same visual language

Refine the existing client rather than replacing its framework or controls.

- Align panel colors, borders, typography, spacing, icons, tooltips, selection
  markers, and map controls with the artwork. Preserve theme contrast.
- Allocate enough map space to the chosen display footprints on desktop and
  phone; verify a full company and readable map at normal browser zoom.
- Keep text as accessible HTML where appropriate, with keyboard navigation,
  visible focus, readable labels, and at least 44-pixel phone targets.
- Update `help worldmap`, `help webclient`, and affected battle help for any
  changed controls/settings; maintain keyword links and tutorial pointers.
  Pure asset changes require no invented new player command or tutorial step.

**Exit:** desktop and phone screenshots show a coherent playable interface;
keyboard/touch checks, help rendering, and tutorial-pointer tests pass for
the behavior changed in this phase.

## Phase 6 — Verify, optimize, and prepare release

- Run the map, class-art, battle, and mobile browser checks as applicable;
  extend them for DPR, source/display independence, loading failures,
  movement, selection, resize, fog, and full-company scenes.
- Check the actual client in Chromium, Firefox, and WebKit where available;
  record any unavailable browser/device rather than claiming coverage.
- Measure cold/warm loading and sustained travel/camp animation against the
  recorded budgets. Optimize packaging and drawing before reducing artwork
  quality; make any necessary visual tradeoff explicit and reviewable.
- Run required generation, lint, validation, and full race checks at the
  implementation completion gate. Obtain the independent phase review
  required by repository workflow; fix confirmed findings with regressions.
- Record screenshots, recordings, performance results, outstanding limits,
  help updates, and review outcome in verification docs and project status.

**Exit:** a reviewable release branch/PR with evidence and no unexplained
quality regressions. Deployment and merging remain separate from completion;
master stays unchanged until the owner authorizes merging.

## Sequence and tracking

1. Phase 1 determines the quality target with a small real-client slice.
2. Phase 2 stabilizes the contract before large-scale production.
3. Phase 3 establishes the environment; Phase 4 expands character coverage.
4. Phase 5 unifies the presentation; Phase 6 validates release readiness.

Every phase records its changed files, focused checks, browser evidence, and
remaining gaps. Each implementation phase receives an independent full-diff
review and fixes confirmed findings before it is considered merge-ready;
Phase 6 additionally reviews the integrated release. A generated concept,
passing asset validator, or attractive
static contact sheet alone is never the visual acceptance artifact.
