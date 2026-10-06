package hooks

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sigils"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 54: sigils (internal/sigils holds the rules; help sigils). A sigil
// is laid before a fight with `cast sigil of [kind]` and belongs to the
// laying company's leader. When a battle begins in the sigil's room while it
// is lit, the battle takes it (internal/battle's Sigil): the ward and
// stillness sigils act at once, and the fire and mending sigils change
// spells for the battle's length (scripting's SpellFactor and HealFactor
// read it; fireBurn leaves the burn). Nobody casts or chooses anything in
// the fight, and enemies never lay sigils.

// sigilNow is the clock sigils fade by; tests replace it.
var sigilNow = time.Now

// UseSigilClockForTest replaces the sigil clock until the returned restore
// is called.
func UseSigilClockForTest(now func() time.Time) (restore func()) {
	prev := sigilNow
	sigilNow = now
	return func() { sigilNow = prev }
}

// startSigil gives a battle that has just begun the sigil its leader laid in
// the room, if it is still lit.
func (sd side) startSigil(room *rooms.Room, enemies []int) {
	uid := sd.user.UserId
	l := sd.user.Character.Sigil
	if !l.In(room.RoomId, sigilNow()) {
		return
	}
	battle.SetSigil(uid, l.Kind, l.Expires)
	switch l.Kind {
	case sigils.Ward:
		if n := wardCompany(sd.user, room); n > 0 {
			room.SendText(fmt.Sprintf("The %s flares, and a ward settles over the company. (%s, %d warded)", l.Kind.Name(), l.Kind.Name(), n))
		}
	case sigils.Stillness:
		chilled := 0
		for _, id := range enemies {
			m := mobs.GetInstance(id)
			if m == nil || m.Character.Health < 1 {
				continue
			}
			events.AddToQueue(events.Buff{MobInstanceId: id, BuffId: status.Windchilled, Source: `spell`, Triggers: stormcraft.Triggers(sigils.StillRounds)})
			chilled++
		}
		if chilled > 0 {
			room.SendText(fmt.Sprintf("The %s turns the air still around the foe. (%s, chants and sling shots a round slower)", l.Kind.Name(), l.Kind.Name()))
		}
	default:
		room.SendText(fmt.Sprintf("The %s glows underfoot. (%s: %s)", l.Kind.Name(), l.Kind.Name(), l.Kind.Effect()))
	}
}

// wardCompany puts a small ward on each standing member of the company that
// has none, and returns how many it warded. (Review: it once warded the
// front row only, which foes going for the weakest mostly walked past.)
func wardCompany(u *users.UserRecord, room *rooms.Room) int {
	warded := 0
	for _, a := range sideActors(u, room) {
		if a.char.Health < 1 {
			continue
		}
		rt := a.char.RTState()
		if rt.Ward > 0 {
			continue
		}
		rt.Ward, rt.WardCap, rt.WardSigil = sigils.WardBlows, sigils.WardCap(a.char.Level), true
		warded++
	}
	return warded
}

// fireBurn leaves the foes a fire spell struck burning, under a fire sigil.
// caster is a company member's battle leader (0 for an enemy's cast).
func fireBurn(leader int, spellId string, targets []int) {
	if leader == 0 || battle.SigilOf(leader) != sigils.Fire {
		return
	}
	if sp := spells.GetSpell(spellId); sp == nil || sp.Element != "fire" {
		return
	}
	for _, id := range targets {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 {
			events.AddToQueue(events.Buff{MobInstanceId: id, BuffId: status.Burning, Source: `spell`})
		}
	}
}

// leaderOfCaster is the battle leader of a company member that has cast: the
// player itself, or a companion's leader (0 for any other mob).
func leaderOfCaster(userId, mobId int) int {
	if userId > 0 {
		return userId
	}
	if mobId > 0 {
		if leader, _, ok := company.LeaderAndKeyForInstance(mobId); ok {
			return leader
		}
	}
	return 0
}
