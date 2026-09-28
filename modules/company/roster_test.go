package company

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 32a2 unit tests: rosters through the module with fakes. Templates
// 980–984 are defined by no fixture world.
func testRosterRules() domain.RosterRules {
	return domain.RosterRules{
		Size:    3,
		StayMin: 450, StayMax: 1800,
		RefillMin: 60, RefillMax: 180,
		PriceBase: 30, PricePerLevel: 30,
		AlignmentMin: -80, AlignmentMax: 80,
		BynamePercent: 50,
		Archetypes: []domain.RosterArchetype{
			{Archetype: "warrior", MobTemplateID: 980, Weight: 1},
			{Archetype: "ranger", MobTemplateID: 984, Weight: 1},
		},
		GivenNames: []string{"Hild", "Anselm", "Tobin", "Wren", "Edda", "Bram", "Ilse", "Corrin", "Maud", "Osric"},
		Bynames:    []string{"Marrow", "of the Ford"},
		Traits:     []string{"Quiet, and quick with a sling."},
	}
}

func newRosterModule(t *testing.T, gold int) (*CompanyModule, *fakeRuntime, *users.UserRecord, *uint64) {
	t.Helper()
	m, runtime, user, _ := newRecruitModule(t, gold)
	recs := testRecruiters()
	rec := recs[hiringRoom]
	rec.Generated = true
	recs[hiringRoom] = rec
	recs[901] = recruiter{RoomID: 901, Name: "the notched hiring post", Candidates: []candidate{{ID: "tamsin", MobTemplateID: 961, Tutorial: true}}}
	m.recruitersForTest = recs
	rules := testRosterRules()
	m.rosterRulesForTest = &rules
	m.rng = rand.New(rand.NewSource(11))
	round := uint64(10000)
	m.roundNow = func() uint64 { return round }
	return m, runtime, user, &round
}

func rosterOf(t *testing.T, m *CompanyModule, leader int) domain.Roster {
	t.Helper()
	record, _ := m.registry.Get(leader)
	ros, ok := record.Roster(hiringRoom)
	require.True(t, ok)
	return ros
}

func TestRosterNoticeListsRegularsThenTheViewersOwn(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 0)
	lines := m.RecruiterLines(user.UserId, hiringRoom)
	ros := rosterOf(t, m, user.UserId)
	require.Len(t, ros.Candidates, 3)
	notice := strings.Join(lines, "\n")
	garrick := strings.Index(notice, "garrick")
	require.GreaterOrEqual(t, garrick, 0)
	for _, c := range ros.Candidates {
		at := strings.Index(notice, c.Name)
		require.Greater(t, at, garrick, "%s is listed after the regulars", c.Name)
	}
	assert.Zero(t, m.store.(*fakeStore).saveCalls, "a look never writes the company file")

	// The same list on the next look, and nothing generated in the tutorial.
	again := m.RecruiterLines(user.UserId, hiringRoom)
	assert.Equal(t, lines, again)
	for _, c := range ros.Candidates {
		assert.NotContains(t, strings.Join(m.RecruiterLines(user.UserId, 901), "\n"), c.Name)
	}
	_, hasTutorialRoster := func() (domain.Roster, bool) {
		record, _ := m.registry.Get(user.UserId)
		return record.Roster(901)
	}()
	assert.False(t, hasTutorialRoster)
}

func TestRosterEachPlayerHasTheirOwn(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 500)
	other := users.NewUserRecord(8, 2)
	other.Character.Gold = 500
	m.RecruiterLines(user.UserId, hiringRoom)
	m.RecruiterLines(other.UserId, hiringRoom)
	mine, theirs := rosterOf(t, m, 7), rosterOf(t, m, 8)
	assert.NotEqual(t, mine.Candidates, theirs.Candidates, "each player rolls their own")

	hire := mine.Candidates[0]
	text, err := m.recruit(user, hiringRoom, hire.Key)
	require.NoError(t, err)
	assert.Contains(t, text, hire.Name+" joins your company (#1).")
	assert.Equal(t, theirs, rosterOf(t, m, 8), "one player's hire leaves another's list alone")
	assert.Len(t, rosterOf(t, m, 7).Candidates, 2)
}

