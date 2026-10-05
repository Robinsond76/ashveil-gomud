package company

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33h1 wiring: companion growth through the real company command,
// spawn, live level-up, and respawn; contracts through the real quest hook.

// growthArchetypes is fakeArchetypes with the shipped growth weights.
type growthArchetypes struct{ fakeArchetypes }

func (growthArchetypes) CompanionGrowth(id string) (map[string]int, bool) {
	switch id {
	case "warrior":
		return map[string]int{"strength": 4, "vitality": 3, "speed": 2, "perception": 1}, true
	case "cleric":
		return map[string]int{"mysticism": 3, "vitality": 3, "smarts": 2, "strength": 2}, true
	}
	return nil, false
}

// growthBrawl is the company brawl with growth weights, Tamsin a warrior
// and Oswin a cleric, everyone respawned so the weights apply.
func growthBrawl(t *testing.T) *brawl {
	t.Helper()
	b := newBrawl(t)
	archetypes.SetProvider(growthArchetypes{})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	for id, arch := range map[int]string{1: "warrior", 2: "cleric"} {
		require.NoError(t, module.registry.SetCompanionArchetype(7, id, arch))
	}
	b.respawn()
	return b
}

// respawn is a logout and login: the live mobs are rebuilt from records.
func (b *brawl) respawn() {
	b.t.Helper()
	module.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	require.NoError(b.t, module.restoreForLeader(7, b.aria.Character.RoomId))
}

// wantTraining is the template's training plus the level's points dealt.
func wantTraining(t *testing.T, mob *mobs.Mob, w domain.GrowthWeights) [6]int {
	t.Helper()
	spec := mobs.GetMobSpec(mob.MobId)
	require.NotNil(t, spec)
	base := trainingOf(&spec.Character)
	dealt := domain.Deal(characters.StatPointsAtLevel(mob.Character.Level), w)
	for i := range base {
		base[i] += dealt[i]
	}
	return base
}

var warriorGrowth = domain.GrowthWeights{4, 2, 0, 3, 0, 1}

