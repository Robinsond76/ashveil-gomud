# Phase 33g — cargo, treasury, and deliberate equipment assignment

Owner instruction: begin 33g. Follow-up decisions: pool all gold into one company
treasury; remove equipment/formation presets from this phase. The standing
33-series authorization delegates routine defaults to the lead.

## Delivery contract

- All unworn items are company cargo. Preserve full item instances (including
  enchantments, uses, edges and quest marks); the leader's existing Items field
  backs the shared cargo for compatibility with normal item commands.
- The leader's Gold field backs the pooled treasury, preserving existing prices,
  purchases and trading. Living companions contribute their carried items/gold;
  dead companions' retained body assets remain subject to resurrection rules.
- Migrate legacy cargo stacks exactly once, with a marker saved alongside cargo.
- Direct commands: `company equip [member] [item reference]`, `company remove
  [member] [slot]`, `company compare [member] [item reference]`, and `company
  treasury`. `company equipment` lists exact cargo references. Legacy player
  equip/remove remain available; follower management directs players to the
  company commands. No presets and no automatic whole-company upgrades.
- Equipment transfers use a durable company operation containing the complete
  resulting leader assets and companion gear. Save the operation before user
  assets; mark its application in the user file; recover before commands/login.
  An applied operation is never replayed after subsequent asset changes.
- Shared cargo affects expedition weight, not the leader's dodge burden. Packs
  in shared cargo count at most one per living member, largest bonuses first.
- `loot` collects eligible corpses into company cargo. Optional `autoloot` is
  opt-in. Battle spoils are reserved for their claimant for two game hours,
  public for two more, then expire. Preserve the existing claimant selection.
- No item use or gear management in battle; enforce through real commands,
  follower orders and stale browser requests.

The approved equipment catalog and tiers are separate content
slices of the equipment roadmap; do not claim their gameplay has shipped with
management. Record their remaining delivery explicitly.

## Tasks

1. Inspect save seams and implement instance-preserving shared cargo, pooled
   treasury, and recoverable equipment operations.
2. Wire explicit assignment/removal/comparison, exact refs, legacy compatibility,
   battle/ownership checks, loot lifecycle and opt-in autoloot.
3. Update Company.Inventory and text/browser controls using the same commands.
4. Ship indexed equipment/loot/treasury help, updated hubs and tutorial pointers.
5. Integration tests: duplicates, cursed/bound/disabled equipment, two hands,
   full capacity, foreign/dead/absent members, battle refusal, user/company save
   failures and restart retries; legacy cargo uses and metadata preservation;
   loot ownership, public/expiry boundaries, autoloot and no clock advancement.
6. Independent full-diff review, fix findings, then generate/validate/full race
   suite and applicable lint. Record exact results in Project Status.

## Completion

Management, cargo migration, treasury, loot, browser/help, integration tests,
and independent review completed. Review findings and
exact final verification are recorded in [Project Status](../PROJECT_STATUS.md).
Presets are excluded; catalog/tier migration remains separate.
