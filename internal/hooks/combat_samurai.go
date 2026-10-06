package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
)

// Phase 39b: the Samurai's lineage. Iaijutsu is in the blow (internal/combat)
// and the opening meter (combat_tempo.go); this file is what a battle round
// keeps for Focus, Zanshin, Vengeance and Bodyguard. All of it is runtime
// battle state on the character's ClassRT, gone with the fight.

// samuraiRound runs at a round's start for one side: Focus counts a quiet
// round for each holder no blow landed on in the last one, and Vengeance
// learns how many of the side have fallen (the most standing in the battle
// less those standing now).
func samuraiRound(side []actor) {
	for _, a := range side {
		fx := a.char.ClassEffects()
		if fx == nil {
			continue
		}
		if fx.Has(classes.Focus) {
			rt := a.char.RTState()
			if rt.QuietStarted {
				if rt.Struck {
					rt.Quiet = 0
				} else {
					rt.Quiet++
				}
			}
			rt.QuietStarted, rt.Struck = true, false
		}
		if fx.Has(classes.Vengeance) {
			rt := a.char.RTState()
			rt.SidePeak = max(rt.SidePeak, len(side))
			a.char.Aura.Fallen = rt.SidePeak - len(side)
		}
	}
}

// samuraiBlow is what a resolved blow does for a Samurai: a blow that lands
// on a holder of Focus resets it, and a Samurai that fells a foe has its
// action meter topped up by Zanshin, once a round.
func samuraiBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if !r.Hit || r.DamageToTarget <= 0 {
		return
	}
	if rt := defender.char.RT; rt != nil && defender.char.ClassEffects().Has(classes.Focus) {
		rt.Struck = true
	}
	n := attacker.char.ClassEffects().Int(classes.Zanshin)
	if n <= 0 || defender.char.Health >= 1 || attacker.char.Health < 1 {
		return
	}
	rt := attacker.char.RTState()
	round := combatRound.Load()
	if rt.ZanshinRound == round {
		return
	}
	who := caster{userId: attacker.ref.UserId, mobId: attacker.ref.MobInstanceId}
	st := tempoMeters[who]
	if st == nil || st.char != attacker.char {
		return
	}
	rt.ZanshinRound = round
	st.meter.Points = min(99, st.meter.Points+float64(n))
	attacker.say("Your stillness after the cut carries you into the next.", "%s's stillness after the cut carries them into the next.", " (zanshin)")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: attacker.char.RoomId, Source: attacker.ref, Status: "Zanshin", Outcome: combatstream.OutcomeSucceeded})
}

// bodyguardLeft is how many times a battle the holder can still step in for
// the company leader.
func bodyguardLeft(c *characters.Character) int {
	n := c.ClassEffects().Int(classes.Bodyguard)
	if n <= 0 {
		return 0
	}
	if c.RT == nil {
		return n
	}
	return max(0, n-c.RT.Bodyguards)
}
