# Bounties Module Guide

Phase 76. Bounty boards: a room tagged `bounty-board` lists bounties for lair bosses and named groups from nearby zones. The rules are in `internal/bounty`; this module owns the saved holdings, the `bounty`/`bounties` command and the `Company.Bounties` GMCP message.

- Reserved room tag: `bounty-board`. A board's list is derived from the board room and the real-time window, never stored.
- Proof is the chronicle (`boss` and `group` deeds, matched by `Ref` and `Zone`, numbered after the bounty was taken). The module never counts kills itself.
- Pay is gold only, and a bounty never changes a foe. Do not scale targets to the company.
- State is per leader and registered with `userstate` and the purge listener.
