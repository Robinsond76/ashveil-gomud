package combat

import (
	"fmt"
	"sync"

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

// fatigueNoted is the penalty last named to each attacker ("u7", "m12"),
// so a strike names fatigue when it starts or changes, not every round.
var fatigueNoted = struct {
	sync.Mutex
	by map[string]int
}{by: map[string]int{}}

func fatigueText(r *AttackResult, penalty int, attacker string) {
	fatigueNoted.Lock()
	defer fatigueNoted.Unlock()
	if penalty <= 0 {
		delete(fatigueNoted.by, attacker)
		return
	}
	if fatigueNoted.by[attacker] == penalty {
		return
	}
	fatigueNoted.by[attacker] = penalty
	suffix := fmt.Sprintf(" (fatigue: hit -%d%%)", penalty)
	for _, lines := range []*[]string{&r.MessagesToSource, &r.MessagesToTarget, &r.MessagesToSourceRoom, &r.MessagesToTargetRoom} {
		for i := range *lines {
			(*lines)[i] += suffix
		}
	}
}
