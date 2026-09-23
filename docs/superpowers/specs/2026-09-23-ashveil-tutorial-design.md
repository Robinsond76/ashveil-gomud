# Ashveil Tutorial

**Status:** Design direction approved 2026-09-23; written spec awaiting owner
review. Part of the [roadmap](2026-09-23-company-life-onboarding-roadmap.md).

## Goal and existing seams

The shipped tutorial has four rooms and a quest whose steps teach `look`,
items, `status`, `inventory`, `experience`, `conditions`, and attacking a
training dummy. Its scripts restrict accepted commands and send graduates
through a portal to the configured start room. It predates Ashveil's
company, formation, survival, camp, weather, load, and archetype systems.
Keep the useful basic lessons, then make each new lesson an action the player
performs and can inspect afterward.

## Course and demonstrations

| Stage | Player action and observed result |
|---|---|
| 1. Character | Choose one of the five archetypes during creation, inspect the one-time starter kit, and use `status`, `exp`, `inventory`, `conditions`, and the prompt. |
| 2. Company | Speak to a tutorial recruiter, recruit two distinct companions, inspect `company status`, and explain the five-character cap, equipment, and dismissal. Explain how this company differs from a multiplayer player party. Each tutorial candidate can be claimed once per character. |
| 3. Formation | Use `formation` and `formation move` or `swap` to place a front-line defender and a rear-line ranged member. Show all nine cells, rows, columns, reach, and who can intercept. Confirm the grid is tactical and does not represent world geography. |
| 4. Survival | Check hunger, thirst, fatigue, `weather`, `temperature`, `strain`, `inventory`, and `cargo`; eat/drink a supplied tutorial ration; compare a sheltered room with an exposed one and inspect clothing, load, and weather effects. Show the difference between local movement and real-time expedition travel without advancing world time. |
| 5. Camp | Establish a camp in an eligible room, light its fire, rest, inspect Rested, and optionally enable automatic whetstone sharpening. Show that one stone prepares all eligible company blades and is consumed once. Contrast inn Well Rested with camp Rested. |
| 6. Combat | Fight safe practice enemies with a visible 3×3 enemy formation. First demonstrate an unreachable target and front-row interception, then move a member and try again. Show legal automatic targeting and target reassignment when an enemy falls. Explain HP, conditions, and the difference between damage and sharpened-strike consumption. |
| 7. Alignment | Once Phase 21 ships, inspect a recruit's alignment and company average, explain drift, loyalty, recruitment gates, and settlement standing. Use a preview or harmless scripted choice so the lesson cannot permanently trap the player. |
| 8. Departure | Review the company, load, supplies, and route status, then exit to the configured Ashveil start room with a once-only completion reward. |

The practice fight has no repeatable XP, gold, item, or chemistry farm. Its
enemy and any injuries reset safely for retries. Climate and weather lessons
use room content and observation rather than mutating the shared clock or
zone weather for one player. The camp lesson uses a real rest session; any
tutorial-specific duration override is content/config scoped and cannot
change ordinary camps. A player can inspect instructions with `tutorial`,
repeat a demonstration, or skip the course; skipping does not grant its
one-time rewards or duplicate a starter kit.

## Progress and accessibility

Store per-character completed steps and one-time claims durably. Progress
survives logout, restart, and copyover; returning players resume at the
current step. Do not rely on room-global JavaScript counters or strict
command allowlists that block chat, help, status, or recovery. Each gate
checks the actual action/result (recruit, formation cell, rest completion,
attack outcome) rather than only matching typed command text. If an NPC or
item is unavailable, offer a reset/retry path without resetting earned
claims. Give clear hints for aliases and target names.

Telnet text and browser UI must expose the same stage, goal, and success
feedback. Screen-reader users receive ordered text for the 3×3 grid and
explicit target legality; visual colour is supplementary. Tutorial content
ships in the default world, with help entries for each demonstrated command.

## Acceptance criteria

- A fresh character can complete or skip the course, resume after each
  stage, and never receive a kit, recruit reward, or graduation reward twice.
- At least two companions can be recruited and repositioned; the practice
  fight shows reach, interception, and automatic targeting through actual
  combat results.
- Food, water, fatigue, weather, temperature, encumbrance, camping, Rested,
  and sharpening are demonstrated with real commands and visible results.
- Alignment instruction is enabled only when Phase 21 behavior exists.
- The same lessons work in Telnet and the browser, including keyboard and
  screen-reader paths; no lesson advances the global clock.
