package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/cookbook"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	lyreID       = 3200
	luteID       = 3201
	fineFiddleID = 3202
	reedPipesID  = 3204
	fineFluteID  = 3205
	frameDrumID  = 3207
	fineDrumID   = 3208
)

var leaderKey = string(survival.LeaderMemberKey)

// teach gives a member a Music skill.
func teach(m *CampingModule, leader int, key string, f camping.Family, songs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.musicSkills[leader] == nil {
		m.musicSkills[leader] = map[string]camping.MusicSkill{}
	}
	m.musicSkills[leader][key] = camping.MusicSkill{Family: f, Songs: songs}
}

// carry makes the company carry the given instruments.
func carry(m *CampingModule, ids ...int) {
	have := map[int]int{}
	for _, id := range ids {
		have[id]++
	}
	m.itemCount = func(_, id int) int { return have[id] }
}

// musicTrio adds three more companions in the camp's room, so the company
// has four members (the leader, Bran, Cor, Dee).
func musicTrio(e *innEnv, roomID int) map[string]*characters.Character {
	bran := e.companion
	cor := &characters.Character{Name: "Cor", RoomId: roomID}
	dee := &characters.Character{Name: "Dee", RoomId: roomID}
	bran.RoomId = roomID
	e.module.companionsOf = func(int) (map[int]*characters.Character, []int) {
		return map[int]*characters.Character{1: bran, 2: cor, 3: dee}, []int{1, 2, 3}
	}
	return map[string]*characters.Character{"Bran": bran, "Cor": cor, "Dee": dee}
}

func musicKeyOf(id int) string { return string(survival.CompanionMemberKey(id)) }

// restThrough runs a camp rest start to finish and one grant pass.
func (e *innEnv) restThrough(t *testing.T, user *users.UserRecord, mutate func()) {
	t.Helper()
	room := eligibleRoom()
	e.module.establish(user, room)
	e.module.lightFire(user, room)
	require.Contains(t, e.module.startRest(user, room), "settle in")
	if mutate != nil {
		mutate()
	}
	*e.now = e.now.Add(camping.RestDuration)
	e.scheduler.fireLatest()
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
}

func TestStringsStretchTheRestedBuff(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 0) // level 1 + lute 2 = 3: 15%
	carry(e.module, luteID)
	e.restThrough(t, user, nil)
	rounds := e.ledger.rounds[buffCall{"Hero", 1033}]
	assert.Greater(t, rounds, 225, "longer than a silent rest")
	assert.InDelta(t, 225*115/100, rounds, 1)
	assert.Equal(t, 15, e.store.saved.Camps[7].Rest.Song.RestedPct())
}

func TestSilentRestIsUnchanged(t *testing.T) {
	e, user := heroEnv(t)
	carry(e.module, luteID) // an instrument with no player
	e.restThrough(t, user, nil)
	assert.Equal(t, 225, e.ledger.rounds[buffCall{"Hero", 1033}])
	assert.Equal(t, []int{camping.FatigueRecovery}, e.surv.amounts)
}

func TestNoInstrumentNoSong(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyWinds, 0)
	carry(e.module) // the skill alone is not a song
	room := eligibleRoom()
	e.module.establish(user, room)
	e.module.lightFire(user, room)
	text := e.module.startRest(user, room)
	assert.Nil(t, e.store.saved.Camps[7].Rest.Song)
	assert.NotContains(t, text, "plays")
}

func TestWindsRestoreExtraFatigue(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyWinds, 0) // 1 + reed pipes 2 = 3
	carry(e.module, reedPipesID)
	e.restThrough(t, user, nil)
	assert.Equal(t, []int{camping.FatigueRecovery + 3}, e.surv.amounts)
}

func TestDrumsGrantDrumbeatAndBattleEndsIt(t *testing.T) {
	e, user := heroEnv(t)
	e.companion.Health, e.companion.RoomId = 10, 100
	teach(e.module, 7, leaderKey, camping.FamilyDrums, 0) // 1 + fine 3 = 4: +2 speed
	carry(e.module, fineDrumID)
	e.restThrough(t, user, nil)
	assert.True(t, e.ledger.held[user.Character][9402], "Drumbeat +2 speed")
	assert.True(t, e.ledger.held[e.companion][9402], "the whole company drums along")

	e.module.onBattleEndedMusic(events.BattleEnded{UserId: 7})
	assert.False(t, e.ledger.held[user.Character][9402], "gone after the first battle")
}

