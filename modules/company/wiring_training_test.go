package company

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35c wiring: "company train" through the real command, store,
// spawn, and restart in the brawl world.

// trainArchetypes is fakeArchetypes with optional skills: Cooking for
// anyone, Map (standing in for 36a's Scribe) for wizards and clerics.
type trainArchetypes struct{ growthArchetypes }

func (trainArchetypes) OptionalSkills() []archetypes.OptionalSkill {
	return []archetypes.OptionalSkill{
		{Skill: "cooking", Archetypes: []string{archetypes.AnyArchetype}, MaxRank: 4},
		{Skill: "map", Archetypes: []string{"wizard", "cleric"}, MaxRank: 4},
	}
}

// fakeCampRest stands in for modules/camping: whether the leader rests,
// and whether their own camp is in the room they stand in.
type fakeCampRest struct{ resting, campHere bool }

func (f *fakeCampRest) MovementBlocked(int) (bool, string) { return f.resting, "resting" }
func (f *fakeCampRest) CampStateOf(int, int, []string) (camping.CampState, bool) {
	return camping.CampState{HasCamp: f.campHere, Here: f.campHere}, true
}

// fakeJourney stands in for modules/expedition.
type fakeJourney struct{ travelling bool }

func (f *fakeJourney) MovementBlocked(int) (bool, string) { return f.travelling, "travelling" }

type trainWorld struct {
	*brawl
	camp    *fakeCampRest
	journey *fakeJourney
}

