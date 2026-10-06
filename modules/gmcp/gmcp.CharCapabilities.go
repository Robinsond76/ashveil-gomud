package gmcp

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type automaticCapability struct {
	Name        string `json:"name"`
	Skill       string `json:"skill"`
	Description string `json:"description"`
	When        string `json:"when"`
	Cooldown    int    `json:"cooldown"`
	Enabled     bool   `json:"enabled"`
	Reason      string `json:"reason,omitempty"`
}

type charCapabilities struct {
	Automatic []automaticCapability   `json:"automatic"`
	Utility   []archetypes.Capability `json:"utility"`
}

func buildCapabilities(user *users.UserRecord) charCapabilities {
	out := charCapabilities{Automatic: []automaticCapability{}, Utility: archetypes.PlayerCapabilities(user.UserId)}
	if cooking, known := camping.CookingCapability(user.UserId); known {
		out.Utility = append(out.Utility, archetypes.Capability{ID: "cooking", Name: "Camp Cooking", Group: "camp", Mode: "manual", Skill: "cooking", Rank: cooking.Rank, Enabled: cooking.Ready, Reason: cooking.Reason, Description: cooking.Description})
	}
	id, _ := archetypes.PlayerArchetype(user.UserId)
	s := strategy.For(user.UserId, "leader", id)
	class, _ := user.Character.ClassState()
	for _, ability := range strategy.WithClass(strategy.PlayerAbilities(user.Character.GetSkillLevel), class, user.Character.Level) {
		spec, ok := strategy.SpecOf(ability)
		if !ok {
			continue
		}
		v := automaticCapability{Name: spec.Name, Skill: spec.Skill, Description: spec.Does, When: spec.When, Cooldown: spec.Cooldown, Enabled: !s.NoAbilities}
		if s.NoAbilities {
			v.Reason = "Automatic abilities disabled by strategy"
		}
		if spec.MinLevel > user.Character.Level {
			v.Enabled, v.Reason = false, fmt.Sprintf("Comes at level %d", spec.MinLevel)
		}
		out.Automatic = append(out.Automatic, v)
	}
	knows := func(id string) bool {
		return user.Character.GetSkillLevel("cast") > 0 && user.Character.HasSpell(id) && spells.GetSpell(id) != nil
	}
	for _, use := range []strategy.Use{strategy.UseHeal, strategy.UseHealAll, strategy.UseAttack, strategy.UseAttackAll, strategy.UseCure, strategy.UseBless, strategy.UseFlame} {
		auto, ok := strategy.SpellFor(strategy.AutoSpells(), use, knows)
		if !ok {
			continue
		}
		spell := spells.GetSpell(auto.ID)
		role := strategy.Caster
		when := "standing foes remain; group spells prefer two or more foes"
		attack := use == strategy.UseAttack || use == strategy.UseAttackAll
		switch use {
		case strategy.UseHeal, strategy.UseHealAll:
			role = strategy.Healer
			when = fmt.Sprintf("an uncovered ally falls below %d%% of its wound limit; group healing prefers two or more hurt allies", strategy.TacticsFor(user.UserId).Resolve().Healing)
		case strategy.UseCure:
			role = strategy.Healer
			when = "an ally carries poison or bleeding and no one is in danger"
		case strategy.UseBless:
			role = strategy.Healer
			when = "no one needs healing and a battle is on: one tonic for each ally without one"
		case strategy.UseFlame:
			role = strategy.Healer
			when = fmt.Sprintf("no one needs healing and more than %d flasks remain", strategy.FlaskKeep)
		}
		v := automaticCapability{Name: spell.Name, Skill: "cast", Description: spell.Description, When: when, Enabled: true}
		switch {
		case s.Role != role:
			v.Reason = "Requires " + string(role) + " strategy role"
		case spell.Flask > 0 && flasks.Remaining(user.Character) < spell.Flask:
			v.Reason = "Satchel empty"
		case user.Character.Mana < spell.Cost:
			v.Reason = "Not enough mana"
		case attack && s.Reserve > 0 && (user.Character.Mana-spell.Cost)*100 < s.Reserve*user.Character.ManaMax.Value:
			v.Reason = "Mana reserve prevents casting"
		}
		v.Enabled = v.Reason == ""
		if spell.Flask > 0 {
			v.Description += fmt.Sprintf("; %d flasks left", flasks.Remaining(user.Character))
		} else {
			v.Description += fmt.Sprintf("; costs %d mana", spell.Cost)
		}
		out.Automatic = append(out.Automatic, v)
	}
	return out
}

func capabilitiesExtra() companyExtra {
	return companyExtra{module: "Char.Capabilities", build: func(user *users.UserRecord) []byte {
		body, _ := json.Marshal(buildCapabilities(user))
		return body
	}}
}
