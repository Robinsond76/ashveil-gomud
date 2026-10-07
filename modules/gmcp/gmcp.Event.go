package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
)

// GMCPEventRequest asks for the story-event page waiting on a user's
// company (Phase 78): the web client sends it once its connection is ready,
// as the page shown at login went out before GMCP was accepted.
type GMCPEventRequest struct {
	UserId int
}

func (g GMCPEventRequest) Type() string { return `GMCPEventRequest` }

func init() {
	events.RegisterListener(GMCPEventRequest{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(GMCPEventRequest); ok {
			storyevents.RequestPush(evt.UserId)
		}
		return events.Continue
	})
}
