package bonds

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

func TestApplyStopsTimeTogetherAtFiftyAndWaryAndRescueAtAHundred(t *testing.T) {
	for _, tc := range []struct {
		name         string
		value, delta int
		src          Source
		want         int
	}{
		{"camp rises", 10, 2, Camp, 12},
		{"camp stops at close", 49, 2, Camp, 50},
		{"a clash at camp can still lower a high bond", 70, -1, Camp, 69}, // a fall is a fall, only the limit is -25
		{"camp does not lift one already past it", 70, 2, Camp, 70},
		{"camp falls to wary", -23, -2, Camp, -25},
		{"camp stops at wary", -24, -2, Camp, -25},
		{"camp never makes rivals", -25, -2, Camp, -25},
		{"battle stops at close", 50, 1, Battle, 50},
		{"talk stops at the limits", -50, -1, Talk, -50},
		{"rescue goes beyond close", 50, 3, Rescue, 53},
		{"rescue caps at the top", 99, 3, Rescue, 100},
		{"refusal goes below the camp floor", -50, -2, Refusal, -52},
		{"refusal floors at the bottom", -99, -2, Refusal, -100},
		{"an opinion clash is a real clash: it goes past wary", -25, -1, Opinion, -26},
		{"an opinion clash goes on for rivals", -50, -1, Opinion, -51},
		{"an opinion agreement stops at 50", 50, 1, Opinion, 50},
		{"a rival's rescue mends it", -60, 3, Rescue, -57},
		{"camp warms a rival that fell past its floor", -80, 2, Camp, -78},
	} {
		if got := Apply(tc.value, tc.delta, tc.src); got != tc.want {
			t.Errorf("%s: Apply(%d, %d, %s) = %d, want %d", tc.name, tc.value, tc.delta, tc.src, got, tc.want)
		}
	}
}

func TestTiersAndWords(t *testing.T) {
	for _, tc := range []struct {
		v    int
		tier int
		feel string
		pair string
	}{
		{100, TierKin, "is like kin to", "are like kin"},
		{75, TierKin, "is like kin to", "are like kin"},
		{74, TierClose, "is close to", "are close"},
		{50, TierClose, "is close to", "are close"},
		{25, TierFriend, "trusts", "trust each other"},
		{24, TierStrangers, "is still getting to know", "are still getting to know each other"},
		{0, TierStrangers, "is still getting to know", "are still getting to know each other"},
		{-24, TierStrangers, "is still getting to know", "are still getting to know each other"},
		{-25, TierWary, "is wary of", "are wary of each other"},
		{-50, TierRival, "can't stand", "can't stand each other"},
		{-84, TierRival, "can't stand", "can't stand each other"},
		{-85, TierBitter, "cannot bear", "cannot bear each other"},
		{-100, TierBitter, "cannot bear", "cannot bear each other"},
	} {
		if got := TierOf(tc.v); got != tc.tier {
			t.Errorf("TierOf(%d) = %d, want %d", tc.v, got, tc.tier)
		}
		if got := Feeling(tc.v); got != tc.feel {
			t.Errorf("Feeling(%d) = %q, want %q", tc.v, got, tc.feel)
		}
		if got := Together(tc.v); got != tc.pair {
			t.Errorf("Together(%d) = %q, want %q", tc.v, got, tc.pair)
		}
	}
	if !IsFriend(25) || IsFriend(24) || !IsRival(-50) || IsRival(-49) {
		t.Error("the friend and rival lines are 25 and -50")
	}
}

func TestGuardsByBond(t *testing.T) {
	for v, want := range map[int]int{-100: 0, 0: 0, 24: 0, 25: 1, 74: 1, 75: 2, 100: 2} {
		if got := Guards(v); got != want {
			t.Errorf("Guards(%d) = %d, want %d", v, got, want)
		}
	}
}

