package archetypes

// Phase 33f2: company specialists. Other modules (walking, market,
// weather, expedition, company) ask the provider who in a leader's
// company is best at a utility, without importing modules/archetype.

// Utility ids added in 33f2 (17b shipped "light" and "traps").
const (
	UtilityTrail      = "trail"
	UtilityPathfinder = "pathfinder"
	UtilityKeenEye    = "keeneye"
	UtilityWeather    = "weather"
	UtilityHaggle     = "haggle"

	// Phase 33f3 camp specialists.
	UtilityWatch      = "watch"
	UtilityFieldSmith = "fieldsmith"
	UtilityVigil      = "vigil"
	UtilityForage     = "forage"
)

// Specialist is the member of a leader's company best at a utility.
type Specialist struct {
	// Name is the member's character name.
	Name string
	// IsLeader is true when the leader is the specialist.
	IsLeader bool
	// Level is the member's utility level, 1..MaxUtilityLevel.
	Level int
}

// SpecialistProvider is optionally implemented by the provider (33f2).
type SpecialistProvider interface {
	// BestSpecialist is the leader's best living, present company member
	// (the leader or a companion standing in one of roomIDs) at a utility,
	// honouring the leader's autoskill switch for it. ok is false when
	// nobody qualifies or the switch is off.
	BestSpecialist(leaderUserID int, utility string, roomIDs ...int) (Specialist, bool)
	// SpecialistsView renders the leader's company specialists.
	SpecialistsView(leaderUserID int) string
}

// MemberUtilityProvider is optionally implemented by the provider (Phase 51
// rest duties): one member's own level at a utility, whether or not they
// are the company's best.
type MemberUtilityProvider interface {
	// MemberUtilityLevel is the level of the leader (companionID 0) or of a
	// companion standing in the leader's room; 0 when they have none.
	MemberUtilityLevel(leaderUserID, companionID int, utility string) int
}

// MemberUtilityLevel is 0 without a provider that resolves it.
func MemberUtilityLevel(leaderUserID, companionID int, utility string) int {
	if mp, ok := current().(MemberUtilityProvider); ok {
		return mp.MemberUtilityLevel(leaderUserID, companionID, utility)
	}
	return 0
}

// BestSpecialist is false without a provider that resolves specialists.
func BestSpecialist(leaderUserID int, utility string, roomIDs ...int) (Specialist, bool) {
	if sp, ok := current().(SpecialistProvider); ok {
		return sp.BestSpecialist(leaderUserID, utility, roomIDs...)
	}
	return Specialist{}, false
}

// SpecialistsView is "" without a provider.
func SpecialistsView(leaderUserID int) string {
	if sp, ok := current().(SpecialistProvider); ok {
		return sp.SpecialistsView(leaderUserID)
	}
	return ""
}

// Subject is how a specialist is named at the start of a sentence: "You"
// for the leader, else the member's name.
func (s Specialist) Subject() string {
	if s.IsLeader {
		return "You"
	}
	return s.Name
}

// Verb picks the verb form agreeing with Subject: "you read", "Mira reads".
func (s Specialist) Verb(you, other string) string {
	if s.IsLeader {
		return you
	}
	return other
}

// PctByLevel is level × perLevel percent, clamped to 0..max.
func PctByLevel(level, perLevel, max int) int {
	pct := level * perLevel
	if pct < 0 {
		return 0
	}
	if pct > max {
		return max
	}
	return pct
}

// AmbushEvader is optionally implemented by the provider (33f2): a tracker
// may lead a travelling company around an ambush.
type AmbushEvader interface {
	EvadeAmbush(leaderUserID int, roomIDs ...int) (Specialist, bool)
}

// EvadeAmbush is false without a provider; tracker is the company's best
// tracker when there is one, whether or not the attempt succeeded.
func EvadeAmbush(leaderUserID int, roomIDs ...int) (tracker Specialist, evaded bool) {
	if ae, ok := current().(AmbushEvader); ok {
		return ae.EvadeAmbush(leaderUserID, roomIDs...)
	}
	return Specialist{}, false
}
