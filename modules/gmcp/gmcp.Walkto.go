package gmcp

// Phase 40d: the Walkto namespace. While a leader's click-to-walk (the
// `walkto` command or a click on a visited tile) is under way it carries
// the target room and the rooms still ahead, so the web map can draw the
// planned path; {} once when it ends. It is built from internal/walkto.ViewOf,
// sent only when it changes, and only to that player. Walkto is a new
// top-level namespace (the existing `travel` command and Char.* payloads
// belong to journeys and the character).

import (
	"encoding/json"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walkto"
)

type walktoPayload struct {
	Target int   `json:"target"`
	Path   []int `json:"path"`
}

// walktoPayloadOf is the payload for a view; nil (sent as {}) when the user
// is not walking.
func walktoPayloadOf(v walkto.View, ok bool) *walktoPayload {
	if !ok {
		return nil
	}
	p := &walktoPayload{Target: v.Target, Path: []int{}}
	p.Path = append(p.Path, v.Path...)
	return p
}

// walktoFeed decides what to send. It runs on the game loop; mu only keeps
// tests honest.
type walktoFeed struct {
	mu        sync.Mutex
	last      map[int]string
	view      func(userID int) (walkto.View, bool)
	send      func(userID int, module string, payload []byte)
	accepting func(userID int) bool
}

func newWalktoFeed() *walktoFeed {
	return &walktoFeed{
		last: map[int]string{},
		view: walkto.ViewOf,
		send: func(userID int, module string, payload []byte) {
			events.AddToQueue(GMCPOut{UserId: userID, Module: module, Payload: payload})
		},
		accepting: nativeAccepting,
	}
}

// update sends a user's Walkto payload when it changed since the last send.
func (f *walktoFeed) update(userID int) {
	if f.accepting != nil && !f.accepting(userID) {
		f.forget(userID)
		return
	}
	body := []byte("{}")
	if payload := walktoPayloadOf(f.view(userID)); payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			mudlog.Error("gmcp: Walkto payload", "error", err)
			return
		}
	}
	f.mu.Lock()
	prev, seen := f.last[userID]
	f.last[userID] = string(body)
	f.mu.Unlock()
	if !seen || prev != string(body) {
		f.send(userID, "Walkto", body)
	}
}

// forget makes the next update send the payload (login, a request) or drops
// the user (logout).
func (f *walktoFeed) forget(userID int) {
	f.mu.Lock()
	delete(f.last, userID)
	f.mu.Unlock()
}

// prune drops users no longer online.
func (f *walktoFeed) prune(online []int) {
	live := make(map[int]bool, len(online))
	for _, id := range online {
		live[id] = true
	}
	f.mu.Lock()
	for id := range f.last {
		if !live[id] {
			delete(f.last, id)
		}
	}
	f.mu.Unlock()
}

// GMCPWalktoRequest asks for a user's Walkto payload now.
type GMCPWalktoRequest struct {
	UserId int
}

func (g GMCPWalktoRequest) Type() string { return `GMCPWalktoRequest` }

var walktoFeeds = newWalktoFeed()

func init() {
	companyview.OnRefresh.Register(func(r companyview.Refreshed) companyview.Refreshed {
		walktoFeeds.update(r.User.UserId)
		return r
	})
	// Each step, a start and a stop reach the map at once, not a round later.
	walkto.OnChanged.Register(func(userID int) int {
		if users.GetByUserId(userID) != nil {
			walktoFeeds.update(userID)
		}
		return userID
	})
	events.RegisterListener(events.PlayerSpawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerSpawn); ok {
			walktoFeeds.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			walktoFeeds.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.NewRound{}, func(events.Event) events.ListenerReturn {
		walktoFeeds.prune(users.GetOnlineUserIds())
		return events.Continue
	})
	events.RegisterListener(GMCPWalktoRequest{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(GMCPWalktoRequest); ok {
			walktoFeeds.forget(evt.UserId)
			walktoFeeds.update(evt.UserId)
		}
		return events.Continue
	})
}
