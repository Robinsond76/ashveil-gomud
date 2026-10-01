// Package assessment is Phase 33i1's company encounter assessment: how a
// player's company, as it stands now, measures up against an enemy group
// they can see. It is a read model. It never changes the game and never
// draws a random number: each member's damage is combat's dice-free
// expected damage, and reach is formationcombat's rule, both read fresh
// on every call. Players see only words (Risk, Close), never the ratio.
//
// Design: docs/designs/2026-10-01-phase-33i-company-assessment-enemy-roles-design.md,
// "Final implementation decisions: 33i1".
package assessment

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
)

// Risk is how dangerous the fight looks to the company.
type Risk string

const (
	Easy     Risk = "easy"
	Fair     Risk = "fair"
	Hard     Risk = "hard"
	Grave    Risk = "grave"
	Hopeless Risk = "hopeless"
)

// Band lines: a ratio at or above a line reads as that band.
const (
	easyAt  = 2.5
	fairAt  = 1.25
	hardAt  = 0.8
	graveAt = 0.4
	// near is how close to a band line (as a ratio, either side) an
	// estimate "could go either way".
	near = 1.1
)

// Phrase is the risk in a sentence: "a hard fight".
func (r Risk) Phrase() string {
	switch r {
	case Easy:
		return "an easy fight"
	case Fair:
		return "a fair fight"
	case Hard:
		return "a hard fight"
	case Grave:
		return "a grave risk"
	}
	return "hopeless"
}

// Member is one fighter on a side.
type Member struct {
	Key   company.MemberKey
	Name  string
	Char  *characters.Character
	Reach formationcombat.Reach
}

// Side is one side of the fight: its formation (HasFormation false when
// it has none, and every blow lands), who is alive in it for reach (Alive;
// nil: every listed member), and the members counted.
type Side struct {
	Formation    company.Formation
	HasFormation bool
	Alive        map[company.MemberKey]bool
	Members      []Member
}

// DamageFunc is a member's expected damage per round against a foe.
type DamageFunc func(atk, def *characters.Character) float64

// Result is the estimate. Ratio is internal (tests and tuning); players
// see Risk and Close only.
type Result struct {
	Risk  Risk
	Close bool // it could go either way
	Ratio float64
	// Unreaching are own members who can reach none of the foes now, and
	// OutOfReach foes none of the own side can reach, in side order.
	Unreaching []company.MemberKey
	OutOfReach []company.MemberKey
}

// Estimate weighs own against foes. Each member deals its expected damage,
// averaged over the members of the other side it can reach from where it
// stands; staying power is current health. The ratio is how many rounds
// the foes need to beat the company over how many the company needs to
// beat the foes.
func Estimate(own, foes Side, damage DamageFunc) Result {
	var res Result
	ownDamage, unreaching := sideDamage(own, foes, damage)
	foeDamage, _ := sideDamage(foes, own, damage)
	res.Unreaching = unreaching
	for _, f := range foes.Members {
		if !reachedByAny(own, foes, f) {
			res.OutOfReach = append(res.OutOfReach, f.Key)
		}
	}

	ownHealth, foeHealth := health(own), health(foes)
	switch {
	case len(foes.Members) == 0:
		res.Risk, res.Ratio = Easy, easyAt*10
		return res
	case ownDamage <= 0 && foeDamage <= 0:
		// Neither side can touch the other from where it stands.
		res.Risk, res.Close, res.Ratio = Hard, true, 1
		return res
	case ownDamage <= 0:
		res.Risk, res.Ratio = Hopeless, 0
		return res
	case foeDamage <= 0:
		res.Risk, res.Ratio = Easy, easyAt*10
		return res
	}
	foesLast := foeHealth / ownDamage // rounds the company needs
	ownLast := ownHealth / foeDamage  // rounds the foes need
	res.Ratio = ownLast / foesLast
	res.Risk = band(res.Ratio)
	res.Close = nearLine(res.Ratio)
	return res
}

func band(ratio float64) Risk {
	switch {
	case ratio >= easyAt:
		return Easy
	case ratio >= fairAt:
		return Fair
	case ratio >= hardAt:
		return Hard
	case ratio >= graveAt:
		return Grave
	}
	return Hopeless
}

// nearLine: near even, or near a band line.
func nearLine(ratio float64) bool {
	if ratio >= hardAt && ratio < fairAt {
		return true
	}
	for _, line := range []float64{easyAt, fairAt, hardAt, graveAt} {
		if ratio >= line/near && ratio < line*near {
			return true
		}
	}
	return false
}

func health(s Side) float64 {
	total := 0.0
	for _, m := range s.Members {
		if m.Char != nil && m.Char.Health > 0 {
			total += float64(m.Char.Health)
		}
	}
	return total
}

// sideDamage is a side's damage per round against the other side, and its
// members who can reach no one.
func sideDamage(atk, def Side, damage DamageFunc) (float64, []company.MemberKey) {
	total := 0.0
	var none []company.MemberKey
	for _, a := range atk.Members {
		sum, n := 0.0, 0
		for _, d := range def.Members {
			if reaches(atk, a, def, d) {
				sum += damage(a.Char, d.Char)
				n++
			}
		}
		if n == 0 {
			none = append(none, a.Key)
			continue
		}
		total += sum / float64(n)
	}
	return total, none
}

func reachedByAny(own, foes Side, f Member) bool {
	for _, a := range own.Members {
		if reaches(own, a, foes, f) {
			return true
		}
	}
	return false
}

// reaches is combat's reach rule (internal/hooks resolveAttackTarget):
// an attacker or target not placed in a formation fails open.
func reaches(atkSide Side, a Member, defSide Side, d Member) bool {
	if !atkSide.HasFormation || !defSide.HasFormation {
		return true
	}
	_, col, placed := atkSide.Formation.Find(a.Key)
	if !placed {
		return true
	}
	if _, _, placed := defSide.Formation.Find(d.Key); !placed {
		return true
	}
	return formationcombat.Legal(col, defSide.Formation, d.Key, aliveOf(defSide), a.Reach)
}

func aliveOf(s Side) map[company.MemberKey]bool {
	if s.Alive != nil {
		return s.Alive
	}
	alive := make(map[company.MemberKey]bool, len(s.Members))
	for _, m := range s.Members {
		alive[m.Key] = true
	}
	return alive
}
