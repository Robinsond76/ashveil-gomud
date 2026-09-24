package company

import "sync"

// FormationProvider is implemented by modules/company. It is a read-only
// query seam — the same shape as survival.CompanyService and
// weather.Provider — so internal/ packages (in particular
// internal/hooks's combat loop) can read a leader's current Formation
// without importing modules/company.
type FormationProvider interface {
	// FormationFor returns leaderUserID's current company Formation. ok is
	// false when the leader has no company record at all (a solo player,
	// or one who has never summoned a companion) — callers should treat
	// that as "no formation concept applies," not as an error.
	FormationFor(leaderUserID int) (Formation, bool)

	// InstanceFor returns the live mob instance ID currently attached to
	// leaderUserID's companionID, if the companion is currently spawned
	// and attached. ok is false otherwise (dismissed, never summoned, or
	// pending restoration).
	InstanceFor(leaderUserID, companionID int) (instanceId int, ok bool)

	// LeaderAndKeyForInstance returns the leader and formation MemberKey a
	// live mob instance is currently attached to as a companion. found is
	// false for anything that isn't a currently-attached companion of any
	// tracked company — a hostile mob, a detached/dismissed instance, or a
	// mob charmed outside the company system entirely.
	LeaderAndKeyForInstance(instanceId int) (leaderUserID int, key MemberKey, found bool)
}

var (
	formationProviderMu sync.RWMutex
	formationProvider   FormationProvider
)

// SetFormationProvider registers the active formation provider. Passing
// nil clears it.
func SetFormationProvider(p FormationProvider) {
	formationProviderMu.Lock()
	defer formationProviderMu.Unlock()
	formationProvider = p
}

// FormationFor calls through to the registered FormationProvider. It
// returns ok=false if no provider is registered (e.g. a test binary that
// never loaded modules/company) or the leader has no company record.
func FormationFor(leaderUserID int) (Formation, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return Formation{}, false
	}
	return p.FormationFor(leaderUserID)
}

// InstanceFor calls through to the registered FormationProvider. See
// FormationFor for the no-provider-registered contract.
func InstanceFor(leaderUserID, companionID int) (int, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return 0, false
	}
	return p.InstanceFor(leaderUserID, companionID)
}

// LeaderAndKeyForInstance calls through to the registered
// FormationProvider. See FormationFor for the no-provider-registered
// contract.
func LeaderAndKeyForInstance(instanceId int) (int, MemberKey, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if p == nil {
		return 0, "", false
	}
	return p.LeaderAndKeyForInstance(instanceId)
}

// ArchetypeProvider is optionally implemented by the registered
// FormationProvider (Phase 17): a read-only view of a companion's durable
// archetype.
type ArchetypeProvider interface {
	CompanionArchetype(leaderUserID, companionID int) (archetype string, ok bool)
}

// CompanionArchetype returns a companion's archetype id. ok is false when
// no provider is registered, the provider doesn't track archetypes, the
// companion is unknown, or it has none yet.
func CompanionArchetype(leaderUserID, companionID int) (string, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	ap, ok := p.(ArchetypeProvider)
	if !ok {
		return "", false
	}
	return ap.CompanionArchetype(leaderUserID, companionID)
}

// AlignmentProvider is optionally implemented by the registered
// FormationProvider (Phase 21b): the company's average alignment (the
// online leader and every companion, engine scale −100..100), as Phase 21a
// computes it for the recruit gate. modules/company runs on the game loop,
// so call this from the game loop only.
type AlignmentProvider interface {
	CompanyAlignment(leaderUserID int) (alignment int, ok bool)
}

// CompanyAlignment returns a leader's company alignment. ok is false when
// no provider is registered, it doesn't track alignment, or there is no one
// to average (an offline leader with no companions).
func CompanyAlignment(leaderUserID int) (int, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	ap, ok := p.(AlignmentProvider)
	if !ok {
		return 0, false
	}
	return ap.CompanyAlignment(leaderUserID)
}

// ChemistryStandingView is one member's Phase 24 chemistry for display.
type ChemistryStandingView struct {
	// Together is how many members of the band are alive and in the
	// member's room, the member included; Tier and Bonus are what that band
	// gives now (TierNone and 0 for a lone member).
	Together int
	Tier     int
	Bonus    int
}

// ChemistryProvider is optionally implemented by the registered
// FormationProvider (Phase 24). modules/company runs on the game loop, so
// call these from the game loop only.
type ChemistryProvider interface {
	// ChemistryHitBonus is a member's hit bonus in percentage points right
	// now: the tier of the band together in its room.
	ChemistryHitBonus(leaderUserID int, key MemberKey) int
	// ChemistryStanding is a member's chemistry for display; ok is false
	// when the leader has no company or the member isn't present.
	ChemistryStanding(leaderUserID int, key MemberKey) (ChemistryStandingView, bool)
}

func chemistryProvider() (ChemistryProvider, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	cp, ok := p.(ChemistryProvider)
	return cp, ok
}

// ChemistryBonusForUser is a player's chemistry hit bonus as the leader of
// their own company; 0 without a provider.
func ChemistryBonusForUser(userID int) int {
	cp, ok := chemistryProvider()
	if !ok || userID <= 0 {
		return 0
	}
	return cp.ChemistryHitBonus(userID, LeaderMemberKey)
}

// ChemistryBonusForInstance is a live mob's chemistry hit bonus when it is
// an attached company companion; 0 otherwise.
func ChemistryBonusForInstance(instanceID int) int {
	cp, ok := chemistryProvider()
	if !ok || instanceID <= 0 {
		return 0
	}
	leaderUserID, key, found := LeaderAndKeyForInstance(instanceID)
	if !found {
		return 0
	}
	return cp.ChemistryHitBonus(leaderUserID, key)
}

// ChemistryStanding is a member's chemistry for display. ok is false without
// a provider, a company, or the member present.
func ChemistryStanding(leaderUserID int, key MemberKey) (ChemistryStandingView, bool) {
	cp, ok := chemistryProvider()
	if !ok {
		return ChemistryStandingView{}, false
	}
	return cp.ChemistryStanding(leaderUserID, key)
}
