package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
