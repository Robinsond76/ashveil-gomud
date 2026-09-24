package company

import (
	"errors"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChemWorld struct {
	online   map[int]bool
	leaderAt map[int]int // living leaders' rooms
	mobAt    map[int]int // living instances' rooms
	told     map[int][]string
}

func newFakeChemWorld() *fakeChemWorld {
	return &fakeChemWorld{online: map[int]bool{}, leaderAt: map[int]int{}, mobAt: map[int]int{}, told: map[int][]string{}}
}

func (w *fakeChemWorld) LeaderOnline(leaderUserID int) bool { return w.online[leaderUserID] }
func (w *fakeChemWorld) LeaderPresence(leaderUserID int) (int, bool) {
	room, ok := w.leaderAt[leaderUserID]
	return room, ok && w.online[leaderUserID]
}
func (w *fakeChemWorld) InstancePresence(instanceID int) (int, bool) {
	room, ok := w.mobAt[instanceID]
	return room, ok
}
func (w *fakeChemWorld) Tell(leaderUserID int, text string) {
	w.told[leaderUserID] = append(w.told[leaderUserID], text)
}

// chemTemplate is a mob template no fixture world defines, so companion
// names stay the fallback "A companion" whatever specs other tests load.
const chemTemplate = 990058

var (
	c1Key = domain.CompanionMemberKey(1)
	c2Key = domain.CompanionMemberKey(2)
)

// newChemistryModule has leader 7 online in room 5 with companions #1
// (instance 101) and #2 (instance 102) beside them, and small tiers: 3, 6,
// and 9 rounds for +2, +4, +6.
func newChemistryModule(t *testing.T) (*CompanyModule, *fakeChemWorld, *fakeRuntime) {
	t.Helper()
	runtime := &fakeRuntime{live: map[int]bool{101: true, 102: true}}
	module := newTestModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: chemTemplate}, {ID: 2, MobTemplateID: chemTemplate}}},
	}}, runtime)
	module.world = newFakeWorld()
	world := newFakeChemWorld()
	world.online[7] = true
	world.leaderAt[7] = 5
	world.mobAt[101] = 5
	world.mobAt[102] = 5
	module.chem = world
	module.chemRulesForTest = &domain.ChemistryRules{TierRounds: [3]int{3, 6, 9}, TierBonus: [3]int{2, 4, 6}}
	module.setInstance(7, 1, 101)
	module.setInstance(7, 2, 102)
	return module, world, runtime
}

// roundFrom sends n NewRound events numbered from *next, advancing it.
func roundFrom(m *CompanyModule, next *uint64, n int) {
	for i := 0; i < n; i++ {
		*next++
		m.onNewRound(events.NewRound{RoundNumber: *next})
	}
}

func served(m *CompanyModule, key domain.MemberKey) int {
	record, _ := m.registry.Get(7)
	s, _ := record.FindService(key)
	return s.Rounds
}

func TestChemistryServiceAccrues(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 2)
	assert.Equal(t, 2, served(module, domain.LeaderMemberKey))
	assert.Equal(t, 2, served(module, c1Key))
	assert.Equal(t, 2, served(module, c2Key))

	// #2 wanders off alone: the leader and #1 keep serving together.
	world.mobAt[102] = 6
	roundFrom(module, &round, 1)
	assert.Equal(t, 3, served(module, domain.LeaderMemberKey))
	assert.Equal(t, 3, served(module, c1Key))
	assert.Equal(t, 2, served(module, c2Key))

	// A round seen twice counts once.
	module.onNewRound(events.NewRound{RoundNumber: round})
	assert.Equal(t, 3, served(module, c1Key))
}

