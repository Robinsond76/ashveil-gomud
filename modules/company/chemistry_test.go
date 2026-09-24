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
		7: {LeaderUserID: 7, Companions: []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}}},
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

func bondRounds(m *CompanyModule, a, b domain.MemberKey) int {
	record, _ := m.registry.Get(7)
	bond, _ := record.FindBond(a, b)
	return bond.Rounds
}

func TestChemistryIndependentBonds(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 2)
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c2Key))
	assert.Equal(t, 2, bondRounds(module, c1Key, c2Key))

	// #2 wanders off: only the leader and #1 keep sharing rounds.
	world.mobAt[102] = 6
	roundFrom(module, &round, 1)
	assert.Equal(t, 3, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c2Key))
	assert.Equal(t, 2, bondRounds(module, c1Key, c2Key))

	// A round seen twice counts once.
	module.onNewRound(events.NewRound{RoundNumber: round})
	assert.Equal(t, 3, bondRounds(module, domain.LeaderMemberKey, c1Key))
}

func TestChemistryPausesWhenNotEligible(t *testing.T) {
	module, world, runtime := newChemistryModule(t)
	round := uint64(1000)

	// The leader is dead: the leader's bonds pause, the companions' don't.
	delete(world.leaderAt, 7)
	roundFrom(module, &round, 1)
	assert.Equal(t, 0, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 1, bondRounds(module, c1Key, c2Key))
	world.leaderAt[7] = 5

	// #1 is dead (its mob at 0 health).
	delete(world.mobAt, 101)
	roundFrom(module, &round, 1)
	assert.Equal(t, 0, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 1, bondRounds(module, c1Key, c2Key))
	assert.Equal(t, 1, bondRounds(module, domain.LeaderMemberKey, c2Key))
	world.mobAt[101] = 5

	// #2 no longer serves the leader.
	runtime.stolen = map[int]bool{102: true}
	roundFrom(module, &round, 1)
	assert.Equal(t, 1, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 1, bondRounds(module, domain.LeaderMemberKey, c2Key))
	assert.Equal(t, 1, bondRounds(module, c1Key, c2Key))
	runtime.stolen = nil

	// The leader signs off: nothing accrues, even for companions together.
	world.online[7] = false
	roundFrom(module, &round, 3)
	assert.Equal(t, 1, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 1, bondRounds(module, c1Key, c2Key))
	assert.Zero(t, module.store.(*fakeStore).saveCalls, "no tier crossed, nothing written")
}

func TestChemistryCompanionPairWhileLeaderElsewhere(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	world.leaderAt[7] = 9
	round := uint64(1000)
	roundFrom(module, &round, 2)
	assert.Equal(t, 0, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 2, bondRounds(module, c1Key, c2Key), "companions together while the leader is signed in")
}

func TestChemistryDismissEndsBonds(t *testing.T) {
	useFakeLifecycle(t, &fakeLifecycle{})
	module, _, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 2)
	_, err := module.dismiss(7, "#1")
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	require.Len(t, record.Bonds, 1)
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c2Key))
	saved := module.store.(*fakeStore).saved.Companies[7]
	assert.Len(t, saved.Bonds, 1, "the ended bonds are gone from disk too")
}

func TestChemistryResumesAfterRespawn(t *testing.T) {
	module, world, runtime := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 2)

	// #1 dies: its mob is untracked and its bonds pause.
	module.onMobDeath(events.MobDeath{InstanceId: 101})
	delete(runtime.live, 101)
	roundFrom(module, &round, 1)
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c1Key))

	// Restored with the same companion ID: the same bond resumes.
	runtime.nextInstanceID = 201
	require.NoError(t, module.restoreForLeader(7, 5))
	world.mobAt[201] = 5
	roundFrom(module, &round, 1)
	assert.Equal(t, 3, bondRounds(module, domain.LeaderMemberKey, c1Key))
}

func TestChemistryTierCrossingSavesAndAnnounces(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	world.mobAt[102] = 6 // only the leader and #1 together
	store := module.store.(*fakeStore)
	round := uint64(1000)
	roundFrom(module, &round, 2)
	assert.Zero(t, store.saveCalls)
	roundFrom(module, &round, 1)
	assert.Equal(t, 1, store.saveCalls, "a new tier is saved at once")
	bond, ok := store.saved.Companies[7].FindBond(domain.LeaderMemberKey, c1Key)
	require.True(t, ok)
	assert.Equal(t, 3, bond.Rounds)
	require.Len(t, world.told[7], 1)
	assert.Contains(t, world.told[7][0], "Familiar")
	assert.Contains(t, world.told[7][0], "#1")
}

