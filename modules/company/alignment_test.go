package company

import (
	"errors"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorld struct {
	templates        map[int]int  // template alignment
	leaders          map[int]int  // online leaders' alignment
	inCombat         map[int]bool // leaders in combat
	instanceFighting map[int]bool
	instanceAlign    map[int]int
	told             map[int][]string
}

func newFakeWorld() *fakeWorld {
	return &fakeWorld{
		templates:        map[int]int{},
		leaders:          map[int]int{},
		inCombat:         map[int]bool{},
		instanceFighting: map[int]bool{},
		instanceAlign:    map[int]int{},
		told:             map[int][]string{},
	}
}

func (w *fakeWorld) TemplateAlignment(templateID int) int { return w.templates[templateID] }
func (w *fakeWorld) LeaderAlignment(leaderUserID int) (int, bool) {
	a, ok := w.leaders[leaderUserID]
	return a, ok
}
func (w *fakeWorld) LeaderInCombat(leaderUserID int) bool { return w.inCombat[leaderUserID] }
func (w *fakeWorld) InstanceInCombat(instanceID int) bool {
	return w.instanceFighting[instanceID]
}
func (w *fakeWorld) SetInstanceAlignment(instanceID, alignment int) {
	w.instanceAlign[instanceID] = alignment
}
func (w *fakeWorld) Tell(leaderUserID int, text string) {
	w.told[leaderUserID] = append(w.told[leaderUserID], text)
}

func newAlignmentModule(registry domain.Registry, world *fakeWorld) (*CompanyModule, *fakeRuntime) {
	runtime := &fakeRuntime{resolved: map[string]int{"training dummy": 58, "paladin": 59}, nextInstanceID: 101}
	module := newTestModule(registry, runtime)
	module.world = world
	return module, runtime
}

func allowAlso59(t *testing.T) {
	t.Helper()
	previous := defaultAllowedTemplates
	defaultAllowedTemplates = map[int]struct{}{58: {}, 59: {}}
	t.Cleanup(func() { defaultAllowedTemplates = previous })
}

func withDisposition(c domain.Companion, alignment, loyalty int) domain.Companion {
	c.Disposition = &domain.Disposition{Alignment: alignment, Loyalty: loyalty}
	return c
}

func runRounds(m *CompanyModule, n int) {
	for i := 0; i < n; i++ {
		m.onNewRound(events.NewRound{})
	}
}

func TestSummonSeedsDispositionFromTemplate(t *testing.T) {
	world := newFakeWorld()
	world.templates[58] = -30
	world.leaders[7] = 0
	module, _ := newAlignmentModule(*domain.NewRegistry(), world)
	_, err := module.summon(7, 12, "training dummy")
	require.NoError(t, err)
	record, _ := module.registry.Get(7)
	require.NotNil(t, record.Companions[0].Disposition)
	assert.Equal(t, domain.Disposition{Alignment: -30, Loyalty: 70}, *record.Companions[0].Disposition)
	saved := module.store.(*fakeStore).saved.Companies[7].Companions[0]
	require.NotNil(t, saved.Disposition, "persisted with the summon")
	assert.Equal(t, -30, saved.Disposition.Alignment)
	assert.Equal(t, -30, world.instanceAlign[101], "the live mob takes the stored alignment")
}

func TestSummonRefusesFarCandidateWithoutWriting(t *testing.T) {
	allowAlso59(t)
	world := newFakeWorld()
	world.templates[59] = 90 // a holy paladin
	world.leaders[7] = -40   // a corrupt leader
	module, runtime := newAlignmentModule(*domain.NewRegistry(), world)
	text, err := module.summon(7, 12, "paladin")
	require.NoError(t, err, "a refusal is a message, not an error")
	assert.Contains(t, text, "won't join")
	assert.Contains(t, text, "paladin (alignment 95, holy) won't join a company of alignment 30, corrupt.")
	_, exists := module.registry.Get(7)
	assert.False(t, exists)
	assert.Zero(t, module.store.(*fakeStore).saveCalls)
	assert.Zero(t, runtime.spawnCalls)

	// The average counts current companions too: a neutral companion pulls
	// a good leader's company average down to within reach.
	world.leaders[8] = 20
	module.registry.Put(domain.Record{LeaderUserID: 8, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -40, 70)}})
	text, err = module.summon(8, 12, "paladin")
	require.NoError(t, err)
	assert.Contains(t, text, "won't join", "average -10 is 100 from 90")
	world.leaders[8] = 100
	text, err = module.summon(8, 12, "paladin")
	require.NoError(t, err)
	assert.Contains(t, text, "Companion summoned", "average 30 is 60 from 90")
}

