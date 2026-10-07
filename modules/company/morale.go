package company

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/interrupt"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func (m *CompanyModule) MoraleMembers(uid int) []domain.MoraleMember {
	r, ok := m.registry.Get(uid)
	if !ok {
		return nil
	}
	var out []domain.MoraleMember
	rules := m.chemistryRules()
	for _, c := range r.Companions {
		id, tracked := m.instance(uid, c.ID)
		if !tracked || c.Dead() || c.PendingReturn || bound(c) || !m.runtime.WithLeader(uid, id) { // Phase 38e: a construct never loses heart
			continue
		}
		loyalty := domain.MaxLoyalty
		if c.Disposition != nil {
			loyalty = c.Disposition.Loyalty
		}
		weak := r.BandTier(m.bandOf(uid, r, domain.CompanionMemberKey(c.ID)), rules) < domain.TierFamiliar
		out = append(out, domain.MoraleMember{ID: c.ID, InstanceID: id, Loyalty: loyalty, WeakChemistry: weak})
	}
	return out
}
func (m *CompanyModule) MercyReaction(uid int, token string, witnesses []int, spare bool) ([]string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	r, ok := m.registry.Get(uid)
	if !ok {
		r = domain.Record{LeaderUserID: uid}
	}
	for _, effect := range r.MercyPending {
		if effect.Token == token {
			return nil, m.moraleDepartures(uid)
		}
	}
	before, existed := m.registry.Get(uid)
	want := map[int]bool{}
	for _, id := range witnesses {
		want[id] = true
	}
	var lines []string
	for i, c := range r.Companions {
		if !want[c.ID] || c.Dead() || bound(c) { // Phase 38e: a construct passes no judgement
			continue
		}
		a := m.companionAlignment(c)
		change := morale.Reaction(a, spare)
		if change == 0 {
			continue
		}
		loyalty := domain.MaxLoyalty
		if c.Disposition != nil {
			loyalty = c.Disposition.Loyalty
		}
		r.Companions[i].Disposition = &domain.Disposition{Alignment: a, Loyalty: max(0, min(100, loyalty+change))}
		if loyalty+change <= 0 {
			r.Companions[i].MoraleDesert = true
		}
		label := nameOf(c, m.companionLabel(uid, c.ID))
		reaction := "approves your decision"
		if change < 0 {
			reaction = "disapproves of your decision"
		}
		lines = append(lines, fmt.Sprintf("%s %s. (loyalty %+d)", label, reaction, change))
	}
	delta := -5
	if spare {
		delta = 5
	}
	r.MercyPending = append(r.MercyPending, domain.MercyEffect{Token: token, Delta: delta})
	m.registry.Put(r)
	if err := m.save(); err != nil {
		if existed {
			m.registry.Put(before)
		} else {
			delete(m.registry.Companies, uid)
		}
		return nil, err
	}
	if err := m.moraleDepartures(uid); err != nil {
		return nil, err
	}
	return lines, nil
}

// BeginFlight saves the complete member state and return debt before removing
// the live instance. A failed save leaves the member in the fight.
func (m *CompanyModule) BeginFlight(uid, cid int) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	before, ok := m.registry.Get(uid)
	if !ok {
		return domain.ErrUnknownMember
	}
	c, ok := findCompanion(before, cid)
	if !ok || c.Dead() {
		return domain.ErrUnknownMember
	}
	if c.PendingReturn {
		return nil
	}
	id, tracked := m.instance(uid, cid)
	mob := mobs.GetInstance(id)
	if !tracked || mob == nil || mob.Character.Health <= 0 {
		return fmt.Errorf("companion is no longer standing")
	}
	m.refreshSnapshot(uid, cid)
	r, _ := m.registry.Get(uid)
	for i, v := range r.Companions {
		if v.ID == cid {
			r.Companions[i].PendingReturn = true
			r.Companions[i].ReturnHP = mob.Character.Health
			r.Companions[i].ReturnMana = mob.Character.Mana
			if a := mob.Character.Aggro; a != nil && a.Type == characters.SpellCast {
				if sp := spells.GetSpell(a.SpellInfo.SpellId); sp != nil {
					r.Companions[i].ReturnMana = min(mob.Character.ManaMax.Value, mob.Character.Mana+interrupt.Refund(sp.Cost))
				}
			}
		}
	}
	m.registry.Put(r)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	m.runtime.Detach(uid, id)
	m.clearInstance(uid, cid)
	return nil
}

