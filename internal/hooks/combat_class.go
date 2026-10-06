package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
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
		applyAuras(sideActors(u, room), f)
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
