package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillBrawl is an equipment brawl with the shipped 35a2 class profiles:
// Aria plays the given class, and her companions are Tamsin (warrior),
// Oswin (cleric), Garrick (warrior) and Ysolde (ranger).
func skillBrawl(t *testing.T, player string) *brawl {
	t.Helper()
	b := equipmentBrawl(t)
	p := balanceHPProvider(t)
	p.fakeArchetypes.player = player
	archetypes.SetProvider(p)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	for id, class := range map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"} {
		if err := module.registry.SetCompanionArchetype(7, id, class); err != nil {
			require.ErrorIs(t, err, domain.ErrArchetypeAlreadySet)
		}
		b.companion(id).Character.HPArchetype = class
	}
	return b
}

// TestClassGearThroughEquip: equip and company equip refuse a shield or
// weapon a class isn't taught, with the reason, and change nothing; what
// a class may use goes on; untrained armor goes on with a warning.
func TestClassGearThroughEquip(t *testing.T) {
	b := skillBrawl(t, "cleric")
	shield, cudgel, mace := items.New(20004), items.New(10015), items.New(10023)
	buckler, tower, plate := items.New(20047), items.New(20048), items.New(20012)
	b.aria.Character.Items = []items.Item{shield, cudgel, mace, buckler, tower, plate}
	b.aria.Character.Equipment.Offhand = items.Item{}

	assert.Contains(t, b.cmd("equip", shield.ShorthandId()), "Clerics don't carry shields.")
	assert.Zero(t, b.aria.Character.Equipment.Offhand.ItemId, "the refused shield stays in cargo")
	assert.Contains(t, b.cmd("equip", cudgel.ShorthandId()), "Clerics fight with staffs, rods and maces.")
	require.Contains(t, b.cmd("equip", mace.ShorthandId()), "equipment updated")
	assert.Equal(t, mace.UUID, b.aria.Character.Equipment.Weapon.UUID)

	assert.Contains(t, b.cmd("company", "equip #4 "+tower.ShorthandId()), "Rangers carry only bucklers.")
	require.Contains(t, b.cmd("company", "equip #4 "+buckler.ShorthandId()), "equipment updated")
	assert.Equal(t, buckler.UUID, b.companion(4).Character.Equipment.Offhand.UUID)
	require.Contains(t, b.cmd("company", "equip #1 "+tower.ShorthandId()), "equipment updated")
	assert.Equal(t, tower.UUID, b.companion(1).Character.Equipment.Offhand.UUID)

	got := b.cmd("equip", plate.ShorthandId())
	require.Contains(t, got, "equipment updated")
	assert.Contains(t, got, "trained for light armor, not heavy", "untrained armor warns")
	assert.True(t, b.aria.Character.UntrainedArmor())
	level := b.aria.Character.Level
	attack, _ := b.aria.Character.BaseSkills(level)
	assert.Equal(t, attack-10, b.aria.Character.AttackSkill(), "untrained armor costs 10 Attack")
}

// TestCompanionSpawnPutsAwayDisallowedGear: a companion saved holding
// what its class may not use spawns with it in its pack, and the leader is
// told once; a second spawn from the saved result says nothing.
func TestCompanionSpawnPutsAwayDisallowedGear(t *testing.T) {
	b := skillBrawl(t, "")
	oswin := b.companion(2)
	state := domain.MemberState{Level: oswin.Character.Level}
	state.Equipment.Weapon = items.New(10015)
	state.Equipment.Offhand = items.New(20004)
	*b.messages = nil
	id, err := nativeRuntime{}.Spawn(7, b.aria.Character.RoomId, int(oswin.MobId), &state, domain.Identity{Archetype: "cleric", Name: "Brother Oswin"}, domain.GrowthWeights{})
	require.NoError(t, err)
	events.ProcessEvents()
	spawned := mobs.GetInstance(id)
	require.NotNil(t, spawned)
	assert.Zero(t, spawned.Character.Equipment.Weapon.ItemId)
	assert.Zero(t, spawned.Character.Equipment.Offhand.ItemId)
	carried := map[int]bool{}
	for _, itm := range spawned.Character.Items {
		carried[itm.ItemId] = true
	}
	assert.True(t, carried[10015] && carried[20004], "moved to its pack")
	// The spawn gives it its class's head start in health (the cleric's 2).
	c := &spawned.Character
	require.Equal(t, 2, c.HPStart())
	base := configs.GetProgressionConfig().HealthAtLevel(c.Level, c.Stats.Vitality.ValueAdj, c.HealthGainPerLevel(), 0)
	assert.Equal(t, base+2+c.StatMod("healthmax"), c.HealthMax.Value, "HPStart counts")
	told := companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
	assert.Contains(t, told, "Brother Oswin puts away")
	assert.Contains(t, told, "Clerics")

	again := domain.MemberState{Level: spawned.Character.Level, Equipment: spawned.Character.Equipment, Items: spawned.Character.Items}
	*b.messages = nil
	_, err = nativeRuntime{}.Spawn(7, b.aria.Character.RoomId, int(oswin.MobId), &again, domain.Identity{Archetype: "cleric", Name: "Brother Oswin"}, domain.GrowthWeights{})
	require.NoError(t, err)
	events.ProcessEvents()
	assert.NotContains(t, strings.Join(*b.messages, "\n"), "puts away", "told once")
}