func TestRecruitGeneratedCandidate(t *testing.T) {
	m, runtime, user, round := newRosterModule(t, 500)
	m.world.(*fakeWorld).leaders[7] = 0
	m.RecruiterLines(user.UserId, hiringRoom)
	ros := rosterOf(t, m, 7)
	var hire domain.Candidate
	for _, c := range ros.Candidates {
		if c.Alignment >= -60 && c.Alignment <= 60 {
			hire = c
			break
		}
	}
	require.NotEmpty(t, hire.Key, "a candidate this company would take")

	text, err := m.recruit(user, hiringRoom, hire.Key)
	require.NoError(t, err)
	assert.Equal(t, "You pay "+strconv.Itoa(hire.Price)+" gold. "+hire.Name+` joins your company (#1). Place them with "formation move".`, text)
	assert.Equal(t, 500-hire.Price, user.Character.Gold)

	record, _ := m.registry.Get(7)
	require.Len(t, record.Companions, 1)
	c := record.Companions[0]
	assert.Equal(t, hire.Name, c.Name)
	assert.Equal(t, hire.Trait, c.Description)
	assert.Equal(t, hire.MobTemplateID, c.MobTemplateID)
	require.NotNil(t, c.Disposition)
	assert.Equal(t, hire.Alignment, c.Disposition.Alignment, "the candidate's own alignment, not the template's")
	require.NotNil(t, c.State)
	assert.Equal(t, hire.Level, c.State.Level)
	assert.Equal(t, domain.Identity{Name: hire.Name, Description: hire.Trait}, runtime.spawnedIdentities[0])
	require.NotNil(t, runtime.spawnedStates[0])
	assert.Equal(t, hire.Level, runtime.spawnedStates[0].Level)

	// Saved with the hire: the companion and the shorter roster.
	saved, _ := m.store.(*fakeStore).saved.Get(7)
	require.Len(t, saved.Companions, 1)
	savedRoster, ok := saved.Roster(hiringRoom)
	require.True(t, ok)
	assert.Len(t, savedRoster.Candidates, 2)
	require.Len(t, savedRoster.Openings, 1)

	// The name is theirs everywhere the company is shown.
	assert.Contains(t, m.status(7), "#1 "+hire.Name+",")
	members, _ := m.CompanyMembers(7)
	assert.Equal(t, hire.Name, members[0].Name)
	assert.Equal(t, hire.Name, m.Roster(7)[1].Name)

	// Hired once: it's gone from the list, and the slot fills after its delay.
	again, err := m.recruit(user, hiringRoom, hire.Key)
	require.NoError(t, err)
	assert.Contains(t, again, "No one called")
	*round = savedRoster.Openings[0]
	m.RecruiterLines(user.UserId, hiringRoom)
	assert.Len(t, rosterOf(t, m, 7).Candidates, 3)
}

func TestRecruitGeneratedRefusalsChangeNothing(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 0)
	m.world.(*fakeWorld).leaders[7] = 100
	m.RecruiterLines(user.UserId, hiringRoom)
	before := rosterOf(t, m, 7)
	far := domain.Candidate{Key: "morrow", Name: "Morrow Black", Archetype: "warrior", MobTemplateID: 980, Level: 1, Alignment: -80, Price: 10, Leaves: 999999}
	before.Candidates[2] = far
	require.NoError(t, m.registry.PutRoster(7, before))

	text, err := m.recruit(user, hiringRoom, "morrow")
	require.NoError(t, err)
	assert.Contains(t, text, "Morrow Black (alignment -80, ")
	assert.Contains(t, text, "won't join a company")
	assert.Contains(t, strings.Join(m.RecruiterLines(7, hiringRoom), "\n"), "Morrow Black</ansi> (won't join you)")
	assert.Contains(t, m.inspectAt(7, hiringRoom, "morrow"), "They won't join a company so far from their ways.")

	m.world.(*fakeWorld).leaders[7] = -80
	text, err = m.recruit(user, hiringRoom, "morrow")
	require.NoError(t, err)
	assert.Equal(t, "Morrow Black asks 10 gold, and you have 0.", text)
	record, _ := m.registry.Get(7)
	assert.Empty(t, record.Companions)
	assert.Equal(t, before, rosterOf(t, m, 7), "nothing taken off the list")
}

func TestRecruitGeneratedRollsBackTheRoster(t *testing.T) {
	m, runtime, user, _ := newRosterModule(t, 500)
	m.RecruiterLines(user.UserId, hiringRoom)
	before := rosterOf(t, m, 7)
	runtime.spawnErr = errors.New("spawn failed")
	_, err := m.recruit(user, hiringRoom, before.Candidates[0].Key)
	require.Error(t, err)
	assert.Equal(t, before, rosterOf(t, m, 7), "a failed hire leaves the candidate on the list")
	assert.Equal(t, 500, user.Character.Gold)
}

