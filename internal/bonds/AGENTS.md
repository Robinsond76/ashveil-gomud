# Bonds Package Guide

Phase 65. GoMud-free: the value's range and words, the sources and their cooldowns, how far each may push (`Apply`), the camp affinity of two temperaments, what a friendship or rivalry does in a battle (`Guards`, `GuardBelowPct`), and the warning rule. `modules/company/bonds.go` keeps and applies the saved value (`Record.Bonds`, `internal/company/bonds.go`); combat reads it through `company.BondValue` and reports rescues and refusals through `company.BondEvent`, and never imports the module. Decisions: `docs/plans/2026-10-07-pillars-phases.md` ("Phase 65 build decisions").

- A bond is one number per pair of companions (ids, `A < B`), never the leader (loyalty is the leader's measure). Match by companion id, never by name.
- Time together (camp, battle, banter, opinions) stops at +50 and -50; only stepping in goes higher and only a refusal (or a clash between a pair already at -50) goes lower. Every source has a per-pair real-time cooldown in `rules`. Do not add a source without both, or the bond becomes a farm.
- A bond never gives gold, experience or power. The only battle effects are the bond guard (a friend at half health, once a battle, twice for kin) and a rival's refused guard; both log a line and a combat event with status `bond`.
- Add a source: a `Source`, its `rules` cooldown, its limits in `Apply`, its real entry point calling `applyBonds`, the help page's table, and a wiring test through that entry point. Retune a threshold only with `help bonds` and `TestBondsHelp`.