func TestSpoiledRestGivesNoSongGifts(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyDrums, 0)
	teach(e.module, 7, musicKeyOf(1), camping.FamilyWinds, 0)
	carry(e.module, fineDrumID, fineFluteID)
	e.restThrough(t, user, func() {
		camp := e.module.camps[7]
		rest := *camp.Rest
		rest.Broken = true
		camp.Rest = &rest
		e.module.camps[7] = camp
	})
	assert.Equal(t, []int{camping.FatigueRecovery}, e.surv.amounts, "no winds bonus")
	assert.Empty(t, *e.buffs, "no Rested, no Drumbeat")
	assert.Zero(t, e.module.musicSkills[7][leaderKey].Songs, "a spoiled rest is no practice")
}

func TestPracticeRaisesLevelAndSaves(t *testing.T) {
	e, user := heroEnv(t)
	messages := captureMessages(t)
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 9)
	carry(e.module, lyreID)
	e.restThrough(t, user, nil)
	assert.Equal(t, 10, e.module.musicSkills[7][leaderKey].Songs)
	assert.Equal(t, 2, e.module.musicSkills[7][leaderKey].Level())
	assert.Equal(t, 10, e.store.saved.MusicSkills[7][leaderKey].Songs, "practice is saved")
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "playing reaches level 2")

	e.restThrough(t, user, nil) // the grant is once per rest
}

func TestEnsembleMakesWellRestedAndLargeTentDoesNotStack(t *testing.T) {
	// Pure: the ensemble upgrade and a large tent give one Well Rested.
	settings := defaultInnSettings()
	ensemble := camping.PlanSong([]camping.Player{
		{Key: "a", Family: camping.FamilyStrings, Level: 2, Tier: camping.InstrumentFine},
		{Key: "b", Family: camping.FamilyWinds, Level: 2, Tier: camping.InstrumentFine},
		{Key: "c", Family: camping.FamilyDrums, Level: 2, Tier: camping.InstrumentFine},
	}, false)
	require.True(t, ensemble.Ensemble())
	tier, _ := restBuffFor(settings, camping.TierRested, nil, &ensemble)
	assert.Equal(t, camping.TierWellRested, tier)

	large := &camping.Tent{WellRested: true, RestedPct: 100}
	tentTier, tentDur := restBuffFor(settings, camping.TierRested, large, nil)
	bothTier, bothDur := restBuffFor(settings, camping.TierRested, large, &camping.Song{Plays: ensemble.Plays[:0]})
	assert.Equal(t, camping.TierWellRested, tentTier)
	assert.Equal(t, tentTier, bothTier)
	assert.Equal(t, tentDur, bothDur)
	// With both, the ensemble adds only the strings' stretch, never a second tier.
	withBoth, durBoth := restBuffFor(settings, camping.TierRested, large, &ensemble)
	assert.Equal(t, camping.TierWellRested, withBoth)
	assert.Equal(t, tentDur*time.Duration(100+ensemble.RestedPct())/100, durBoth)
}

func TestEnsembleRestGrantsWellRested(t *testing.T) {
	e, user := heroEnv(t)
	musicTrio(e, 100)
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 25) // level 3
	teach(e.module, 7, musicKeyOf(1), camping.FamilyWinds, 25)
	teach(e.module, 7, musicKeyOf(2), camping.FamilyDrums, 25)
	carry(e.module, luteID, reedPipesID, frameDrumID) // 5, 5, 5: three strong families
	e.restThrough(t, user, nil)
	assert.True(t, e.ledger.held[user.Character][1030], "Well Rested from the ensemble")
	assert.False(t, e.ledger.held[user.Character][1033])
}

func TestSongRaisesRaidChanceInTheSingleRoll(t *testing.T) {
	// A roll of 54 misses a 50% raid and is caught by 50 * 1.2 = 60.
	run := func(songFamily camping.Family, id int) int {
		w := newRaidWorld(t, 50, 54)
		w.m.itemCount = func(_, item int) int {
			if item == id {
				return 1
			}
			return 0
		}
		if id != 0 {
			teach(w.m, 7, leaderKey, songFamily, 0)
		}
		w.rest(t)
		if w.camp().Rest.Raid != nil {
			return 1
		}
		return 0
	}
	assert.Equal(t, 0, run(camping.FamilyDrums, 0), "no song, no raid")
	assert.Equal(t, 1, run(camping.FamilyDrums, frameDrumID), "drums add 20%")
}

