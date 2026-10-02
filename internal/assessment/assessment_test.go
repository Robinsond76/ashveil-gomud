package assessment

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fighter is a member whose damage is its Level, so the tests set damage
// without combat's formula.
func fighter(key string, health, damage int, reach formationcombat.Reach) Member {
	c := &characters.Character{Name: key, Health: health, Level: damage}
	c.HealthMax.Value = health
	return Member{Key: company.MemberKey(key), Name: key, Char: c, Reach: reach}
}

func byLevel(atk, def *characters.Character) float64 { return float64(atk.Level) }

func loose(members ...Member) Side { return Side{Members: members} }

func TestEstimateBands(t *testing.T) {
	// ratio = (ownHP / foeDmg) / (foeHP / ownDmg); every side deals 10.
	cases := []struct {
		name  string
		foeHP int // against the company's 100
		risk  Risk
		close bool
	}{
		{"crushing", 10, Easy, false},            // 10
		{"at the easy line", 40, Easy, true},     // 2.5
		{"fair", 60, Fair, false},                // 1.67
		{"even", 100, Hard, true},                // 1
		{"grave", 160, Grave, false},             // 0.625
		{"just past grave", 260, Hopeless, true}, // 0.385
		{"hopeless", 400, Hopeless, false},       // 0.25
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := Estimate(loose(fighter("leader", 100, 10, 0)), loose(fighter("m:1", c.foeHP, 10, 0)), byLevel)
			assert.Equal(t, c.risk, res.Risk, "ratio %.3f", res.Ratio)
			assert.Equal(t, c.close, res.Close, "ratio %.3f", res.Ratio)
		})
	}
}

func TestEstimateWoundsAndNumbersChangeTheWord(t *testing.T) {
	foes := loose(fighter("m:1", 100, 10, 0), fighter("m:2", 100, 10, 0))
	whole := loose(fighter("leader", 100, 10, 0), fighter("companion:1", 100, 10, 0))
	assert.Equal(t, Hard, Estimate(whole, foes, byLevel).Risk)

	hurt := loose(fighter("leader", 30, 10, 0), fighter("companion:1", 30, 10, 0))
	assert.Equal(t, Hopeless, Estimate(hurt, foes, byLevel).Risk, "wounded members last less")

	alone := loose(fighter("leader", 100, 10, 0))
	assert.Equal(t, Hopeless, Estimate(alone, foes, byLevel).Risk, "a missing member is missed")
}

func TestEstimateReachFromTheFormation(t *testing.T) {
	var own company.Formation
	require.NoError(t, own.Place(company.LeaderMemberKey, 0, 0))
	require.NoError(t, own.Place("companion:1", 2, 2)) // back row, far column
	var foe company.Formation
	require.NoError(t, foe.Place("m:1", 0, 0))
	require.NoError(t, foe.Place("m:2", 1, 0)) // behind m:1

	ownSide := Side{Formation: own, HasFormation: true, Members: []Member{
		fighter("leader", 100, 10, formationcombat.ReachNone),
		fighter("companion:1", 100, 10, formationcombat.ReachNone),
	}}
	foeSide := Side{Formation: foe, HasFormation: true, Members: []Member{
		fighter("m:1", 100, 10, formationcombat.ReachNone),
		fighter("m:2", 100, 10, formationcombat.ReachNone),
	}}
	res := Estimate(ownSide, foeSide, byLevel)
	assert.Equal(t, []company.MemberKey{"companion:1"}, res.Unreaching, "the back-row melee reaches no one")
	assert.Equal(t, []company.MemberKey{"m:2"}, res.OutOfReach, "the one behind is shielded")

	// A bow reaches any depth of a column within one of its own: moved to
	// the middle column, no one is out of reach.
	own.Clear("companion:1")
	require.NoError(t, own.Place("companion:1", 2, 1))
	ownSide.Formation = own
	ownSide.Members[1].Reach = formationcombat.ReachAny
	res2 := Estimate(ownSide, foeSide, byLevel)
	assert.Empty(t, res2.Unreaching)
	assert.Empty(t, res2.OutOfReach)
	assert.Greater(t, res2.Ratio, res.Ratio, "more of the company can strike")

	// A member not placed fails open, as in combat.
	ownSide.Members[1].Reach = formationcombat.ReachNone
	ownSide.Members = append(ownSide.Members, fighter("companion:2", 100, 10, formationcombat.ReachNone))
	res3 := Estimate(ownSide, foeSide, byLevel)
	assert.NotContains(t, res3.Unreaching, company.MemberKey("companion:2"))
}