// ReturnFlight pays the loyalty debt and clears its durable marker together.
// Runtime restoration happens only after that save; failed spawn is retryable.
func (m *CompanyModule) ReturnFlight(uid int) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	before, ok := m.registry.Get(uid)
	if !ok {
		return nil
	}
	r, _ := m.registry.Get(uid)
	changed := false
	for i, c := range r.Companions {
		if !c.PendingReturn {
			continue
		}
		changed = true
		r.Companions[i].PendingReturn = false
		if c.Dead() {
			r.Companions[i].ReturnHP = 0
			r.Companions[i].ReturnMana = 0
			continue
		}
		loyalty := domain.MaxLoyalty
		if c.Disposition != nil {
			loyalty = c.Disposition.Loyalty
		}
		r.Companions[i].Disposition = &domain.Disposition{Alignment: m.companionAlignment(c), Loyalty: max(0, loyalty-5)}
		if loyalty <= 5 {
			r.Companions[i].MoraleDesert = true
		}
	}
	if !changed {
		if u := users.GetByUserId(uid); u != nil && u.Character.Health > 0 {
			for _, c := range r.Companions {
				if c.ReturnHP > 0 {
					return m.restoreForLeader(uid, u.Character.RoomId)
				}
			}
		}
		return nil
	}
	m.registry.Put(r)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	if u := users.GetByUserId(uid); u != nil && u.Character.Health > 0 {
		if err := m.restoreForLeader(uid, u.Character.RoomId); err != nil {
			return err
		}
		u.SendText("The companions who fled rejoin you. (loyalty -5)")
	}
	return nil
}
func (m *CompanyModule) retryReturns() {
	for uid := range m.registry.Companies {
		if err := m.moraleDepartures(uid); err != nil {
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(err.Error())
			}
		}
	}
	for uid, r := range m.registry.Companies {
		if _, busy := battle.Current(uid); busy {
			continue
		}
		pending := false
		for _, c := range r.Companions {
			pending = pending || c.PendingReturn || c.ReturnHP > 0
		}
		if pending {
			if err := m.ReturnFlight(uid); err != nil {
				if u := users.GetByUserId(uid); u != nil {
					u.SendText(err.Error())
				}
			}
		}
	}
}

func (m *CompanyModule) PendingMercy() map[int][]domain.MercyEffect {
	out := map[int][]domain.MercyEffect{}
	for uid, r := range m.registry.Companies {
		if len(r.MercyPending) > 0 {
			out[uid] = append([]domain.MercyEffect(nil), r.MercyPending...)
		}
	}
	return out
}
func (m *CompanyModule) CompleteMercy(uid int, token string) error {
	before, ok := m.registry.Get(uid)
	if !ok {
		return nil
	}
	r, _ := m.registry.Get(uid)
	r.MercyPending = nil
	for _, e := range before.MercyPending {
		if e.Token != token {
			r.MercyPending = append(r.MercyPending, e)
		}
	}
	m.registry.Put(r)
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err
	}
	return nil
}

// Zero loyalty caused by morale is a departure even when ordinary drift would
// otherwise increase it. A failed departure keeps zero and retries each round.
func (m *CompanyModule) moraleDepartures(uid int) error {
	r, ok := m.registry.Get(uid)
	if !ok {
		return nil
	}
	for _, c := range r.Companions {
		if !c.MoraleDesert || c.Dead() || c.Disposition == nil || c.Disposition.Loyalty > 0 {
			continue
		}
		if m.inCombat(uid, c.ID) {
			continue
		}
		current, _ := m.registry.Get(uid)
		if err := m.removeCompanion(uid, current, c); err != nil {
			return err
		}
		m.alignmentWorld().Tell(uid, fmt.Sprintf("%s has lost faith in your company and deserts.", nameOf(c, m.companionLabel(uid, c.ID)))+m.riteHint(uid, c.ID))
	}
	return nil
}
