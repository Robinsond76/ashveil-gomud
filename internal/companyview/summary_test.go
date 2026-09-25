package companyview

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
