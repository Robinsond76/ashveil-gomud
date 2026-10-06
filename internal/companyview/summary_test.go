package companyview

import (
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noSources reports nothing, as with no modules registered.
func noSources() sources {
	return sources{
		members:            func(int) ([]company.MemberView, bool) { return nil, false },
		needs:              func(int) []survival.MemberNeeds { return nil },
		load:               func(int) (encumbrance.Load, bool) { return encumbrance.Load{}, false },
		band:               func(int) (encumbrance.LoadBand, bool) { return encumbrance.LoadBand{}, false },
		journey:            func(int) (expedition.Progress, bool) { return expedition.Progress{}, false },
		rest:               func(int) (camping.RestActivity, bool) { return camping.RestActivity{}, false },
		tier:               func(int) (camping.Tier, time.Duration, bool) { return camping.TierNone, 0, false },
		exposure:           func(int, string) (int, bool) { return 0, false },
		light:              func(*users.UserRecord) (int, bool) { return 0, false },
		archetype:          func(int) (string, bool) { return "", false },
		archetypeReporting: func() bool { return false },
		journeyReporting:   func() bool { return false },
		restReporting:      func() bool { return false },
		name:               func(string) (string, bool) { return "", false },
		room:               func(int) *rooms.Room { return nil },
		formation:          func(int) (company.Formation, bool) { return company.Formation{}, false },
		strategy:           func(int, company.MemberKey) strategy.Strategy { return strategy.Strategy{} },
	}
}

func testUser() *users.UserRecord {
	u := users.NewUserRecord(7, 1)
	u.Character.Name = "Wren"
	u.Character.Level = 4
	u.Character.Alignment = 40
	u.Character.Health = 20
	u.Character.HealthMax.Value = 30
	return u
}

// fullSources: a leader and three companions (present, awaiting, dead),
// needs, load, a camp rest, Rested, dim light, a checkpoint.
func fullSources() sources {
	src := noSources()
	src.members = func(int) ([]company.MemberView, bool) {
		return []company.MemberView{
			{ID: 1, Name: "Bran", Status: company.MemberPresent, Level: 3, Archetype: "warrior", HP: 12, HPMax: 25, Placed: true, Row: 0, Col: 1},
			{ID: 2, Name: "Bran", Status: company.MemberAwaiting, Level: 2},
			{ID: 3, Name: "Ysolde", Status: company.MemberDead, Level: 5, RescueSeconds: 5400},
		}, true
	}
	src.needs = func(int) []survival.MemberNeeds {
		return []survival.MemberNeeds{
			{Key: company.LeaderMemberKey, Needs: survival.Needs{Hunger: 40, Thirst: 90, Fatigue: 40}},
			{Key: company.CompanionMemberKey(1), Needs: survival.Needs{Hunger: 70, Thirst: 70, Fatigue: 70}},
		}
	}
	src.load = func(int) (encumbrance.Load, bool) {
		return encumbrance.Load{PersonalGrams: 5000, CargoGrams: 3000, CapacityGrams: 10000}, true
	}
	src.band = func(int) (encumbrance.LoadBand, bool) {
		return encumbrance.LoadBand{MinRatio: 0.75, TravelDurationPct: 120, FatiguePct: 120}, true
	}
	src.rest = func(int) (camping.RestActivity, bool) {
		return camping.RestActivity{Resting: true, Remaining: 12 * time.Minute}, true
	}
	src.tier = func(int) (camping.Tier, time.Duration, bool) { return camping.TierRested, time.Hour, true }
	src.exposure = func(_ int, key string) (int, bool) {
		if key == "leader" {
			return -30, true
		}
		return 0, true
	}
	src.light = func(*users.UserRecord) (int, bool) { return 1, true }
	src.archetype = func(int) (string, bool) { return "ranger", true }
	src.archetypeReporting = func() bool { return true }
	src.journeyReporting = func() bool { return true }
	src.restReporting = func() bool { return true }
	src.name = func(id string) (string, bool) {
		return map[string]string{"ranger": "Ranger", "warrior": "Warrior"}[id], true
	}
	src.room = func(id int) *rooms.Room {
		if id == 2007 {
			return &rooms.Room{RoomId: 2007, Title: "The Chapel of the Wayfarer"}
		}
		return nil
	}
	return src
}

func TestSummaryLeaderAndCompanions(t *testing.T) {
	user := testUser()
	user.Character.SetMiscData(death.CheckpointKey, 2007)
	s := fullSources().summary(user)

	assert.Equal(t, "Wren", s.Leader.Name)
	assert.Equal(t, "Ranger", s.Leader.Archetype)
	assert.Equal(t, 20, s.Leader.HP)
	assert.Equal(t, Need{Known: true, Value: 40, Label: "Hungry"}, s.Leader.Hunger)
	assert.Equal(t, "Tired", s.Leader.Fatigue.Label)
	assert.Equal(t, "Chilled", s.Leader.Warmth)
	assert.Equal(t, company.DisplayAlignment(40), s.Alignment)

	require.True(t, s.CompanyKnown)
	require.Len(t, s.Companions, 3)
	assert.Equal(t, 3, s.Alive, "the leader and two living")
	assert.Equal(t, 1, s.Dead)
	bran := s.Companions[0]
	assert.True(t, bran.HasHP)
	assert.Equal(t, "Warrior", bran.Archetype)
	assert.True(t, bran.Hunger.Known)
	assert.True(t, bran.Placed)
	assert.False(t, s.Companions[1].HasHP, "awaiting: no live health")
	assert.False(t, s.Companions[1].Hunger.Known, "no needs reported")
	dead := s.Companions[2]
	assert.Equal(t, company.MemberDead, dead.Status)
	assert.Equal(t, 90*time.Minute, dead.RescueLeft)
	assert.False(t, dead.Hunger.Known)

	assert.Equal(t, "Burdened", s.LoadLabel)
	assert.Equal(t, "Resting 12m", s.Activity.Label())
	assert.Equal(t, camping.TierRested, s.RestTier)
	assert.Equal(t, 1, s.Light)
	assert.Equal(t, "The Chapel of the Wayfarer", s.Checkpoint)
	assert.Equal(t, []string{"Hungry", "Tired", "Dim", "Chilled"}, s.WarnWords())
}

func TestSummaryJourneyWinsOverCamp(t *testing.T) {
	src := fullSources()
	src.journey = func(int) (expedition.Progress, bool) { return expedition.Progress{Percent: 42}, true }
	assert.Equal(t, "Travelling 42%", src.summary(testUser()).Activity.Label())
	src.journey = func(int) (expedition.Progress, bool) { return expedition.Progress{Interrupted: true}, true }
	assert.Equal(t, "Stopped", src.summary(testUser()).Activity.Label())
	src = fullSources()
	src.rest = func(int) (camping.RestActivity, bool) {
		return camping.RestActivity{Inn: true, Resting: true, Remaining: time.Minute}, true
	}
	assert.Equal(t, "At inn 1m", src.summary(testUser()).Activity.Label())
	src.rest = func(int) (camping.RestActivity, bool) { return camping.RestActivity{}, true }
	assert.Equal(t, "Camped", src.summary(testUser()).Activity.Label())
}

func TestSummaryMissingProvidersAreUnknown(t *testing.T) {
	s := noSources().summary(testUser())
	assert.False(t, s.CompanyKnown)
	assert.Empty(t, s.Companions)
	assert.Equal(t, 1, s.Alive, "the leader")
	assert.False(t, s.Leader.Hunger.Known, "never a healthy default")
	assert.False(t, s.Leader.WarmthKnown)
	assert.False(t, s.LoadKnown)
	assert.Empty(t, s.LoadLabel)
	assert.False(t, s.RestKnown)
	assert.False(t, s.LightKnown)
	assert.False(t, s.ActivityKnown, "no provider can say what the company is doing")
	assert.False(t, s.Leader.ArchetypeKnown)
	assert.Empty(t, s.Checkpoint)
	assert.Empty(t, WarnCluster(s.WarnWords()), "nothing known, nothing warned")
}

func TestSummaryKeysByMemberID(t *testing.T) {
	s := fullSources().summary(testUser())
	assert.Equal(t, s.Companions[0].Name, s.Companions[1].Name, "two companions share a name")
	assert.NotEqual(t, s.Companions[0].Key, s.Companions[1].Key)
	assert.Equal(t, company.CompanionMemberKey(2), s.Companions[1].Key)
}

// TestSummaryIdleAndNoArchetypeAreKnown: with the providers present but
// nothing to report, the company is known to be idle and the leader known
// to have no archetype (review findings 1 and 4).
func TestSummaryIdleAndNoArchetypeAreKnown(t *testing.T) {
	src := noSources()
	src.journeyReporting = func() bool { return true }
	src.restReporting = func() bool { return true }
	src.archetypeReporting = func() bool { return true }
	s := src.summary(testUser())
	assert.True(t, s.ActivityKnown)
	assert.Equal(t, Idle, s.Activity.Kind)
	assert.True(t, s.Leader.ArchetypeKnown)
	assert.Empty(t, s.Leader.Archetype)
}

// TestCheckpointTitleRemembered: the church's title is read once, so a
// refresh never reloads an unloaded room (review finding 7).
func TestCheckpointTitleRemembered(t *testing.T) {
	loads := 0
	src := noSources()
	src.room = func(id int) *rooms.Room { loads++; return &rooms.Room{RoomId: id, Title: "Shrine"} }
	user := testUser()
	user.Character.SetMiscData(death.CheckpointKey, 424242)
	assert.Equal(t, "Shrine", src.summary(user).Checkpoint)
	assert.Equal(t, "Shrine", src.summary(user).Checkpoint)
	assert.Equal(t, 1, loads)
}

func TestSummaryLeaderCell(t *testing.T) {
	src := noSources()
	src.formation = func(int) (company.Formation, bool) {
		var f company.Formation
		_ = f.Place(company.LeaderMemberKey, 1, 2)
		return f, true
	}
	s := src.summary(testUser())
	assert.True(t, s.Leader.Placed)
	assert.Equal(t, [2]int{1, 2}, [2]int{s.Leader.Row, s.Leader.Col})
	assert.False(t, noSources().summary(testUser()).Leader.Placed)
}

// TestSummaryMana (Phase 32g): the leader's mana from their character; a
// present companion's from its view; none for a companion with no mana,
// one awaiting, or the fallen.
func TestSummaryMana(t *testing.T) {
	user := testUser()
	user.Character.Mana = 6
	user.Character.ManaMax.Value = 14
	src := fullSources()
	src.members = func(int) ([]company.MemberView, bool) {
		return []company.MemberView{
			{ID: 1, Name: "Oswin", Status: company.MemberPresent, HP: 10, HPMax: 20, MP: 8, MPMax: 20},
			{ID: 2, Name: "Bran", Status: company.MemberPresent, HP: 10, HPMax: 20},
			{ID: 3, Name: "Tamsin", Status: company.MemberAwaiting, MP: 5, MPMax: 5},
			{ID: 4, Name: "Ysolde", Status: company.MemberDead, RescueSeconds: 60},
		}, true
	}
	s := src.summary(user)

	assert.True(t, s.Leader.HasMP)
	assert.Equal(t, 6, s.Leader.MP)
	assert.Equal(t, 14, s.Leader.MPMax)
	require.Len(t, s.Companions, 4)
	assert.True(t, s.Companions[0].HasMP)
	assert.Equal(t, 8, s.Companions[0].MP)
	assert.Equal(t, 20, s.Companions[0].MPMax)
	assert.False(t, s.Companions[1].HasMP, "no mana to show")
	assert.False(t, s.Companions[2].HasMP, "awaiting: not live")
	assert.False(t, s.Companions[3].HasMP, "fallen")

	user.Character.ManaMax.Value = 0
	assert.False(t, src.summary(user).Leader.HasMP, "a leader with no mana")
}

// TestSummaryStrategies (Phase 32g): each member carries the strategy a
// battle would use; with no strategy source it stays unknown (zero).
func TestSummaryStrategies(t *testing.T) {
	src := fullSources()
	src.strategy = func(_ int, key company.MemberKey) strategy.Strategy {
		if key == company.LeaderMemberKey {
			return strategy.Strategy{Role: strategy.Caster, Rule: strategy.Weakest}
		}
		return strategy.Strategy{Role: strategy.Healer, Rule: strategy.Defend}
	}
	s := src.summary(testUser())
	assert.Equal(t, strategy.Strategy{Role: strategy.Caster, Rule: strategy.Weakest}, s.Leader.Strategy)
	require.Len(t, s.Companions, 3)
	for _, c := range s.Companions {
		assert.Equal(t, strategy.Strategy{Role: strategy.Healer, Rule: strategy.Defend}, c.Strategy, c.Name)
	}
	assert.True(t, noSources().summary(testUser()).Leader.Strategy.IsZero(), "no source: unknown")
}

// Phase 30b: the leader's and a present companion's wound limits.
func TestSummaryWoundLimits(t *testing.T) {
	user := testUser()
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 4}}
	src := fullSources()
	src.members = func(int) ([]company.MemberView, bool) {
		return []company.MemberView{
			{ID: 1, Name: "Tamsin", Status: company.MemberPresent, Level: 3, HP: 10, HPMax: 16, HPLimit: 13},
			{ID: 2, Name: "Bran", Status: company.MemberPresent, Level: 3, HP: 10, HPMax: 16},
		}, true
	}
	s := src.summary(user)
	assert.Equal(t, 26, s.Leader.HPLimit)
	assert.Equal(t, 13, s.Companions[0].HPLimit)
	assert.Equal(t, 16, s.Companions[1].HPLimit, "a view with no limit reads as unwounded")
}