func TestChemistryPausesWhenNotEligible(t *testing.T) {
	module, world, runtime := newChemistryModule(t)
	round := uint64(1000)

	// The leader is dead: the companions still serve together.
	delete(world.leaderAt, 7)
	roundFrom(module, &round, 1)
	assert.Equal(t, 0, served(module, domain.LeaderMemberKey))
	assert.Equal(t, 1, served(module, c1Key))
	assert.Equal(t, 1, served(module, c2Key))
	world.leaderAt[7] = 5

	// #1 is dead.
	delete(world.mobAt, 101)
	roundFrom(module, &round, 1)
	assert.Equal(t, 1, served(module, domain.LeaderMemberKey))
	assert.Equal(t, 1, served(module, c1Key))
	assert.Equal(t, 2, served(module, c2Key))
	world.mobAt[101] = 5

	// #2 no longer serves the leader.
	runtime.stolen = map[int]bool{102: true}
	roundFrom(module, &round, 1)
	assert.Equal(t, 2, served(module, domain.LeaderMemberKey))
	assert.Equal(t, 2, served(module, c1Key))
	assert.Equal(t, 2, served(module, c2Key))
	runtime.stolen = nil

	// Everyone apart: no one serves "together".
	world.mobAt[101], world.mobAt[102] = 6, 8
	roundFrom(module, &round, 1)
	assert.Equal(t, 2, served(module, domain.LeaderMemberKey))
	world.mobAt[101], world.mobAt[102] = 5, 5

	// The leader signs off: nothing accrues.
	world.online[7] = false
	roundFrom(module, &round, 3)
	assert.Equal(t, 2, served(module, c1Key))
	assert.Equal(t, 2, served(module, c2Key))
	assert.Zero(t, module.store.(*fakeStore).saveCalls, "no tier crossed, nothing written")
}

func TestChemistryCompanionsServeWhileLeaderElsewhere(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	world.leaderAt[7] = 9
	round := uint64(1000)
	roundFrom(module, &round, 2)
	assert.Equal(t, 0, served(module, domain.LeaderMemberKey))
	assert.Equal(t, 2, served(module, c1Key), "companions together while the leader is signed in")
}

func TestChemistryDismissEndsService(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	module, _, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 2)
	_, err := module.dismiss(7, "#1")
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	_, found := record.FindService(c1Key)
	assert.False(t, found)
	assert.Equal(t, 2, served(module, c2Key))
	assert.Len(t, module.store.(*fakeStore).saved.Companies[7].Service, 2, "gone from disk too")
}

func TestChemistryResumesAfterRespawn(t *testing.T) {
	module, world, runtime := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 2)

	// #1 dies: its mob is untracked and its service pauses.
	module.onMobDeath(events.MobDeath{InstanceId: 101})
	delete(runtime.live, 101)
	roundFrom(module, &round, 1)
	assert.Equal(t, 2, served(module, c1Key))

	// Restored with the same companion ID: its service resumes.
	runtime.nextInstanceID = 201
	require.NoError(t, module.restoreForLeader(7, 5))
	world.mobAt[201] = 5
	roundFrom(module, &round, 1)
	assert.Equal(t, 3, served(module, c1Key))
}

func TestChemistryTierUpTakesEffectAtNextSave(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	store := module.store.(*fakeStore)
	round := uint64(1000)
	roundFrom(module, &round, 3)
	assert.Zero(t, store.saveCalls, "chemistry adds no save of its own")
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "not on disk yet")
	assert.Empty(t, world.told[7])

	require.NoError(t, module.save()) // the autosave, a drift tick, a logout, a command
	s, ok := store.saved.Companies[7].FindService(c1Key)
	require.True(t, ok)
	assert.Equal(t, 3, s.Rounds)
	assert.Equal(t, 2, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
	assert.Equal(t, 2, module.ChemistryHitBonus(7, c2Key), "everyone in the band")
	require.Len(t, world.told[7], 1)
	assert.Equal(t, "Your band grows closer: Familiar (+2% to hit fighting together).", world.told[7][0])

	require.NoError(t, module.save())
	assert.Len(t, world.told[7], 1, "announced once")
}

