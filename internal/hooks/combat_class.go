package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 38b: class auras. Each combat round, before any blow, a company
// member who has fallen gives none; a standing holder gives the allies in
// its row its Evasion (Watchful row) and damage reduction (Aura of
// Resolve, Shelter), and the rest of the company its Rallying share.
// Auras of one kind don't stack: a row takes the best.

// auraPass sets every company member's aura for the round.
func auraPass() {
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || u == nil || u.Character == nil || u.Character.RoomId != b.RoomId {
			continue
		}
		room := rooms.LoadRoom(b.RoomId)
		if room == nil {
			continue
		}
		f, _ := company.FormationFor(uid)
		side := sideActors(u, room)
		applyAuras(side, f)
		for _, a := range side {
			if a.char.ClassEffects() != nil {
				a.char.RTState()
			}
		}
		rejuvPass(side)
	}
}

// applyAuras computes the round's auras for a side from its standing
// holders and its formation, replacing last round's.
func applyAuras(side []actor, f company.Formation) {
	rowOf := func(a actor) int {
		r, _, placed := f.Find(a.key)
		if !placed {
			return -1
		}
		return r
	}
	evade := map[int]int{}
	resolve := map[int]int{}
	rally := 0
	for _, a := range side {
		fx := a.char.ClassEffects()
		if fx == nil || a.char.Health < 1 {
			continue
		}
		row := rowOf(a)
		rally = max(rally, fx.Int(classes.AuraCompan))
		if row < 0 {
			continue // an unplaced member holds no row
		}
		evade[row] = max(evade[row], fx.Int(classes.AuraEvade))
		resolve[row] = max(resolve[row], fx.Int(classes.AuraResolv))
	}
	for _, a := range side {
		row := rowOf(a)
		a.char.Aura = characters.ClassAura{
			Evasion: evade[row],
			Resolve: max(resolve[row], rally),
		}
	}
}

// rejuvPass heals each ally that carries Rejuvenation by one round's share
// of it, and counts the round off. Healing stops at the wound limit.
func rejuvPass(side []actor) {
	for _, a := range side {
		rt := a.char.RT
		if rt == nil || rt.Rejuv <= 0 {
			continue
		}
		rt.Rejuv--
		if a.char.Health < 1 {
			continue
		}
		healed := a.char.ApplyHealthChange(rt.Per)
		if a.who.userId > 0 {
			events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
		}
		if healed > 0 {
			a.holder.say(fmt.Sprintf("Rejuvenation mends you. (%d healed)", healed),
				"Rejuvenation mends %s. ("+fmt.Sprint(healed)+" healed)", "")
		}
	}
}

// thornsBlow hurts a foe that struck an ally under Barkskin's Thornhide.
func thornsBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if defender.char.RT == nil || defender.char.RT.Thorns <= 0 || !r.Hit || r.DamageToTarget <= 0 || attacker.char.Health < 1 {
		return
	}
	dealt := -attacker.char.ApplyHealthChange(-defender.char.RT.Thorns)
	if dealt <= 0 {
		return
	}
	attacker.say(fmt.Sprintf("Thorns tear at you. (%d damage)", dealt), "Thorns tear at %s. ("+fmt.Sprint(dealt)+" damage)", "")
	if attacker.user != nil {
		roundExtraPlayers = append(roundExtraPlayers, attacker.user.UserId)
		events.AddToQueue(events.CharacterVitalsChanged{UserId: attacker.user.UserId})
	} else {
		roundExtraMobs = append(roundExtraMobs, attacker.mob.InstanceId)
	}
}