// Phase 35d: a player with no saved focus is shown the level's default, the
// focus the battle really uses.
func TestSummaryTacticsShowLevelDefaultFocus(t *testing.T) {
	src := fullSources()
	src.tactics = func(int) strategy.Tactics { return strategy.Tactics{}.Resolve() }
	u := testUser()
	u.Character.Level = 12
	s := src.summary(u)
	assert.Equal(t, strategy.Weakest, s.Tactics.Focus)
	assert.Equal(t, strategy.DefaultPatch, s.Tactics.Patch)
	u.Character.Level = 4
	assert.Equal(t, strategy.NoFocus, src.summary(u).Tactics.Focus)
}

// TestCompanyLevelAveragesLeaderAndCompanions (37c): the level a zone band
// is rated against is the rounded average of everyone, the fallen included.
func TestCompanyLevelAveragesLeaderAndCompanions(t *testing.T) {
	mk := func(known bool, leader int, companions ...int) Summary {
		s := Summary{CompanyKnown: known, Leader: Member{Level: leader}}
		for _, l := range companions {
			s.Companions = append(s.Companions, Member{Level: l})
		}
		return s
	}
	assert.Equal(t, 8, CompanyLevel(mk(true, 8)), "a leader alone")
	assert.Equal(t, 8, CompanyLevel(mk(true, 6, 8, 10)))
	assert.Equal(t, 11, CompanyLevel(mk(true, 6, 12, 12, 12, 12)), "10.8 rounds up")
	assert.Equal(t, 6, CompanyLevel(mk(true, 12, 5, 5, 5, 5)), "6.4 rounds down")
	assert.Equal(t, 12, CompanyLevel(mk(false, 12, 1, 1, 1)), "unreadable company: the leader alone")
	assert.Equal(t, 1, CompanyLevel(mk(true, 0)), "never below 1")
}

