package company

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35a2 acceptance rows (skill over hit points). ASHVEIL_BALANCE=1
// runs them; ASHVEIL_BALANCE_FIGHTS sets the fights per cell (default 100).

func balanceFights() int {
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		return v
	}
	return 100
}

// TestBalanceSkillWins (acceptance 3): a level-20 company beats a level-10
// group of 3 almost always and cheaply; a level-10 company loses to a
// level-20 group of 2 most of the time.
func TestBalanceSkillWins(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the skill rows")
	}
	fights := balanceFights()
	cells := []struct {
		name         string
		level, enemy int
		count        int
	}{{"L20 vs three L10", 20, 10, 3}, {"L10 vs two L20", 10, 20, 2}}
	for _, cell := range cells {
		var results []balanceResult
		for i := 0; i < fights; i++ {
			t.Run(fmt.Sprintf("%s-%d", cell.name, i), func(t *testing.T) {
				f := newBalanceFightWithOptions(t, cell.level, companyDefault, enemyDefault,
					balanceFightOptions{EnemyCount: cell.count, EnemyLevels: []int{cell.enemy}, Coordination: 1, LegacyMirror: true})
				results = append(results, f.run())
			})
		}
		t.Log(balanceRow(cell.level, companyDefault, cell.name, results))
		_, winPct := balanceMedianAndWins(results)
		shares := make([]float64, 0, len(results))
		for _, r := range results {
			shares = append(shares, 100*float64(r.HPRemoved[sideCompany])/float64(max(1, r.StartHP[sideCompany])))
		}
		lost := balanceMean(shares)
		t.Logf("%s: won %d%%, company lost %.1f%% of its health on average", cell.name, winPct, lost)
		if cell.level == 20 {
			assert.GreaterOrEqual(t, winPct, 99, "%s wins", cell.name)
			assert.LessOrEqual(t, lost, 15.0, "%s loses little health", cell.name)
		} else {
			assert.LessOrEqual(t, winPct, 30, "%s loses at least 70%% of fights", cell.name)
		}
	}
}

// tankKits are each class's best trained shipped armor (and shield).
var tankKits = map[string]map[items.ItemType]int{
	"warrior": {items.Body: 20038, items.Head: 20023, items.Neck: 20017, items.Legs: 20029, items.Feet: 20028, items.Offhand: 20048},
	"rogue":   {items.Body: 20022, items.Head: 20020, items.Neck: 20045, items.Legs: 20029, items.Feet: 20028},
	"wizard":  {items.Body: 20022, items.Head: 20020, items.Neck: 20045, items.Legs: 20029, items.Feet: 20028},
}

// TestBalanceWarriorsAreTheTanks (acceptance 5): at level 30 in their best
// trained armor, a warrior with a shield takes at least 30% less damage per
// enemy swing than a rogue and at most half of a wizard's; a rogue in heavy
// armor gets at least 20% fewer turns than one in light.
func TestBalanceWarriorsAreTheTanks(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the tank row")
	}
	swings := balanceFights() * 40
	b := newBrawl(t)
	mob := mobs.GetInstance(b.bandits["bandit cutthroat"][0])
	require.NotNil(t, mob)
	even := func(c *characters.Character) {
		for _, s := range []*stats.StatInfo{&c.Stats.Strength, &c.Stats.Speed, &c.Stats.Smarts, &c.Stats.Perception} {
			s.ValueAdj = 10
		}
	}
	perSwing := map[string]float64{}
	for class, kit := range tankKits {
		p := balanceHPProvider(t)
		p.fakeArchetypes.player = class
		archetypes.SetProvider(p)
		a := b.aria.Character
		a.Equipment = characters.Worn{}
		for slot, id := range kit {
			a.Equipment.Set(slot, items.New(id))
		}
		a.Level, mob.Character.Level = 30, 30
		mob.Character.AttackOffset, mob.Character.EvasionOffset = 0, 0
		a.RecalculateStats()
		even(a)
		even(&mob.Character)
		require.False(t, a.UntrainedArmor(), class)
		total, landed, defended, crits := 0, 0, 0, 0
		for i := 0; i < swings; i++ {
			a.Health, a.HealthMax.Value = 10000, 10000
			r := combat.AttackMobVsPlayer(mob, b.aria)
			total += r.DamageToTarget
			if r.DamageToTarget > 0 {
				landed++
			}
			if r.Crit {
				crits++
			}
			defended += len(r.Defenses)
		}
		t.Logf("DEBUG %s: landed %d defended %d crits %d shield %v", class, landed, defended, crits, a.HasShield())
		perSwing[class] = float64(total) / float64(swings)
		t.Logf("%s: defense %d, Evasion %d, bulk %s: %.2f damage per swing", class, a.GetDefense(), a.Evasion(), a.ArmorBulk(), perSwing[class])
	}
	archetypes.SetProvider(nil)
	assert.LessOrEqual(t, perSwing["warrior"], 0.7*perSwing["rogue"], "a warrior takes at least 30%% less than a rogue")
	assert.LessOrEqual(t, perSwing["warrior"], 0.5*perSwing["wizard"], "and at most half a wizard's")

	p := balanceHPProvider(t)
	p.fakeArchetypes.player = "rogue"
	archetypes.SetProvider(p)
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	rogue := b.aria.Character
	rogue.Equipment = characters.Worn{}
	rogue.Equipment.Body = items.New(20022)
	light := combat.Tempo(rogue)
	rogue.Equipment.Body = items.New(20038)
	heavy := combat.Tempo(rogue)
	t.Logf("rogue tempo: light %.2f, heavy %.2f", light, heavy)
	assert.LessOrEqual(t, heavy, 0.8*light, "a heavy-armored rogue gets at least 20%% fewer turns")
}
