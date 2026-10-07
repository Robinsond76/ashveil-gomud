package storyevents

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 72 review: a choice a life story opens says so in its note (web
// modal and terminal), and one without an authored hint still reads in
// words rather than as a raw tag.
func shippedLifeStory(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "lifestory.yaml"))
	require.NoError(t, err)
	require.NoError(t, lifestory.LoadBytes(raw))
	t.Cleanup(func() { lifestory.SetData(nil) })
}

func TestABackgroundChoiceSaysWhatOpenedIt(t *testing.T) {
	shippedLifeStory(t)

	page := storyevents.Page{Choices: []storyevents.Choice{
		{Label: "Talk soldier to soldier", Require: storyevents.Requirement{Tag: "trade-soldier", Leader: true}},
		{Label: "Read the hills", Require: storyevents.Requirement{Tag: "homeland-hill-clans", Leader: true}},
	}}
	members := []storyevents.Facts{{Key: "leader", Name: "Mara", Leader: true, Tags: []string{"trade-soldier"}}}
	views := (&Module{}).choiceViews(page, members, storyevents.Company{})
	require.Len(t, views, 2)
	assert.Equal(t, choiceView{N: 1, Label: "Talk soldier to soldier", Open: true, Who: "Mara", Because: "life story: a soldier"}, views[0])
	assert.Equal(t, choiceView{N: 2, Label: "Read the hills", Needs: "you, life story: the hill clans"}, views[1])
	assert.Contains(t, render(payload{Choices: views}), "Talk soldier to soldier <ansi fg=\"black-bold\">(Mara, life story: a soldier)</ansi>")
}

// Life stories are the only tag source, so a shipped choice's tag must name
// a real pick: a misspelt one would load, never open, and show its raw tag.
func TestEveryShippedChoiceTagIsALifeStoryPick(t *testing.T) {
	shippedLifeStory(t)
	files, err := fs.Glob(shipped, "events/*.yaml")
	require.NoError(t, err)
	found := 0
	for _, f := range files {
		raw, err := shipped.ReadFile(f)
		require.NoError(t, err)
		evs, err := storyevents.Parse(raw)
		require.NoError(t, err, f)
		for _, ev := range evs {
			for pageID, page := range ev.Pages {
				for _, c := range page.Choices {
					if c.Require.Tag == "" {
						continue
					}
					found++
					_, ok := lifestory.TagName(c.Require.Tag)
					assert.True(t, ok, "%s %s/%s %q: tag %q is no life-story pick", f, ev.ID, pageID, c.Label, c.Require.Tag)
				}
			}
		}
	}
	assert.Positive(t, found, "the test scenes carry background choices")
}

// A tag a companion carries is never called a life story: only the leader
// has one.
func TestACompanionsTagIsNotCalledALifeStory(t *testing.T) {
	shippedLifeStory(t)
	page := storyevents.Page{Choices: []storyevents.Choice{
		{Label: "Drill them", Require: storyevents.Requirement{Tag: "trade-soldier"}},
	}}
	members := []storyevents.Facts{
		{Key: "leader", Name: "Mara", Leader: true},
		{Key: "companion:3", Name: "Sera", Tags: []string{"trade-soldier"}},
	}
	views := (&Module{}).choiceViews(page, members, storyevents.Company{})
	require.Len(t, views, 1)
	assert.Equal(t, "Sera", views[0].Who)
	assert.Empty(t, views[0].Because)
}