func TestChemistryFailedCrossingSaveHoldsShort(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	world.mobAt[102] = 6
	store := module.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	round := uint64(1000)
	roundFrom(module, &round, 5)
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c1Key), "held one round short of Familiar")
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "no tier used before it's on disk")
	assert.Empty(t, world.told[7], "nothing announced")

	store.saveErr = nil
	roundFrom(module, &round, 1)
	assert.Equal(t, 3, bondRounds(module, domain.LeaderMemberKey, c1Key))
	assert.Equal(t, 2, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
	require.Len(t, world.told[7], 1)
}

func TestChemistrySurvivesReload(t *testing.T) {
	module, world, runtime := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 4)
	require.NoError(t, module.save()) // the autosave/logout/copyover path

	useFakeLifecycle(t, &fakeLifecycle{})
	restarted := newTestModule(*domain.NewRegistry(), runtime)
	restarted.store = module.store
	restarted.world = module.world
	restarted.chem = world
	restarted.chemRulesForTest = module.chemRulesForTest
	restarted.load()
	require.NoError(t, restarted.loadErr)
	restarted.setInstance(7, 1, 101)
	restarted.setInstance(7, 2, 102)
	assert.Equal(t, 4, bondRounds(restarted, domain.LeaderMemberKey, c1Key))

	// The counter came back a round behind: that round isn't counted again.
	restarted.onNewRound(events.NewRound{RoundNumber: round - 1})
	assert.Equal(t, 4, bondRounds(restarted, domain.LeaderMemberKey, c1Key))
	restarted.onNewRound(events.NewRound{RoundNumber: round + 1})
	assert.Equal(t, 5, bondRounds(restarted, domain.LeaderMemberKey, c1Key))
}

func TestDecodeCompaniesReadsBonds(t *testing.T) {
	registry := domain.NewRegistry()
	data := []byte(`companies:
  7:
    leader_user_id: 7
    companions:
      - {id: 1, mob_template_id: 58}
    bonds:
      - {a: "companion:1", b: leader, rounds: 950, last_round: 1314500}
`)
	require.NoError(t, decodeCompanies(data, registry))
	bond, ok := registry.Companies[7].FindBond(domain.LeaderMemberKey, c1Key)
	require.True(t, ok)
	assert.Equal(t, 950, bond.Rounds)
	assert.Equal(t, uint64(1314500), bond.LastRound)
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

func TestChemistryBonusLeaderAloneNone(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 9) // every pair Sworn
	delete(world.mobAt, 101)
	delete(world.mobAt, 102)
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "no partner at the leader's side")
	assert.Zero(t, module.ChemistryHitBonus(8, domain.LeaderMemberKey), "no company")
}

func TestChemistryBonusCapNeverStacks(t *testing.T) {
	module, _, _ := newChemistryModule(t)
	round := uint64(1000)
	roundFrom(module, &round, 20)
	assert.Equal(t, 6, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "two Sworn partners still give the top bonus once")
	assert.Equal(t, 6, module.ChemistryHitBonus(7, c1Key))
}

func TestChemistryBonusAbsentPartnerNone(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	world.mobAt[102] = 6
	roundFrom(module, &round, 9) // leader and #1 Sworn; #2 a stranger
	world.mobAt[102] = 5
	world.mobAt[101] = 6
	assert.Zero(t, module.ChemistryHitBonus(7, domain.LeaderMemberKey), "#1 is elsewhere; #2 is no bond")
	world.mobAt[101] = 5
	assert.Equal(t, 6, module.ChemistryHitBonus(7, domain.LeaderMemberKey))
	assert.Equal(t, 6, module.ChemistryHitBonus(7, c1Key))
	assert.Zero(t, module.ChemistryHitBonus(7, c2Key))
}

// modulePackageInstance is the module init registered.
func modulePackageInstance() *CompanyModule { return module }

