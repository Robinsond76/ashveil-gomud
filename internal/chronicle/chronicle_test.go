package chronicle

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProseReadsEveryKindAsASentence(t *testing.T) {
	cases := []struct {
		e    Entry
		want string
	}{
		{Entry{Kind: Joined, Members: []string{"Mara"}, Place: "the Waymark Inn"}, "Mara joined the company at the Waymark Inn."},
		{Entry{Kind: Dismissed, Members: []string{"Mara", "Tobin"}}, "Mara and Tobin were sent away."},
		{Entry{Kind: Deserted, Members: []string{"Mara"}}, "Mara lost faith in the company and left."},
		{Entry{Kind: Fell, Members: []string{"Mara"}, Subject: "a wolf", Place: "the ford"}, "Mara fell to a wolf at the ford."},
		{Entry{Kind: Fell, Members: []string{"Mara"}}, "Mara fell."},
		{Entry{Kind: Raised, Members: []string{"Mara"}}, "Mara was raised from the dead."},
		{Entry{Kind: Lost, Members: []string{"Mara"}}, "Mara was lost for good; no one raised them in time."},
		{Entry{Kind: Defeated, Detail: "They were left for dead.", Place: "the ford"}, "The company was beaten at the ford. They were left for dead."},
		{Entry{Kind: Boss, Subject: "the Hollow King", Place: "the Throne Room"}, "The company slew the Hollow King at the Throne Room."},
		{Entry{Kind: Relic, Subject: "the Pale Crown", Detail: "the Hollow King"}, "The company took the Pale Crown from the Hollow King."},
		{Entry{Kind: Spared, Subject: "a bandit"}, "The company showed mercy to a bandit."},
		{Entry{Kind: Executed, Subject: "a bandit"}, "The company put a bandit to the sword."},
		{Entry{Kind: Promoted, Members: []string{"Mara"}, Subject: "Samurai"}, "Mara became a Samurai."},
		{Entry{Kind: Promoted, Members: []string{"Mara"}, Subject: "Alchemist"}, "Mara became an Alchemist."},
		{Entry{Kind: Story, Members: []string{"Mara"}, Subject: "The Cliff", Detail: "send the climber"}, "At The Cliff, Mara chose: Send the climber."},
		{Entry{Kind: Story, Subject: "The Cliff", Detail: "turn back."}, "At The Cliff, the company chose: Turn back."},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, Prose(c.e), string(c.e.Kind))
		assert.NotContains(t, Prose(c.e), "<", "prose is plain text")
	}
	for _, k := range Kinds {
		assert.NotEmpty(t, Prose(Entry{Kind: k.Kind}), "%s reads with nothing filled in", k.Kind)
	}
}

func TestWho(t *testing.T) {
	assert.Equal(t, "The company", Who(nil))
	assert.Equal(t, "The company", Who([]string{" "}))
	assert.Equal(t, "Mara", Who([]string{"Mara"}))
	assert.Equal(t, "Mara and Tobin", Who([]string{"Mara", "Tobin"}))
	assert.Equal(t, "Mara, Tobin and Ysolde", Who([]string{"Mara", "Tobin", "Ysolde"}))
}

func TestAgo(t *testing.T) {
	now := int64(1_000_000)
	for _, c := range []struct {
		d    int64
		want string
	}{{0, "just now"}, {89, "just now"}, {120, "2 minutes ago"}, {59 * 60, "59 minutes ago"}, {3600, "1 hour ago"}, {2 * 3600, "2 hours ago"}, {23 * 3600, "23 hours ago"}, {3 * 86400, "3 days ago"}, {40 * 3600, "2 days ago"}} {
		assert.Equal(t, c.want, Ago(now-c.d, now), fmt.Sprint(c.d))
	}
	assert.Equal(t, "1 hour ago", Ago(now-3700, now))
}

func TestKindWordsResolveAndAreUnique(t *testing.T) {
	seen := map[string]Kind{}
	for _, k := range Kinds {
		assert.True(t, k.Kind.Valid())
		require.NotEmpty(t, k.Words, k.Kind)
		for _, w := range k.Words {
			if other, dup := seen[w]; dup {
				t.Errorf("word %q names both %s and %s", w, other, k.Kind)
			}
			seen[w] = k.Kind
			got, ok := KindByWord(strings.ToUpper(w))
			assert.True(t, ok)
			assert.Equal(t, k.Kind, got)
		}
	}
	_, ok := KindByWord("nonsense")
	assert.False(t, ok)
	assert.False(t, Kind("nope").Valid())
}

