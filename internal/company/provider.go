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