func TestAffinity(t *testing.T) {
	if got := Affinity("stoic", "devout", 0, 0); got != 2 {
		t.Errorf("suited = %d", got)
	}
	if got := Affinity("devout", "stoic", 0, 0); got != 2 {
		t.Errorf("the order does not matter: %d", got)
	}
	if got := Affinity("grim", "grim", 10, -10); got != 1 {
		t.Errorf("the same temperament = %d", got)
	}
	if got := Affinity("cheerful", "grim", 0, 0); got != -1 {
		t.Errorf("clashing = %d", got)
	}
	if got := Affinity("Cheerful", "GRIM", 0, 0); got != -1 {
		t.Errorf("case does not matter: %d", got)
	}
	if got := Affinity("stoic", "devout", 80, -10); got != 1 {
		t.Errorf("a saint and a villain rub even when suited: %d", got)
	}
	if got := Affinity("cheerful", "grim", 70, 0); got != -2 {
		t.Errorf("a clash across the alignment gap = %d", got)
	}
	if got := Affinity("stoic", "devout", 40, -29); got != 2 {
		t.Errorf("just inside the gap = %d", got)
	}
	if BattleGain(2) != 1 || BattleGain(1) != 1 || BattleGain(0) != 0 || BattleGain(-2) != 0 {
		t.Error("a battle is a point unless the pair rubs")
	}
	// Every personality pair has an affinity: no pair is undefined.
	for _, a := range banter.Personalities {
		for _, b := range banter.Personalities {
			if n := Affinity(a, b, 0, 0); n < -1 || n > 2 {
				t.Errorf("Affinity(%s, %s) = %d", a, b, n)
			}
		}
	}
}

func TestEveryPersonalityHasAFriendAndMostPairsAreNotClashes(t *testing.T) {
	clashing := 0
	for _, a := range banter.Personalities {
		suited := 0
		for _, b := range banter.Personalities {
			switch Affinity(a, b, 0, 0) {
			case 2:
				suited++
			case -1:
				clashing++
			}
		}
		if suited == 0 {
			t.Errorf("%s suits no one", a)
		}
	}
	if clashing > 10 {
		t.Errorf("%d clashing ordered pairs: rivalries should be the exception", clashing)
	}
}

func TestOpinionDelta(t *testing.T) {
	if OpinionDelta(opinions.Likes, opinions.Likes) != 1 || OpinionDelta(opinions.Dislikes, opinions.Dislikes) != 1 {
		t.Error("agreeing is +1")
	}
	if OpinionDelta(opinions.Likes, opinions.Dislikes) != -1 || OpinionDelta(opinions.Dislikes, opinions.Likes) != -1 {
		t.Error("splitting is -1")
	}
	if OpinionDelta(opinions.Likes, opinions.Neutral) != 0 {
		t.Error("one with no view moves nothing")
	}
}

func TestWarningComesBeforeLeaving(t *testing.T) {
	if w, l, m := Warning(-84, false); w || l || m {
		t.Error("-84 is not yet a warning")
	}
	if w, l, _ := Warning(-85, false); !w || l {
		t.Error("-85 warns")
	}
	if w, l, _ := Warning(-100, false); !w || l {
		t.Error("an unwarned pair at the bottom is warned first, never sent away unseen")
	}
	if w, l, _ := Warning(-100, true); w || !l {
		t.Error("a warned pair at the bottom leaves")
	}
	if w, l, m := Warning(-90, true); w || l || m {
		t.Error("a warned pair between does nothing")
	}
	if _, _, m := Warning(-60, true); !m {
		t.Error("a warned pair that recovered to -60 is forgiven")
	}
	if _, _, m := Warning(-60, false); m {
		t.Error("nothing to mend without a warning")
	}
}

func TestSourcesAreKnownAndHaveCooldowns(t *testing.T) {
	for _, s := range Sources {
		if !Known(s) || CooldownOf(s) <= 0 {
			t.Errorf("source %q has no cooldown", s)
		}
	}
	if Known("nonsense") || CooldownOf("nonsense") != 0 {
		t.Error("an unknown source is not known")
	}
}
