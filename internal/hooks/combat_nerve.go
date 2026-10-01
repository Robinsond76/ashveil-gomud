package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/interrupt"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var nerveSkip = map[int]bool{}

func noteMoraleDeaths() {
	for _, f := range moraleFights {
		for _, id := range f.ids {
			if m := mobs.GetInstance(id); m != nil && m.Character.Health <= 0 {
				f.dead[id] = true
			}
		}
	}
}
func nervePass() {
	clear(nerveSkip)
	for _, uid := range battle.Players() {
		b, _ := battle.Current(uid)
		f := moraleFights[b.FightID]
		if f == nil || f.nerveChecked {
			continue
		}
		hp, maximum, down := 0, 0, 0
		for _, ref := range f.startCompany {
			var c *characters.Character
			if ref.UserId > 0 {
				if u := users.GetByUserId(ref.UserId); u != nil {
					c = u.Character
				}
			} else {
				if m := mobs.GetInstance(ref.MobInstanceId); m != nil {
					c = &m.Character
				}
			}
			if c == nil || c.Health <= 0 {
				down++
				continue
			}
			hp += c.Health
			maximum += c.HealthMax.Value
		}
		if !morale.Losing(len(f.startCompany), down, hp, maximum) {
			continue
		}
		f.nerveChecked = true
		members := company.MoraleMembers(uid)
		for _, v := range members {
			m := mobs.GetInstance(v.InstanceID)
			if m == nil || m.Character.Health <= 0 {
				continue
			}
			if v.Loyalty >= 25 && !v.WeakChemistry {
				continue
			}
			switch morale.Nerve(v.Loyalty, v.WeakChemistry, moraleRoll(100)) {
			case morale.Hesitate:
				if _, lost := status.LostAction(&m.Character); lost {
					continue
				}
				nerveSkip[m.InstanceId] = true
				cancelNerveCast(m)
				moraleSay(m, "hesitates as the company falters.")
				emitCombat(combatstream.Event{Kind: combatstream.StatusTick, Target: mobRef(m), RoomId: b.RoomId, Outcome: combatstream.OutcomeLostAction, Status: "hesitation"})
			case morale.Flee:
				ref := mobRef(m)
				if err := company.BeginFlight(uid, v.ID); err != nil {
					if u := users.GetByUserId(uid); u != nil {
						u.SendText(err.Error())
					}
					continue
				}
				cancelNerveCast(m)
				emitCombat(combatstream.Event{Kind: combatstream.Flee, Source: ref, RoomId: b.RoomId})
				moraleSay(m, "loses nerve and flees the fight.")
				clearAimsAt(v.InstanceID)
			}
		}
	}
}
func cancelNerveCast(m *mobs.Mob) {
	if a := m.Character.Aggro; a != nil && a.Type == characters.SpellCast {
		_, cost := spellName(a.SpellInfo.SpellId)
		m.Character.ApplyManaChange(interrupt.Refund(cost))
		emitCast(combatstream.CastComplete, mobRef(m), a.SpellInfo.SpellId, combatstream.OutcomeInterrupted, m.Character.RoomId)
		endCast(&m.Character, caster{mobId: m.InstanceId})
	}
	delete(castAims, caster{mobId: m.InstanceId})
	delete(windUps, m.InstanceId)
}