func TestChemistryFailedSaveKeepsTierUntilSaved(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	store := module.store.(*fakeStore)
	round := uint64(1000)
	roundFrom(module, &round, 7)
	store.saveErr = errors.New("disk full")
	require.Error(t, module.save())
	assert.Equal(t, 7, served(module, c1Key), "service keeps counting")
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "no tier before it's on disk")
	standing, _ := module.ChemistryStanding(7, domain.LeaderMemberKey)
	assert.Equal(t, domain.TierNone, standing.Tier)
	assert.Empty(t, world.told[7], "nothing announced")

	store.saveErr = nil
	require.NoError(t, module.save())
	assert.Equal(t, 4, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "7 rounds saved: Trusted")
	require.Len(t, world.told[7], 1, "one announcement, at the highest tier reached")
	assert.Contains(t, world.told[7][0], "Trusted")
}

// Review finding (band rework) 3: a tier made durable by any save is
// announced; one the band no longer reaches is dropped.
func TestChemistryTierUpDroppedWhenBandShrinks(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	world.mobAt[102] = 6
	roundFrom(module, &round, 3)      // the leader and #1 due Familiar
	_, err := module.dismiss(7, "#1") // the dismissal's own save
	require.NoError(t, err)
	assert.Empty(t, world.told[7], "the band that earned it is gone")
}

// A failed dismissal save restores the companion with its saved service.
func TestChemistryDismissRollbackKeepsSavedService(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	module, _, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 4)
	require.NoError(t, module.save())
	store := module.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	_, err := module.dismiss(7, "#1")
	require.Error(t, err)
	record, _ := module.registry.Get(7)
	s, ok := record.FindService(c1Key)
	require.True(t, ok)
	assert.Equal(t, domain.Service{Member: c1Key, Rounds: 4, LastRound: 1004, Saved: 4}, s)
	assert.Equal(t, 2, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
}

