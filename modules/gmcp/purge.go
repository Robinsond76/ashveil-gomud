package gmcp

import "github.com/GoMudEngine/GoMud/internal/events"

// Phase 32b: a purged user's GMCP caches go. They are dropped on logout
// already; this makes the purge complete on its own.
func init() {
	events.RegisterListener(events.UserPurged{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.UserPurged); ok {
			companyFeeds.forget(evt.UserId)
			tutorialFeeds.forget(evt.UserId)
			bestiaryForget(evt.UserId)
		}
		return events.Continue
	})
	// Phase 66: a client that logged out no longer asks for the bestiary.
	events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		if evt, ok := e.(events.PlayerDespawn); ok {
			bestiaryForget(evt.UserId)
		}
		return events.Continue
	})
}

// userPurgedHandler forgets whether a purged user's client was Mudlet.
func (g *GMCPMudletModule) userPurgedHandler(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.UserPurged); ok {
		delete(g.mudletUsers, evt.UserId)
	}
	return events.Continue
}
