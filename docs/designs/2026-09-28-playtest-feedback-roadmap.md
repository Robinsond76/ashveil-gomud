# Phase 32 Play-Test Decisions

The 2026-09-28 play-test roadmap has been implemented (32a–32h, including
32a2 and 32g2). Project Status retains a review note for 32b and the known
limitations. This document preserves owner decisions, not a queue of new work.

## Decisions still useful to contributors

- The company carries cargo collectively; all living members share one
  capacity derived from members, packs, horses, and saddles.
- Gear, packs, and cargo appear together in `company inventory`.
- Each player has their own changing recruit notice with generated names.
- Tutorial replay uses a temporary level-1 copy. Leaving returns to the real
  character unchanged and transfers no rewards or inventory.
- Players initiate combat against a named group, not an individual member.
  Strategies choose targets and actions within formation constraints.
- Combat plays automatically; later 30c1 added the limited company-focus
  change. Use current help and Project Status for allowed commands.
- Every participating company member receives the full experience award.
- The browser dock collects company management, communication, and combat;
  32g2's battle view replaced the separate Phase 31 window.
- Character deletion confirms identity and purges every module's saved state.

Detailed shipped designs remain alongside this file. The obsolete diagnosis
of pre-Phase-32 behavior and completed build order are available in git history
at `792455ea`.