// trainingBrawl is the brawl with Tamsin a level-6 warrior and Oswin a
// cleric, respawned from their records, no camp and no trainer yet.
func trainingBrawl(t *testing.T) *trainWorld {
	t.Helper()
	b := newBrawl(t)
	archetypes.SetProvider(trainArchetypes{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	for id, arch := range map[int]string{1: "warrior", 2: "cleric"} {
		require.NoError(t, module.registry.SetCompanionArchetype(7, id, arch))
	}
	// The logout snapshots the live level into the record; the login
	// spawns from it.
	b.companion(1).Character.Level = 6
	b.respawn()
	require.Equal(t, 6, b.companion(1).Character.Level)

	w := &trainWorld{brawl: b, camp: &fakeCampRest{}, journey: &fakeJourney{}}
	camping.SetMovementProvider(w.camp)
	expedition.SetMovementProvider(w.journey)
	t.Cleanup(func() {
		camping.SetMovementProvider(nil)
		expedition.SetMovementProvider(nil)
	})
	b.road.SkillTraining = nil
	t.Cleanup(func() { b.road.SkillTraining = nil })
	return w
}

func (w *trainWorld) trainer(lo, hi int) {
	w.road.SkillTraining = map[string]rooms.TrainingRange{"cooking": {Min: lo, Max: hi}}
}

func rankOf(t *testing.T, companionID int, skill string) int {
	t.Helper()
	record, ok := module.registry.Get(7)
	require.True(t, ok)
	for _, c := range record.Companions {
		if c.ID == companionID {
			return c.SkillRank(skill)
		}
	}
	t.Fatalf("no companion %d", companionID)
	return 0
}

// TestCompanyTrainPreviewConfirmAndRepeat is the plan's acceptance: a
// level-6 companion has 6 points; Cooking 1 then 2 leaves 3. The preview
// changes nothing, the confirm saves and teaches the live mob, and the same
// confirm again does nothing.
func TestCompanyTrainPreviewConfirmAndRepeat(t *testing.T) {
	w := trainingBrawl(t)
	w.camp.campHere = true
	round := util.GetRoundCount()

	view := w.cmd("company", "train")
	assert.Contains(t, view, "#1 Tamsin Reed, Warrior, level 6: 6 training points")
	assert.Contains(t, view, "Cooking rank 0: rank 1 costs 1")
	assert.NotContains(t, w.cmd("company", "train tamsin"), "Map", "a warrior can't learn the casters' skill")
	assert.Contains(t, view, "Preview with company train [member] [skill], then add confirm.")
	assert.Contains(t, w.cmd("company", "train"), "#2 Brother Oswin, Cleric")

	preview := w.cmd("company", "train tamsin cooking")
	assert.Contains(t, preview, "Tamsin Reed can learn Cooking rank 1 at your camp for 1 training point (6 now, 5 after).")
	assert.Contains(t, preview, "Cooking: Turn raw game")
	assert.Contains(t, preview, "Type company train #1 cooking 1 confirm to train.")
	assert.Zero(t, rankOf(t, 1, "cooking"), "a preview changes nothing")

	assert.Contains(t, w.cmd("company", "train #1 cooking 1 confirm"), "Tamsin Reed learns Cooking rank 1 at your camp (5 training points left).")
	assert.Equal(t, 1, rankOf(t, 1, "cooking"))
	assert.Equal(t, 1, w.companion(1).Character.GetSkillLevel("cooking"), "the live companion knows it at once")
	stored := domain.NewRegistry()
	require.NoError(t, module.store.Load(stored))
	rec, _ := stored.Get(7)
	assert.Equal(t, map[string]int{"cooking": 1}, rec.Companions[0].Skills, "saved at once")

	assert.Contains(t, w.cmd("company", "train #1 cooking 1 confirm"), "already has Cooking rank 1; there is nothing to do")
	assert.Equal(t, 1, rankOf(t, 1, "cooking"), "a repeated confirm is idempotent")
	assert.Contains(t, w.cmd("company", "train #1 cooking 3 confirm"), "must learn Cooking rank 2 first")

	assert.Contains(t, w.cmd("company", "train tamsin cooking 2 confirm"), "learns Cooking rank 2 at your camp (3 training points left)")
	assert.Contains(t, w.cmd("company", "train"), "Tamsin Reed, Warrior, level 6: 3 training points")
	inspect := w.cmd("company", "inspect #1")
	assert.Contains(t, inspect, "Optional skills: Cooking rank 2.")
	assert.Contains(t, inspect, "Training: 3 training points.")
	assert.Equal(t, round, util.GetRoundCount(), "training never touches the clock")

	// Restart or copyover: only the ranks were saved, so the points are
	// worked out again and none appear.
	w.respawn()
	require.NoError(t, module.save())
	module.instances = map[int]map[int]int{}
	module.load()
	require.NoError(t, module.loadErr)
	require.NoError(t, module.restoreForLeader(7, w.road.RoomId))
	assert.Equal(t, 2, w.companion(1).Character.GetSkillLevel("cooking"), "the rank survives a restart")
	assert.Contains(t, w.cmd("company", "train"), "Tamsin Reed, Warrior, level 6: 3 training points")
	members, ok := domain.CompanyMembers(7)
	require.True(t, ok)
	assert.Equal(t, map[string]int{"cooking": 2}, members[0].Skills)
	assert.Equal(t, 3, members[0].TrainingPoints)
	assert.Contains(t, w.cmd("experience", ""), "(3 training points)")
}

// TestCompanyTrainWhere: ranks 1-2 at the leader's own camp or a trainer
// teaching that rank; ranks 3-4 only at such a trainer.
func TestCompanyTrainWhere(t *testing.T) {
	w := trainingBrawl(t)
	assert.Contains(t, w.cmd("company", "train tamsin cooking"), "can't learn Cooking rank 1 here. Rank 1 can be learned at your own camp or from a trainer who teaches it.")
	w.trainer(1, 4)
	assert.Contains(t, w.cmd("company", "train tamsin cooking 1 confirm"), "learns Cooking rank 1 at the trainer here")
	w.road.SkillTraining = nil
	w.camp.campHere = true
	assert.Contains(t, w.cmd("company", "train tamsin cooking 2 confirm"), "learns Cooking rank 2 at your camp")

	// Rank 3: never at camp, only from a trainer teaching rank 3.
	w.companion(1).Character.Level = 9
	w.respawn()
	assert.Contains(t, w.cmd("company", "train tamsin cooking"), "Rank 3 can only be learned from a trainer who teaches it.")
	w.trainer(1, 2)
	assert.Contains(t, w.cmd("company", "train tamsin cooking"), "Rank 3 can only be learned from a trainer")
	w.trainer(3, 4)
	assert.Contains(t, w.cmd("company", "train tamsin cooking 3 confirm"), "learns Cooking rank 3 at the trainer here")
	assert.Equal(t, 3, rankOf(t, 1, "cooking"))
}

// TestCompanyTrainRefusals: each refusal names its reason and trains
// nothing.
func TestCompanyTrainRefusals(t *testing.T) {
	w := trainingBrawl(t)
	w.trainer(1, 4)
	refuse := func(rest, want string) {
		t.Helper()
		assert.Contains(t, w.cmd("company", "train "+rest), want)
		assert.Zero(t, rankOf(t, 1, "cooking"), rest)
	}
	refuse("nobody cooking", "You have no companion like that.")
	refuse("r cooking", "More than one companion answers to that; use their number (#N).")
	refuse("tamsin brawling", "can't be trained in that. Companions learn only optional skills")
	refuse("tamsin map", "A Warrior can't learn Map.")

	w.journey.travelling = true
	refuse("tamsin cooking 1 confirm", "Not while your company is travelling.")
	w.journey.travelling = false
	w.camp.resting = true
	refuse("tamsin cooking 1 confirm", "Not while your company is resting.")
	w.camp.resting = false

	// A member who isn't with the leader, or is fighting.
	tamsin := w.companion(1)
	require.True(t, nativeRuntime{}.Relocate(tamsin.InstanceId, 920102))
	refuse("tamsin cooking 1 confirm", "Tamsin Reed isn't here with you to train.")
	require.True(t, nativeRuntime{}.Relocate(tamsin.InstanceId, w.road.RoomId))
	tamsin.Character.SetAggro(0, w.bandits["bandit captain"][0], characters.DefaultAttack)
	refuse("tamsin cooking 1 confirm", "Tamsin Reed is busy fighting.")
	tamsin.Character.Aggro = nil

	// The fallen and the separated.
	record, _ := module.registry.Get(7)
	record.Companions[0].Death = &domain.CompanionDeath{Remaining: 60}
	module.registry.Put(record)
	refuse("tamsin cooking 1 confirm", "Tamsin Reed has fallen")
	record.Companions[0].Death = nil
	record.Companions[0].Separation = &domain.Separation{RoundsLeft: 5}
	module.registry.Put(record)
	refuse("tamsin cooking 1 confirm", "Tamsin Reed is separated from the company")
	record.Companions[0].Separation = nil
	module.registry.Put(record)

	// Not enough points: Ysolde is level 2 (2 points); rank 1 costs 1, so
	// rank 2 (2 more) is short.
	assert.Contains(t, w.cmd("company", "train ysolde cooking 1 confirm"), "learns Cooking rank 1")
	assert.Contains(t, w.cmd("company", "train ysolde cooking 2 confirm"), "Ysolde needs 2 training points for Cooking rank 2 and has 1 training point.")
	assert.Equal(t, 1, rankOf(t, 4, "cooking"))

	// Already at the most a companion can learn.
	require.NoError(t, module.registry.SetSkillRank(7, 1, "cooking", 4))
	assert.Contains(t, w.cmd("company", "train tamsin cooking"), "the most a companion can learn")
	assert.Contains(t, w.cmd("company", "train"), "Cooking rank 4 (mastered)")
}

// TestCompanyTrainInBattle: refused in a battle, the view stays open.
func TestCompanyTrainInBattle(t *testing.T) {
	w := trainingBrawl(t)
	w.trainer(1, 4)
	w.start()
	assert.Contains(t, w.cmd("company", "train tamsin cooking 1 confirm"), "Not in the middle of a battle")
	assert.Zero(t, rankOf(t, 1, "cooking"))
	assert.Contains(t, w.cmd("company", "train"), "Company training")
}

// TestCompanyTrainSaveFailureRollsBack: a failed save trains nothing, in
// the record or on the live mob, and says so.
func TestCompanyTrainSaveFailureRollsBack(t *testing.T) {
	w := trainingBrawl(t)
	w.trainer(1, 4)
	real := module.store
	module.store = &fakeStore{saveErr: errors.New("disk full")}
	t.Cleanup(func() { module.store = real })
	assert.Contains(t, w.cmd("company", "train tamsin cooking 1 confirm"), "couldn't be saved; nothing was trained")
	assert.Zero(t, rankOf(t, 1, "cooking"))
	assert.Zero(t, w.companion(1).Character.GetSkillLevel("cooking"))
	module.store = real
	assert.Contains(t, w.cmd("company", "train tamsin cooking 1 confirm"), "learns Cooking rank 1")
}

// TestLostLevelKeepsRanksAndOwesPoints: a level lost to death keeps the
// ranks; the points go negative, show as owed, and train nothing until
// later levels repay them. Nothing mints points.
func TestLostLevelKeepsRanksAndOwesPoints(t *testing.T) {
	w := trainingBrawl(t)
	w.trainer(1, 4)
	w.cmd("company", "train tamsin cooking 1 confirm")
	w.cmd("company", "train tamsin cooking 2 confirm")
	w.cmd("company", "train tamsin cooking 3 confirm") // costs 3 of the 3 left
	require.Equal(t, 3, rankOf(t, 1, "cooking"))

	w.companion(1).Character.Level = 5
	assert.Contains(t, w.cmd("company", "train"), "level 5: 0 training points (1 owed from a lost level)")
	assert.Contains(t, w.cmd("company", "train tamsin cooking"), "needs 4 training points for Cooking rank 4 and has 0 training points (1 owed")
	assert.Equal(t, 3, rankOf(t, 1, "cooking"), "the ranks are never removed")
	members, _ := domain.CompanyMembers(7)
	assert.Zero(t, members[0].TrainingPoints, "the views never show a negative")

	w.companion(1).Character.Level = 10
	assert.Contains(t, w.cmd("company", "train"), "level 10: 4 training points", "10 earned, 6 spent")
}
