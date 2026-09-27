// Package combatstream is Ashveil's combat event stream (Phase 29b). Every
// combat happening becomes one structured Event. Combat code produces them;
// narration, pacing, the battle summary, the battle panel, and balancing
// data consume them.
//
// The stream only reports. It is never the source of truth for health,
// aggro, or any other state, it never advances the clock, and nothing in
// it is persisted: a copyover mid-fight loses the fight's summary, not the
// fight.
//
// It depends only on the standard library; internal/hooks adapts live
// users and mobs into Refs.
package combatstream

import "strconv"

// Kind names one kind of combat happening.
type Kind string

const (
	FightStart   Kind = "fight-start"
	FightEnd     Kind = "fight-end"
	TargetChange Kind = "target-change"
	// Attack is one weapon attack round (combat.AttackResult). Outcome is
	// OutcomeHit, OutcomeMiss, or OutcomeCrit.
	Attack Kind = "attack"
	// SpellHit is damage a spell did to one target.
	SpellHit Kind = "spell-hit"
	// Heal is health a spell restored to one target. HeldBack is what a
	// wound limit kept back (Phase 30b).
	Heal         Kind = "heal"
	CastStart    Kind = "cast-start" // Phase 30d
	CastProgress Kind = "cast-progress"
	// CastComplete ends a cast. Outcome is OutcomeCast or OutcomeFizzled.
	CastComplete   Kind = "cast-complete"
	WindUpStart    Kind = "windup-start" // Phase 30d
	WindUpLand     Kind = "windup-land"  // Phase 30d
	Interrupt      Kind = "interrupt"    // Phase 30d
	StatusApplied  Kind = "status-applied"
	StatusExpired  Kind = "status-expired"  // Phase 30a
	WoundChange    Kind = "wound-change"    // Phase 30b
	GuardUsed      Kind = "guard-used"      // Phase 30c
	GuardExhausted Kind = "guard-exhausted" // Phase 30c
	Yield          Kind = "yield"           // Phase 30e
	Flee           Kind = "flee"
	// Death is a death or an incapacitation: Outcome is OutcomeSlain,
	// OutcomeBeaten (a practice foe), or OutcomeIncapacitated. An empty
	// Source is filled with the victim's last damager in the fight.
	Death Kind = "death"
	Mercy Kind = "mercy" // Phase 30e
)

// Outcomes.
const (
	OutcomeHit           = "hit"
	OutcomeMiss          = "miss"
	OutcomeCrit          = "crit"
	OutcomeCast          = "cast"
	OutcomeFizzled       = "fizzled"
	OutcomeSlain         = "slain"
	OutcomeIncapacitated = "incapacitated"
	OutcomeBeaten        = "beaten"    // a practice foe, beaten without a death
	OutcomeSucceeded     = "succeeded" // interrupts, Phase 30d
	OutcomeFailed        = "failed"    // interrupts, Phase 30d

	// Fight endings (FightEnd's Outcome).
	OutcomeVictory   = "victory"
	OutcomeDefeat    = "defeat"
	OutcomeBrokenOff = "broken-off"
)

// Ref names one combatant: a player (UserId) or a mob instance
// (MobInstanceId). LeaderUserId and MemberKey are set when the actor is a
// member of a company (the leader, or an attached companion).
type Ref struct {
	UserId        int
	MobInstanceId int
	MobId         int
	Name          string
	LeaderUserId  int
	MemberKey     string
}

// Zero reports whether r names nobody.
func (r Ref) Zero() bool { return r.UserId == 0 && r.MobInstanceId == 0 }

// Key identifies r uniquely: "u:<id>" for a player, "m:<instance>" for a
// mob, "" for nobody.
func (r Ref) Key() string {
	switch {
	case r.UserId > 0:
		return "u:" + strconv.Itoa(r.UserId)
	case r.MobInstanceId > 0:
		return "m:" + strconv.Itoa(r.MobInstanceId)
	}
	return ""
}

// Event is one combat happening. Fields a kind doesn't use are zero.
type Event struct {
	Seq     uint64 // stamped by Emit, in emission order
	Kind    Kind
	Round   uint64
	FightID uint64 // 0 when the happening is not part of a tracked fight
	PartyID string // the enemy party (Phase 11a) of the fight, if any
	RoomId  int

	Source   Ref
	Target   Ref
	Previous Ref // TargetChange: the target before

	Outcome    string
	Damage     int
	Crit       bool
	WeaponType string
	SpellId    string
	Amount     int // Heal: health restored
	HeldBack   int // Heal: kept back by a wound limit (Phase 30b)
	BuffId     int
	Status     string // StatusApplied/Expired: the buff's name

	// Summary is set on FightEnd.
	Summary *Summary
}
