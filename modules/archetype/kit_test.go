package archetype

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// backpackIDs lists the item ids in a user's backpack, in order.
func backpackIDs(u *users.UserRecord) []int {
	out := []int{}
	for _, itm := range u.Character.Items {
		out = append(out, itm.ItemId)
	}
	return out
}

func kitOf(t *testing.T, m *ArchetypeModule, id string) []int {
	t.Helper()
	a, ok := m.table.Get(id)
	require.True(t, ok)
	return a.Kit
}

func TestParseKit(t *testing.T) {
	list := parseArchetypes([]any{
		map[any]any{"ArchetypeId": "warrior", "Name": "Warrior", "Skills": []any{"brawling"}, "Kit": []any{10002, 30001, 30001}},
		map[any]any{"ArchetypeId": "rogue", "Name": "Rogue", "Skills": []any{"skulduggery"}},
	})
	require.Len(t, list, 2)
	assert.Equal(t, []int{10002, 30001, 30001}, list[0].Kit)
	assert.Empty(t, list[1].Kit, "a kit is optional")
}

func TestBuildTableDropsUnknownKitItems(t *testing.T) {
	m, _ := testModule(t)
	table := m.buildTable([]archetypes.Archetype{
		{ID: "fine", Name: "Fine", Skills: []string{"brawling"}, CompanionLevels: []int{1, 2, 3, 4}, Kit: []int{10002, 99999999, 30001, 30001}},
	})
	a, ok := table.Get("fine")
	require.True(t, ok, "an unknown kit item doesn't drop the archetype")
	assert.Equal(t, []int{10002, 30001, 30001}, a.Kit)
}

// TestShippedKitsResolveAndBalance pins the shipped kits (Phase 22a design
// decision 2): every id resolves, and the largest kit is worth at most
// 1.25x the smallest.
func TestShippedKitsResolveAndBalance(t *testing.T) {
	m, _ := testModule(t)
	want := map[string][]int{
		"warrior": {10002, 20004, 20020, 30004, 30015},
		"rogue":   {10004, 8, 20029, 20003, 23, 30004, 30015},
		"wizard":  {10021, 20020, 20008, 20039, 30004, 30015, 30014, 30014},
		"cleric":  {10015, 20004, 20008, 30004, 30015, 30001, 30001},
		"ranger":  {10014, 20020, 20024, 20003, 30019, 30015, 23},
	}
	lowest, highest := 0, 0
	for id, kit := range want {
		assert.Equal(t, kit, kitOf(t, m, id), id)
		total := 0
		for _, itemID := range kit {
			spec := items.GetItemSpec(itemID)
			require.NotNil(t, spec, "%s kit item %d", id, itemID)
			total += spec.Value
		}
		if lowest == 0 || total < lowest {
			lowest = total
		}
		if total > highest {
			highest = total
		}
	}
	assert.LessOrEqual(t, float64(highest), 1.25*float64(lowest), "kit values are balanced (%d..%d)", lowest, highest)

	staff := items.GetItemSpec(10021)
	require.NotNil(t, staff)
	assert.Equal(t, items.Weapon, staff.Type)
	assert.Equal(t, 2, staff.Hands)
}

func TestChooseOwesKitInSameSave(t *testing.T) {
	m, store := testModule(t)
	u := newUser(201)
	text := m.choose(u, "warrior", true)
	assert.Contains(t, text, "You are now a Warrior.")
	assert.Contains(t, text, "starter kit")
	assert.Contains(t, text, "broadsword")
	require.NotNil(t, store.saved)
	assert.Equal(t, 1, store.saves, "the choice and the owed kit are one save")
	assert.Equal(t, "warrior", store.saved.Players[201])
	assert.Equal(t, "warrior", store.saved.Kits[201])
	assert.Equal(t, kitOf(t, m, "warrior"), backpackIDs(u))
	assert.Equal(t, "warrior", u.Character.GetMiscData(kitMarkerKey))
}