func TestSummonGateSkipsDisallowedTemplate(t *testing.T) {
	world := newFakeWorld()
	world.templates[59] = 90
	world.leaders[7] = -100
	module, _ := newAlignmentModule(*domain.NewRegistry(), world)
	_, err := module.summon(7, 12, "paladin")
	assert.ErrorIs(t, err, domain.ErrTemplateNotAllowed, "the allow list answers first; the gate doesn't reveal alignment")
}

func TestLoadSeedsLegacyDispositionAndSaves(t *testing.T) {
	world := newFakeWorld()
	world.templates[58] = 25
	module, _ := newAlignmentModule(*domain.NewRegistry(), world)
	store := module.store.(*fakeStore)
	store.saved = domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			{ID: 1, MobTemplateID: 58},
			withDisposition(domain.Companion{ID: 2, MobTemplateID: 58}, -10, 40),
		}},
	}}
	useFakeLifecycle(t, &fakeLifecycle{})
	module.load()
	require.NoError(t, module.loadErr)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.Disposition{Alignment: 25, Loyalty: 100}, *record.Companions[0].Disposition, "legacy companions have been serving: full loyalty")
	assert.Equal(t, domain.Disposition{Alignment: -10, Loyalty: 40}, *record.Companions[1].Disposition, "stored values kept")
	assert.Equal(t, 1, store.saveCalls, "the upgrade is saved at once")
	require.NotNil(t, store.saved.Companies[7].Companions[0].Disposition)

	store.saveCalls = 0
	module.load()
	assert.Zero(t, store.saveCalls, "nothing to seed, nothing written")
}

func TestSpawnAppliesStoredAlignment(t *testing.T) {
	world := newFakeWorld()
	world.templates[58] = 0
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -44, 50)}},
	}}, world)
	require.NoError(t, module.restoreForLeader(7, 12))
	assert.Equal(t, -44, world.instanceAlign[101])
}

func TestDecodeCompaniesReadsDispositionAndDriftIn(t *testing.T) {
	registry := domain.NewRegistry()
	require.NoError(t, decodeCompanies([]byte(`
drift_in: 33
companies:
  7:
    companions:
      - id: 1
        mob_template_id: 58
        disposition: {alignment: -12, loyalty: 64}
`), registry))
	assert.Equal(t, 33, registry.DriftIn)
	record, _ := registry.Get(7)
	require.NotNil(t, record.Companions[0].Disposition)
	assert.Equal(t, domain.Disposition{Alignment: -12, Loyalty: 64}, *record.Companions[0].Disposition)
}

func TestDriftRunsAfterConfiguredRounds(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 60
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 0, 50)}},
	}}, world)
	module.setInstance(7, 1, 99)
	module.runtime.(*fakeRuntime).live = map[int]bool{99: true}
	store := module.store.(*fakeStore)

	runRounds(module, defaultDriftEveryRounds-1)
	record, _ := module.registry.Get(7)
	assert.Equal(t, 0, record.Companions[0].Disposition.Alignment, "not yet")
	assert.Zero(t, store.saveCalls)

	runRounds(module, 1)
	record, _ = module.registry.Get(7)
	assert.Equal(t, domain.Disposition{Alignment: 2, Loyalty: 52}, *record.Companions[0].Disposition)
	assert.Equal(t, 1, store.saveCalls, "one save per tick")
	assert.Equal(t, 2, store.saved.Companies[7].Companions[0].Disposition.Alignment)
	assert.Equal(t, 2, world.instanceAlign[99], "the live mob follows")

	runRounds(module, defaultDriftEveryRounds)
	record, _ = module.registry.Get(7)
	assert.Equal(t, 4, record.Companions[0].Disposition.Alignment, "every interval")
}