func TestTwoGeneratedWarriorsAreBothAddressable(t *testing.T) {
	m, runtime, user, _ := newRosterModule(t, 5000)
	m.world.(*fakeWorld).leaders[7] = 0
	rules := testRosterRules()
	rules.Archetypes = rules.Archetypes[:1] // warriors only
	rules.AlignmentMin, rules.AlignmentMax = 0, 0
	m.rosterRulesForTest = &rules
	m.RecruiterLines(user.UserId, hiringRoom)
	ros := rosterOf(t, m, 7)
	for _, c := range ros.Candidates[:2] {
		_, err := m.recruit(user, hiringRoom, c.Key)
		require.NoError(t, err)
		runtime.nextInstanceID++
	}
	record, _ := m.registry.Get(7)
	require.Len(t, record.Companions, 2)
	assert.Equal(t, record.Companions[0].MobTemplateID, record.Companions[1].MobTemplateID)
	for _, c := range record.Companions {
		got, ok := resolveCompanion(record, givenKey(c.Name))
		require.True(t, ok, c.Name)
		assert.Equal(t, c.ID, got.ID)
		got, ok = resolveCompanion(record, "#"+strconv.Itoa(c.ID))
		require.True(t, ok)
		assert.Equal(t, c.Name, got.Name)
		assert.Contains(t, m.gearView(7, givenKey(c.Name)), "#"+strconv.Itoa(c.ID)+" "+c.Name)
	}
	grid := m.renderFormation(7)
	for _, c := range record.Companions {
		assert.Contains(t, grid, c.Name)
	}
	// Both restore under their own names after a logout.
	runtime.spawnedIdentities = nil
	m.instances = map[int]map[int]int{}
	require.NoError(t, m.restoreForLeader(7, hiringRoom))
	require.Len(t, runtime.spawnedIdentities, 2)
	assert.Equal(t, record.Companions[0].Identity(), runtime.spawnedIdentities[0])
	assert.Equal(t, record.Companions[1].Identity(), runtime.spawnedIdentities[1])
}

func TestRosterNamesAvoidTheCompanyAndOtherRosters(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 0)
	rules := testRosterRules()
	rules.GivenNames = []string{"Hild", "Garrick", "Tobin", "Wren", "Edda"}
	rules.BynamePercent = 0
	m.rosterRulesForTest = &rules
	require.NoError(t, m.registry.PutRoster(7, domain.Roster{RoomID: 2005, Candidates: []domain.Candidate{{Key: "wren", Name: "Wren", Leaves: 999999}}}))
	reg := m.registry
	record, _ := reg.Get(7)
	record.Companions = []domain.Companion{{ID: 1, MobTemplateID: 980, Name: "Tobin Reyes"}}
	m.registry.Put(record)
	m.RecruiterLines(user.UserId, hiringRoom)
	ros := rosterOf(t, m, 7)
	require.Len(t, ros.Candidates, 2)
	for _, c := range ros.Candidates {
		assert.Contains(t, []string{"hild", "edda"}, c.Key, "not a companion's, a regular's, or another roster's")
	}
}

func TestLookAndInspectAGeneratedCandidate(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 0)
	m.world.(*fakeWorld).leaders[7] = 0
	m.RecruiterLines(user.UserId, hiringRoom)
	c := rosterOf(t, m, 7).Candidates[0]
	text, ok := m.LookCandidate(7, hiringRoom, c.Key)
	require.True(t, ok)
	assert.Contains(t, text, c.Name)
	assert.Contains(t, text, c.Trait)
	assert.Contains(t, text, "asking "+strconv.Itoa(c.Price)+" gold")
	assert.Contains(t, text, "company inspect "+c.Key)

	out := m.inspectAt(7, hiringRoom, c.Key)
	assert.Contains(t, out, c.Name+", "+archetypeLabel(c.Archetype)+", level "+strconv.Itoa(c.Level)+": alignment ")
	assert.Contains(t, out, "They ask "+strconv.Itoa(c.Price)+" gold.")
	// Elsewhere, inspect doesn't know them.
	assert.NotContains(t, m.inspectAt(7, 12, c.Key), "They ask")
}

func TestRosterRefreshesAfterAStay(t *testing.T) {
	m, _, user, round := newRosterModule(t, 0)
	m.RecruiterLines(user.UserId, hiringRoom)
	old := rosterOf(t, m, 7)
	*round += 1800 * 4
	notice := strings.Join(m.RecruiterLines(user.UserId, hiringRoom), "\n")
	fresh := rosterOf(t, m, 7)
	require.Len(t, fresh.Candidates, 3)
	for _, c := range fresh.Candidates {
		assert.Contains(t, notice, c.Name)
		assert.Greater(t, c.Leaves, *round)
	}
	assert.NotEqual(t, old.Candidates, fresh.Candidates)
}

