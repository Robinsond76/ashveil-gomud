package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/bestiary"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 66: the bestiary reads the shipped world's creature templates, so
// no entry is written by hand per creature.

func bestiaryWorld(t *testing.T) {
	t.Helper()
	useWorld(t, "default")
	races.LoadDataFiles()
	items.LoadDataFiles()
	spells.LoadSpellFiles()
	mobs.LoadDataFiles()
}

func TestBestiaryEntriesComeFromEveryShippedTemplate(t *testing.T) {
	bestiaryWorld(t)
	built := 0
	for _, m := range mobs.GetAllMobInfo() {
		e, ok := bestiary.Build(mobs.GetMobSpec(m.MobId), 99)
		if m.Practice {
			assert.False(t, ok, "%s: a practice foe is never entered", m.Character.Name)
			continue
		}
		require.True(t, ok, m.Character.Name)
		built++
		assert.Equal(t, bestiary.Habits, e.Tier)
		assert.NotEmpty(t, e.Lore, m.Character.Name)
		assert.NotEmpty(t, e.Defences, m.Character.Name)
		assert.NotEmpty(t, e.Habits, m.Character.Name)
		text := tagPattern.ReplaceAllString(BestiaryEntry(e), "")
		assert.NotContains(t, text, "%!", m.Character.Name)
		assert.NotContains(t, text, "<nil>", m.Character.Name)
		assert.NotContains(t, text, "  ,", m.Character.Name)
		assert.NotContains(t, text, "disabled", m.Character.Name, "an empty slot is not gear")
		assert.NotContains(t, text, "A undead", m.Character.Name)
	}
	assert.Greater(t, built, 20, "the shipped world has creatures")
}

func TestBestiaryKnowsOnlyWhatTheTierEarned(t *testing.T) {
	bestiaryWorld(t)
	spec := mobs.GetMobSpec(85) // the forest ogre: a boss with Crushing Blow
	require.NotNil(t, spec)

	lore, ok := bestiary.Build(spec, 1)
	require.True(t, ok)
	assert.Equal(t, bestiary.Lore, lore.Tier)
	assert.NotEmpty(t, lore.Lore)
	assert.Empty(t, lore.Defences)
	assert.Empty(t, lore.Habits)
	assert.Empty(t, lore.Notes)

	def, _ := bestiary.Build(spec, 2) // a boss: its second kill
	assert.Equal(t, bestiary.Defences, def.Tier)
	assert.NotEmpty(t, def.Defences)
	assert.Empty(t, def.Habits)

	hab, _ := bestiary.Build(spec, 3)
	assert.Equal(t, bestiary.Habits, hab.Tier)
	assert.Contains(t, strings.Join(hab.Habits, "\n"), "Crushing Blow")
	assert.Contains(t, hab.Notes, "winds up crushing blow")
	assert.Equal(t, 0, hab.NextAt)

	plain := mobs.GetMobSpec(1) // a rat: not a boss
	one, _ := bestiary.Build(plain, 5)
	assert.Equal(t, bestiary.Defences, one.Tier, "an ordinary kind needs six kills for its habits")
	assert.Equal(t, 6, one.NextAt)

	_, ok = bestiary.Build(spec, 0)
	assert.False(t, ok, "nothing of a kind never beaten")
	assert.Empty(t, bestiary.NotesFor(map[int]int{85: 2}, 85), "habit notes wait for the habits tier")
	assert.Equal(t, []string{"winds up crushing blow"}, bestiary.NotesFor(map[int]int{85: 3}, 85))
}

func TestBestiaryCommandAndConsiderLine(t *testing.T) {
	bestiaryWorld(t)
	user := users.NewUserRecord(7, 71)
	user.Character.Name = "Aria"
	out := func(arg string) string {
		return tagPattern.ReplaceAllString(heard(t, func() {
			_, err := Bestiary(arg, user, nil, 0)
			require.NoError(t, err)
		}), "")
	}
	assert.Contains(t, out(""), "Your bestiary is empty")

	user.Character.KD.Kills = map[int]int{85: 3, 1: 1}
	list := out("")
	assert.Contains(t, list, "Your bestiary: 2 kinds.")
	assert.Contains(t, list, "forest ogre")
	assert.Contains(t, list, "habits")
	assert.Contains(t, list, "lore")

	entry := out("forest ogre")
	assert.Contains(t, entry, "forest ogre, level 22, Dark Forest")
	assert.Contains(t, entry, "Known: habits (3 kills); everything is known.")
	assert.Contains(t, entry, "Habits and weaknesses")
	assert.Contains(t, entry, "Heavy force breaks it")

	assert.Contains(t, out("dark forest"), "forest ogre", "a zone name filters")
	assert.Contains(t, out("lich"), `Nothing called "lich" is in your bestiary`)

	// The foe line: what is known of the visible foes' habits, and what is new.
	ogre := mobs.GetMobSpec(85)
	rat := mobs.GetMobSpec(12)
	line := bestiary.FoeLine(user.Character.KD.Kills, []*mobs.Mob{ogre, ogre, rat})
	assert.Contains(t, line, "Known: forest ogre (habits: winds up crushing blow).")
	assert.Contains(t, line, "New to you: big rat.")
	assert.NotContains(t, bestiary.FoeLine(user.Character.KD.Kills, []*mobs.Mob{rat}), "Known")
}

func TestBestiaryHelpRenders(t *testing.T) {
	bestiaryWorld(t)
	keywords.LoadAliases()
	var combat []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" {
			combat = append(combat, topic.Command)
		}
	}
	assert.Contains(t, combat, "bestiary", "the combat category lists the page")

	text, err := GetHelpContents("bestiary")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for bestiary", "1st", "3rd", "6th", "boss", "orders [who] add foe healer then break", "Bestiary tab"} {
		assert.Contains(t, plain, want)
	}
	for _, alias := range []string{"monsters", "beast-lore", "foe-lore"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help bestiary", alias)
	}
	for _, hub := range []string{"combat", "consider", "webclient"} {
		page, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		assert.Contains(t, tagPattern.ReplaceAllString(page, ""), "bestiary", "help %s points at the bestiary", hub)
	}
}
