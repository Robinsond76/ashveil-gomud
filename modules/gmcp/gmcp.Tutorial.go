package gmcp

// Phase 27d: the Ashveil Tutorial namespace. It carries the player's place
// in the tutorial course (stage, goal, checklist, hints) from
// internal/tutorial.ViewOf, the same data the terminal "tutorial" command
// shows. It is built on each companyview refresh (every round and after
// every command, on the game loop), sent only when it changes, and only to
// that player; {} once for a player not in the course. See
// docs/superpowers/specs/2026-09-25-phase-27d-tutorial-alignment-panel-design.md.

import (
	"encoding/json"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/tutorial"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type tutorialCheck struct {
	Label string `json:"label"`
	Done  bool   `json:"done"`
}

type tutorialPayload struct {
	Active    bool            `json:"active"`
	Stage     int             `json:"stage"`
	Stages    int             `json:"stages"`
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	Goal      string          `json:"goal"`
	Checklist []tutorialCheck `json:"checklist"`
	Hints     []string        `json:"hints"`
}

// tutorialPayloadOf is the payload for a view; nil (sent as {}) when the
// player isn't in the course.
func tutorialPayloadOf(v tutorial.View, ok bool) *tutorialPayload {
	if !ok {
		return nil
	}
	p := &tutorialPayload{Active: true, Stage: v.Stage, Stages: v.Stages, ID: v.ID, Title: v.Title, Goal: v.Goal,
		Checklist: []tutorialCheck{}, Hints: []string{}}
	for _, c := range v.Checklist {
		p.Checklist = append(p.Checklist, tutorialCheck{Label: c.Label, Done: c.Done})
	}
	p.Hints = append(p.Hints, v.Hints...)
	return p
}

// tutorialFeed decides what to send. It runs on the game loop; mu only
// keeps tests honest.
type tutorialFeed struct {
	mu        sync.Mutex
	last      map[int]string
	view      func(userID int) (tutorial.View, bool)
	send      func(userID int, module string, payload []byte)
	accepting func(userID int) bool
}

func newTutorialFeed() *tutorialFeed {
	return &tutorialFeed{
		last: map[int]string{},
		view: tutorial.ViewOf,
		send: func(userID int, module string, payload []byte) {
			events.AddToQueue(GMCPOut{UserId: userID, Module: module, Payload: payload})
		},
		accepting: nativeAccepting,
	}
}

// update sends a user's Tutorial payload when it changed since the last
// send.
func (f *tutorialFeed) update(userID int) {
	if f.accepting != nil && !f.accepting(userID) {
		f.forget(userID)
		return
	}
	payload := tutorialPayloadOf(f.view(userID))
	body := []byte("{}")
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			mudlog.Error("gmcp: Tutorial payload", "error", err)
			return
		}
	}
	f.mu.Lock()
	prev, seen := f.last[userID]
	f.last[userID] = string(body)
	f.mu.Unlock()
	if !seen || prev != string(body) {
		f.send(userID, "Tutorial", body)
	}
}

// forget makes the next update send the payload (login, copyover, a
// request) or drops the user (logout).
func (f *tutorialFeed) forget(userID int) {
	f.mu.Lock()
	delete(f.last, userID)
	f.mu.Unlock()
}

// prune drops users no longer online.
func (f *tutorialFeed) prune(online []int) {
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

// GMCPTutorialRequest asks for a user's Tutorial payload now.
type GMCPTutorialRequest struct {
	UserId int
}

func (g GMCPTutorialRequest) Type() string { return `GMCPTutorialRequest` }

var tutorialFeeds = newTutorialFeed()

func init() {
	companyview.OnRefresh.Register(func(r companyview.Refreshed) companyview.Refreshed {
		tutorialFeeds.update(r.User.UserId)
		return r
	})
	// A stage passed after this refresh's update (the tutorial's gates run
	// on the same refresh) is sent at once, not a round later.
	tutorial.OnChanged.Register(func(userID int) int {
		tutorialFeeds.update(userID)
		return userID
	})
	events.RegisterListener(events.PlayerSpawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerSpawn); ok {
			tutorialFeeds.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			tutorialFeeds.forget(evt.UserId)
		}
		return events.Continue
	})
	events.RegisterListener(events.NewRound{}, func(events.Event) events.ListenerReturn {
		tutorialFeeds.prune(users.GetOnlineUserIds())
		return events.Continue
	})
	events.RegisterListener(GMCPTutorialRequest{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(GMCPTutorialRequest); ok {
			tutorialFeeds.forget(evt.UserId)
			companyview.RefreshUser(evt.UserId)
		}
		return events.Continue
	})
}
