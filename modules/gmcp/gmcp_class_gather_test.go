package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCharInfoNamesThePromotedClass (Phase 45): Char.Info's class is the
// promoted class once promoted, with the lineage kept in lineage_name for
// the Character window's hover; before promotion it is the archetype and no
// lineage_name is sent.
func TestCharInfoNamesThePromotedClass(t *testing.T) {
	archetypes.SetProvider(fakeArchetypes{chosen: map[int]string{94971: "warrior", 94972: "cleric"}})
	t.Cleanup(func() { archetypes.SetProvider(nil) })
	classes.SetProvider(fakeClasses{class: map[int]string{94972: "priest"}})
	t.Cleanup(func() { classes.SetProvider(nil) })

	info := func(uid int) map[string]any {
		u := users.NewUserRecord(uid, 0)
		u.Character.Name = "Hero"
		node, _ := (&GMCPCharModule{}).GetCharNode(u, "Char.Info")
		raw, err := json.Marshal(node)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(raw, &got))
		return got
	}
	plain := info(94971)
	assert.Equal(t, "warrior", plain["class"], "the archetype name (the fake names it by id)")
	assert.NotContains(t, plain, "lineage_name")

	promoted := info(94972)
	assert.Equal(t, "Priest", promoted["class"], "the promoted class names the character")
	assert.Equal(t, "cleric", promoted["lineage_name"], "and the lineage rides along")
	assert.Equal(t, "priest", promoted["classid"])
}

// TestRoomGatherCarriesTheWorksProgress (Phase 45): a GatherProgress event
// reaches the leader's client as Room.Gather through the real listener.
func TestRoomGatherCarriesTheWorksProgress(t *testing.T) {
	var out []GMCPOut
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out = append(out, e.(GMCPOut))
		return events.Cancel // the dispatcher needs a connection; only the payload is under test
	})
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })

	g := &GMCPRoomModule{}
	g.gatherProgressHandler(events.GatherProgress{UserId: 7, Kind: "herbs", Label: "gathering herbs", Phase: "start", Seconds: 20})
	g.gatherProgressHandler(events.GatherProgress{UserId: 7, Kind: "herbs", Label: "gathering herbs", Phase: "done", Lines: []string{`Your company gathers <ansi fg="itemname">2 thyme</ansi>.`}})
	events.ProcessEvents()

	require.Len(t, out, 2)
	assert.Equal(t, 7, out[0].UserId)
	assert.Equal(t, "Room.Gather", out[0].Module)
	start := out[0].Payload.(GMCPRoomGatherPayload)
	assert.Equal(t, "start", start.Phase)
	assert.Equal(t, 20, start.Seconds)
	raw, err := json.Marshal(out[1].Payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{"phase":"done","kind":"herbs","label":"gathering herbs","lines":["Your company gathers 2 thyme."]}`, string(raw), "colour tags are stripped for the panel")
}
