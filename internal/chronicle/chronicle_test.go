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
		{Entry{Kind: Relic, Subject: "the Pale Crown", Detail: "the Hollow King"}, "The company found the Pale Crown on the Hollow King."},
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

// A later phase tells apart two companions with one name by their keys.
func TestFilterByMemberKey(t *testing.T) {
	var l Log
	l.Add(Entry{Kind: Fell, Members: []string{"Mara"}, Keys: []string{"companion:1"}})
	l.Add(Entry{Kind: Fell, Members: []string{"Mara"}, Keys: []string{"companion:4"}})
	l.Add(Entry{Kind: Dismissed, Members: []string{"Mara", "Tobin"}, Keys: []string{"companion:4", "companion:5"}})
	assert.Len(t, l.Query(Filter{Member: "mara"}), 3)
	assert.Len(t, l.Query(Filter{Key: "companion:4"}), 2)
	assert.Len(t, l.Query(Filter{Key: "companion:1", Kinds: []Kind{Fell}}), 1)
	assert.Empty(t, l.Query(Filter{Key: "leader"}))
	c := l.Clone()
	c.Entries[2].Keys[0] = "changed"
	assert.Equal(t, "companion:4", l.Entries[2].Keys[0], "a clone shares no keys")
}

// Phase 67: a relic's awakening is a deed, with a sentence and filter words,
// and OnRecord observers see every deed a caller records.
func TestAwakenedDeedHasProseWordsAndObservers(t *testing.T) {
	e := Entry{Kind: Awakened, Members: []string{"Mara"}, Subject: "Ogrebane", Detail: "Giant-Slayer", Place: "Hollowweb Deep"}
	assert.Equal(t, "Mara's Ogrebane awoke to Giant-Slayer at Hollowweb Deep.", Prose(e))
	assert.Equal(t, "The company's relic awoke.", Prose(Entry{Kind: Awakened}))
	for _, w := range []string{"awakened", "awakenings", "awakening"} {
		k, ok := KindByWord(w)
		assert.True(t, ok, w)
		assert.Equal(t, Awakened, k)
	}

	var seen []Entry
	OnRecord(func(_ int, got Entry) { seen = append(seen, got) })
	Record(7, Entry{Kind: Boss, Ref: "mob:3"}) // no provider installed: still observed
	require.Len(t, seen, 1)
	assert.Equal(t, "mob:3", seen[0].Ref)
	Record(0, Entry{Kind: Boss})
	Record(7, Entry{Kind: "nonsense"})
	assert.Len(t, seen, 1, "an invalid deed is not observed")
}

// Phase 70: an errand deed reads as a sentence and answers to its filter words.
func TestErrandDeedProseAndWords(t *testing.T) {
	e := Entry{Kind: Errand, Members: []string{"Ysolde"}, Subject: "a hunt", Place: "Brindle Downs", Detail: "with 45 gold"}
	assert.Equal(t, "Ysolde came back from a hunt at Brindle Downs with 45 gold.", Prose(e))
	assert.Equal(t, "Ysolde came back from an errand.", Prose(Entry{Kind: Errand, Members: []string{"Ysolde"}}))
	for _, w := range []string{"errand", "errands"} {
		k, ok := KindByWord(w)
		assert.True(t, ok, w)
		assert.Equal(t, Errand, k)
	}
	assert.True(t, Errand.Valid())
}

// Phase 74: rites read as sentences, held or let pass, and answer to their
// filter words.
func TestRitesProse(t *testing.T) {
	held := Entry{Kind: Rites, Subject: "Hild", Detail: "lost for good", Ref: "rite:held"}
	assert.Equal(t, "The company held rites for Hild (lost for good).", Prose(held))
	assert.Equal(t, "The company let Hild go without a word (left the company).", Prose(Entry{Kind: Rites, Subject: "Hild", Detail: "left the company", Ref: "rite:skipped"}))
	assert.Equal(t, "The company held rites for one of their own.", Prose(Entry{Kind: Rites}))
	for _, w := range []string{"rites", "rite", "funerals"} {
		k, ok := KindByWord(w)
		assert.True(t, ok, w)
		assert.Equal(t, Rites, k)
	}
}

func TestGroupAndBountyDeedsReadAndFilterByZone(t *testing.T) {
	assert.Equal(t, "The company broke Road Brigands at the green.", Prose(Entry{Kind: Group, Subject: "Road Brigands", Place: "the green"}))
	assert.Equal(t, "The company claimed the bounty on the Hollow King at the board: 160 gold.", Prose(Entry{Kind: Bounty, Subject: "the Hollow King", Place: "the board", Detail: "160 gold"}))
	for _, w := range []string{"group", "bands", "packs"} {
		k, ok := KindByWord(w)
		assert.True(t, ok)
		assert.Equal(t, Group, k)
	}
	if k, ok := KindByWord("bounties"); assert.True(t, ok) {
		assert.Equal(t, Bounty, k)
	}

	var l Log
	l.Add(Entry{Kind: Boss, Ref: "mob:1", Zone: "Dark Forest"})
	l.Add(Entry{Kind: Boss, Ref: "mob:1", Zone: "Catacombs"})
	l.Add(Entry{Kind: Boss, Ref: "mob:1", Zone: "Dark Forest"})
	assert.Len(t, l.Query(Filter{Ref: "mob:1", Zone: "Dark Forest"}), 2, "a boss slain in another zone is not this zone's lair")
	assert.Len(t, l.Query(Filter{Ref: "mob:1", Zone: "Dark Forest", AfterSeq: 1}), 1, "only deeds numbered after a point count")
}

func TestMasteryDeedsReadWithOrWithoutTheArticle(t *testing.T) {
	assert.Equal(t, "The company learned the habits of the big rat at Old King's Road.", Prose(Entry{Kind: Mastered, Subject: "big rat", Place: "Old King's Road"}))
	assert.Equal(t, "The company learned the habits of Rodric at the ford.", Prose(Entry{Kind: Mastered, Subject: "Rodric", Place: "the ford"}))
	assert.Equal(t, "The company learned the habits of a creature.", Prose(Entry{Kind: Mastered}))
	for _, w := range []string{"lore", "mastered", "mastery", "beasts", "beast"} {
		k, ok := KindByWord(w)
		assert.True(t, ok, w)
		assert.Equal(t, Mastered, k, w)
	}
	assert.True(t, Mastered.Valid())
}

func TestMasteryDeedsAreCappedWithinTheLog(t *testing.T) {
	var l Log
	l.Add(Entry{Kind: Boss, Ref: "mob:1"})
	for i := 1; i <= 35; i++ {
		l.Add(Entry{Kind: Mastered, Ref: fmt.Sprintf("mob:%d", i)})
		if i == 10 {
			l.Add(Entry{Kind: Relic, Subject: "ring"})
		}
	}
	got := l.Query(Filter{Kinds: []Kind{Mastered}})
	require.Len(t, got, KindCaps[Mastered], "no more than the cap is kept")
	assert.Equal(t, "mob:35", got[0].Ref, "the newest stays")
	assert.Equal(t, "mob:6", got[len(got)-1].Ref, "the oldest five went first")
	assert.Equal(t, 1, l.Tally[Boss])
	assert.Len(t, l.Query(Filter{Kinds: []Kind{Boss}}), 1, "other kinds are left alone")
	assert.Len(t, l.Query(Filter{Kinds: []Kind{Relic}}), 1)
	assert.Equal(t, 35, l.Tally[Mastered], "the tally still counts every deed")
	assert.Equal(t, 37, l.NextSeq)
}