func TestLogKeepsTheNewestAndCountsEverything(t *testing.T) {
	var l Log
	for i := 1; i <= MaxEntries+25; i++ {
		l.Add(Entry{Kind: Joined, At: int64(i), Members: []string{fmt.Sprint("m", i)}})
	}
	l.Add(Entry{Kind: Boss, At: 9999, Ref: "mob:5"})
	assert.Len(t, l.Entries, MaxEntries)
	assert.Equal(t, 1, l.Tally[Boss])
	assert.Equal(t, MaxEntries+25, l.Tally[Joined], "the tally never forgets")
	assert.Equal(t, MaxEntries+26, l.Entries[len(l.Entries)-1].Seq, "sequence numbers keep counting")
	assert.Equal(t, "mob:5", l.Query(Filter{Limit: 1})[0].Ref, "newest first")
	assert.Equal(t, "m27", l.Entries[0].Members[0], "the oldest were dropped")
}

func TestFilterAndQuery(t *testing.T) {
	var l Log
	l.Add(Entry{Kind: Joined, At: 10, Members: []string{"Mara"}, Ref: "mob:1"})
	l.Add(Entry{Kind: Fell, At: 20, Members: []string{"Mara"}, Ref: "mob:1"})
	l.Add(Entry{Kind: Boss, At: 30, Ref: "mob:9"})
	l.Add(Entry{Kind: Boss, At: 40, Ref: "mob:10"})

	assert.Len(t, l.Query(Filter{}), 4)
	assert.Equal(t, int64(40), l.Query(Filter{})[0].At)
	assert.Len(t, l.Query(Filter{Kinds: []Kind{Boss}}), 2)
	assert.Len(t, l.Query(Filter{Kinds: []Kind{Boss, Fell}}), 3)
	assert.Len(t, l.Query(Filter{Ref: "mob:9"}), 1)
	assert.Len(t, l.Query(Filter{Member: "mara"}), 2, "members match without regard to case")
	assert.Len(t, l.Query(Filter{Since: 30}), 2)
	assert.Len(t, l.Query(Filter{Limit: 3}), 3)
	assert.Empty(t, l.Query(Filter{Member: "Tobin"}))
}

func TestCloneSharesNothing(t *testing.T) {
	var l Log
	l.Add(Entry{Kind: Joined, Members: []string{"Mara"}})
	c := l.Clone()
	c.Entries[0].Members[0] = "Other"
	c.Tally[Joined] = 50
	assert.Equal(t, "Mara", l.Entries[0].Members[0])
	assert.Equal(t, 1, l.Tally[Joined])
}

func TestTheSeamDoesNothingWithoutAProviderAndReadsWithOne(t *testing.T) {
	SetProvider(nil)
	Record(1, Entry{Kind: Joined})
	assert.Empty(t, Query(1, Filter{}))
	assert.False(t, Has(1, Filter{}))
	assert.Zero(t, Count(1, Filter{}))
	assert.Zero(t, Total(1, Joined))

	mem := NewMemory()
	SetProvider(mem)
	t.Cleanup(func() { SetProvider(nil) })
	Record(1, Entry{Kind: Boss, Subject: "Hollow King", Ref: "mob:9"})
	Record(1, Entry{Kind: Boss, Subject: "Hollow Queen", Ref: "mob:10"})
	Record(1, Entry{Kind: "bogus"})
	Record(0, Entry{Kind: Boss})
	Record(2, Entry{Kind: Relic})
	assert.True(t, Has(1, Filter{Ref: "mob:9"}))
	assert.False(t, Has(1, Filter{Ref: "mob:99"}))
	assert.Equal(t, 2, Count(1, Filter{Kinds: []Kind{Boss}, Limit: 1}), "Count ignores the limit")
	assert.Equal(t, 2, Total(1, Boss))
	assert.Equal(t, 1, Total(2, Relic))
	assert.Zero(t, Total(1, Relic), "companies are separate")
	assert.NotZero(t, Query(1, Filter{})[0].At, "the memory stamps the time")
	assert.Equal(t, "Hollow Queen", Query(1, Filter{Limit: 1})[0].Subject)
}
