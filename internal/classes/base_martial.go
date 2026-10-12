package classes

// The skill schedule (2026-10-12, docs/plans/2026-10-12-skill-schedule.md):
// a level-1 character knows only its plain attack (a caster its first
// spell), and its lineage's skills come as it levels, about one every other
// level until promotion. The warrior, rogue and ranger signatures are
// automatic abilities (internal/strategy, whose MinLevel matches); they are
// base ranks too so a level-up names them in the "New rank" line and the
// level-up report's "Next:" points at them. They set no effects.

func init() {
	registerBase("warrior",
		rank(2, "Tackle", "knocks its foe down (it loses its next action), breaking a chant or wind-up; the whole turn, ready every 4 rounds"),
	)
	registerBase("rogue",
		rank(2, "Opening Strike", "its first blow that lands in a round on a foe that is knocked down, stunned, staggered or exposed is a critical hit, ready every 2 rounds"),
	)
	registerBase("ranger",
		rank(2, "Aimed Shot", "with a shooting weapon, its first shot that lands in a round is a critical hit that leaves the foe exposed, ready every 3 rounds"),
	)
}