func TestDriftSkipsOfflineLeaders(t *testing.T) {
	world := newFakeWorld()
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -100, 10)}},
	}}, world)
	runRounds(module, defaultDriftEveryRounds*3)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.Disposition{Alignment: -100, Loyalty: 10}, *record.Companions[0].Disposition)
	assert.Zero(t, module.store.(*fakeStore).saveCalls, "nothing changed, nothing written")
}

func TestDriftInPersistsAndResumes(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 60
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 0, 50)}},
	}}, world)
	runRounds(module, 50)
	require.NoError(t, module.save()) // the autosave/copyover/shutdown path
	store := module.store.(*fakeStore)
	assert.Equal(t, defaultDriftEveryRounds-50, store.saved.DriftIn)

	// Restart: a fresh module resumes from the stored countdown.
	restarted, _ := newAlignmentModule(*domain.NewRegistry(), world)
	restarted.store = store
	useFakeLifecycle(t, &fakeLifecycle{})
	restarted.load()
	require.NoError(t, restarted.loadErr)
	runRounds(restarted, defaultDriftEveryRounds-51)
	record, _ := restarted.registry.Get(7)
	assert.Equal(t, 0, record.Companions[0].Disposition.Alignment)
	runRounds(restarted, 1)
	record, _ = restarted.registry.Get(7)
	assert.Equal(t, 2, record.Companions[0].Disposition.Alignment, "the tick lands where it would have without the restart")

	// An out-of-range stored countdown means a full interval.
	restarted.registry.DriftIn = defaultDriftEveryRounds * 10
	runRounds(restarted, defaultDriftEveryRounds-1)
	record, _ = restarted.registry.Get(7)
	assert.Equal(t, 2, record.Companions[0].Disposition.Alignment)
	runRounds(restarted, 1)
	record, _ = restarted.registry.Get(7)
	assert.Equal(t, 4, record.Companions[0].Disposition.Alignment)
}

func TestLoyaltyWarningAndDesertion(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 100
	lifecycle := &fakeLifecycle{}
	useFakeLifecycle(t, lifecycle)
	var formation domain.Formation
	require.NoError(t, formation.Place(domain.CompanionMemberKey(1), 0, 0))
	module, runtime := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Formation: formation, Companions: []domain.Companion{
			withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -100, 28),
			withDisposition(domain.Companion{ID: 2, MobTemplateID: 58}, 100, 90),
		}},
	}}, world)
	module.setInstance(7, 1, 99)
	runtime.live = map[int]bool{99: true}

	runRounds(module, defaultDriftEveryRounds)
	require.Len(t, world.told[7], 1)
	assert.Contains(t, world.told[7][0], "#1")
	assert.Contains(t, world.told[7][0], "uneasy")

	// 23 -> 18 -> 13 -> 8 -> 3 -> 0.
	runRounds(module, defaultDriftEveryRounds*5)
	record, _ := module.registry.Get(7)
	require.Len(t, record.Companions, 1, "#1 deserted")
	assert.Equal(t, 2, record.Companions[0].ID)
	_, _, placed := record.Formation.Find(domain.CompanionMemberKey(1))
	assert.False(t, placed, "formation cleared")
	assert.Contains(t, lifecycle.removed, [2]int{7, 1}, "survival state removed")
	assert.Equal(t, 1, runtime.detachCalls, "live mob detached")
	_, tracked := module.instance(7, 1)
	assert.False(t, tracked)
	assert.Contains(t, world.told[7][len(world.told[7])-1], "deserts")
	saved := module.store.(*fakeStore).saved.Companies[7]
	assert.Len(t, saved.Companions, 1, "desertion persisted")
}

func TestDesertionWaitsForCombat(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 100
	world.inCombat[7] = true
	useFakeLifecycle(t, &fakeLifecycle{})
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -100, 3)}},
	}}, world)
	runRounds(module, defaultDriftEveryRounds*2)
	record, _ := module.registry.Get(7)
	require.Len(t, record.Companions, 1, "no desertion mid-fight")
	assert.Equal(t, 0, record.Companions[0].Disposition.Loyalty)

	world.inCombat[7] = false
	runRounds(module, defaultDriftEveryRounds)
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Companions, "deserts on the first tick after the fight")
}

