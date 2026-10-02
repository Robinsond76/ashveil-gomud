package gmcp

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

type condition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Duration    string `json:"duration"`
	// SecondsLeft and SecondsTotal are set for an effect timed in game
	// rounds; the client counts SecondsLeft down itself. Duration states the
	// same as of the send.
	SecondsLeft  int            `json:"seconds_left,omitempty"`
	SecondsTotal int            `json:"seconds_total,omitempty"`
	Harmful      bool           `json:"harmful,omitempty"`
	Helpful      bool           `json:"helpful,omitempty"`
	Mods         map[string]int `json:"mods,omitempty"`
	Stacks       int            `json:"stacks,omitempty"`
	// ExpiresRound is the game round a timed effect ends. It is only in the
	// change key, which compares it instead of the ticking countdown.
	ExpiresRound uint64 `json:"expires_round,omitempty"`
	expires      uint64
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
		effect.Harmful, effect.Helpful = spec.Effect()
		switch {
		case b.PermaBuff:
			effect.Duration = "Persistent"
			out.Bonuses = append(out.Bonuses, effect)
			continue
		case spec.CombatRounds:
			effect.Duration = fmt.Sprintf("%d combat rounds remaining", max(b.TriggersLeft, 0))
		case spec.RoundInterval > 0 && b.TriggersLeft < buffs.TriggersLeftUnlimited:
			left, total := buffs.GetDurations(b, spec)
			timing := configs.GetTimingConfig()
			effect.SecondsLeft = max(timing.RoundsToSeconds(left), 0)
			effect.SecondsTotal = max(timing.RoundsToSeconds(total), effect.SecondsLeft)
			effect.Duration = fmt.Sprintf("%d seconds remaining", effect.SecondsLeft)
			effect.expires = util.GetRoundCount() + uint64(max(left, 0))
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

// conditionsKey is the payload with each timed effect's countdown replaced
// by the round it ends, so the message is resent only when an effect starts,
// ends, is refreshed or otherwise changes, never because a second passed.
// The end round a refresh computes can differ by one with the refresh's
// phase (before or after buffs tick that round, or a member that skips a
// tick while withdrawn), so an end within one round of the one last keyed
// for the same effect keeps that one (Phase 34 review).
func conditionsKey(all map[string]memberConditions, prev []byte) []byte {
	var last map[string]memberConditions
	if prev != nil {
		_ = json.Unmarshal(prev, &last)
	}
	stable := make(map[string]memberConditions, len(all))
	for key, member := range all {
		effects := append([]condition(nil), member.Effects...)
		before := last[key].Effects
		for i := range effects {
			if effects[i].expires == 0 {
				continue
			}
			end := effects[i].expires
			if i < len(before) && before[i].Name == effects[i].Name && before[i].ExpiresRound > 0 {
				if old := before[i].ExpiresRound; end+1 >= old && end <= old+1 {
					end = old
				}
			}
			effects[i].Duration, effects[i].SecondsLeft, effects[i].ExpiresRound = "", 0, end
		}
		member.Effects = effects
		stable[key] = member
	}
	key, _ := json.Marshal(stable)
	return key
}

func conditionsExtra() companyExtra {
	return companyExtra{module: "Company.Conditions", buildKeyed: func(user *users.UserRecord, prev []byte) ([]byte, []byte) {
		all := buildConditions(user)
		body, _ := json.Marshal(all)
		return body, conditionsKey(all, prev)
	}}
}