func TestRosterWaitsForPersistence(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 0)
	m.loadErr = errors.New("unreadable")
	m.RecruiterLines(user.UserId, hiringRoom)
	_, ok := m.registry.Get(7)
	assert.False(t, ok, "nothing is rolled onto a company that failed to load")
}

func TestParseRosterRules(t *testing.T) {
	raw := map[string]any{
		"RosterSize":             "4",
		"CandidateStayRoundsMin": 900,
		"CandidateStayRoundsMax": 100, // below the min: the min falls to it
		"RecruitAlignmentMin":    -500,
		"RecruitTemplates": []any{
			map[any]any{"MobTemplateId": 80, "Weight": 2, "PricePercent": 110},
			map[any]any{"MobTemplateId": 81},
			map[any]any{"MobTemplateId": 99}, // no archetype
			map[any]any{"Weight": 1},
		},
		"RecruitGivenNames": []any{"Hild", " ", 7, "Wren"},
	}
	rules := parseRosterRules(func(k string) any { return raw[k] }, map[int]string{80: "warrior", 81: "rogue"})
	assert.Equal(t, 4, rules.Size)
	assert.Equal(t, uint64(100), rules.StayMin)
	assert.Equal(t, uint64(100), rules.StayMax)
	assert.Equal(t, defaultAlignmentMin, rules.AlignmentMin, "out of range falls back")
	assert.Equal(t, []domain.RosterArchetype{
		{Archetype: "warrior", MobTemplateID: 80, Weight: 2, PricePercent: 110},
		{Archetype: "rogue", MobTemplateID: 81, Weight: 1},
	}, rules.Archetypes)
	assert.Equal(t, []string{"Hild", "Wren"}, rules.GivenNames)

	empty := parseRosterRules(func(string) any { return nil }, nil)
	assert.Equal(t, defaultRosterSize, empty.Size)
	assert.Empty(t, empty.Archetypes, "no templates, no generated candidates")

	assert.Equal(t, map[string]int{"ranger": 4, "rogue": 0}, parseArchetypeWeights([]any{
		map[any]any{"Archetype": " Ranger ", "Weight": 4},
		map[any]any{"Archetype": "rogue", "Weight": 0},
		map[any]any{"Archetype": "cleric", "Weight": -1},
	}, 2005))
	assert.Nil(t, parseArchetypeWeights(nil, 2005))
}

// TestShippedRosters pins the shipped roster content: the rules parse, each
// archetype has a farm-proof template, only the settlement recruiters roll
// rosters, and there are enough names that no one runs short.
func TestShippedRosters(t *testing.T) {
	world := shippedWorld(t)
	archetypesByTemplate := parseCompanionArchetypes(shippedModuleConfig(t, "CompanionArchetypes"))
	get := func(key string) any { return shippedModuleConfig(t, key) }
	rules := parseRosterRules(get, archetypesByTemplate)
	assert.Equal(t, 3, rules.Size)
	seen := map[string]bool{}
	allowed := allowedTemplateIDs(shippedModuleConfig(t, "AllowedCompanionMobIDs"))
	for _, a := range rules.Archetypes {
		seen[a.Archetype] = true
		_, summonable := allowed[a.MobTemplateID]
		assert.False(t, summonable, "template %d must not be summonable for free", a.MobTemplateID)
		path := globOne(t, filepath.Join(world, "mobs", "*", strconv.Itoa(a.MobTemplateID)+"-*.yaml"))
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var mob mobs.Mob
		require.NoError(t, yaml.Unmarshal(data, &mob))
		assert.Equal(t, mob.Filepath(), filepath.ToSlash(path[len(filepath.Join(world, "mobs"))+1:]), "the loader's path rule")
		assert.Zero(t, mob.ItemDropChance, a.Archetype)
		assert.Empty(t, mob.LootCategory, a.Archetype)
		assert.Empty(t, mob.ScriptTag, a.Archetype)
		assert.Empty(t, mob.Character.Items, a.Archetype)
		assert.Zero(t, mob.Character.Gold, a.Archetype)
		assert.False(t, mob.Hostile, a.Archetype)
		for _, itm := range mob.Character.Equipment.GetAllItems() {
			assert.True(t, itemFileExists(t, world, itm.ItemId), "%s: item %d exists", a.Archetype, itm.ItemId)
		}
	}
	assert.Equal(t, map[string]bool{"warrior": true, "rogue": true, "wizard": true, "cleric": true, "ranger": true}, seen)
	assert.GreaterOrEqual(t, len(rules.GivenNames), 20)
	assert.NotEmpty(t, rules.Bynames)
	assert.NotEmpty(t, rules.Traits)

	recs := parseRecruiters(shippedModuleConfig(t, "Recruiters"))
	generated := map[int]bool{}
	for roomID, rec := range recs {
		generated[roomID] = rec.Generated
		for _, c := range rec.Candidates {
			for _, n := range rules.GivenNames {
				assert.NotEqual(t, c.ID, strings.ToLower(n), "a generated name can't shadow %s", c.ID)
			}
		}
		for archetype := range rec.Weights {
			assert.True(t, seen[archetype], "room %d weights a known archetype", roomID)
		}
	}
	assert.Equal(t, map[int]bool{2003: true, 2005: true, 901: false, 907: false}, generated)
}

