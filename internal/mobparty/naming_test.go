package mobparty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlural(t *testing.T) {
	cases := map[string]string{
		"ruffian":          "ruffians",
		"bandit cutthroat": "bandit cutthroats",
		"rat":              "rats",
		"fly":              "flies",
		"monkey":           "monkeys",
		"lynx":             "lynxes",
		"witch":            "witches",
		"dire wolf":        "dire wolves",
		"thief":            "thieves",
		"guardsman":        "guardsmen",
		"swordswoman":      "swordswomen",
		"field mouse":      "field mice",
		"sheep":            "sheep",
		"boss":             "bosses",
		"":                 "",
	}
	for in, want := range cases {
		assert.Equal(t, want, Plural(in), in)
	}
}

func TestNameGroupGenerated(t *testing.T) {
	n := NameGroup([]MobSummary{
		{InstanceId: 1, Name: "ruffian", EHP: 10},
		{InstanceId: 2, Name: "rat", Noun: "swarm", EHP: 3},
		{InstanceId: 3, Name: "ruffian", EHP: 10},
	})
	assert.Equal(t, "a band of ruffians", n.Name)
	assert.Equal(t, "band", n.Noun)
	assert.Equal(t, "ruffians", n.Kind)
	assert.Equal(t, "ruffians", n.Keyword)
	assert.False(t, n.Authored)
}

func TestNameGroupTieGoesToTheToughest(t *testing.T) {
	n := NameGroup([]MobSummary{
		{InstanceId: 1, Name: "rat", Noun: "swarm", EHP: 3},
		{InstanceId: 2, Name: "wolf", Noun: "pack", EHP: 20},
	})
	assert.Equal(t, "a pack of wolves", n.Name)
}

func TestNameGroupNounAndArticle(t *testing.T) {
	n := NameGroup([]MobSummary{{InstanceId: 1, Name: "skeleton", Noun: "host"}, {InstanceId: 2, Name: "skeleton", Noun: "host"}})
	assert.Equal(t, "a host of skeletons", n.Name)
	n = NameGroup([]MobSummary{{InstanceId: 1, Name: "orc", Noun: "army"}, {InstanceId: 2, Name: "orc"}})
	assert.Equal(t, "an army of orcs", n.Name)
}

func TestNameGroupKeepsItsNameAsMembersFall(t *testing.T) {
	// The ruffians have fallen; only the rat is left, still carrying the name.
	n := NameGroup([]MobSummary{{InstanceId: 2, Name: "rat", Noun: "swarm", GroupName: "a band of ruffians"}})
	assert.Equal(t, "a band of ruffians", n.Name)
	assert.Equal(t, "ruffians", n.Keyword)
	assert.False(t, n.Solo)
	assert.True(t, n.Matches("ruffians"))
	assert.True(t, n.Matches("band"))
}

func TestNameGroupAuthored(t *testing.T) {
	n := NameGroup([]MobSummary{
		{InstanceId: 1, Name: "rat", GroupName: "the Rat King's court", GroupDesc: "Rats in paper crowns."},
		{InstanceId: 2, Name: "rat king", GroupName: "the Rat King's court"},
	})
	assert.True(t, n.Authored)
	assert.Equal(t, "the Rat King's court", n.Name)
	assert.Equal(t, "court", n.Keyword)
	assert.Equal(t, "Rats in paper crowns.", n.Desc)
	for _, s := range []string{"court", "king", "rat king's court", "the rat king's court"} {
		assert.True(t, n.Matches(s), s)
	}
}

func TestNameGroupLoneMob(t *testing.T) {
	n := NameGroup([]MobSummary{{InstanceId: 1, Name: "troll"}})
	assert.True(t, n.Solo)
	assert.Equal(t, "troll", n.Name)
	assert.True(t, n.Matches("troll"))
}

func TestMatchesNotAMember(t *testing.T) {
	n := NameGroup([]MobSummary{{InstanceId: 1, Name: "ruffian"}, {InstanceId: 2, Name: "cutpurse"}, {InstanceId: 3, Name: "ruffian"}})
	for _, s := range []string{"band", "ruffians", "band of ruffians", "a band of ruffians", "Ruffians"} {
		assert.True(t, n.Matches(s), s)
	}
	for _, s := range []string{"ruffian", "cutpurse", "rat"} {
		assert.False(t, n.Matches(s), s)
	}
	assert.True(t, n.MatchesPrefix("ruff"))
	assert.False(t, n.MatchesPrefix("r"))
}

func TestWithOrdinal(t *testing.T) {
	assert.Equal(t, "a band of ruffians", WithOrdinal("a band of ruffians", 1))
	assert.Equal(t, "a second band of ruffians", WithOrdinal("a band of ruffians", 2))
	assert.Equal(t, "a third army of orcs", WithOrdinal("an army of orcs", 3))
	assert.Equal(t, "the straw squad (second)", WithOrdinal("the straw squad", 2))
}

func TestListKinds(t *testing.T) {
	assert.Equal(t, "two ruffians, a cutpurse, and a rat", ListKinds([]string{"ruffian", "cutpurse", "ruffian", "rat"}))
	assert.Equal(t, "an archer and three footmen", ListKinds([]string{"archer", "footman", "footman", "footman"}))
	assert.Equal(t, "two rats", ListKinds([]string{"rat", "rat"}))
}

func TestMatchesTheLastWordOfAKind(t *testing.T) {
	n := NameGroup([]MobSummary{{InstanceId: 1, Name: "big rat"}, {InstanceId: 2, Name: "big rat"}})
	assert.Equal(t, "a band of big rats", n.Name)
	assert.True(t, n.Matches("rats"))
	assert.True(t, n.Matches("big rats"))
	assert.False(t, n.Matches("rat"))
}