func TestCompanionsSpawnWithTheirArchetypesGrowth(t *testing.T) {
	b := growthBrawl(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = 9
	require.True(t, module.RetrainCompanion(tamsin.InstanceId))
	assert.Equal(t, wantTraining(t, tamsin, warriorGrowth), trainingOf(&tamsin.Character))
	assert.Zero(t, tamsin.Character.Stats.Smarts.Training-trainingOf(&mobs.GetMobSpec(tamsin.MobId).Character)[2], "a warrior puts nothing in smarts")
	assert.Zero(t, tamsin.Character.StatPoints)

	// Garrick has no archetype here: an even spread, as before.
	garrick := b.companion(3)
	assert.Equal(t, wantTraining(t, garrick, domain.EvenGrowth), trainingOf(&garrick.Character))
}

func TestCompanyGrowthCommandSetsAFocusAndRetrains(t *testing.T) {
	b := growthBrawl(t)
	view := b.cmd("company", "growth")
	assert.Contains(t, view, "Tamsin Reed, Warrior")
	assert.Contains(t, view, "weights: strength 4, speed 2, vitality 3, perception 1")
	assert.Contains(t, view, "focus balanced")

	assert.Contains(t, b.cmd("company", "growth tamsin reed vit"), "Tamsin Reed favours vitality now", "a full name with a space")
	record, _ := module.registry.Get(7)
	assert.Equal(t, "vitality", record.Companions[0].GrowthFocus)
	tamsin := b.companion(1)
	focused := warriorGrowth.WithFocus("vitality")
	assert.Equal(t, wantTraining(t, tamsin, focused), trainingOf(&tamsin.Character))
	assert.LessOrEqual(t, tamsin.Character.Health, tamsin.Character.HealthMax.Value)

	// The focus is saved: a respawn deals the same points the same way.
	before := trainingOf(&tamsin.Character)
	b.respawn()
	assert.Equal(t, before, trainingOf(&b.companion(1).Character))

	assert.Contains(t, b.cmd("company", "growth tamsin vitality"), "already grows that way")
	assert.Contains(t, b.cmd("company", "growth tamsin luck"), "Usage: company growth")
	assert.Contains(t, b.cmd("company", "growth nobody strength"), "no companion like that")
	assert.Contains(t, b.cmd("company", "growth tamsin balanced"), "grows by Warrior training alone")
	assert.Equal(t, wantTraining(t, b.companion(1), warriorGrowth), trainingOf(&b.companion(1).Character))
}

func TestCompanyGrowthIsRefusedInABattle(t *testing.T) {
	b := growthBrawl(t)
	b.start()
	assert.Contains(t, b.cmd("company", "growth tamsin strength"), "Not in the middle of a battle")
	record, _ := module.registry.Get(7)
	assert.Empty(t, record.Companions[0].GrowthFocus)
	assert.Contains(t, b.cmd("company", "growth"), "Company growth", "the view stays open")
}

// TestLiveLevelUpDealsTheSamePointsAsARespawn is the no-minting check: a
// companion levelled by the real combat award has exactly the training a
// respawn at that level gives, and losing a level and earning it back ends
// where it started.
func TestLiveLevelUpDealsTheSamePointsAsARespawn(t *testing.T) {
	b := growthBrawl(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	tamsin := b.companion(1)
	start := tamsin.Character.Level
	before := tamsin.Character

	paid, lines := mobcommands.AwardCompanyXP(7, b.aria.Character, tamsin.Character.XPTL(start+2), b.road.RoomId)
	require.GreaterOrEqual(t, paid, 1)
	require.NotEmpty(t, lines)
	require.Greater(t, tamsin.Character.Level, start)
	levelled := trainingOf(&tamsin.Character)
	assert.Equal(t, wantTraining(t, tamsin, warriorGrowth), levelled, "live level-up uses the warrior's weights")
	assert.Zero(t, tamsin.Character.StatPoints)
	var report string
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, tamsin.Character.Name+" reaches level") {
			report = line
			count++
		}
	}
	require.Equal(t, 1, count, "one line spans a multi-level gain")
	assert.Contains(t, report, fmt.Sprintf("Health %d -> %d", before.HealthMax.Value, tamsin.Character.HealthMax.Value))
	assert.Contains(t, report, fmt.Sprintf("Mana %d -> %d", before.ManaMax.Value, tamsin.Character.ManaMax.Value))
	// Phase 35a2: the report names what the levels made it better at.
	require.Greater(t, tamsin.Character.AttackSkill(), before.AttackSkill())
	assert.Contains(t, report, fmt.Sprintf("Attack %d -> %d", before.AttackSkill(), tamsin.Character.AttackSkill()))
	assert.Contains(t, report, fmt.Sprintf("Evasion %d -> %d", before.Evasion(), tamsin.Character.Evasion()))
	require.NotEqual(t, before.Stats.Strength.ValueAdj, tamsin.Character.Stats.Strength.ValueAdj)
	assert.Contains(t, report, fmt.Sprintf("Strength %d -> %d", before.Stats.Strength.ValueAdj, tamsin.Character.Stats.Strength.ValueAdj))

	b.respawn()
	assert.Equal(t, levelled, trainingOf(&b.companion(1).Character), "a respawn changes nothing")

	// Death's level loss: the record drops a level, the respawn deals fewer
	// points, and earning the level again deals them back, no more.
	// (The despawn snapshots the live mob, so the record is lowered after.)
	module.onPlayerDespawn(events.PlayerDespawn{UserId: 7})
	record, _ := module.registry.Get(7)
	level := record.Companions[0].State.Level
	record.Companions[0].State.Level = level - 1
	record.Companions[0].State.Experience = 0
	module.registry.Put(record)
	require.NoError(t, module.restoreForLeader(7, b.aria.Character.RoomId))
	lower := b.companion(1)
	require.Equal(t, level-1, lower.Character.Level)
	assert.Equal(t, wantTraining(t, lower, warriorGrowth), trainingOf(&lower.Character))
	mobcommands.AwardCompanyXP(7, b.aria.Character, lower.Character.XPTNL()-lower.Character.Experience, b.road.RoomId)
	require.Equal(t, level, lower.Character.Level)
	assert.Equal(t, levelled, trainingOf(&lower.Character), "regaining the level mints nothing")
}