func TestStringsRaiseRaidChanceByTenPercent(t *testing.T) {
	w := newRaidWorld(t, 50, 54) // 50 * 1.1 = 55: 54 raids
	w.m.itemCount = func(_, id int) int {
		if id == luteID {
			return 1
		}
		return 0
	}
	teach(w.m, 7, leaderKey, camping.FamilyStrings, 0)
	w.rest(t)
	require.NotNil(t, w.camp().Rest.Raid)
}

func TestWetWeatherHalvesStringsUnlessTent(t *testing.T) {
	e, user := heroEnv(t)
	e.module.weatherIn = func(string) (weather.Condition, bool) { return weather.Condition{Name: "rain"}, true }
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 0)
	carry(e.module, fineFiddleID) // 1 + 3 = 4, wet: 2
	room := &rooms.Room{RoomId: 100, Zone: "Road", Tags: []string{"camping"}}
	dry := e.module.planSong(user, room, true)
	wet := e.module.planSong(user, room, false)
	assert.Equal(t, 4, dry.Strength(camping.FamilyStrings), "a pitched tent keeps the weather off")
	assert.Equal(t, 2, wet.Strength(camping.FamilyStrings))
}

func TestMusicOffSilencesTheSong(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 0)
	carry(e.module, luteID)
	assert.Contains(t, e.module.musicArgs(user, eligibleRoom(), []string{"off"}), "silence")
	assert.True(t, e.store.saved.MusicOff[7])
	assert.True(t, e.module.planSong(user, eligibleRoom(), false).Empty())
	e.module.musicArgs(user, eligibleRoom(), []string{"on"})
	assert.False(t, e.store.saved.MusicOff[7])
	assert.False(t, e.module.planSong(user, eligibleRoom(), false).Empty())
}

func teacherRoom() *rooms.Room {
	return &rooms.Room{RoomId: 2114, Title: "Alderbrook Green", Zone: "Alderbrook", Tags: []string{musicTeacherTag}}
}

func TestLearnPaysTeachesAndChangingFamilyResets(t *testing.T) {
	e, user := heroEnv(t)
	room := teacherRoom()
	user.Character.RoomId = room.RoomId
	user.Character.Gold = 100
	assert.Contains(t, e.module.musicArgs(user, eligibleRoom(), []string{"learn", "strings"}), "no music teacher")

	text := e.module.musicArgs(user, room, []string{"learn", "strings"})
	assert.Contains(t, text, "You pay 30 gold")
	assert.Equal(t, 70, user.Character.Gold)
	assert.Equal(t, camping.FamilyStrings, e.store.saved.MusicSkills[7][leaderKey].Family)
	assert.Contains(t, e.module.musicArgs(user, room, []string{"learn", "strings"}), "already plays")

	teach(e.module, 7, leaderKey, camping.FamilyStrings, 12)
	text = e.module.musicArgs(user, room, []string{"learn", "drums"})
	assert.Contains(t, text, "starting over")
	assert.Equal(t, 70, user.Character.Gold, "nothing paid without confirm")
	assert.Equal(t, 12, e.module.musicSkills[7][leaderKey].Songs)
	e.module.musicArgs(user, room, []string{"learn", "drums", "confirm"})
	assert.Equal(t, 40, user.Character.Gold)
	assert.Equal(t, camping.MusicSkill{Family: camping.FamilyDrums}, e.module.musicSkills[7][leaderKey])
}

func TestLearnNeedsGold(t *testing.T) {
	e, user := heroEnv(t)
	user.Character.Gold = 5
	assert.Contains(t, e.module.musicArgs(user, teacherRoom(), []string{"learn", "voice"}), "asks 30 gold")
	assert.Empty(t, e.module.musicSkills[7])
}

func TestMusicStatusShowsPlayersAndEffects(t *testing.T) {
	e, user := heroEnv(t)
	e.companion.RoomId = 100
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 0)
	carry(e.module, luteID)
	text := e.module.musicArgs(user, eligibleRoom(), nil)
	assert.Contains(t, text, "travelling lute")
	assert.Contains(t, text, "Rested and Well Rested last 15% longer")
	assert.Contains(t, text, "Bran: no music yet")
}

