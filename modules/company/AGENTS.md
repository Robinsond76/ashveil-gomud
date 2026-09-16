# Company Module Guide

`MobTemplateID` is durable company ownership; `InstanceId` is runtime-only. `PlayerSpawn` restores companions for both normal login and copyover. Normal GoMud charm movement remains the only follower mechanism.

Player-facing management is provided by `company summon <mob-id-or-name>`, `company status`, and `company dismiss`. Durable records are written immediately when these commands change company state.
