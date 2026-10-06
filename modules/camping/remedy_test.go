package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 55: `camp prepare remedy` makes the remedy for each ailing member
// from gathered herbs the company carries, and ends the ailment.

const (
	thymeID    = 30018
	mushroomID = 30007
	mintID     = 30009
)

// cureFake is a survival seam whose needs a cure edits.
type cureFake struct {
	*fakeSurvival
	cured []string
}

func (f *cureFake) CureAilment(_ int, key survival.MemberKey, kind string) (bool, error) {
	for i := range f.needs {
		if f.needs[i].Key != key || survival.AilmentBattles(f.needs[i].Needs, kind) == 0 {
			continue
		}
		r := survival.NewRegistry()
		_ = r.Ensure(7, key)
		_ = r.PutNeeds(7, key, f.needs[i].Needs)
		cured, err := r.CureAilment(7, key, kind)
		f.needs[i].Needs = r.MustNeedsFor(7, key)
		if cured {
			f.cured = append(f.cured, string(key)+":"+kind)
		}
		return cured, err
	}
	return false, nil
}

func ailingWorld(t *testing.T, herbs stock, needs survival.Needs) (*raidWorld, *cureFake) {
	t.Helper()
	w, _, _ := prepWorld(t, herbs)
	f := &cureFake{fakeSurvival: &fakeSurvival{needs: []survival.MemberNeeds{{Key: survival.LeaderMemberKey, Name: w.user.Character.Name, Needs: needs}}}}
	w.m.survival = f
	return w, f
}

func fit() survival.Needs { return survival.FullNeeds() }

func TestARemedyCuresTheAilmentAndSpendsItsHerbs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ail   func(*survival.Needs)
		herbs stock
		left  stock
		text  string
	}{
		{"chill", func(n *survival.Needs) { n.Chill = 3 }, stock{thymeID: 3}, stock{thymeID: 1}, "thyme tea"},
		{"gut-ache", func(n *survival.Needs) { n.GutAche = 2 }, stock{thymeID: 1, mushroomID: 1}, stock{thymeID: 0, mushroomID: 0}, "thyme and mushroom tisane"},
		{"fever", func(n *survival.Needs) { n.Fever = 5 }, stock{mintID: 1, thymeID: 1}, stock{mintID: 0, thymeID: 0}, "cooling fever draught"},
	} {
		n := fit()
		tc.ail(&n)
		w, f := ailingWorld(t, tc.herbs, n)
		text := prepare(w, "remedy")
		assert.Contains(t, text, tc.text, tc.name)
		assert.Contains(t, text, "breaks", tc.name)
		assert.Equal(t, tc.left, tc.herbs, tc.name+": only its herbs are spent")
		require.Len(t, f.cured, 1, tc.name)
		assert.False(t, survival.HasAilment(f.needs[0].Needs), tc.name)
	}
}

func TestARemedyShortOfHerbsConsumesNothing(t *testing.T) {
	n := fit()
	n.Fever, n.Chill = 5, 2
	herbs := stock{thymeID: 2, mintID: 1} // chill 2 + fever 1 thyme = 3 needed
	w, f := ailingWorld(t, herbs, n)
	text := prepare(w, "remedy")
	assert.Contains(t, text, "You need 3 wild thyme (you carry 2)")
	assert.Contains(t, text, "Nothing was used")
	assert.Equal(t, stock{thymeID: 2, mintID: 1}, herbs)
	assert.Empty(t, f.cured)
	assert.True(t, survival.HasAilment(f.needs[0].Needs))
}

func TestARemedyForAWellMemberChangesNothing(t *testing.T) {
	herbs := stock{thymeID: 5}
	w, f := ailingWorld(t, herbs, fit())
	assert.Contains(t, prepare(w, "remedy"), "is not ill")
	assert.Equal(t, 5, herbs[thymeID])
	assert.Empty(t, f.cured)
}

func TestARemedyIsRefusedMidRestAndWithoutACampHere(t *testing.T) {
	n := fit()
	n.Chill = 2
	w, _ := ailingWorld(t, stock{thymeID: 4}, n)
	w.m.inBattle = func(int) bool { return true }
	assert.Contains(t, prepare(w, "remedy"), "middle of a fight")
}

func TestSuppliesAndStatusMentionRemediesAndAilments(t *testing.T) {
	n := fit()
	n.Chill = 2
	w, _ := ailingWorld(t, stock{thymeID: 1}, n)
	supplies := w.m.suppliesCommand(w.user)
	assert.Contains(t, supplies, "thyme tea for chill: 2 wild thyme (you carry wild thyme 1/2)")
	assert.Contains(t, prepare(w, "status"), "has a chill (2 battles left): camp prepare remedy")
}
