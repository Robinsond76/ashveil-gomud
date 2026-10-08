package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MemberTempo is a company member's combat tempo as the next round would
// use it (Phase 82d): the leader's own, or a companion's by its member key,
// with the stance it would fight in. ok is false for a member who is not
// here (away, dead and gone, unknown). It is the number the Company panel,
// the battle screen's caption and the Combat tab show, rounded to two
// decimals so a change the player cannot see does not resend the snapshot.
func MemberTempo(leaderUserId int, key company.MemberKey) (tempo float64, ok bool) {
	if key == company.LeaderMemberKey {
		user := users.GetByUserId(leaderUserId)
		if user == nil || user.Character == nil {
			return 0, false
		}
		return roundTempo(tempoWithStance(caster{userId: leaderUserId}, user.Character)), true
	}
	id, found := company.InstanceForKey(leaderUserId, key)
	if !found {
		return 0, false
	}
	m := mobs.GetInstance(id)
	if m == nil {
		return 0, false
	}
	return roundTempo(tempoWithStance(caster{mobId: id}, &m.Character)), true
}

// tempoWithStance is the tempo the fight would use: the stance is applied
// as fillTempo applies it (between battles it is read afresh), on a copy so
// reading never changes the live character.
func tempoWithStance(who caster, c *characters.Character) float64 {
	cp := *c
	if c.RT != nil {
		rt := *c.RT
		cp.RT = &rt
		cp.RT.StanceRead = false
	}
	applyStance(who, &cp)
	return tempoRate(&cp)
}

func roundTempo(t float64) float64 { return math.Round(t*100) / 100 }

// tellRoundStart gives every player in a live battle one dim line naming
// the fight's round (Phase 82d), before the round's first turn, so the text
// alone shows where rounds break, in every client. The number counts the
// fight's own rounds from 1, as the battle screen's title does.
func tellRoundStart(round uint64) {
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		if !ok || round < b.StartRound {
			continue
		}
		user := users.GetByUserId(uid)
		if user == nil {
			continue
		}
		user.SendText(fmt.Sprintf(`<ansi fg="black-bold">Round %d</ansi>`, round-b.StartRound+1))
	}
}
