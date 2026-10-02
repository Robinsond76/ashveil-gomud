# Phase 34 save migration

Phase 34a backfills missing formation cells once; existing arrangements and
subsequent manual clears are retained. Solo leaders occupy 2,2.

Phase 34b introduces assigned Pack equipment. Existing suitable cargo packs
are assigned once, largest first, to the leader and then living companions.
A member without one receives a cloth knapsack providing 10 kg. Companion
grants are retained across death; an older fallen companion without a grant
receives it upon recovery. Existing equipped packs are retained. The asset
journal saves markers and exact resulting items together, and retries safely.

Every item retains its instance, quality, uses, quest marks and sharpening.
Nothing is deleted, sold or distributed between separate bags. Excess cargo
remains overloaded. Worn equipment is excluded from cargo load; assigned pack
weight is excluded from combat burden. Capacity is supplied by living,
present pack carriers and eligible horses. Removing packs or losing carriers
can leave the company overloaded; sell/drop/consume cargo or assign capacity
to recover. Ordinary additions and equipment changes cannot worsen an excess.
