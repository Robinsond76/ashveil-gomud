package opinions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryPersonalityReactsToAtLeastTwoKindsAndLikesAndDislikes(t *testing.T) {
	for _, p := range banter.Personalities {
		likes, dislikes := Leanings(p)
		assert.GreaterOrEqual(t, len(likes), 2, "%s likes", p)
		assert.GreaterOrEqual(t, len(dislikes), 2, "%s dislikes", p)
	}
}

func TestEveryKindHasSomeoneWhoLikesItAndEveryVerdictHasALine(t *testing.T) {
	for _, info := range Kinds {
		liked := false
		for _, p := range banter.Personalities {
			switch Verdict(p, info.Kind) {
			case Likes:
				liked = true
			case Neutral:
				assert.Empty(t, LineFor(&banter.Narrator{}, "Maren", p, info.Kind), "%s has no view on %s", p, info.Kind)
				continue
			}
			line := LineFor(&banter.Narrator{}, "Maren", p, info.Kind)
			require.NotEmpty(t, line, "%s on %s needs a line", p, info.Kind)
			assert.Contains(t, line, "Maren", "the speaker is named")
			assert.Contains(t, line, `"`, "the words are in quotes")
		}
		assert.True(t, liked, "someone should like %s", info.Kind)
		assert.Greater(t, info.Cooldown.Seconds(), 0.0)
	}
}

func TestLeaningsAreOnlyForRealPersonalitiesAndKinds(t *testing.T) {
	for p, kinds := range leanings {
		assert.True(t, banter.ValidPersonality(p), p)
		for k := range kinds {
			_, ok := InfoOf(k)
			assert.True(t, ok, "%s names unknown kind %s", p, k)
		}
	}
	for p, texts := range lines {
		assert.True(t, banter.ValidPersonality(p), p)
		for k := range texts {
			assert.NotEqual(t, Neutral, Verdict(p, k), "%s has a line for %s but no verdict", p, k)
		}
	}
}

func TestStancesAreTheStoryKinds(t *testing.T) {
	assert.True(t, IsStance("courage"))
	assert.False(t, IsStance("spare"), "mercy is answered by the mercy prompt, not tagged by authors")
	assert.False(t, IsStance("nonsense"))
	assert.Len(t, Stances(), 6)
}

func TestTheLoyaltyBoundsLeaveRoomForNerve(t *testing.T) {
	assert.Less(t, Floor, Ceiling)
	assert.GreaterOrEqual(t, Floor, 25, "battle nerve starts to waver under 25")
	assert.LessOrEqual(t, Nudge, 3)
}

func TestObserversAreToldOfEveryBatchButNotOfNothing(t *testing.T) {
	var got [][]Reaction
	Observe(func(_ int, c Choice, rs []Reaction) {
		if c.Op == "observer-test" {
			got = append(got, rs)
		}
	})
	Announce(1, Choice{Op: "observer-test"}, nil)
	assert.Empty(t, got)
	Announce(1, Choice{Op: "observer-test"}, []Reaction{{CompanionID: 3}})
	require.Len(t, got, 1)
	assert.Equal(t, 3, got[0][0].CompanionID)
}
