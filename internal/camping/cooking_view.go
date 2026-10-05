package camping

// CookingView is the manual camp cook capability, supplied by its owner.
// Ready refers to camp, battle, recipe ingredients/ranks and final load.
type CookingView struct {
	// Rank is the best cook's (Phase 35c): the leader or a companion
	// present, named by Cook.
	Rank         int
	Cook         string
	CookIsLeader bool
	Ready        bool
	Description  string
	Reason       string
}

type CookingProvider interface {
	CookingCapability(userID int) (CookingView, bool)
}

func CookingCapability(userID int) (CookingView, bool) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	if cp, ok := p.(CookingProvider); ok {
		return cp.CookingCapability(userID)
	}
	return CookingView{}, false
}