func musicItemSpecs(t *testing.T) {
	t.Helper()
	for _, spec := range []*items.ItemSpec{
		{ItemId: 3200, Name: "gut-strung lyre", Type: items.Object, Weight: 800},
		{ItemId: 3202, Name: "inlaid fiddle", Type: items.Object, Weight: 900},
		{ItemId: 201, Name: "elk antlers", Type: items.Object, Weight: 300},
		{ItemId: 29, Name: "raw game meat", Type: items.Food, Weight: 500},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
}

func TestCraftRefusals(t *testing.T) {
	musicItemSpecs(t)
	e, user := heroEnv(t)
	assert.Contains(t, e.module.musicArgs(user, eligibleRoom(), []string{"craft", "tuba"}), "no instrument called")
	assert.Contains(t, e.module.musicArgs(user, eligibleRoom(), []string{"craft", "fiddle"}), "recipe page")
	assert.Contains(t, e.module.musicArgs(user, eligibleRoom(), []string{"craft", "lyre"}), "make camp first")
	room := eligibleRoom()
	e.module.establish(user, room)
	assert.Contains(t, e.module.musicArgs(user, room, []string{"craft", "lyre"}), "You need elk antlers, raw game meat")
}

func TestCraftMakesACrudeInstrumentFromGatheredGoods(t *testing.T) {
	musicItemSpecs(t)
	e, user := heroEnv(t)
	room := eligibleRoom()
	e.module.establish(user, room)
	have := map[int]int{201: 1, 29: 1}
	e.module.itemCount = func(_, id int) int { return have[id] }
	spent := []int{}
	e.module.spendItem = func(_, id int) bool { spent = append(spent, id); have[id]--; return true }
	text := e.module.musicArgs(user, room, []string{"craft", "lyre"})
	assert.Contains(t, text, "You make a")
	assert.ElementsMatch(t, []int{201, 29}, spent)
	assert.Equal(t, 1, len(user.Character.GetAllBackpackItems())+len(room.Items), "the lyre is in the pack or by the fire")
}

// --- persistence ---

func TestMusicStateSurvivesReload(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyDrums, 11)
	e.module.musicArgs(user, eligibleRoom(), []string{"off"})
	carry(e.module, frameDrumID)
	e.module.musicArgs(user, eligibleRoom(), []string{"on"})
	e.module.musicArgs(user, eligibleRoom(), []string{"off"})

	child := newTestModule(e.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *e.now })
	child.load()
	require.NoError(t, child.loadErr)
	assert.Equal(t, camping.MusicSkill{Family: camping.FamilyDrums, Songs: 11}, child.musicSkills[7][leaderKey])
	assert.True(t, child.musicOff[7])
}

func TestPendingSongSurvivesReloadAndGrantsOnce(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyDrums, 0)
	carry(e.module, fineDrumID)
	room := eligibleRoom()
	e.module.establish(user, room)
	e.module.lightFire(user, room)
	e.module.startRest(user, room)
	*e.now = e.now.Add(camping.RestDuration)
	e.scheduler.fireLatest()
	require.True(t, e.store.saved.RestedPending[7])
	require.NotEmpty(t, e.store.saved.RestedSongs[7].Plays, "the song waits with the pending grant")

	child := newTestModule(e.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *e.now })
	child.load()
	require.NoError(t, child.loadErr)
	calls := []buffCall{}
	ledger := installLedger(child, &calls)
	child.companionsOf = e.module.companionsOf
	child.onNewRound(events.NewRound{RoundNumber: 1})
	child.onNewRound(events.NewRound{RoundNumber: 2})
	assert.True(t, ledger.held[user.Character][9402])
	assert.Equal(t, 1, child.musicSkills[7][leaderKey].Songs, "practised once")
	assert.Empty(t, e.store.saved.RestedSongs, "cleared with the grant")
}

func TestPurgeForgetsMusic(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyDrums, 11)
	e.module.musicArgs(user, eligibleRoom(), []string{"off"})
	e.module.mu.Lock()
	assert.True(t, e.module.purgeLocked(7))
	e.module.mu.Unlock()
	assert.Empty(t, e.module.musicSkills[7])
	assert.False(t, e.module.musicOff[7])
}

// --- inn gigs ---

type gigEnv struct {
	*innEnv
	user *users.UserRecord
	hour int
	day  uint64
}