func TestDriftSaveFailureRollsBack(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 100
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 0, 50)}},
	}}, world)
	module.setInstance(7, 1, 99)
	module.runtime.(*fakeRuntime).live = map[int]bool{99: true}
	store := module.store.(*fakeStore)
	store.saveErr = errors.New("disk full")
	runRounds(module, defaultDriftEveryRounds)
	record, _ := module.registry.Get(7)
	assert.Equal(t, domain.Disposition{Alignment: 0, Loyalty: 50}, *record.Companions[0].Disposition, "memory matches disk")
	assert.NotContains(t, world.instanceAlign, 99, "live mob untouched")
	assert.Equal(t, defaultDriftEveryRounds, module.registry.DriftIn, "retried at the next interval")

	store.saveErr = nil
	runRounds(module, defaultDriftEveryRounds)
	record, _ = module.registry.Get(7)
	assert.Equal(t, 2, record.Companions[0].Disposition.Alignment)
}

func TestDesertionSaveFailureKeepsCompanion(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 100
	lifecycle := &fakeLifecycle{}
	useFakeLifecycle(t, lifecycle)
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -100, 0)}},
	}}, world)
	require.NoError(t, lifecycle.EnsureCompanyMember(7, 1))
	store := module.store.(*fakeStore)
	// The drift save succeeds; the desertion save fails.
	store.failSaveOnCall = 2
	runRounds(module, defaultDriftEveryRounds)
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	require.Len(t, record.Companions, 1, "the companion stays when the desertion can't be saved")
	assert.Len(t, lifecycle.restored, 1, "survival state restored")
}

func TestCompanyInspectShowsCandidate(t *testing.T) {
	allowAlso59(t)
	world := newFakeWorld()
	world.templates[58] = 0
	world.templates[59] = 90
	world.leaders[7] = 60
	module, _ := newAlignmentModule(*domain.NewRegistry(), world)
	out := module.inspect(7, "paladin")
	assert.Contains(t, out, "alignment 95 (holy)")
	assert.Contains(t, out, "Your company: 80 (good)")
	assert.Contains(t, out, "would join")
	world.leaders[7] = -60
	assert.Contains(t, module.inspect(7, "paladin"), "won't join")
	world.leaders[7] = 30
	rulesTolerance := domain.DefaultAlignmentRules().LoyaltyToleranceGap
	require.Equal(t, 60, rulesTolerance)
	assert.Contains(t, module.inspect(7, "paladin"), "They would join.", "gap 60 is content")
	module.loadErr = errors.New("cannot read companies")
	assert.Contains(t, module.inspect(7, "paladin"), "unavailable", "never judged against an unloaded company")
	module.loadErr = nil
	assert.Contains(t, module.inspect(7, "nobody"), "no one")
	defaultAllowedTemplates = map[int]struct{}{58: {}}
	assert.Contains(t, module.inspect(7, "paladin"), "isn't available to recruit")
}

func TestCompanyAlignmentView(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 60
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{
			withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 40, 70),
			withDisposition(domain.Companion{ID: 2, MobTemplateID: 58}, -60, 20),
		}},
	}}, world)
	out := module.alignmentView(7)
	assert.Contains(t, out, "Company alignment: 56 (neutral)", "avg(60, 40, -60) = 13")
	assert.Contains(t, out, "You: 80 (good)")
	assert.Regexp(t, `#1 [^\n]+: 70 \(virtuous\), loyalty 70, content`, out)
	assert.Regexp(t, `#2 [^\n]+: 20 \(evil\), loyalty 20, uneasy`, out)

	world.leaders[8] = -20
	assert.Contains(t, module.alignmentView(8), "Company alignment: 40 (misguided)", "a leader alone still sees their own")
	world.leaders = map[int]int{}
	assert.Contains(t, module.alignmentView(9), "No companions.")
}

func TestCompanyStatusShowsAlignmentAndLoyalty(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 60
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, 40, 70)}},
	}}, world)
	out := module.status(7)
	assert.Contains(t, out, "Company alignment: 75 (virtuous)")
	assert.Contains(t, out, "alignment 70 (virtuous), loyalty 70")
}

