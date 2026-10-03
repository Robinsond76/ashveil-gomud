package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Game-loop-owned, never serialized. Membership includes every battle a
// shared enemy fights; one actor fills once, independent of player count.
type tempoFight struct {
	user      int
	start, id uint64
}
type tempoState struct {
	char   *characters.Character
	epoch  uint64
	fights map[tempoFight]bool
	meter  combat.Meter
}

var tempoMeters = map[caster]*tempoState{}
var tempoTurns = map[caster]int{}
var tempoBlocked = map[caster]bool{}
var tempoChanted = map[caster]bool{}
var tempoLostTold = map[caster]bool{}
var tempoFights = map[caster]map[tempoFight]bool{}
var tempoActive bool
var tempoRate = combat.Tempo

// UseTempoForTest pins cadence for fixtures testing other round mechanics.
// Balance and tempo integration tests restore the real calculation.
func UseTempoForTest(f func(*characters.Character) float64) (restore func()) {
	prev := tempoRate
	if f == nil {
		tempoRate = combat.Tempo
	} else {
		tempoRate = f
	}
	return func() { tempoRate = prev }
}

func ResetTempoForTest() {
	clear(tempoMeters)
	clear(tempoTurns)
	clear(tempoBlocked)
	clear(tempoChanted)
	clear(tempoLostTold)
	tempoActive = false
	clear(tempoFights)
}

// Snapshot managed membership once per pass, not once per actor per battle.
func snapshotTempoFights() {
	clear(tempoFights)
	add := func(who caster, f tempoFight) {
		if tempoFights[who] == nil {
			tempoFights[who] = map[tempoFight]bool{}
		}
		tempoFights[who][f] = true
	}
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		if !ok {
			continue
		}
		f := tempoFight{uid, b.StartRound, b.FightID}
		add(caster{userId: uid}, f)
		for id := range b.Enemies {
			add(caster{mobId: id}, f)
		}
	}
}

func tempoMembership(who caster) map[tempoFight]bool {
	if tempoActive {
		if who.mobId > 0 {
			if owner, _, ok := company.LeaderAndKeyForInstance(who.mobId); ok {
				if m := mobs.GetInstance(who.mobId); m != nil {
					if b, live := battle.Current(owner); live && m.Character.RoomId == b.RoomId {
						return tempoFights[caster{userId: owner}]
					}
				}
			}
		}
		return tempoFights[who]
	}
	owner := who.userId
	if who.mobId > 0 {
		owner, _, _ = company.LeaderAndKeyForInstance(who.mobId)
		if m := mobs.GetInstance(who.mobId); m != nil {
			if b, live := battle.Current(owner); !live || m.Character.RoomId != b.RoomId {
				owner = 0
			}
		}
	}
	result := map[tempoFight]bool{}
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		if ok && (uid == owner || who.mobId > 0 && b.Has(who.mobId)) {
			result[tempoFight{uid, b.StartRound, b.FightID}] = true
		}
	}
	return result
}

func sharesTempoFight(a, b map[tempoFight]bool) bool {
	for f := range a {
		if b[f] {
			return true
		}
	}
	return false
}

func fillTempo(who caster, c *characters.Character) {
	if _, allocated := tempoTurns[who]; allocated {
		return
	}
	fights := tempoMembership(who)
	st := tempoMeters[who]
	if st == nil || st.char != c ||
		(len(fights) > 0 || len(st.fights) > 0) && !sharesTempoFight(fights, st.fights) ||
		len(fights) == 0 && st.epoch != c.CombatEpoch {
		st = &tempoState{char: c}
		tempoMeters[who] = st
	}
	st.epoch, st.fights = c.CombatEpoch, fights
	tempoTurns[who] = st.meter.Fill(tempoRate(c), int(configs.GetCombatConfig().MaxTurnsPerRound))
	if a := c.Aggro; a != nil && (a.Type == characters.SpellCast || a.RoundsWaiting > 0) {
		tempoBlocked[who] = true
		if a.Type == characters.SpellCast {
			tempoChanted[who] = true
		}
	}
}

func beginTempoRound() {
	tempoActive = true
	snapshotTempoFights()
	clear(tempoTurns)
	clear(tempoBlocked)
	clear(tempoChanted)
	clear(tempoLostTold)
	pruneTempo()
	for _, id := range users.GetOnlineUserIds() {
		u := users.GetByUserId(id)
		if u != nil && u.Character != nil && u.Character.Health > 0 && (u.Character.Aggro != nil || len(tempoMembership(caster{userId: id})) > 0) {
			fillTempo(caster{userId: id}, u.Character)
		}
	}
	for _, id := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(id)
		if m != nil && m.Character.Health > 0 && !m.Character.CombatWithdrawn && (m.Character.Aggro != nil || len(tempoMembership(caster{mobId: id})) > 0) {
			fillTempo(caster{mobId: id}, &m.Character)
		}
	}
}

func pruneTempo() {
	for who, st := range tempoMeters {
		var c *characters.Character
		if who.userId > 0 {
			if u := users.GetByUserId(who.userId); u != nil {
				c = u.Character
			}
		} else if m := mobs.GetInstance(who.mobId); m != nil {
			c = &m.Character
		}
		fights := tempoMembership(who)
		if c == nil || c != st.char || c.Health < 1 || c.CombatWithdrawn ||
			len(fights) == 0 && (c.Aggro == nil || len(st.fights) > 0) ||
			len(st.fights) > 0 && !sharesTempoFight(fights, st.fights) {
			delete(tempoMeters, who)
		}
	}
}

// A second pass can perform only an earned physical turn. Spells, waits,
// wind-ups, withdrawals and skipped actions never run twice.
func extraTempoTurn(who caster, c *characters.Character) bool {
	return tempoTurns[who] > 1 && !tempoBlocked[who] && c.Health > 0 && c.Aggro != nil &&
		(c.Aggro.Type == characters.DefaultAttack || c.Aggro.Type == characters.Shooting || c.Aggro.Type == characters.BackStab) && c.Aggro.RoundsWaiting == 0
}

func tempoStatusCostsAction(h statusHolder) bool {
	who := caster{userId: h.ref.UserId, mobId: h.ref.MobInstanceId}
	if tempoLostTold[who] {
		return true
	}
	if statusCostsAction(h) {
		tempoLostTold[who], tempoBlocked[who] = true, true
		return true
	}
	return false
}