// TestSkillEdgeDecidesRealBlows: through AttackPlayerVsMob and
// AttackMobVsPlayer, with every stat even, ten levels of skill make a
// veteran land far more of its blows than a novice does.
func TestSkillEdgeDecidesRealBlows(t *testing.T) {
	b := newBrawl(t)
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.SkillEdgeSpan = 20
	cfg.Combat.DefaultAttackRate, cfg.Combat.DefaultEvasionRate = 1, 1
	cfg.Combat.ToHitMin, cfg.Combat.ToHitEven, cfg.Combat.ToHitMax = 10, 60, 95
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceEven, cfg.Combat.DodgeChanceMax = 3, 12, 40
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceEven, cfg.Combat.ParryChanceMax = 3, 12, 40
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceEven, cfg.Combat.BlockChanceMax = 8, 15, 55
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	mob := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
	require.NotNil(t, mob)
	even := func(c *characters.Character) {
		for _, s := range []*stats.StatInfo{&c.Stats.Strength, &c.Stats.Speed, &c.Stats.Smarts, &c.Stats.Perception} {
			s.ValueAdj = 10
		}
	}
	landed := func(ariaLevel, mobLevel int) (aria, foe int) {
		b.aria.Character.Level, mob.Character.Level = ariaLevel, mobLevel
		even(b.aria.Character)
		even(&mob.Character)
		mob.Character.Equipment.Offhand = items.Item{}
		b.aria.Character.Equipment.Offhand = items.Item{}
		for i := 0; i < 400; i++ {
			mob.Character.Health, mob.Character.HealthMax.Value = 10000, 10000
			b.aria.Character.Health, b.aria.Character.HealthMax.Value = 10000, 10000
			if combat.AttackPlayerVsMob(b.aria, mob).DamageToTarget > 0 {
				aria++
			}
			if combat.AttackMobVsPlayer(mob, b.aria).DamageToTarget > 0 {
				foe++
			}
		}
		return aria, foe
	}
	veteran, novice := landed(30, 20)
	assert.Greater(t, veteran, novice*3/2, "level 30 lands far more on level 20 (%d) than it takes (%d)", veteran, novice)
	novice2, veteran2 := landed(20, 30)
	assert.Greater(t, veteran2, novice2*3/2, "and the other way round (%d against %d)", veteran2, novice2)
}

// TestSkillShownToThePlayer: consider says in words how a group's skill
// compares (status's rows are in usercommands' TestStatusShowsSkillAndBulk).
func TestSkillShownToThePlayer(t *testing.T) {
	b := skillBrawl(t, "warrior")
	g, kw := b.banditGroup()
	for _, m := range g.Visible() {
		m.Character.Level = b.aria.Character.Level
		m.Character.AttackOffset, m.Character.EvasionOffset = 0, 0
	}
	assert.Contains(t, b.cmd("consider", kw), "They are evenly matched with you.")
	for _, m := range g.Visible() {
		m.Character.Level = b.aria.Character.Level + 15
	}
	assert.Contains(t, b.cmd("consider", kw), "They are far more skilled than you.")
	for _, m := range g.Visible() {
		m.Character.Level = max(1, b.aria.Character.Level-15)
	}
	b.aria.Character.Level = max(b.aria.Character.Level, 16)
	assert.Contains(t, b.cmd("consider", kw), "They are novices next to you.")

}

func TestSkillGapWords(t *testing.T) {
	for gap, want := range map[int]string{15: "far more skilled than you", 10: "far more skilled than you", 9: "more skilled than you",
		4: "more skilled than you", 3: "evenly matched with you", 0: "evenly matched with you", -3: "evenly matched with you",
		-4: "less skilled than you", -9: "less skilled than you", -10: "novices next to you", -30: "novices next to you"} {
		assert.Equal(t, want, usercommands.SkillGapWords(gap), "%d", gap)
	}
}
