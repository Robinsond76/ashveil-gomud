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

// LoyaltyAdjuster is optionally implemented by the formation provider
// (Phase 60 story events): it moves present companions' loyalty up or down
// once per operation, kept between 0 and 100.
type LoyaltyAdjuster interface {
	AdjustLoyaltyOnce(leaderUserID int, op string, companionIDs []int, delta int) ([]string, error)
}

// AdjustLoyaltyOnce changes each named living companion's loyalty by delta
// (negative lowers it), clamped to 0..100 and saved with op so a repeat
// changes nothing. It returns the names of those changed; nil without a
// provider that can.
func AdjustLoyaltyOnce(leaderUserID int, op string, companionIDs []int, delta int) ([]string, error) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if la, ok := p.(LoyaltyAdjuster); ok {
		return la.AdjustLoyaltyOnce(leaderUserID, op, companionIDs, delta)
	}
	return nil, nil
}