func TestChemistryVeteranCannotHideARecruit(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	module.registry.Put(domain.Record{
		LeaderUserID: 7,
		Companions:   []domain.Companion{{ID: 1, MobTemplateID: chemTemplate}, {ID: 2, MobTemplateID: chemTemplate}},
		Service:      []domain.Service{{Member: domain.LeaderMemberKey, Rounds: 100, Saved: 100}},
	})
	world.mobAt[102] = 6
	// Capped at Sworn's 9: (9 + 0) / 2 = 4, Familiar, not Sworn.
	assert.Equal(t, 2, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
}

func TestChemistryRecruitDilutesTheBand(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	module.registry.Put(domain.Record{
		LeaderUserID: 7,
		Companions:   []domain.Companion{{ID: 1, MobTemplateID: chemTemplate}, {ID: 2, MobTemplateID: chemTemplate}},
		Service: []domain.Service{
			{Member: domain.LeaderMemberKey, Rounds: 9, Saved: 9},
			{Member: c1Key, Rounds: 9, Saved: 9},
		},
	})
	// Veterans plus a fresh #2: average 6, Trusted, for all three.
	assert.Equal(t, 4, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
	assert.Equal(t, 4, module.ChemistryHitBonus(7, c2Key), "the recruit fights with the band")
	// #2 steps away: the veterans are Sworn.
	world.mobAt[102] = 6
	assert.Equal(t, 6, module.ChemistryHitBonus(7, c1Key))
	assert.Zero(t, module.ChemistryHitBonus(7, c2Key), "alone, no band")
	// The leader alone: nothing.
	world.mobAt[101] = 6
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
	assert.Zero(t, module.ChemistryHitBonus(8, domain.LeaderMemberKey), "no company")
}

func TestChemistryCompanionBandAnnounced(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	world.leaderAt[7] = 9 // the leader is elsewhere but signed in
	round := uint64(1000)
	roundFrom(module, &round, 3)
	require.NoError(t, module.save())
	require.Len(t, world.told[7], 1)
	assert.Equal(t, "Your companions A companion (#1) and A companion (#2) grow closer: Familiar (+2% to hit fighting together).", world.told[7][0])

	// A tier worth nothing is announced without a "+0%".
	module.chemRulesForTest = &domain.ChemistryRules{TierRounds: [3]int{3, 4, 9}, TierBonus: [3]int{0, 0, 6}}
	roundFrom(module, &round, 1)
	require.NoError(t, module.save())
	require.Len(t, world.told[7], 2)
	assert.Equal(t, "Your companions A companion (#1) and A companion (#2) grow closer: Trusted.", world.told[7][1])
}

func TestBandsOrderMembersNumerically(t *testing.T) {
	present := map[domain.MemberKey]int{
		domain.CompanionMemberKey(10): 5, domain.CompanionMemberKey(2): 5, domain.LeaderMemberKey: 5,
	}
	assert.Equal(t, []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(2), domain.CompanionMemberKey(10)}, band(present, 5))
}

// The ordering onNewRound relies on: a drift tick whose save fails in the
// same round as a charge keeps the charge.
func TestChemistryChargeKeptWhenDriftSaveFails(t *testing.T) {
	module, _, _ := newChemistryModule(t)
	module.world.(*fakeWorld).leaders[7] = 100
	round := uint64(1000)
	roundFrom(module, &round, 1)
	module.registry.DriftIn = 1 // the next round is a drift tick
	store := module.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	roundFrom(module, &round, 1)
	assert.Equal(t, 2, served(module, c1Key))
	record, _ := module.registry.Get(7)
	assert.Nil(t, record.Companions[0].Disposition, "the drift itself rolled back")
}

func TestChemistrySurvivesReload(t *testing.T) {
	module, world, runtime := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 4)
	require.NoError(t, module.save()) // the autosave/logout/copyover path

	useFakeLifecycle(t, &fakeLifecycle{})
	restarted := newTestModule(*domain.NewRegistry(), runtime)
	restarted.store = &fakeStore{saved: module.store.(*fakeStore).saved}
	// A real store doesn't keep in-memory Saved values.
	for id, record := range restarted.store.(*fakeStore).saved.Companies {
		for i := range record.Service {
			record.Service[i].Saved = 0
		}
		restarted.store.(*fakeStore).saved.Companies[id] = record
	}
	restarted.world = module.world
	restarted.chem = world
	restarted.chemRulesForTest = module.chemRulesForTest
	restarted.load()
	require.NoError(t, restarted.loadErr)
	restarted.setInstance(7, 1, 101)
	restarted.setInstance(7, 2, 102)
	assert.Equal(t, 4, served(restarted, c1Key))
	assert.Equal(t, 2, restarted.ChemistryHitBonus(7, domain.LeaderMemberKey), "loaded service counts as saved")

	// The counter came back a round behind: that round isn't counted again.
	restarted.onNewRound(events.NewRound{RoundNumber: round - 1})
	assert.Equal(t, 4, served(restarted, c1Key))
	restarted.onNewRound(events.NewRound{RoundNumber: round + 1})
	assert.Equal(t, 5, served(restarted, c1Key))
}

func TestDecodeCompaniesReadsService(t *testing.T) {
	registry := domain.NewRegistry()
	data := []byte(`companies:
  7:
    leader_user_id: 7
    companions:
      - {id: 1, mob_template_id: 58}
    service:
      - {member: "companion:1", rounds: 950, last_round: 1314500}
`)
	require.NoError(t, decodeCompanies(data, registry))
	s, ok := registry.Companies[7].FindService(c1Key)
	require.True(t, ok)
	assert.Equal(t, 950, s.Rounds)
	assert.Equal(t, uint64(1314500), s.LastRound)
}

