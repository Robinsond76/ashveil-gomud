package company

// MoraleMember is a living, present companion's nerve input.
// MercyEffect is a saved alignment obligation after loyalty reactions were saved.
// It contains no instance IDs or rewards and cannot restore prisoners.
type MercyEffect struct {
	Token string `yaml:"token"`
	Delta int    `yaml:"delta"`
}
type MoraleMember struct {
	ID, InstanceID, Loyalty int
	WeakChemistry           bool
}

// MoraleProvider owns durable reactions and pending companion returns.
type MoraleProvider interface {
	MoraleMembers(int) []MoraleMember
	MercyReaction(int, string, []int, bool) ([]string, error)
	PendingMercy() map[int][]MercyEffect
	CompleteMercy(int, string) error
	BeginFlight(int, int) error
	ReturnFlight(int) error
}

func moraleProvider() MoraleProvider {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	m, _ := p.(MoraleProvider)
	return m
}
func MoraleMembers(uid int) []MoraleMember {
	if p := moraleProvider(); p != nil {
		return p.MoraleMembers(uid)
	}
	return nil
}
func MercyReaction(uid int, token string, witnesses []int, spare bool) ([]string, error) {
	if p := moraleProvider(); p != nil {
		return p.MercyReaction(uid, token, witnesses, spare)
	}
	return nil, nil
}
func BeginFlight(uid, id int) error {
	if p := moraleProvider(); p != nil {
		return p.BeginFlight(uid, id)
	}
	return nil
}
func ReturnFlight(uid int) error {
	if p := moraleProvider(); p != nil {
		return p.ReturnFlight(uid)
	}
	return nil
}

func PendingMercy() map[int][]MercyEffect {
	if p := moraleProvider(); p != nil {
		return p.PendingMercy()
	}
	return nil
}
func CompleteMercy(uid int, token string) error {
	if p := moraleProvider(); p != nil {
		return p.CompleteMercy(uid, token)
	}
	return nil
}
