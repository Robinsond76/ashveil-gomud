package camping

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/items"
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
	for _, spec := range []*items.ItemSpec{
		{ItemId: thymeID, Name: "wild thyme", NameSimple: "thyme", Type: items.Botanical},
		{ItemId: mushroomID, Name: "mushroom", Type: items.Food},
		{ItemId: mintID, Name: "glacial mint", NameSimple: "mint", Type: items.Botanical},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
	// Phase 56: these tests are about the cure, so the leader knows every remedy.
	for _, kind := range []string{survival.AilmentChill, survival.AilmentGutAche, survival.AilmentFever} {
		cookbook.LearnRemedy(w.user.Character, kind)
	}
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

// Phase 56: remedies are discovered like dishes. The chill's thyme tea is
// common knowledge; the rest must be worked out with a mix of herbs.
func forgetRemedies(w *raidWorld) { w.user.Character.SetMiscData(cookbook.BookKey, "") }

func TestABareRemedyOnlyMakesKnownRemediesAndSaysSo(t *testing.T) {
	n := fit()
	n.GutAche, n.Chill = 2, 2
	herbs := stock{thymeID: 4, mushroomID: 1}
	w, f := ailingWorld(t, herbs, n)
	forgetRemedies(w)
	text := prepare(w, "remedy")
	assert.Contains(t, text, "no remedy for it", "gut-ache is not known yet")
	assert.Contains(t, text, "thyme tea", "the chill's tea is common knowledge")
	assert.Equal(t, stock{thymeID: 2, mushroomID: 1}, herbs, "only the chill's herbs went")
	require.Len(t, f.cured, 1)
	assert.Greater(t, f.needs[0].Needs.GutAche, 0)
}

func TestTryingTheRightMixCuresAndLearnsTheRemedyOnce(t *testing.T) {
	n := fit()
	n.GutAche = 2
	herbs := stock{thymeID: 2, mushroomID: 2}
	w, f := ailingWorld(t, herbs, n)
	forgetRemedies(w)
	text := prepare(w, "remedy", "with", "thyme", "mushroom")
	assert.Contains(t, text, "breaks")
	assert.Contains(t, text, "worked out a new remedy")
	assert.Equal(t, stock{thymeID: 1, mushroomID: 1}, herbs)
	require.Len(t, f.cured, 1)
	assert.True(t, cookbook.KnowsRemedy(w.user.Character, survival.AilmentGutAche, false))
	// The next ailment is cured without naming herbs.
	f.needs[0].Needs.GutAche = 3
	assert.Contains(t, prepare(w, "remedy"), "tisane")
	assert.Equal(t, stock{thymeID: 0, mushroomID: 0}, herbs)
}

func TestAWrongMixSpendsItsHerbsAndTeachesNothing(t *testing.T) {
	n := fit()
	n.GutAche = 2
	herbs := stock{thymeID: 2, mushroomID: 1}
	w, f := ailingWorld(t, herbs, n)
	forgetRemedies(w)
	text := prepare(w, "remedy", "with", "thyme", "thyme", "mushroom")
	assert.Contains(t, text, "does nothing")
	assert.Equal(t, stock{thymeID: 0, mushroomID: 0}, herbs)
	assert.Empty(t, f.cured)
	assert.False(t, cookbook.KnowsRemedy(w.user.Character, survival.AilmentGutAche, false))
}

func TestTheRightMixForNobodyIllIsAlsoSpentAndNotLearned(t *testing.T) {
	herbs := stock{thymeID: 1, mushroomID: 1}
	w, f := ailingWorld(t, herbs, fit())
	forgetRemedies(w)
	assert.Contains(t, prepare(w, "remedy", "with", "thyme", "mushroom"), "does nothing")
	assert.Equal(t, stock{thymeID: 0, mushroomID: 0}, herbs, "a guess always costs")
	assert.Empty(t, f.cured)
	assert.False(t, cookbook.KnowsRemedy(w.user.Character, survival.AilmentGutAche, false))
}

func TestSuppliesListOnlyKnownRemedies(t *testing.T) {
	n := fit()
	w, _ := ailingWorld(t, stock{thymeID: 1}, n)
	forgetRemedies(w)
	supplies := w.m.suppliesCommand(w.user)
	assert.Contains(t, supplies, "thyme tea")
	assert.NotContains(t, supplies, "tisane")
	assert.NotContains(t, supplies, "fever draught")
}

// 56 review: a mix is one dose. With two members ill, the right mix cures
// the first and spends exactly the herbs named, never more, and never
// refuses in a way that would confirm the mix for free.
func TestAMixIsOneDoseForOneMember(t *testing.T) {
	n := fit()
	n.GutAche = 2
	herbs := stock{thymeID: 1, mushroomID: 1}
	w, f := ailingWorld(t, herbs, n)
	forgetRemedies(w)
	mira := &characters.Character{Name: "Mira", RoomId: w.user.Character.RoomId}
	w.m.companionsOf = func(int) (map[int]*characters.Character, []int) {
		return map[int]*characters.Character{1: mira}, []int{1}
	}
	f.needs = append(f.needs, survival.MemberNeeds{Key: survival.CompanionMemberKey(1), Name: "Mira", Needs: n})
	text := prepare(w, "remedy", "with", "thyme", "mushroom")
	assert.Contains(t, text, "breaks")
	assert.NotContains(t, text, "Nothing was used")
	assert.Equal(t, stock{thymeID: 0, mushroomID: 0}, herbs, "exactly the named herbs")
	assert.Len(t, f.cured, 1, "one dose, one member")
	assert.True(t, cookbook.KnowsRemedy(w.user.Character, survival.AilmentGutAche, false))
}