func TestChooseSaveFailureOwesNothing(t *testing.T) {
	m, store := testModule(t)
	store.saveErr = errors.New("disk full")
	u := newUser(202)
	text := m.choose(u, "rogue", true)
	assert.Contains(t, text, "retry")
	_, chosen := m.PlayerArchetype(202)
	assert.False(t, chosen)
	_, owed := m.registry.Kits[202]
	assert.False(t, owed, "the owed kit rolls back with the choice")
	assert.Empty(t, u.Character.Items)
	assert.Nil(t, u.Character.GetMiscData(kitMarkerKey))
}

func TestGrantKitExactlyOnce(t *testing.T) {
	m, _ := testModule(t)
	saves := 0
	m.saveUser = func(*users.UserRecord) error { saves++; return nil }
	u := newUser(203)
	m.choose(u, "cleric", true)
	assert.Equal(t, 1, saves, "the user is saved right after the grant")
	kit := kitOf(t, m, "cleric")
	assert.Equal(t, kit, backpackIDs(u))

	assert.Empty(t, m.grantKit(u), "a repeat grant gives nothing")
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(203) })
	m.onPlayerSpawn(eventsPlayerSpawn(203))
	m.onPlayerSpawn(eventsPlayerSpawn(203))
	assert.Equal(t, kit, backpackIDs(u), "spawns never grant a second kit")
	assert.Equal(t, 1, saves)
}

func TestGrantKitSaveFailureKeepsGrantInMemory(t *testing.T) {
	m, _ := testModule(t)
	m.saveUser = func(*users.UserRecord) error { return errors.New("disk full") }
	u := newUser(204)
	m.choose(u, "ranger", true)
	assert.Equal(t, kitOf(t, m, "ranger"), backpackIDs(u), "items stay for the next user save")
	assert.Equal(t, "ranger", u.Character.GetMiscData(kitMarkerKey), "and so does the marker")
	assert.Empty(t, m.grantKit(u))
}

func TestGrantKitSkipsLegacyChoice(t *testing.T) {
	m, _ := testModule(t)
	// A choice made before Phase 22a: recorded, but owing no kit.
	m.registry.Players[205] = "warrior"
	u := newUser(205)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(205) })
	m.onPlayerSpawn(eventsPlayerSpawn(205))
	assert.Empty(t, u.Character.Items, "existing characters keep their gear and get no kit")
	assert.Equal(t, 1, u.Character.GetSkillLevel("brawling"), "grants still recover")
}

func TestResetAndRechooseNoSecondKit(t *testing.T) {
	m, store := testModule(t)
	u := newUser(206)
	m.choose(u, "warrior", true)
	_, err := m.reset(206)
	require.NoError(t, err)
	_, owed := store.saved.Kits[206]
	assert.False(t, owed, "reset clears the owed kit with the choice")

	text := m.choose(u, "rogue", true)
	assert.Contains(t, text, "You are now a Rogue.")
	assert.NotContains(t, text, "starter kit")
	assert.Equal(t, kitOf(t, m, "warrior"), backpackIDs(u), "the character keeps only its first kit")
}

func TestResetSaveFailureRestoresOwedKit(t *testing.T) {
	m, store := testModule(t)
	u := newUser(207)
	m.choose(u, "warrior", true)
	store.saveErr = errors.New("disk full")
	_, err := m.reset(207)
	require.Error(t, err)
	assert.Equal(t, "warrior", m.registry.Kits[207])
	assert.Equal(t, "warrior", m.registry.Players[207])
}