// contractQuests writes a contract and a personal quest into the brawl's
// data and loads them.
func contractQuests(t *testing.T) {
	t.Helper()
	dir := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "quests")
	require.NoError(t, os.MkdirAll(dir, 0755))
	steps := "steps:\n  - id: start\n    description: Begin.\n  - id: end\n    description: Done.\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "9301-road_contract.yaml"), []byte("questid: 9301\nname: Road Contract\ndescription: Clear the road.\n"+steps+"rewards:\n  experience: 50\n  companyexperience: 300\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "9302-private_errand.yaml"), []byte("questid: 9302\nname: Private Errand\ndescription: An errand.\n"+steps+"rewards:\n  experience: 50\n"), 0600))
	quests.LoadDataFiles()
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(dir, "9301-road_contract.yaml"))
		_ = os.Remove(filepath.Join(dir, "9302-private_errand.yaml"))
		quests.LoadDataFiles()
	})
	require.Equal(t, 300, quests.GetQuest("9301-start").Rewards.CompanyExperience)
}

func completeQuest(b *brawl, id string) string {
	b.t.Helper()
	*b.messages = nil
	for _, step := range []string{"start", "end"} {
		hooks.HandleQuestUpdate(events.Quest{UserId: 7, QuestToken: id + "-" + step})
		events.ProcessEvents()
	}
	return companyTagPattern.ReplaceAllString(joinLines(*b.messages), "")
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

func TestAContractPaysTheCompanionsWithTheLeader(t *testing.T) {
	b := growthBrawl(t)
	contractQuests(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	// Ysolde waits on the verge, out of the leader's room.
	ysolde := b.companion(4)
	require.True(t, nativeRuntime{}.Relocate(ysolde.InstanceId, 920102))
	xp := map[int]int{}
	for id := 1; id <= 4; id++ {
		xp[id] = b.companion(id).Character.Experience
	}
	leaderBefore := b.aria.Character.Experience

	out := completeQuest(b, "9302")
	assert.NotContains(t, out, "shares the contract")
	for id := 1; id <= 4; id++ {
		assert.Equal(t, xp[id], b.companion(id).Character.Experience, "a personal quest pays no companion (#%d)", id)
	}
	assert.Equal(t, leaderBefore+50, b.aria.Character.Experience)

	out = completeQuest(b, "9301")
	assert.Contains(t, out, "Your company shares the contract: 300 experience (before scaling) for each companion with you.")
	for id := 1; id <= 3; id++ {
		assert.Equal(t, xp[id]+300, b.companion(id).Character.Experience, "#%d with the leader earns the contract in full", id)
	}
	assert.Equal(t, xp[4], b.companion(4).Character.Experience, "an absent companion earns nothing")
	assert.Equal(t, leaderBefore+100, b.aria.Character.Experience, "the leader's own experience is unchanged by the contract")

	// Turning it in again pays nothing: the quest is done.
	again := b.companion(1).Character.Experience
	assert.NotContains(t, completeQuest(b, "9301"), "shares the contract")
	assert.Equal(t, again, b.companion(1).Character.Experience)
}

// TestCompanyArchetypeRetrainsTheLiveCompanion: giving a companion its
// archetype deals its points by that archetype at once.
func TestCompanyArchetypeRetrainsTheLiveCompanion(t *testing.T) {
	b := growthBrawl(t)
	garrick := b.companion(3)
	garrick.Character.Level = 8
	require.True(t, module.RetrainCompanion(garrick.InstanceId))
	require.Equal(t, wantTraining(t, garrick, domain.EvenGrowth), trainingOf(&garrick.Character))
	assert.Contains(t, b.cmd("company", "archetype garrick warrior"), "is now a Warrior")
	assert.Equal(t, wantTraining(t, garrick, warriorGrowth), trainingOf(&garrick.Character))
}
