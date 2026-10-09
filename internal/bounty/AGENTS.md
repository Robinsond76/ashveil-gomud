# Bounty Package Guide

Phase 76. GoMud-free rules for bounty boards: how a board's postings are chosen from the zones' encounter tables for a real-time window, what a bounty pays, and the saved shapes. `modules/bounties` binds it to rooms, the chronicle and the command.

- Real time only. Nothing here reads or advances the world clock.
- Postings are derived, not stored: the same board and window always give the same list (`Post`), so a restart or copyover keeps the board.
- A bounty never scales a foe. It names an existing lair boss or an existing ordinary group in a zone's tables; the zone's band sets the pay, nothing else.
- Proof is the chronicle: a `boss` deed (`mob:<id>`) or `group` deed (`group:<composition id>`) in the target's zone after the bounty was taken.
