package company

// LoyaltyKeeper is optionally implemented by the formation provider (Phase
// 33f3 Vigil): it raises present companions' loyalty once per operation.
type LoyaltyKeeper interface {
	RaiseLoyaltyOnce(leaderUserID int, op string, companionIDs []int, delta, cap int) ([]string, error)
}

// RaiseLoyaltyOnce raises each named living companion's loyalty by delta,
// never above cap, saved with op so a repeat changes nothing. It returns
// the names of those raised; nil without a keeper.
func RaiseLoyaltyOnce(leaderUserID int, op string, companionIDs []int, delta, cap int) ([]string, error) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if lk, ok := p.(LoyaltyKeeper); ok {
		return lk.RaiseLoyaltyOnce(leaderUserID, op, companionIDs, delta, cap)
	}
	return nil, nil
}