func TestLostGrantRecoveredOnSpawn(t *testing.T) {
	m, _ := testModule(t)
	u := newUser(208)
	m.choose(u, "wizard", true)

	// The server crashed before any user save: the reloaded character has
	// neither the items nor the marker, but the registry owes the kit.
	fresh := newUser(208)
	users.SetTestUser(fresh)
	t.Cleanup(func() { users.RemoveTestUser(208) })
	m.onPlayerSpawn(eventsPlayerSpawn(208))
	assert.Equal(t, kitOf(t, m, "wizard"), backpackIDs(fresh))
	m.onPlayerSpawn(eventsPlayerSpawn(208))
	assert.Equal(t, kitOf(t, m, "wizard"), backpackIDs(fresh), "recovered exactly once")
}

func TestPermadeathClearsOwedKit(t *testing.T) {
	m, store := testModule(t)
	u := newUser(209)
	m.choose(u, "warrior", true)
	require.NoError(t, m.clearCharacter(209))
	_, owed := store.saved.Kits[209]
	assert.False(t, owed)

	// The replacement character (no marker) earns its own kit.
	next := newUser(209)
	m.choose(next, "cleric", true)
	assert.Equal(t, kitOf(t, m, "cleric"), backpackIDs(next))
}

func TestClearCharacterWithOnlyOwedKit(t *testing.T) {
	m, store := testModule(t)
	m.registry.Kits[210] = "warrior"
	require.NoError(t, m.clearCharacter(210))
	require.NotNil(t, store.saved)
	_, owed := store.saved.Kits[210]
	assert.False(t, owed)
}

func TestKitMarkerSurvivesUserYAML(t *testing.T) {
	m, _ := testModule(t)
	u := newUser(211)
	m.choose(u, "rogue", true)

	// User saves and copyover both persist the record as YAML.
	data, err := yaml.Marshal(u)
	require.NoError(t, err)
	loaded := users.NewUserRecord(211, 211)
	require.NoError(t, yaml.Unmarshal(data, loaded))
	assert.Equal(t, "rogue", loaded.Character.GetMiscData(kitMarkerKey))
	assert.Equal(t, kitOf(t, m, "rogue"), backpackIDs(loaded))
	assert.Empty(t, m.grantKit(loaded), "a reloaded character isn't granted again")
}

func TestDecodeRegistryKits(t *testing.T) {
	data := []byte("players:\n  5: warrior\nkits:\n  5: \" Warrior \"\n  0: rogue\n  6: \"\"\n")
	r := NewRegistry()
	require.NoError(t, decodeRegistry(data, r))
	assert.Equal(t, map[int]string{5: "warrior"}, r.Kits)

	clone := r.Clone()
	clone.Kits[7] = "rogue"
	assert.NotContains(t, r.Kits, 7, "Clone is deep")
}

func TestListAndPreviewShowKit(t *testing.T) {
	m, _ := testModule(t)
	list := m.list(212)
	assert.Contains(t, list, "Starter kit: guardsman's broadsword")
	assert.Contains(t, list, "small blue potion (x2)")

	preview := m.choose(newUser(212), "wizard", false)
	assert.Contains(t, preview, "Starter kit: ash quarterstaff")
	assert.Contains(t, preview, "permanent")
}

func TestCreationChoicesAndChooseAtCreation(t *testing.T) {
	m, _ := testModule(t)
	choices := m.CreationChoices()
	require.Len(t, choices, 5)
	assert.Equal(t, "cleric", choices[0].ID)
	assert.Contains(t, choices[0].Kit, "small red potion (x2)")

	_, ok := m.ChooseAtCreation(213, "wizard")
	assert.False(t, ok, "an offline user can't choose")

	u := newUser(213)
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(213) })
	text, ok := m.ChooseAtCreation(213, "wizard")
	assert.True(t, ok)
	assert.Contains(t, text, "Wizard")
	assert.Equal(t, kitOf(t, m, "wizard"), backpackIDs(u))

	_, ok = m.ChooseAtCreation(213, "rogue")
	assert.False(t, ok, "the choice is permanent")
}

func eventsPlayerSpawn(userID int) events.PlayerSpawn {
	return events.PlayerSpawn{UserId: userID}
}
