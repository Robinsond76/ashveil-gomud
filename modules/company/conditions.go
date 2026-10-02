package company

import (
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// CompanyConditions reads only this leader's roster. A transferred charm
// exposes neither live conditions nor saved private state to the old owner.
func (m *CompanyModule) CompanyConditions(leaderUserID int) ([]domain.MemberConditions, bool) {
	if m.persistenceAvailable() != nil {
		return nil, false
	}
	record, _ := m.registry.Get(leaderUserID)
	out := []domain.MemberConditions{}
	for _, c := range record.Companions {
		v := domain.MemberConditions{Key: domain.CompanionMemberKey(c.ID), State: "away"}
		switch {
		case c.Dead():
			v.State = "dead"
		case c.Separated():
			v.State = "separated"
			if c.State != nil {
				v.Wounds = wounds.Lasting(c.State.Wounds)
			}
		default:
			inst, tracked := m.instance(leaderUserID, c.ID)
			if tracked && m.runtime.CharmedByOther(leaderUserID, inst) {
				v.State = "unavailable"
			} else if tracked && !c.PendingReturn && m.runtime.IsLive(inst) && m.runtime.IsAttached(leaderUserID, inst) {
				if mob := mobs.GetInstance(inst); mob != nil {
					v.State = "live"
					if !m.runtime.WithLeader(leaderUserID, inst) {
						v.State = "away-live"
					}
					for _, b := range mob.Character.GetBuffs() {
						if b != nil {
							v.Buffs = append(v.Buffs, *b)
						}
					}
					v.Wounds = append([]wounds.Wound(nil), mob.Character.Wounds...)
				}
			}
			if v.State == "away" && c.State != nil {
				v.Wounds = wounds.Lasting(c.State.Wounds)
			}
		}
		out = append(out, v)
	}
	return out, true
}