func newGigEnv(t *testing.T) *gigEnv {
	e, user := heroEnv(t)
	g := &gigEnv{innEnv: e, user: user, hour: 19, day: 40}
	e.module.worldClock = func() (int, uint64) { return g.hour, g.day }
	e.module.zoneBand = func(string) int { return 10 }
	user.Character.RoomId = innRoom().RoomId
	e.companion.RoomId = innRoom().RoomId
	teach(e.module, 7, leaderKey, camping.FamilyStrings, 0)
	teach(e.module, 7, musicKeyOf(1), camping.FamilyDrums, 0)
	carry(e.module, luteID, frameDrumID)
	return g
}

func (g *gigEnv) finish() {
	*g.now = g.now.Add(camping.GigDuration)
	g.scheduler.fireLatest()
}

func TestGigNeedsTheWindowAndTwoFamilies(t *testing.T) {
	g := newGigEnv(t)
	g.hour = 12
	assert.Contains(t, g.module.innGig(g.user, innRoom()), "19:00 to 21:00")
	g.hour = 19
	teach(g.module, 7, musicKeyOf(1), camping.FamilyStrings, 0) // one family only
	assert.Contains(t, g.module.innGig(g.user, innRoom()), "at least 2 families")
	assert.Empty(t, g.module.gigLogs[7].Current)
	assert.Contains(t, g.module.innGig(g.user, ineligibleRoom()), "no inn here")
}

func TestGigPaysOnceOnTheGameLoop(t *testing.T) {
	g := newGigEnv(t)
	g.user.Character.Gold = 0
	text := g.module.innGig(g.user, innRoom())
	require.Contains(t, text, "takes the floor")
	require.NotNil(t, g.store.saved.Gigs[7].Current, "the gig is saved at once")
	pay := g.store.saved.Gigs[7].Current.Pay
	assert.Greater(t, pay, 0)
	assert.Zero(t, g.user.Character.Gold, "nothing until it ends")

	g.finish()
	assert.Zero(t, g.user.Character.Gold, "the timer never pays")
	g.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, pay, g.user.Character.Gold)
	g.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, pay, g.user.Character.Gold, "settled once")
	assert.True(t, g.store.saved.Gigs[7].Current.Settled)
	assert.Equal(t, 1, g.module.musicSkills[7][leaderKey].Songs, "a gig is practice too")
}

func TestGigLimitsPerEveningAndPerThreeHours(t *testing.T) {
	g := newGigEnv(t)
	require.Contains(t, g.module.innGig(g.user, innRoom()), "takes the floor")
	g.finish()
	g.module.onNewRound(events.NewRound{RoundNumber: 1})

	assert.Contains(t, g.module.innGig(g.user, innRoom()), "already played here this evening")
	other := innRoom()
	other.RoomId = 2004
	assert.Contains(t, g.module.innGig(g.user, other), "not long ago", "the 3 hour limit holds at another inn")

	*g.now = g.now.Add(camping.GigCooldown + time.Minute)
	g.day++
	assert.Contains(t, g.module.innGig(g.user, innRoom()), "takes the floor", "the next evening, three hours on")
}

func TestGigHoldsTheCompanyUntilItEnds(t *testing.T) {
	g := newGigEnv(t)
	require.Contains(t, g.module.innGig(g.user, innRoom()), "takes the floor")
	g.module.mu.Lock()
	blocked, why := g.module.gigBlockLocked(7)
	g.module.mu.Unlock()
	assert.True(t, blocked)
	assert.Contains(t, why, "won't let you leave")
	assert.Contains(t, g.module.innGig(g.user, innRoom()), "already playing")
	assert.Contains(t, g.module.innRest(g.user, innRoom()), "playing")

	g.finish()
	g.module.onNewRound(events.NewRound{RoundNumber: 1})
	g.module.mu.Lock()
	blocked, _ = g.module.gigBlockLocked(7)
	g.module.mu.Unlock()
	assert.False(t, blocked)
}

