package gmcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gathering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoomGatherCarriesTheWorksProgress (Phase 45): a GatherProgress event
// reaches the leader's client as Room.Gather through the real listener.
func TestRoomGatherCarriesTheWorksProgress(t *testing.T) {
	var out []GMCPOut
	freshEvents(t)
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

// Phase 46: a client that reconnects mid-work asks for Room.Gather and gets a
// start payload carrying how much of the work is already done.
func TestRoomGatherResumesWorkInProgress(t *testing.T) {
	gathering.SetProgressProvider(func(userID int) (gathering.Progress, bool) {
		if userID != 7 {
			return gathering.Progress{}, false
		}
		return gathering.Progress{Kind: "herbs", Label: "gathering herbs", Total: 20 * time.Second, Remaining: 12*time.Second + 300*time.Millisecond}, true
	})
	t.Cleanup(func() { gathering.SetProgressProvider(nil) })

	resume, ok := gatherResume(7)
	require.True(t, ok)
	assert.Equal(t, GMCPRoomGatherPayload{Phase: "start", Kind: "herbs", Label: "gathering herbs", Seconds: 20, Elapsed: 7}, resume)

	_, ok = gatherResume(8)
	assert.False(t, ok, "no work, nothing to resume")
}
