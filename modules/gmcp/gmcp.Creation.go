package gmcp

// Ashveil Phase 72a: the Char.Creation namespace. It carries the looks and
// life story step a player is on (the question, its answers, the picks so
// far and a live preview) from internal/creation, so the web client's
// creation panel can show the same step the telnet prompt asks. It is sent
// whenever a step is asked and once as {"active":false} when the steps end,
// and again on a request (login, copyover), only to that player. The panel
// answers by sending the same input a telnet player types.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// GMCPCreationRequest asks for a user's Char.Creation payload now.
type GMCPCreationRequest struct {
	UserId int
}

func (g GMCPCreationRequest) Type() string { return `GMCPCreationRequest` }

// creationSend queues a Char.Creation payload; tests replace it.
var creationSend = func(userID int, payload []byte) {
	events.AddToQueue(GMCPOut{UserId: userID, Module: `Char.Creation`, Payload: payload})
}

func sendCreation(userID int, v creation.View) {
	if !nativeAccepting(userID) {
		return
	}
	body, err := json.Marshal(v)
	if err != nil {
		mudlog.Error("gmcp: Char.Creation payload", "error", err)
		return
	}
	creationSend(userID, body)
}

func init() {
	creation.Subscribe(sendCreation)
	events.RegisterListener(GMCPCreationRequest{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(GMCPCreationRequest); ok {
			if v, open := creation.Get(evt.UserId); open {
				sendCreation(evt.UserId, v)
			} else {
				sendCreation(evt.UserId, creation.View{Active: false})
			}
		}
		return events.Continue
	})
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			creation.Forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.UserPurged{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.UserPurged); ok {
			creation.Forget(evt.UserId)
		}
		return events.Continue
	})
}
