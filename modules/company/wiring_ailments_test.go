package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 55 wiring: an ailment sets the battle condition through the real
// beginBattle and counts a battle off, and a lasting puncture left open
// through three battles festers into a Fever that counts in that battle.

// ailingNeeds is fareNeeds whose ailment service edits the fixed needs, so a
// catch made as the battle begins shows in the condition it reads next.
type ailingNeeds struct {
	*fareNeeds
	caught []survival.MemberKey
}

func (f *ailingNeeds) CatchAilment(_ int, key survival.MemberKey, kind string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.needs {
		if f.needs[i].Key != key {
			continue
		}
		r := survival.NewRegistry()
		_ = r.Ensure(7, key)
		_ = r.PutNeeds(7, key, f.needs[i].Needs)
		caught, err := r.CatchAilment(7, key, kind)
		f.needs[i].Needs = r.MustNeedsFor(7, key)
		if caught {
			f.caught = append(f.caught, key)
		}
		return caught, err
	}
	return false, survival.ErrUnknownMember
}

func (f *ailingNeeds) CureAilment(int, survival.MemberKey, string) (bool, error) { return false, nil }

func useAilingNeeds(t *testing.T, needs map[survival.MemberKey]survival.Needs) *ailingNeeds {
	t.Helper()
	f := &ailingNeeds{fareNeeds: useFareNeeds(t, needs)}
	survival.SetAilmentService(f)
	t.Cleanup(func() { survival.SetAilmentService(nil) })
	return f
}

func TestAnAilingMemberGoesInWorseAndCountsABattleOff(t *testing.T) {
	b := sigilBrawl(t)
	chilled := survival.FullNeeds()
	chilled.Chill = 2
	sick := survival.FullNeeds()
	sick.GutAche = 3
	f := useAilingNeeds(t, map[survival.MemberKey]survival.Needs{
		survival.LeaderMemberKey:       chilled,
		survival.CompanionMemberKey(1): sick,
		survival.CompanionMemberKey(2): survival.FullNeeds(),
	})
	out := b.beginFight()

	assert.Equal(t, -10, b.aria.Character.RTState().FareDamage, "a chill cuts the damage dealt")
	assert.Equal(t, -10, b.companion(1).Character.RTState().FareGuard, "a gut-ache lowers the guard")
	two := b.companion(2).Character.RTState()
	assert.Zero(t, two.FareDamage+two.FareGuard, "a well member fights as it always has")
	assert.Contains(t, out, "Chill: -10% damage (2 battles)")
	assert.Contains(t, out, "Gut-ache: +10% damage taken (3 battles)")
	assert.Contains(t, battle.FareOf(7)["leader"], "Chill")
	require.Len(t, f.spent, 1)
	assert.ElementsMatch(t, []survival.MemberKey{survival.LeaderMemberKey, survival.CompanionMemberKey(1)}, f.spent[0], "only the ailing have a battle counted off")
}

func TestAPunctureLeftOpenFestersIntoAFever(t *testing.T) {
	b := sigilBrawl(t)
	f := useAilingNeeds(t, map[survival.MemberKey]survival.Needs{
		survival.LeaderMemberKey:       survival.FullNeeds(),
		survival.CompanionMemberKey(1): survival.FullNeeds(),
		survival.CompanionMemberKey(2): survival.FullNeeds(),
	})
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Puncture, Place: "arm", Points: 3, Battles: survival.FeverBattles - 1}}
	// A cut, however old, never festers; a young puncture is only counted.
	b.companion(1).Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 3, Battles: 9}}
	b.companion(2).Character.Wounds = []wounds.Wound{{Kind: wounds.Puncture, Place: "leg", Points: 3}}
	out := b.beginFight()

	assert.Equal(t, []survival.MemberKey{survival.LeaderMemberKey}, f.caught)
	assert.Contains(t, out, "has caught a fever")
	assert.Equal(t, -15, b.aria.Character.RTState().FareDamage, "the fever counts in the battle it starts in")
	assert.Zero(t, b.aria.Character.Wounds[0].Battles, "the count starts over")
	assert.Equal(t, 1, b.companion(2).Character.Wounds[0].Battles, "a young puncture is counted")
	assert.Zero(t, b.companion(1).Character.RTState().FareDamage)
}

func TestAFeverishMemberDoesNotCountItsWoundAgain(t *testing.T) {
	b := sigilBrawl(t)
	febrile := survival.FullNeeds()
	febrile.Fever = 4
	f := useAilingNeeds(t, map[survival.MemberKey]survival.Needs{survival.LeaderMemberKey: febrile})
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Puncture, Place: "arm", Points: 3, Battles: 2}}
	b.beginFight()
	assert.Empty(t, f.caught)
	assert.Equal(t, 2, b.aria.Character.Wounds[0].Battles)
}