func TestGigSurvivesRestartAndPaysOnce(t *testing.T) {
	g := newGigEnv(t)
	g.user.Character.Gold = 0
	require.Contains(t, g.module.innGig(g.user, innRoom()), "takes the floor")
	pay := g.store.saved.Gigs[7].Current.Pay
	*g.now = g.now.Add(camping.GigDuration + time.Second) // finished while the server was down

	child := newTestModule(g.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *g.now })
	child.load()
	require.NoError(t, child.loadErr)
	child.onNewRound(events.NewRound{RoundNumber: 1})
	child.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Equal(t, pay, g.user.Character.Gold, "paid once after the reload")

	again := newTestModule(g.store, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return *g.now })
	again.load()
	again.onNewRound(events.NewRound{RoundNumber: 3})
	assert.Equal(t, pay, g.user.Character.Gold, "a second reload doesn't pay again")
}

func TestGigStatusBoard(t *testing.T) {
	g := newGigEnv(t)
	text := g.module.gigStatus(g.user, innRoom())
	assert.Contains(t, text, "Your company can play now")
	g.hour = 8
	text = g.module.gigStatus(g.user, innRoom())
	assert.Contains(t, text, "19:00 to 21:00")
	assert.Contains(t, text, "would earn about")
}

func TestGigNoticeForTheWebClient(t *testing.T) {
	g := newGigEnv(t)
	notice, song := g.module.gigNotice(g.user, innRoom())
	assert.True(t, notice.Ready)
	assert.Equal(t, 2, notice.Families)
	assert.Equal(t, camping.GigPay(10, song), notice.Pay)
}

// --- review regressions ---

// Review: the five Drumbeat sizes are separate buffs, so a second rest
// before any battle must replace the first lift, never stack on it.
func TestASecondRestReplacesTheDrumbeat(t *testing.T) {
	e, user := heroEnv(t)
	teach(e.module, 7, leaderKey, camping.FamilyDrums, 0)
	carry(e.module, fineDrumID) // 1 + 3 = 4: +2 speed
	e.restThrough(t, user, nil)
	require.True(t, e.ledger.held[user.Character][9402])
	carry(e.module, frameDrumID) // 1 + 2 = 3: +1 speed
	e.restThrough(t, user, nil)
	assert.True(t, e.ledger.held[user.Character][9401], "the new rest's lift")
	assert.False(t, e.ledger.held[user.Character][9402], "the old lift is gone, not stacked")
}

// Review: an older character (Legacy, who keeps every dish) still needs a
// fine instrument's page; the page teaches it without ending the book.
func TestFineInstrumentNeedsItsPageEvenForAnOlderCharacter(t *testing.T) {
	musicItemSpecs(t)
	for _, spec := range []*items.ItemSpec{
		{ItemId: luteID, Name: "travelling lute", Type: items.Object, Weight: 1500},
		{ItemId: ashwoodPlankItemID, Name: "ashwood plank", Type: items.Object, Weight: 500},
	} {
		items.SetTestItemSpec(spec)
		id := spec.ItemId
		t.Cleanup(func() { items.RemoveTestItemSpec(id) })
	}
	e, user := heroEnv(t)
	user.Character.Created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	delete(user.Character.MiscData, cookbook.BookKey)
	require.True(t, cookbook.Legacy(user.Character))
	room := eligibleRoom()
	e.module.establish(user, room)
	have := map[int]int{luteID: 1, ashwoodPlankItemID: 2}
	e.module.itemCount = func(_, id int) int { return have[id] }
	e.module.spendItem = func(_, id int) bool { have[id]--; return true }
	assert.Contains(t, e.module.musicArgs(user, room, []string{"craft", "fiddle"}), "recipe page")

	assert.True(t, cookbook.LearnPattern(user.Character, fineFiddleID))
	assert.True(t, cookbook.Legacy(user.Character), "learning a pattern keeps the old recipe book")
	assert.Contains(t, e.module.musicArgs(user, room, []string{"craft", "fiddle"}), "You make a")
	assert.Equal(t, map[int]int{luteID: 0, ashwoodPlankItemID: 0}, have, "the lute and planks went into it")
}

// Review: a leader who dies mid-gig is never paid for it, and neither the
// evening nor the three-hour limit is spent.
func TestDeathMidGigPaysNothingAndSpendsNoLimit(t *testing.T) {
	g := newGigEnv(t)
	g.user.Character.Gold = 0
	require.Contains(t, g.module.innGig(g.user, innRoom()), "takes the floor")
	require.NoError(t, g.module.AbandonForDeath(7))
	g.finish()
	g.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Zero(t, g.user.Character.Gold, "no pay for a gig cut short")
	assert.Contains(t, g.module.innGig(g.user, innRoom()), "takes the floor", "the evening and cooldown are unspent")
}