func TestChemistryBonusForInstance(t *testing.T) {
	module, _, _ := newChemistryModule(t)
	domain.SetFormationProvider(module)
	// Restore the module init registered, which later wiring tests use.
	registered := modulePackageInstance()
	t.Cleanup(func() { domain.SetFormationProvider(registered) })
	round := uint64(1000)
	roundFrom(module, &round, 6)
	assert.Equal(t, 4, domain.ChemistryBonusForInstance(102))
	assert.Equal(t, 4, domain.ChemistryBonusForUser(7))
	assert.Zero(t, domain.ChemistryBonusForInstance(999))
}

func TestChemistryViewShowsTierPartnerProgress(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	round := uint64(1000)
	world.mobAt[102] = 6
	roundFrom(module, &round, 4) // leader and #1 Familiar (4 of 3..6)
	view := module.chemistryView(7)
	lines := strings.Split(view, "\n")
	require.GreaterOrEqual(t, len(lines), 4, view)
	assert.Contains(t, lines[1], "You: Familiar with")
	assert.Contains(t, lines[1], "#1")
	assert.Contains(t, lines[1], "+2% to hit now")
	assert.Contains(t, lines[1], "33% of the way to Trusted")
	assert.Contains(t, lines[2], "Familiar with you")
	assert.Contains(t, lines[3], "#2")
	assert.Contains(t, lines[3], "no shared service yet")

	world.mobAt[101] = 6
	view = module.chemistryView(7)
	assert.Contains(t, strings.Split(view, "\n")[1], "not at your side")
	assert.Equal(t, "No companions.", module.chemistryView(8))

	standing, ok := module.ChemistryStanding(7, domain.LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, domain.TierFamiliar, standing.Tier)
	assert.Contains(t, standing.Partner, "#1")
	assert.Zero(t, standing.Bonus, "#1 is elsewhere")
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

// Review finding 3: the bonus is credited to the bond that gives it.
func TestChemistryDisplayCreditsActivePartner(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	module.registry.Put(domain.Record{
		LeaderUserID: 7,
		Companions:   []domain.Companion{{ID: 1, MobTemplateID: 58}, {ID: 2, MobTemplateID: 58}},
		Bonds: []domain.Bond{
			{A: c1Key, B: domain.LeaderMemberKey, Rounds: 3},  // Familiar, present
			{A: c2Key, B: domain.LeaderMemberKey, Rounds: 20}, // Sworn, away
		},
	})
	world.mobAt[102] = 6
	standing, ok := module.ChemistryStanding(7, domain.LeaderMemberKey)
	require.True(t, ok)
	assert.Equal(t, domain.ChemistryStandingView{Tier: domain.TierFamiliar, Partner: "A companion (#1)", Bonus: 2}, standing)
	line := strings.Split(module.chemistryView(7), "\n")[1]
	assert.Contains(t, line, "You: Sworn with A companion (#2) (+2% to hit now, from Familiar with A companion (#1))")

	// Nobody beside the leader: the strongest bond, no bonus, "your side".
	world.mobAt[101] = 6
	standing, _ = module.ChemistryStanding(7, domain.LeaderMemberKey)
	assert.Equal(t, domain.ChemistryStandingView{Tier: domain.TierSworn, Partner: "A companion (#2)"}, standing)
	assert.Contains(t, strings.Split(module.chemistryView(7), "\n")[1], "(not at your side)")
	assert.Contains(t, strings.Split(module.chemistryView(7), "\n")[3], "(not at their side)")
}

func TestChemistryCompanionPairCrossingAnnounced(t *testing.T) {
	module, world, _ := newChemistryModule(t)
	world.leaderAt[7] = 9 // the leader is elsewhere but signed in
	round := uint64(1000)
	roundFrom(module, &round, 3)
	require.Len(t, world.told[7], 1)
	assert.Equal(t, "A companion (#1) and A companion (#2) have grown Familiar (+2% to hit fighting side by side).", world.told[7][0])

	// A tier worth nothing is announced without a "+0%".
	module.chemRulesForTest = &domain.ChemistryRules{TierRounds: [3]int{3, 4, 9}, TierBonus: [3]int{0, 0, 6}}
	roundFrom(module, &round, 1)
	require.Len(t, world.told[7], 2)
	assert.Equal(t, "A companion (#1) and A companion (#2) have grown Trusted.", world.told[7][1])
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
	assert.Equal(t, 2, bondRounds(module, domain.LeaderMemberKey, c1Key))
	record, _ := module.registry.Get(7)
	assert.Nil(t, record.Companions[0].Disposition, "the drift itself rolled back")
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