func TestEstimateWhenNoOneCanStrike(t *testing.T) {
	var own company.Formation
	require.NoError(t, own.Place(company.LeaderMemberKey, 0, 0))
	var foe company.Formation
	require.NoError(t, foe.Place("m:1", 0, 2))
	ownSide := Side{Formation: own, HasFormation: true, Members: []Member{fighter("leader", 100, 10, formationcombat.ReachNone)}}
	foeSide := Side{Formation: foe, HasFormation: true, Members: []Member{fighter("m:1", 10, 10, formationcombat.ReachNone)}}
	// Columns 1 and 3 are out of each other's range.
	res := Estimate(ownSide, foeSide, byLevel)
	assert.Equal(t, Hard, res.Risk)
	assert.True(t, res.Close, "neither can strike: it can't be told")

	// m:1 in the middle column's second rank, behind a foe the viewer
	// can't see (alive, but not listed): it strikes the leader, and the
	// leader's fists can't reach past the front.
	foe = company.Formation{}
	require.NoError(t, foe.Place("m:2", 0, 1))
	require.NoError(t, foe.Place("m:1", 1, 1))
	foeSide.Formation = foe
	foeSide.Alive = map[company.MemberKey]bool{"m:1": true, "m:2": true}
	assert.Equal(t, Hopeless, Estimate(ownSide, foeSide, byLevel).Risk, "they strike, you can't")

	assert.Equal(t, Easy, Estimate(ownSide, Side{}, byLevel).Risk, "no foes")

	// 33i1 review finding 1: nobody of the company can fight.
	res = Estimate(Side{}, foeSide, byLevel)
	assert.Equal(t, Hopeless, res.Risk)
	assert.False(t, res.Close)
}

func TestHeadlineAndLines(t *testing.T) {
	r := Report{Result: Result{Risk: Grave, Close: true}, Counted: []string{"you", "Tamsin"}, Missing: []string{"Ysolde (fled)"},
		Hurt: []string{"Tamsin (badly wounded)"}, NoReach: []string{"Tamsin"}, OutOfReach: []string{"the slinger"}, Allies: true}
	assert.Equal(t, "A grave risk for your company; it could go either way.", r.Headline())
	text := r.Text()
	for _, want := range []string{
		"Counted: you and Tamsin. Not with you: Ysolde (fled).",
		"Hurt: Tamsin (badly wounded).",
		"Can't reach any of them from where they stand: Tamsin.",
		"Out of your company's reach: the slinger.",
		"Allies here aren't counted.",
		"Not judged: spells, healing, guards and abilities, hidden foes, and anyone yet to come.",
	} {
		assert.Contains(t, text, want)
	}
	assert.NotContains(t, text, "Burdened")
	r.Burdened = []string{"you (burdened)"}
	assert.Contains(t, r.Text(), "Burdened among you: you (burdened).")
	assert.Equal(t, "Hopeless for your company; the odds look clear.", Report{Result: Result{Risk: Hopeless}}.Headline())
}

// Phase 33i2: the coordination line, with and without roles, and none for
// a lone foe.
func TestCoordinationLine(t *testing.T) {
	assert.Equal(t, "", Report{}.CoordinationLine())
	assert.Equal(t, "They fight as a band.", Report{Coordination: "a band"}.CoordinationLine())
	r := Report{Coordination: "a drilled company", Roles: []string{"two healers", "a guardian"}}
	assert.Equal(t, "They fight as a drilled company: two healers and a guardian among them.", r.CoordinationLine())
	assert.Contains(t, r.Text(), "\n  They fight as a drilled company")
}
