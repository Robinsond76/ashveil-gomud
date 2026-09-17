# Company Module Guide

`MobTemplateID` and companion `ID` are durable company ownership; `InstanceId` is runtime-only. `PlayerSpawn` restores every companion for both normal login and copyover. Normal GoMud charm movement remains the only follower mechanism.

The company roster is one leader plus up to `MaxCompanions` (default 4) companions, for a five-member cap. Durable records are written immediately when commands change company state.

Player-facing management: `company summon <mob-id-or-name>`, `company status`, `company dismiss <member|all>`, and `formation` / `formation move <member> <row> <col>` / `formation swap <a> <b>` / `formation clear <member>`. Formation rows and columns are 1-based to players; row 1 is the front row.

Legacy single-`companion` records migrate to `companions[0]` with ID 1 on load. Formation cells for dismissed companions are pruned automatically. Travel, rest, and formation must never change global game time.
