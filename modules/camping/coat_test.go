package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43b weapon poisons: the coat command and the camp preparation list.

const (
	bitterleafVial = 280
	leechbaneVial  = 281
	leadrootVial   = 282
)

// coatFixture is a leader with a sword and a live companion Bran (a sword
// and a dagger) in the same room, holding the given vials.
func coatFixture(t *testing.T, vials stock) (*CampingModule, *users.UserRecord, *characters.Character, *fakeStore) {
	t.Helper()
	bran := member(t, "Bran", 100, testSwordID, testDaggerID)
	module, user, store := sharpenFixture(t, map[int]*characters.Character{1: bran})
	armed(user.Character, testSwordID, 0)
	vials.install(module)
	return module, user, bran, store
}

func coat(t *testing.T, module *CampingModule, user *users.UserRecord, cmd string) string {
	t.Helper()
	messages := captureMessages(t)
	_, err := module.coatCommand(cmd, user, eligibleRoom(), 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return strings.Join(*messages, "\n")
}

func campPoison(t *testing.T, module *CampingModule, user *users.UserRecord, cmd string) string {
	t.Helper()
	messages := captureMessages(t)
	_, err := module.userCommand("poison "+cmd, user, eligibleRoom(), 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return strings.Join(*messages, "\n")
}

func TestCoatSpendsOneDoseAndCoatsOneBlade(t *testing.T) {
	vials := stock{bitterleafVial: 2}
	module, user, bran, _ := coatFixture(t, vials)

	text := coat(t, module, user, "bitterleaf")
	assert.Contains(t, text, "You coat your")
	assert.Equal(t, 1, vials[bitterleafVial], "one dose")
	main := user.Character.Equipment.Weapon
	assert.True(t, main.Coated(module.now()))
	assert.Equal(t, "bitterleaf", main.CoatKind)
	assert.Equal(t, items.CoatContacts, main.CoatContacts)
	assert.Equal(t, module.now().Add(10*time.Minute).Unix(), main.CoatExpires)
	assert.False(t, bran.Equipment.Weapon.Coated(module.now()), "only the one blade")

	again := coat(t, module, user, "bitterleaf")
	assert.Contains(t, again, "already carries a coating")
	assert.Equal(t, 1, vials[bitterleafVial], "a refused coat spends nothing")
	assert.Equal(t, items.CoatContacts, user.Character.Equipment.Weapon.CoatContacts, "never refreshed")

	other := coat(t, module, user, "leechbane")
	assert.Contains(t, other, "already carries a coating", "never replaced either")
	assert.Equal(t, "bitterleaf", user.Character.Equipment.Weapon.CoatKind)
}

func TestCoatCompanionOffHandAndSharpenCoexist(t *testing.T) {
	vials := stock{leadrootVial: 1}
	module, user, bran, _ := coatFixture(t, vials)
	bran.Equipment.Offhand.Sharpen(1, 20)

	text := coat(t, module, user, "leadroot bran off")
	assert.Contains(t, text, "Bran's")
	assert.Equal(t, "leadroot", bran.Equipment.Offhand.CoatKind)
	assert.Equal(t, 20, bran.Equipment.Offhand.SharpStrikes, "the edge is untouched")
	assert.Equal(t, 0, vials[leadrootVial])
	assert.Contains(t, coat(t, module, user, "status"), "Bran off")
}

func TestCoatClearLosesTheDoseAndAllowsRecoat(t *testing.T) {
	vials := stock{bitterleafVial: 2}
	module, user, _, _ := coatFixture(t, vials)
	coat(t, module, user, "bitterleaf")
	assert.Contains(t, coat(t, module, user, "clear"), "The dose is lost")
	assert.Equal(t, 1, vials[bitterleafVial], "clearing returns no dose")
	assert.False(t, user.Character.Equipment.Weapon.Coated(module.now()))
	assert.Contains(t, coat(t, module, user, "clear"), "carries no coating")
	coat(t, module, user, "bitterleaf")
	assert.Equal(t, 0, vials[bitterleafVial])
}

func TestCoatRefusals(t *testing.T) {
	vials := stock{bitterleafVial: 1}
	module, user, bran, _ := coatFixture(t, vials)
	assert.Contains(t, coat(t, module, user, "nightshade"), "no poison called")
	assert.Contains(t, coat(t, module, user, "leechbane"), "no Leechbane vial")
	assert.Contains(t, coat(t, module, user, "bitterleaf nobody"), "Nobody in your company")
	assert.Contains(t, coat(t, module, user, "bitterleaf self off"), "no bladed weapon in the off hand")
	armed(user.Character, testClubID, 0)
	assert.Contains(t, coat(t, module, user, "bitterleaf"), "Poison goes on blades only")

	armed(user.Character, testSwordID, 0)
	bran.SetAggro(0, 55, characters.DefaultAttack)
	assert.Contains(t, coat(t, module, user, "bitterleaf"), "middle of a fight")
	bran.Aggro = nil
	module.travelling = func(int) bool { return true }
	assert.Contains(t, coat(t, module, user, "bitterleaf"), "while travelling")
	module.travelling = func(int) bool { return false }
	module.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100, Rest: &camping.RestSession{State: camping.Resting}}
	assert.Contains(t, coat(t, module, user, "bitterleaf"), "while resting")
	assert.Equal(t, 1, vials[bitterleafVial], "every refusal spent nothing")
	assert.False(t, user.Character.Equipment.Weapon.Coated(module.now()))
}

func TestLapsedCoatingCanBeReplaced(t *testing.T) {
	vials := stock{bitterleafVial: 1}
	module, user, _, _ := coatFixture(t, vials)
	w := &user.Character.Equipment.Weapon
	w.CoatKind, w.CoatContacts, w.CoatExpires = "leadroot", 3, module.now().Add(-time.Minute).Unix()
	require.False(t, user.Character.Equipment.Weapon.Coated(module.now()), "the time ran out")
	coat(t, module, user, "bitterleaf")
	assert.Equal(t, "bitterleaf", user.Character.Equipment.Weapon.CoatKind)
	assert.Equal(t, items.CoatContacts, user.Character.Equipment.Weapon.CoatContacts)
}

func TestCampPoisonNeedsACamp(t *testing.T) {
	module, user, _, _ := coatFixture(t, stock{})
	assert.Contains(t, campPoison(t, module, user, ""), "You have no camp")
}

func campedFixture(t *testing.T, vials stock) (*CampingModule, *users.UserRecord, *characters.Character, *fakeStore) {
	module, user, bran, store := coatFixture(t, vials)
	module.camps[7] = camping.Camp{LeaderUserID: 7, RoomID: 100}
	return module, user, bran, store
}

func TestCampPoisonAssignPreviewApply(t *testing.T) {
	vials := stock{bitterleafVial: 1, leechbaneVial: 1, leadrootVial: 1}
	module, user, bran, store := campedFixture(t, vials)

	assert.Contains(t, campPoison(t, module, user, "assign self main bitterleaf"), "Nothing is spent")
	campPoison(t, module, user, "assign bran main leechbane")
	campPoison(t, module, user, "assign bran off leadroot")
	assert.Equal(t, 3, vials[bitterleafVial]+vials[leechbaneVial]+vials[leadrootVial], "assigning spends nothing")
	assert.False(t, user.Character.Equipment.Weapon.Coated(module.now()))
	require.Len(t, store.saved.PoisonPlans[7], 3, "the list persists")

	preview := campPoison(t, module, user, "preview")
	assert.Contains(t, preview, "Needed: Bitterleaf 1, Leechbane 1, Leadroot 1.")
	assert.Contains(t, preview, "Ready to apply to 3 blades.")
	assert.Equal(t, 1, vials[leechbaneVial], "preview spends nothing")

	assert.Contains(t, campPoison(t, module, user, "apply"), "You coat 3 blades")
	assert.Equal(t, "bitterleaf", user.Character.Equipment.Weapon.CoatKind)
	assert.Equal(t, "leechbane", bran.Equipment.Weapon.CoatKind)
	assert.Equal(t, "leadroot", bran.Equipment.Offhand.CoatKind, "each hand its own poison")
	assert.Equal(t, 0, vials[bitterleafVial]+vials[leechbaneVial]+vials[leadrootVial])

	// A repeat finds every blade already coated: no dose, no refresh.
	again := campPoison(t, module, user, "apply")
	assert.Contains(t, again, "no application is needed")
	assert.Equal(t, items.CoatContacts, user.Character.Equipment.Weapon.CoatContacts)
}

func TestCampPoisonBlockersSpendNothingAtAll(t *testing.T) {
	vials := stock{bitterleafVial: 1, leechbaneVial: 0}
	module, user, bran, _ := campedFixture(t, vials)
	campPoison(t, module, user, "assign self main bitterleaf")
	campPoison(t, module, user, "assign bran main leechbane")

	text := campPoison(t, module, user, "apply")
	assert.Contains(t, text, "Blocked, nothing spent")
	assert.Contains(t, text, "Short of Leechbane: need 1, have 0.")
	assert.Equal(t, 1, vials[bitterleafVial], "no partial application")
	assert.False(t, user.Character.Equipment.Weapon.Coated(module.now()))

	// Changed gear and a foreign coating are blockers too, all listed together.
	vials[leechbaneVial] = 1
	bran.Equipment.Weapon = items.Item{}
	user.Character.Equipment.Weapon.Coat("mirethorn", items.CoatExpiry(module.now()), 4, module.now())
	text = campPoison(t, module, user, "apply")
	assert.Contains(t, text, "Bran main: no usable blade here.")
	assert.Contains(t, text, "already carries another coating")
	assert.Equal(t, 1, vials[bitterleafVial])
	assert.Equal(t, 1, vials[leechbaneVial])
	assert.Equal(t, "mirethorn", user.Character.Equipment.Weapon.CoatKind, "the old coating is untouched")
}

func TestCampPoisonUnassignAndDeadRowsAreIgnored(t *testing.T) {
	vials := stock{bitterleafVial: 2, leechbaneVial: 2}
	module, user, _, store := campedFixture(t, vials)
	campPoison(t, module, user, "assign self main bitterleaf")
	campPoison(t, module, user, "assign bran main leechbane")
	assert.Contains(t, campPoison(t, module, user, "unassign bran main"), "Cleared")
	assert.Contains(t, campPoison(t, module, user, "unassign bran main"), "has no assignment")
	assert.Len(t, store.saved.PoisonPlans[7], 1)

	campPoison(t, module, user, "assign bran off leechbane")
	module.companionsOf = func(int) (map[int]*characters.Character, []int) { return nil, nil } // Bran left
	text := campPoison(t, module, user, "preview")
	assert.Contains(t, text, "Ready to apply to 1 blades.", "a departed member's row is ignored")
	assert.Contains(t, campPoison(t, module, user, "apply"), "You coat 1 blades")
	assert.Equal(t, 2, vials[leechbaneVial])
}

func TestCampPoisonPlanSurvivesReload(t *testing.T) {
	module, user, _, store := campedFixture(t, stock{})
	campPoison(t, module, user, "assign self main leadroot")

	reloaded := newTestModule(store, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	reloaded.load()
	assert.Equal(t, []PoisonAssign{{Member: 0, Hand: "main", Poison: "leadroot"}}, reloaded.poisonPlan(7))
}

func TestCampPoisonAssignRejectsBadInput(t *testing.T) {
	module, user, _, store := campedFixture(t, stock{})
	assert.Contains(t, campPoison(t, module, user, "assign self main nightshade"), "no poison called")
	assert.Contains(t, campPoison(t, module, user, "assign self off bitterleaf"), "no bladed weapon in the off hand")
	assert.Contains(t, campPoison(t, module, user, "assign self"), "Usage")
	assert.Empty(t, store.saved.PoisonPlans)
}
