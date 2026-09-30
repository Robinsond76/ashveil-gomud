# Strategy Module Guide

Phase 32d battle strategies. The registry (user id -> member key -> the role and target rule that differ from that character's default) lives in the plugin file `strategy`, saved on every change (rolled back if the save fails) and on the plugin save callback, so it survives restart and copyover. `AutoSpells` in `files/data-overlays/config.yaml` lists the only spells ever cast automatically, each with its use.

- The engine reads strategies only through `internal/strategy` (`For`, `AutoSpells`); engine code never imports this module.
- Member keys are the company's formation keys: `leader` for the player, `companion:<id>` for a companion. The command prunes keys for companions no longer on the record; `UserPurged` drops the user.
- `m.mu` is a leaf lock: never call the engine (users, mobs, company) while holding it.
- The command reads company members through `company.CompanyMembers` (game loop only). Changes are refused in a battle (`usercommands.InBattle`); reading is always allowed.
- Tests replace `m.env` and `m.store`; they never write into `_datafiles`.
- Phase 30c: the registry also holds each player's `Tactics` (company focus and healing threshold; only values that differ from the defaults), saved and rolled back like strategies (`SetTactics`), dropped on purge, and served to the engine as `internal/strategy`'s `TacticsProvider`. The `company tactics` command lives in `modules/company` and writes through `strategy.SaveTactics`.
- Phase 30c2: `strategy <who> guard [<other>]` (also `guardian`) makes a guardian, with `Strategy.Ward` the other's member key (blank: the most hurt). Only a guardian keeps a ward (another role, `default`, or a load with any other role drops it); a ward naming a member no longer on the record is pruned. After `target`, only a rule is read. `strategy` warns when a guardian and its set ward stand more than one column apart (`formationcombat.InLateralRange`; unplaced fails open), reading cells from `company.FormationFor`.
