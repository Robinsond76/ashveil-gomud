package storyevents

import (
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
func TestABackgroundChoiceSaysWhatOpenedIt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "world", "default", "lifestory.yaml"))
	require.NoError(t, err)
	require.NoError(t, lifestory.LoadBytes(raw))
	t.Cleanup(func() { lifestory.SetData(nil) })

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
