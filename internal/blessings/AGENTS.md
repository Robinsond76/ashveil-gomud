# Blessings Package Guide

Phase 77. Account blessings: data (`blessings.yaml` in the world files, loaded by `main.go`), validation, what a character's chronicle has earned (`Earned`), applying a blessing to a new character (`Apply`) and the recruit discount (`DiscountPercent`, `RecruitPrice`). The module `modules/blessings` owns the saved per-account list and installs the `Provider`; callers read through `EarnedFor` and never import the module.

- A blessing is a chronicle deed kind, a lifetime count and one perk: a starting item (`item`, `qty` 1-3) or a recruit discount (`discount`, percent). `iron: true` counts only an Iron character's deeds. Nothing here adds combat power; starting items are camp supplies markets never buy back.
- `MaxDiscount` (10%) caps a character's total discount, enforced in `DiscountPercent` and per blessing in `Validate`. A discounted price is never below 1 gold, and a free price stays free.
- A character carries the ids it was given (`Character.Blessings`), set once at creation by `Apply`; perks are read from the character, so a blessing earned later by the same character waits for its successor.
- Add a deed phrase in `deedPhrase` when a blessing uses a new kind. The shipped file's items must exist (`TestShippedBlessingsValidate`).
