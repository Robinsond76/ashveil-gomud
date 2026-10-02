package company

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// MemberConditions is a read-only game-loop view. Unknown live state must
// never be replaced with saved buffs; saved lasting wounds are marked separately.
type MemberConditions struct {
	Key    MemberKey
	State  string
	Buffs  []buffs.Buff
	Wounds []wounds.Wound
}

type ConditionsProvider interface {
	CompanyConditions(leaderUserID int) ([]MemberConditions, bool)
}

func CompanyConditions(leaderUserID int) ([]MemberConditions, bool) {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if cp, ok := p.(ConditionsProvider); ok {
		return cp.CompanyConditions(leaderUserID)
	}
	return nil, false
}