// Phase 38c1: the summary names each member's class, tier, rank and
// promotion state, derived from lineage, class, level and alignment.
func TestSummaryMemberClasses(t *testing.T) {
	user := testUser()
	user.Character.Level = 30
	user.Character.Alignment = 41
	src := fullSources()
	src.members = func(int) ([]company.MemberView, bool) {
		return []company.MemberView{
			{ID: 1, Name: "Bran", Status: company.MemberPresent, Level: 52, Archetype: "warrior", Class: "warlord", Alignment: 0},
			{ID: 2, Name: "Tamsin", Status: company.MemberPresent, Level: 30, Archetype: "warrior", Class: "knight", Alignment: 29},
			{ID: 3, Name: "Oswin", Status: company.MemberPresent, Level: 30, Archetype: "warrior", Class: "mercenary", Alignment: -80},
			{ID: 4, Name: "Ysolde", Status: company.MemberPresent, Level: 12, Archetype: "cleric"},
		}, true
	}
	s := src.summary(user)
	assert.Empty(t, s.Leader.Class, "no class record: unpromoted")
	warlord := s.Companions[0]
	assert.Equal(t, [4]string{"warlord", "Warlord", "elite", ""}, [4]string{warlord.Class, warlord.ClassName, warlord.ClassTier, warlord.Promotion})
	assert.Equal(t, 50, warlord.ClassRank)
	waiting := s.Companions[1]
	assert.Equal(t, "advanced", waiting.ClassTier)
	assert.Equal(t, 25, waiting.ClassRank)
	assert.Equal(t, "waiting-gate", waiting.Promotion, "+29 does not meet the Knight's gate")
	assert.Equal(t, "ready", s.Companions[2].Promotion, "a Mercenary has no gate")
	assert.Equal(t, "ready", s.Companions[3].Promotion, "a base character at level 12 may take an advanced route")
}

type fakeClasses struct{ class string }

func (f fakeClasses) PlayerClass(int) classes.State { return classes.State{Class: f.class} }

// TestSummaryCarriesPromotedClass (Phase 40s5): the leader's class comes from
// the class provider, a companion's from its member view.
func TestSummaryCarriesPromotedClass(t *testing.T) {
	classes.SetProvider(fakeClasses{class: "knight"})
	t.Cleanup(func() { classes.SetProvider(nil) })
	src := fullSources()
	members, _ := src.members(1)
	members[0].Class = "druid"
	src.members = func(int) ([]company.MemberView, bool) { return members, true }
	s := src.summary(testUser())
	assert.Equal(t, "knight", s.Leader.Class)
	require.NotEmpty(t, s.Companions)
	assert.Equal(t, "druid", s.Companions[0].Class)
	assert.Equal(t, "", s.Companions[1].Class)
}
