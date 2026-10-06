package company

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/require"
)

type narrationOutcome struct {
	Events                 []narrationMechanic
	Mana, Experience, Gold int
}
type narrationMechanic struct {
	Round                                    uint64
	Kind                                     combatstream.Kind
	Source, Target, Previous, Outcome, Spell string
	Damage, Amount                           int
	Crit                                     bool
}

// The golden was captured by running this same real-round fixture against
// b9729809 (before Phase 29d). Names, sequence/fight IDs, and fresh runtime
// instance IDs are excluded; attacks, casts, rewards and endings remain.
func TestNarrationPreservesCombatOutcome(t *testing.T) {
	t.Setenv("GODEBUG", "randseednop=0")
	b := newBrawl(t)
	// Recaptured for Phase 30g4: stepped stats change hit and damage rolls.
	// 30g6a updated one target-change event (the next foe is selected on
	// the kill round). Recaptured for 30g6: stat edges, Strength damage and
	// the HP shape change every roll and the fight's length; the same kinds
	// of events, deaths and rewards remain. Recaptured for 35b: the hit,
	// dodge, parry and block chances, Minor Heal's one-round chant and
	// spell power change the rolls and the fight's length. This remains an
	// outcome lock for subsequent narration-only changes.
	// The golden predates Phase 30d1, and its fixture keeps a cutthroat
	// chanting for ever as a placeholder: blows breaking chants would
	// change what it records.
	t.Cleanup(hooks.DisableInterruptsForTest())
	b.cmd("company", "dismiss all")
	for name, ids := range b.bandits {
		if name == "bandit cutthroat" {
			continue
		}
		for _, id := range ids {
			b.road.RemoveMob(id)
			mobs.DestroyInstance(id)
		}
		delete(b.bandits, name)
	}
	// One attacking mob and one chanting mob avoid the engine's existing
	// unsorted mob-map turn order changing which actor consumes each RNG draw.
	second := mobs.GetInstance(b.bandits["bandit cutthroat"][1])
	second.Character.SetCast(1000000, characters.SpellAggroInfo{SpellId: "mm", TargetUserIds: []int{7}})

	roles := map[int]string{}
	for id, instance := range module.instances[7] {
		roles[instance] = fmt.Sprintf("companion-%d", id)
	}
	for name, ids := range b.bandits {
		for i, id := range ids {
			roles[id] = fmt.Sprintf("%s-%d", name, i+1)
		}
	}
	role := func(ref combatstream.Ref) string {
		if ref.UserId > 0 {
			return fmt.Sprintf("user-%d", ref.UserId)
		}
		return roles[ref.MobInstanceId]
	}
	seen := b.listen()
	rand.Seed(29)
	b.aimAt("bandit cutthroat")
	b.aria.Character.ManaMax.Value = 100
	b.aria.Character.Mana = 97
	b.aria.Character.SpellBook["heal"] = 5000
	b.aria.Character.SetCast(1, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{7}})
	mobs.GetInstance(b.bandits["bandit cutthroat"][0]).Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	b.fightItOut(200)
	result := narrationOutcome{Mana: b.aria.Character.Mana, Experience: b.aria.Character.Experience, Gold: b.aria.Character.Gold}
	for _, e := range *seen {
		result.Events = append(result.Events, narrationMechanic{e.Round, e.Kind, role(e.Source), role(e.Target), role(e.Previous), e.Outcome, e.SpellId, e.Damage, e.Amount, e.Crit})
	}
	sort.SliceStable(result.Events, func(i, j int) bool {
		a, b := result.Events[i], result.Events[j]
		if a.Round != b.Round {
			return a.Round < b.Round
		}
		return fmt.Sprint(a.Kind, a.Source, a.Target) < fmt.Sprint(b.Kind, b.Source, b.Target)
	})
	if path := os.Getenv("PHASE29D_CAPTURE_OUTCOME"); path != "" {
		data, err := json.MarshalIndent(result, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, append(data, '\n'), 0600))
		return
	}
	data, err := os.ReadFile(filepath.Join("modules", "company", "testdata", "phase29d-outcome-before.json"))
	require.NoError(t, err)
	var before narrationOutcome
	require.NoError(t, json.Unmarshal(data, &before))
	require.Equal(t, before, result, "Narration-only changes must preserve the captured mechanics")
}