// Phase 32a2 review: inspect and recruit read a partial name the same way,
// even when a regular at another recruiter shares its start.
func TestInspectAndRecruitAgreeOnAPartialName(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 500)
	m.world.(*fakeWorld).leaders[7] = 0
	m.recruitersForTest[907] = recruiter{RoomID: 907, Name: "the notices", Candidates: []candidate{{ID: "corvin", MobTemplateID: 969, Price: 150}}}
	require.NoError(t, m.registry.PutRoster(7, domain.Roster{RoomID: hiringRoom, Candidates: []domain.Candidate{
		{Key: "corrin", Name: "Corrin Pike", Archetype: "warrior", MobTemplateID: 980, Level: 1, Price: 60, Leaves: 999999},
		{Key: "hild", Name: "Hild", Archetype: "warrior", MobTemplateID: 980, Level: 1, Price: 60, Leaves: 999999},
		{Key: "edda", Name: "Edda", Archetype: "warrior", MobTemplateID: 980, Level: 1, Price: 60, Leaves: 999999},
	}}))
	assert.Contains(t, m.inspectAt(7, hiringRoom, "cor"), "Corrin Pike, warrior, level 1")
	text, err := m.recruit(user, hiringRoom, "cor")
	require.NoError(t, err)
	assert.Contains(t, text, "Corrin Pike joins your company")
	// This recruiter's own regular is still the regular.
	assert.NotContains(t, m.inspectAt(7, hiringRoom, "garrick"), "They ask")
}

// Phase 32a2 review: no generated name shadows any recruiter's regular.
func TestRosterNamesAvoidEveryRecruitersRegulars(t *testing.T) {
	m, _, user, _ := newRosterModule(t, 0)
	m.recruitersForTest[907] = recruiter{RoomID: 907, Name: "the notices", Candidates: []candidate{{ID: "corvin", MobTemplateID: 969}}}
	rules := testRosterRules()
	rules.GivenNames = []string{"Corvin", "Hild"}
	m.rosterRulesForTest = &rules
	m.RecruiterLines(user.UserId, hiringRoom)
	ros := rosterOf(t, m, 7)
	require.Len(t, ros.Candidates, 1)
	assert.Equal(t, "hild", ros.Candidates[0].Key)
}

// Phase 32a2 review: a hire whose company save fails leaves the candidate
// on the list, the gold, and the company as they were.
func TestRecruitGeneratedSaveFailureRollsBack(t *testing.T) {
	m, runtime, user, _ := newRosterModule(t, 500)
	m.world.(*fakeWorld).leaders[7] = 0
	m.RecruiterLines(user.UserId, hiringRoom)
	before := rosterOf(t, m, 7)
	var hire domain.Candidate
	for _, c := range before.Candidates {
		if c.Alignment >= -60 && c.Alignment <= 60 {
			hire = c
		}
	}
	require.NotEmpty(t, hire.Key)
	m.store.(*fakeStore).failSaveOnCall = m.store.(*fakeStore).saveCalls + 1
	_, err := m.recruit(user, hiringRoom, hire.Key)
	require.Error(t, err)
	assert.Equal(t, before, rosterOf(t, m, 7))
	assert.Equal(t, 500, user.Character.Gold)
	record, _ := m.registry.Get(7)
	assert.Empty(t, record.Companions)
	assert.Equal(t, 1, runtime.detachCalls, "the spawned mob is removed")
}
