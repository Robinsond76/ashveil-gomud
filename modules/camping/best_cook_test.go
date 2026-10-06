package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// presentCompanions is the formation seam with the presence one (Phase
// 35c): these companion IDs walk with the leader.
type presentCompanions struct {
	formationMap
	present []int
}

func (p presentCompanions) CompanionsWithLeader(int) []int { return p.present }

// bestCookWorld is a lit camp with the two-recipe menu, the meat in the
// cargo and the thyme in the pack, and live companion mobs found through
// the native company seams.
func bestCookWorld(t *testing.T) (*raidWorld, *fakeCargo) {
	t.Helper()
	forageSpecs(t)
	w := newRaidWorld(t, 0)
	w.m.campCfg.Recipes = []campRecipe{
		{Output: 30020, Inputs: []int{29, 30018}, Skill: "cooking", MinLevel: 2},
		{Output: 30021, Inputs: []int{29}, Skill: "cooking", MinLevel: 1},
	}
	w.m.inBattle = func(int) bool { return false }
	cargo := newFakeCargo(100000)
	useCargo(t, cargo)
	cargo.stacks[29] = 1
	w.user.Character.StoreItem(items.New(30018))
	skills.SetTestData([]*skills.Skill{{SkillId: "cooking", Name: "Cooking", MaxLevel: 4}}, nil)
	t.Cleanup(func() { skills.SetTestData(nil, nil) })
	return w, cargo
}

// TestCampCookUsesTheBestCookPresent: the highest Cooking among the leader
// and the living companions in the camp room cooks, named; ties go to the
// leader, then the lowest companion ID. The companions' ranks are read from
// their live mobs through the real company seams.
func TestCampCookUsesTheBestCookPresent(t *testing.T) {
	w, cargo := bestCookWorld(t)
	brannoc := testMob(t, 97301, "Brannoc", 100)
	brannoc.Character.Health = 10
	mira := testMob(t, 97302, "Mira", 100)
	mira.Character.Health = 10
	fallen := testMob(t, 97303, "Hild", 100)
	fallen.Character.Health = 0
	setPresence(t, presentCompanions{formationMap: formationMap{1: 97301, 2: 97302, 3: 97303}, present: []int{3, 2, 1}})

	// Nobody can cook: the refusal names the best rank there is.
	text := w.m.cook(w.user, w.room, nil)
	assert.Contains(t, text, "it needs cooking 1", "no Cooking at all")
	assert.Contains(t, text, "the best in your company here is you, with cooking 0")

	// A companion with Cooking 1 cooks the simpler dish, named.
	mira.Character.SetSkill("cooking", 1)
	text = w.m.cook(w.user, w.room, nil)
	assert.Contains(t, text, "Mira cooks")
	assert.Contains(t, text, "seared game meat")
	assert.Equal(t, 1, cargo.stacks[30021])

	// A tie at Cooking 2 goes to the lowest companion ID; a fallen member
	// (no health) never cooks, whatever its rank.
	cargo.stacks[29] = 1
	mira.Character.SetSkill("cooking", 2)
	brannoc.Character.SetSkill("cooking", 2)
	fallen.Character.SetSkill("cooking", 4)
	text = w.m.cook(w.user, w.room, nil)
	assert.Contains(t, text, "Brannoc cooks")
	assert.Contains(t, text, "thyme-roasted game")

	// The leader wins a tie.
	cargo.stacks[29] = 1
	w.user.Character.StoreItem(items.New(30018))
	w.user.Character.Skills = map[string]int{"cooking": 2}
	text = w.m.cook(w.user, w.room, nil)
	assert.True(t, strings.HasPrefix(text, "You cook"), text)
}

// TestCookingViewNamesTheBestCook: the capability read model (GMCP
// Char.Capabilities) shows the best cook's rank and name.
func TestCookingViewNamesTheBestCook(t *testing.T) {
	w, _ := bestCookWorld(t)
	brannoc := testMob(t, 97301, "Brannoc", 100)
	brannoc.Character.Health = 10
	brannoc.Character.SetSkill("cooking", 3)
	setPresence(t, presentCompanions{formationMap: formationMap{1: 97301}, present: []int{1}})

	v, ok := w.m.CookingCapability(w.user.UserId)
	require.True(t, ok)
	assert.Equal(t, camping.CookingView{Rank: 3, Ready: true, Description: v.Description}, v)
	assert.Contains(t, v.Description, "Brannoc is the best cook here (cooking rank 3)")
}

// TestCampCookCommandUsesTheBestCook: the real "camp cook" entry point
// cooks with the best companion present and names them.
func TestCampCookCommandUsesTheBestCook(t *testing.T) {
	w, cargo := bestCookWorld(t)
	mira := testMob(t, 97302, "Mira", 100)
	mira.Character.Health = 10
	mira.Character.SetSkill("cooking", 2)
	setPresence(t, presentCompanions{formationMap: formationMap{1: 97302}, present: []int{1}})

	messages := captureMessages(t)
	_, err := w.m.userCommand("cook", w.user, w.room, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	out := strings.Join(*messages, "")
	assert.Contains(t, out, "Mira cooks")
	assert.Contains(t, out, "thyme-roasted game")
	assert.Equal(t, 1, cargo.stacks[30020])
}

func setPresence(t *testing.T, p presentCompanions) {
	t.Helper()
	company.SetFormationProvider(p)
	t.Cleanup(func() { company.SetFormationProvider(nil) })
}
