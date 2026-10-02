package archetype

import (
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PlayerCapabilities shares the specialist level resolver and durable
// autoskill settings. It reads without spending mana or starting cooldowns.
func (m *ArchetypeModule) PlayerCapabilities(userID int) []archetypes.Capability {
	out := []archetypes.Capability{}
	u := users.GetByUserId(userID)
	if u == nil || u.Character == nil {
		return out
	}
	m.mu.Lock()
	id, chosen := m.registry.Players[userID]
	a, known := m.table.Get(id)
	cfg := m.config
	mapped := map[string]string{}
	for key, value := range cfg.UtilitySkills {
		mapped[key] = value
	}
	m.mu.Unlock()
	_, fighting := battle.Current(userID)
	for _, spec := range specialistOrder {
		skill, configured := mapped[spec.utility]
		if !configured || !chosen || !known || !a.HasUtility(spec.utility) {
			continue
		}
		mb := []member{{user: u}}
		m.levels(mb, spec.utility)
		group := "field"
		switch spec.utility {
		case archetypes.UtilityWatch, archetypes.UtilityFieldSmith, archetypes.UtilityVigil, archetypes.UtilityForage:
			group = "camp"
		}
		v := archetypes.Capability{ID: spec.utility, Name: spec.name, Group: group, Description: spec.does, Skill: skill, Rank: mb[0].Level, Enabled: true}
		switch {
		case v.Rank <= 0:
			v.Reason = "Requires trained " + skill
		case !m.autoskillOn(userID, spec.utility):
			v.Reason = "Autoskill off"
		case fighting || u.Character.Aggro != nil:
			v.Reason = "Unavailable in battle"
		case u.Character.IsDisabled():
			v.Reason = "Unable to act"
		}
		if v.Reason == "" && spec.utility == utilityLight {
			spell := spells.GetSpell(cfg.AutoLightSpell)
			switch {
			case spell == nil || !u.Character.HasSpell(cfg.AutoLightSpell):
				v.Reason = "Requires " + cfg.AutoLightSpell
			case u.Character.Mana < spell.Cost:
				v.Reason = "Not enough mana"
			}
		}
		v.Enabled = v.Reason == ""
		out = append(out, v)
	}
	return out
}
