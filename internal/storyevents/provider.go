package storyevents

import (
	"errors"
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/util"
)

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

// FlagProvider is implemented by modules/storyevents: a company's flags,
// the marks scenes leave (Phase 68 towns that remember reads and sets them).
type FlagProvider interface {
	CompanyFlags(leaderUserID int) []string
	SetCompanyFlag(leaderUserID int, flag string) error
}

var flagProvider FlagProvider

// SetFlagProvider registers the flag provider; nil clears it.
func SetFlagProvider(p FlagProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	flagProvider = p
}

// CompanyFlags lists a company's flags, sorted; nil without a provider.
func CompanyFlags(leaderUserID int) []string {
	providerMu.RLock()
	p := flagProvider
	providerMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.CompanyFlags(leaderUserID)
}

// HasCompanyFlag reports whether a company carries a flag.
func HasCompanyFlag(leaderUserID int, flag string) bool {
	for _, f := range CompanyFlags(leaderUserID) {
		if f == flag {
			return true
		}
	}
	return false
}

// SetCompanyFlag gives a company a flag, saved. A flag must be lowercase
// words joined by dashes.
func SetCompanyFlag(leaderUserID int, flag string) error {
	if !idPattern.MatchString(flag) {
		return fmt.Errorf("storyevents: flag %q must be lowercase words joined by dashes", flag)
	}
	providerMu.RLock()
	p := flagProvider
	providerMu.RUnlock()
	if p == nil {
		return errors.New("storyevents: no flag provider")
	}
	return p.SetCompanyFlag(leaderUserID, flag)
}

// OnPushRequest fires, on the game loop, when the web client asks for the
// page waiting on its company (it asks once the connection is ready, as a
// page shown at login is sent before GMCP is accepted and lost). The
// module sends the page's GMCP message again, with no text.
var OnPushRequest util.Hook[int]

// RequestPush asks for the waiting page to be sent to the web client again.
func RequestPush(leaderUserID int) { OnPushRequest.Fire(leaderUserID) }