func TestParseAlignmentConfig(t *testing.T) {
	get := func(values map[string]any) func(string) any {
		return func(key string) any { return values[key] }
	}
	rules, every := parseAlignmentConfig(get(nil))
	assert.Equal(t, domain.DefaultAlignmentRules(), rules)
	assert.Equal(t, defaultDriftEveryRounds, every)

	rules, every = parseAlignmentConfig(get(map[string]any{
		"DriftEveryRounds": 10, "DriftStep": "4", "LoyaltyToleranceGap": 30,
		"LoyaltyLoss": 0, "LoyaltyGain": 7, "StartLoyalty": 55,
		"LoyaltyWarnBelow": 10, "RecruitMaxGap": 200,
	}))
	assert.Equal(t, 10, every)
	assert.Equal(t, domain.AlignmentRules{DriftStep: 4, LoyaltyToleranceGap: 30, LoyaltyLoss: 0, LoyaltyGain: 7, StartLoyalty: 55, LoyaltyWarnBelow: 10, RecruitMaxGap: 200}, rules)

	rules, every = parseAlignmentConfig(get(map[string]any{
		"DriftEveryRounds": 0, "DriftStep": 51, "LoyaltyToleranceGap": 201,
		"LoyaltyLoss": -1, "LoyaltyGain": "x", "StartLoyalty": 0,
		"LoyaltyWarnBelow": 101, "RecruitMaxGap": -5,
	}))
	assert.Equal(t, defaultDriftEveryRounds, every)
	assert.Equal(t, domain.DefaultAlignmentRules(), rules, "out of range falls back")
}

func TestInspectWarnsWhenRecruitWouldBeUneasy(t *testing.T) {
	allowAlso59(t)
	world := newFakeWorld()
	world.templates[59] = 90
	world.leaders[7] = 0
	module, _ := newAlignmentModule(*domain.NewRegistry(), world)
	module.plug = nil
	// A config with RecruitMaxGap above the tolerance lets an uneasy recruit in.
	module.rulesForTest = &domain.AlignmentRules{DriftStep: 2, LoyaltyToleranceGap: 60, LoyaltyLoss: 5, LoyaltyGain: 2, StartLoyalty: 70, LoyaltyWarnBelow: 25, RecruitMaxGap: 100}
	assert.Contains(t, module.inspect(7, "paladin"), "would join, but uneasily")
}

func TestSummonFullCompanyRefusedForCapacityNotAlignment(t *testing.T) {
	allowAlso59(t)
	world := newFakeWorld()
	world.templates[59] = 90
	world.leaders[7] = -100
	companions := []domain.Companion{}
	for id := 1; id <= domain.MaxCompanions; id++ {
		companions = append(companions, withDisposition(domain.Companion{ID: id, MobTemplateID: 58}, -100, 70))
	}
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{7: {LeaderUserID: 7, Companions: companions}}}, world)
	_, err := module.summon(7, 12, "paladin")
	assert.ErrorIs(t, err, domain.ErrCompanyFull)
}

func TestDesertionWaitsForCompanionCombat(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 100
	useFakeLifecycle(t, &fakeLifecycle{})
	module, runtime := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -100, 3)}},
	}}, world)
	module.setInstance(7, 1, 99)
	runtime.live = map[int]bool{99: true}
	world.instanceFighting[99] = true
	runRounds(module, defaultDriftEveryRounds)
	record, _ := module.registry.Get(7)
	require.Len(t, record.Companions, 1, "a fighting companion doesn't walk off mid-fight")
	world.instanceFighting[99] = false
	runRounds(module, defaultDriftEveryRounds)
	record, _ = module.registry.Get(7)
	assert.Empty(t, record.Companions)
}

func TestCompanyAlignmentProviderUsesCompanyAverage(t *testing.T) {
	world := newFakeWorld()
	world.leaders[7] = 60
	module, _ := newAlignmentModule(domain.Registry{Companies: map[int]domain.Record{
		7: {LeaderUserID: 7, Companions: []domain.Companion{withDisposition(domain.Companion{ID: 1, MobTemplateID: 58}, -20, 70)}},
	}}, world)
	got, ok := module.CompanyAlignment(7)
	assert.True(t, ok)
	assert.Equal(t, 20, got)
	_, ok = module.CompanyAlignment(8)
	assert.False(t, ok, "offline leader without companions")
	module.loadErr = errors.New("cannot read companies")
	_, ok = module.CompanyAlignment(7)
	assert.False(t, ok, "never judged against an unloaded company")
}