func TestParseChemistryConfig(t *testing.T) {
	cfg := map[string]any{
		"ChemistryFamiliarRounds": 10, "ChemistryTrustedRounds": "20", "ChemistrySwornRounds": 30,
		"ChemistryFamiliarBonus": 1, "ChemistryTrustedBonus": 3, "ChemistrySwornBonus": 5,
	}
	rules, ok := parseChemistryConfig(func(k string) any { return cfg[k] })
	assert.True(t, ok)
	assert.Equal(t, domain.ChemistryRules{TierRounds: [3]int{10, 20, 30}, TierBonus: [3]int{1, 3, 5}}, rules)

	// Out of order: all defaults.
	cfg["ChemistryTrustedRounds"] = 5
	rules, ok = parseChemistryConfig(func(k string) any { return cfg[k] })
	assert.False(t, ok)
	assert.Equal(t, domain.DefaultChemistryRules(), rules)

	// Out of range: that knob's default.
	rules, ok = parseChemistryConfig(func(k string) any {
		if k == "ChemistrySwornBonus" {
			return 50
		}
		return nil
	})
	assert.True(t, ok)
	assert.Equal(t, domain.DefaultChemistryRules(), rules)
	rules, ok = parseChemistryConfig(func(string) any { return nil })
	assert.True(t, ok)
	assert.Equal(t, domain.DefaultChemistryRules(), rules)
}

func TestChemistryBonusForInstance(t *testing.T) {
	module, _, _ := newChemistryModule(t)
	domain.SetFormationProvider(module)
	// Restore the module init registered, which later wiring tests use.
	registered := modulePackageInstance()
	t.Cleanup(func() { domain.SetFormationProvider(registered) })
	round := uint64(1000)
	roundFrom(module, &round, 6)
	require.NoError(t, module.save())
	assert.Equal(t, 4, domain.ChemistryBonusForInstance(102))
	assert.Equal(t, 4, domain.ChemistryBonusForUser(7))
	assert.Zero(t, domain.ChemistryBonusForInstance(999))
}

// modulePackageInstance is the module init registered.
func modulePackageInstance() *CompanyModule { return module }

func TestChemistryViewShowsBandAndService(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	world.mobAt[102] = 6
	roundFrom(module, &round, 4) // leader and #1: 4 rounds each, saved at 3
	require.NoError(t, module.save())
	view := module.chemistryView(7)
	lines := strings.Split(view, "\n")
	require.GreaterOrEqual(t, len(lines), 6, view)
	assert.Equal(t, "  With you: 2 together, Familiar (+2% to hit); 33% of the way to Trusted.", lines[1])
	assert.Equal(t, "Service with the band:", lines[2])
	assert.Contains(t, lines[3], "You: ")
	assert.Contains(t, lines[5], "A companion (#2): 0.0 days")

	// #2 joins: the band is diluted.
	world.mobAt[102] = 5
	lines = strings.Split(module.chemistryView(7), "\n")
	assert.Equal(t, "  With you: 3 together, Strangers; 66% of the way to Familiar.", lines[1])
	standing, ok := module.ChemistryStanding(7, domain.LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, domain.ChemistryStandingView{Together: 3}, standing)

	// The companions apart from the leader.
	world.leaderAt[7] = 9
	lines = strings.Split(module.chemistryView(7), "\n")
	assert.Equal(t, "  With you: no one from the band.", lines[1])
	assert.Equal(t, "  Apart: A companion (#1) and A companion (#2), 2 together, Strangers; 66% of the way to Familiar.", lines[2])
	assert.Equal(t, "No companions.", module.chemistryView(8))
}

func TestChemistryUnavailableWhileCompanyDataFailed(t *testing.T) {
	module, _, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 9)
	module.loadErr = errors.New("unreadable")
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
	_, ok := module.ChemistryStanding(7, domain.LeaderMemberKey)
	assert.False(t, ok)
	assert.Contains(t, module.chemistryView(7), "unavailable")
}

func TestChemistryRulesCachedAndRefreshedEachRound(t *testing.T) {
	module, _, _ := newChemistryModule(t)
	module.chemRulesForTest = nil
	cached := domain.ChemistryRules{TierRounds: [3]int{1, 2, 3}, TierBonus: [3]int{1, 1, 1}}
	module.chemRules = &cached
	assert.Equal(t, cached, module.chemistryRules(), "combat reads the cache, not the config")
	module.refreshChemistryRules() // no plugin: the cache stays
	assert.Equal(t, cached, module.chemistryRules())
}
