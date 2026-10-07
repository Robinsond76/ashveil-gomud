package storyevents

import "sync"

// MovementProvider is implemented by modules/storyevents. While a page is
// waiting for the company's answer, ordinary movement refuses with the
// returned text, as an active rest does.
type MovementProvider interface {
	MovementBlocked(leaderUserID int) (blocked bool, message string)
}

var (
	providerMu       sync.RWMutex
	movementProvider MovementProvider
)

// SetMovementProvider registers the movement-block provider; nil clears it.
func SetMovementProvider(p MovementProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	movementProvider = p
}

// MovementBlocked reports whether a waiting page must refuse ordinary
// movement. Without a provider movement is unchanged.
func MovementBlocked(leaderUserID int) (bool, string) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	if p == nil {
		return false, ""
	}
	return p.MovementBlocked(leaderUserID)
}

// TagSource names a member's tags for requirements (Phase 72 backgrounds
// and later hooks register one): the member is the leader or a companion,
// by its member key.
type TagSource func(leaderUserID int, memberKey string) []string

var tagSources []TagSource

// RegisterTagSource adds a tag source.
func RegisterTagSource(fn TagSource) {
	providerMu.Lock()
	defer providerMu.Unlock()
	tagSources = append(tagSources, fn)
}

// TagsFor is every registered source's tags for one member.
func TagsFor(leaderUserID int, memberKey string) []string {
	providerMu.RLock()
	fns := append([]TagSource(nil), tagSources...)
	providerMu.RUnlock()
	var out []string
	for _, fn := range fns {
		out = append(out, fn(leaderUserID, memberKey)...)
	}
	return out
}
