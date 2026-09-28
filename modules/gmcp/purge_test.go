package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestUserPurgedForgetsGMCPCaches (Phase 32b): the Company and Tutorial
// feeds' last sends and the Mudlet flag, through the registered listeners.
func TestUserPurgedForgetsGMCPCaches(t *testing.T) {
	companyFeeds.mu.Lock()
	companyFeeds.last[932007] = companySent{}
	companyFeeds.mu.Unlock()
	tutorialFeeds.mu.Lock()
	tutorialFeeds.last[932007] = "{}"
	tutorialFeeds.mu.Unlock()
	g := &GMCPMudletModule{mudletUsers: map[int]bool{932007: true, 932008: true}}

	events.AddToQueue(events.UserPurged{UserId: 932007})
	events.ProcessEvents()
	g.userPurgedHandler(events.UserPurged{UserId: 932007})

	companyFeeds.mu.Lock()
	_, company := companyFeeds.last[932007]
	companyFeeds.mu.Unlock()
	tutorialFeeds.mu.Lock()
	_, tutorial := tutorialFeeds.last[932007]
	tutorialFeeds.mu.Unlock()
	assert.False(t, company)
	assert.False(t, tutorial)
	assert.Equal(t, map[int]bool{932008: true}, g.mudletUsers)
}
