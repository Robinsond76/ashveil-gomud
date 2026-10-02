package gmcp

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

type condition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Duration    string         `json:"duration"`
	Mods        map[string]int `json:"mods,omitempty"`
	Stacks      int            `json:"stacks,omitempty"`
}

type memberConditions struct {
	State   string      `json:"state"`
	Effects []condition `json:"effects"`
	Bonuses []condition `json:"bonuses"`
	Wounds  []condition `json:"wounds"`
}

func conditionsOf(v company.MemberConditions) memberConditions {
	out := memberConditions{State: v.State, Effects: []condition{}, Bonuses: []condition{}, Wounds: []condition{}}
	for i := range v.Buffs {
		b := &v.Buffs[i]
		spec := buffs.GetBuffSpec(b.BuffId)
		if spec == nil || b.Expired() {
			continue
		}
		name, desc := spec.VisibleNameDesc()
		effect := condition{Name: name, Description: desc, Duration: "Until removed"}
		if !spec.Secret {
			effect.Stacks = max(b.Stacks, 1)
			effect.Mods = map[string]int{}
			for name, value := range spec.StatMods {
				effect.Mods[name] = value
			}
		}
		switch {
		case b.PermaBuff:
			effect.Duration = "Persistent"
			out.Bonuses = append(out.Bonuses, effect)
			continue
		case spec.CombatRounds:
			effect.Duration = fmt.Sprintf("%d combat rounds remaining", max(b.TriggersLeft, 0))
		case spec.RoundInterval > 0 && b.TriggersLeft < buffs.TriggersLeftUnlimited:
			left, _ := buffs.GetDurations(b, spec)
			effect.Duration = fmt.Sprintf("%d seconds remaining", max(configs.GetTimingConfig().RoundsToSeconds(left), 0))
		}
		out.Effects = append(out.Effects, effect)
	}
	sort.SliceStable(out.Effects, func(i, j int) bool { return out.Effects[i].Name < out.Effects[j].Name })
	sort.SliceStable(out.Bonuses, func(i, j int) bool { return out.Bonuses[i].Name < out.Bonuses[j].Name })
	for _, w := range v.Wounds {
		if w.Points <= 0 {
			continue
		}
		duration := "Until treated or rested away"
		if w.Light {
			duration = "Until fight ends"
		}
		out.Wounds = append(out.Wounds, condition{Name: wounds.Describe(w), Description: fmt.Sprintf("Holds back %d health; healing stops at the wound limit", w.Points), Duration: duration})
	}
	return out
}

func buildConditions(user *users.UserRecord) map[string]memberConditions {
	leader := company.MemberConditions{Key: company.LeaderMemberKey, State: "live", Wounds: user.Character.Wounds}
	for _, b := range user.Character.GetBuffs() {
		if b != nil {
			leader.Buffs = append(leader.Buffs, *b)
		}
	}
	out := map[string]memberConditions{"leader": conditionsOf(leader)}
	members, known := company.CompanyConditions(user.UserId)
	if !known {
		out["companions"] = memberConditions{State: "unknown"}
	}
	for _, v := range members {
		out[string(v.Key)] = conditionsOf(v)
	}
	return out
}

func conditionsExtra() companyExtra {
	return companyExtra{module: "Company.Conditions", build: func(user *users.UserRecord) []byte {
		body, _ := json.Marshal(buildConditions(user))
		return body
	}}
}
