package combat

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/survival"
)

func fatigueFor(uid int, key company.MemberKey) int {
	if uid <= 0 {
		return 0
	}
	for _, member := range survival.CompanyNeeds(uid) {
		if member.Key == key {
			return formationcombat.FatiguePenalty(member.Needs.Fatigue)
		}
	}
	return 0
}

func mobFatigue(id int) int {
	owner, key, ok := company.LeaderAndKeyForInstance(id)
	if !ok {
		return 0
	}
	return fatigueFor(owner, key)
}

func fatigueText(r *AttackResult, penalty int) {
	if penalty <= 0 {
		return
	}
	suffix := fmt.Sprintf(" (fatigue: hit -%d%%)", penalty)
	for _, lines := range []*[]string{&r.MessagesToSource, &r.MessagesToTarget, &r.MessagesToSourceRoom, &r.MessagesToTargetRoom} {
		for i := range *lines {
			(*lines)[i] += suffix
		}
	}
}
