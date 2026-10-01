# Phase 33a execution plan

Owner authorization: the current request instructs implementation of all 33
phases and delegates open decisions to the lead. Start with 33a after the
already-shipped 30g2; no additional design confirmation is required.

Decisions: keep `ask [member] [order]` as the documented safe compatibility
route for present companions and temporary charms. Keep company management
surfaces (`company gear`, `company inventory`, `company meal`) as primary
views/provisioning. Do not introduce a second command parser. Refuse all
requested follower hostility, including outside battle; starting a fight is
`attack [group]`. Retain observation and conversation during battles.
Temporary charms get the same order restrictions without becoming companions.

1. Implement a dependency-safe action policy and tag queued follower orders
   with requester, origin room and stable member key. Revalidate at enqueue
   and execution. Preserve autonomous world NPC and administrative input.
2. Gate battle management before script interception and prevent aid/tame
   from replacing battle actions. Revalidate legacy skill casts at execution;
   refuse taming allied, yielded, fighting and battle-owned targets.
3. Test the real player dispatcher, ask, script wrappers and queued mob
   dispatcher across battle start, movement, ownership change, dismissal,
   death and logout; retain out-of-battle management and conversation.
4. Update company/ask/combat/equipment/consumable/skill help and tutorial
   pointers; test rendering and tutorial links.
5. No durable field changes or migration: provenance is runtime input only,
   dropped with the existing queue at restart/copyover. Existing company,
   user and charm records stay authoritative.
6. Independent full-diff review, fix verified findings, then generate,
   validate, applicable lint, and the complete race suite. Record outcomes,
   commit on the feature branch, merge to master and push origin.
