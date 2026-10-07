package camping

import (
	"sync"
	"time"
)

// ViewProvider is implemented by modules/camping. RenderCampView returns
// handled=true when it rendered an active resting session's view in place of
// ordinary room rendering.
type ViewProvider interface {
	RenderCampView(leaderUserID int) (handled bool, err error)
}

// MovementProvider is implemented by modules/camping. MovementBlocked reports
// an active rest and the refusal text for ordinary movement.
type MovementProvider interface {
	MovementBlocked(leaderUserID int) (blocked bool, message string)
}

var (
	providerMu       sync.RWMutex
	viewProvider     ViewProvider
	movementProvider MovementProvider
)

// SetViewProvider registers the active view provider. Passing nil clears it.
func SetViewProvider(p ViewProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	viewProvider = p
}

// SetMovementProvider registers the active movement-block provider. Passing
// nil clears it.
func SetMovementProvider(p MovementProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	movementProvider = p
}

// CampView consults the registered view provider. Without one, native look
// behavior is unchanged.
func CampView(leaderUserID int) (bool, error) {
	providerMu.RLock()
	p := viewProvider
	providerMu.RUnlock()
	if p == nil {
		return false, nil
	}
	return p.RenderCampView(leaderUserID)
}

// MovementBlocked reports whether an active rest must refuse ordinary
// movement, and the refusal text to show. Without a provider, movement is
// unchanged.
func MovementBlocked(leaderUserID int) (bool, string) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	if p == nil {
		return false, ""
	}
	return p.MovementBlocked(leaderUserID)
}

// AbandonProvider is implemented by modules/camping (Phase 25a).
// AbandonForDeath removes a dead leader's camp and any inn stay, resting or
// not, and saves that at once. An error means they are still there.
type AbandonProvider interface {
	AbandonForDeath(leaderUserID int) error
}

var abandonProvider AbandonProvider

// SetAbandonProvider registers the active abandon provider. Passing nil
// clears it.
func SetAbandonProvider(p AbandonProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	abandonProvider = p
}

// AbandonForDeath removes a dead leader's camp and inn stay. Without a
// provider there is nothing to remove.
func AbandonForDeath(leaderUserID int) error {
	providerMu.RLock()
	p := abandonProvider
	providerMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.AbandonForDeath(leaderUserID)
}

// RestActivity is a leader's camp or inn stay as the information surfaces
// show it (Phase 26a).
type RestActivity struct {
	// Inn is true for an inn stay, false for a camp.
	Inn bool
	// Resting is true while a rest runs; a pitched camp that isn't resting
	// is false.
	Resting   bool
	Remaining time.Duration
}

// RestProvider is optionally implemented by the registered movement
// provider (Phase 26a). Both read state only.
type RestProvider interface {
	// LeaderRest reports the leader's camp or inn stay; ok is false with
	// neither.
	LeaderRest(leaderUserID int) (RestActivity, bool)
	// RestTierOf reports a character's rest tier buff (Rested, Well
	// Rested, or TierNone) and its time left; ok is false when it can't
	// tell.
	RestTierOf(userID int) (tier Tier, remaining time.Duration, ok bool)
}

func restProvider() (RestProvider, bool) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	rp, ok := p.(RestProvider)
	return rp, ok
}

// RestReporting reports whether a registered provider can report camps and
// inn stays (Phase 26a), so "none" can be told from "can't tell".
func RestReporting() bool {
	_, ok := restProvider()
	return ok
}

// LeaderRest reports a leader's camp or inn stay. ok is false without a
// provider or either.
func LeaderRest(leaderUserID int) (RestActivity, bool) {
	rp, ok := restProvider()
	if !ok {
		return RestActivity{}, false
	}
	return rp.LeaderRest(leaderUserID)
}

// RestTierOf reports a character's rest tier. ok is false without a
// provider or a tier.
func RestTierOf(userID int) (Tier, time.Duration, bool) {
	rp, ok := restProvider()
	if !ok {
		return TierNone, 0, false
	}
	return rp.RestTierOf(userID)
}

// CampAbandoner is implemented by modules/camping (Phase 27b). AbandonCamp
// removes a leader's camp, resting or not, and saves at once; a finished
// rest keeps its recovery and any rest tier owed. Inn stays are untouched.
// An error means the camp is still there.
type CampAbandoner interface {
	AbandonCamp(leaderUserID int) error
}

var campAbandoner CampAbandoner

// SetCampAbandoner registers the active camp abandoner. Passing nil clears
// it.
func SetCampAbandoner(p CampAbandoner) {
	providerMu.Lock()
	defer providerMu.Unlock()
	campAbandoner = p
}

// AbandonCamp removes a leader's camp. Without a provider there is nothing
// to remove.
func AbandonCamp(leaderUserID int) error {
	providerMu.RLock()
	p := campAbandoner
	providerMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.AbandonCamp(leaderUserID)
}

// CampState is a leader's camp as the web client's Camp tab shows it
// (Phase 32g), seen from the room the leader stands in.
type CampState struct {
	// HasCamp is true when the leader has a camp; Here when it is in this
	// room. RoomTitle is the camp's room.
	HasCamp, Here bool
	RoomTitle     string
	// RoomID is the camp's room (Phase 40b), so the map can mark its tile.
	RoomID  int
	FireLit bool
	Resting bool
	// Rested is a camp whose last rest is done. Its fire has burned to
	// embers (Embers, Phase 40a3): feed it (camp fire) to rest again.
	Rested bool
	Embers bool
	// Tent is an oiled canvas tent pitched at the camp (Phase 40a3).
	Tent bool
	// TentKind and TentNote (Phase 52) name the pitched tent and what it
	// does; Tents are the tents carried, for the Camp tab's picker.
	TentKind TentKind
	TentNote string
	Tents    []TentChoice
	// Gear is the camp gear the company carries (Phase 40a4), one short
	// label each, for the web Camp tab.
	Gear []string
	// Supplies are the camp supplies the company carries (Phase 43a), one
	// label each ("Broth x2"); Prepared what is queued for the next rest
	// ("Broth for Tamsin", "Watch incense").
	Supplies []string
	Prepared []string
	// TheftRisk is set when thieves work the camp's road and the company
	// carries no camp bells (40a4 review).
	TheftRisk bool
	// RestPercent and RestSeconds are a running rest's progress and time
	// left.
	RestPercent, RestSeconds int
	// CanCamp is true when the leader has no camp and this room allows
	// one; Inn when this room has an inn.
	CanCamp, Inn bool
	// Duties (Phase 51) are the camp rest duties of the members at the
	// camp, for the Camp tab's picker; Locked is set while a rest runs
	// (its duties are fixed then).
	Duties       []DutyRow
	DutiesLocked bool
	// Recipes (Phase 56) are the dishes the leader has learned, one line
	// each ("Hunter's stew: 2 raw game meat, 1 wild thyme (cooking 3)").
	Recipes []string
	// Music (camp music) is the Music block of the Camp tab: who plays
	// what and what the next song gives. Gig is the inn's gig board, set
	// only in a room with an inn.
	Music MusicState
	Gig   *GigNotice
}

// MusicState is a company's Music at the camp, for the Camp tab.
type MusicState struct {
	// Known is false when the leader is offline and nothing can be read.
	Known bool
	// Off is the leader's switch: the camp song is silenced.
	Off bool
	// Players is each member at the camp, with their skill if any.
	Players []MusicRow
	// Covered is the families that play ("3 of 4") and Effects the lines
	// the next camp song would give.
	Covered string
	Effects []string
	// Cost is the song's added raid and thief chance in words ("" for none).
	Cost string
	// Teacher is true in a room with a music teacher; TeachPrice is its fee.
	Teacher    bool
	TeachPrice int
}

// MusicRow is one member's Music for the Camp tab.
type MusicRow struct {
	Key, Name string
	// Family and Level are empty and 0 for a member with no music; Label
	// is the skill in words ("Strings 2 (3 songs to level 3)"); Instrument
	// is the instrument they would play.
	Family     string
	Level      int
	Label      string
	Instrument string
}

// GigNotice is an inn's gig board for a company.
type GigNotice struct {
	// Window is the evening window ("19:00 to 21:00"); Open whether the
	// world clock is inside it now.
	Window string
	Open   bool
	// Ready is whether the company can start a gig now; Reason why not.
	Ready  bool
	Reason string
	// Families is how many families the company fields; Pay what a gig
	// would earn now.
	Families int
	Pay      int
}

// MusicProvider is optionally implemented by the registered movement
// provider: a member's Music for the Company panel, "" with none.
type MusicProvider interface {
	MusicLabelOf(leaderUserID int, memberKey string) string
}

// MusicLabelOf reports a member's Music in words. "" without a provider or
// any music.
func MusicLabelOf(leaderUserID int, memberKey string) string {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	if mp, ok := p.(MusicProvider); ok {
		return mp.MusicLabelOf(leaderUserID, memberKey)
	}
	return ""
}

// DutyRow is one member's rest duty for the Camp tab's picker.
type DutyRow struct {
	// Key is the survival member key; Name the display name; Command the
	// word "camp duties" takes for them ("me" for the leader).
	Key, Name, Command string
	// Duty is their duty now; Options the duties they can take.
	Duty    string
	Options []string
}

// CampStateProvider is optionally implemented by the registered movement
// provider (Phase 32g). It reads state only.
type CampStateProvider interface {
	CampStateOf(leaderUserID, roomID int, roomTags []string) (CampState, bool)
}

// CampStateOf reports a leader's camp from a room with those tags. ok is
// false without a provider.
func CampStateOf(leaderUserID, roomID int, roomTags []string) (CampState, bool) {
	providerMu.RLock()
	p := movementProvider
	providerMu.RUnlock()
	cp, ok := p.(CampStateProvider)
	if !ok {
		return CampState{}, false
	}
	return cp.CampStateOf(leaderUserID, roomID, roomTags)
}

// TentChoice is one carried tent on the Camp tab: its kind, name, effect and
// whether it is the one pitched.
type TentChoice struct {
	Kind    TentKind
	Name    string
	Effect  string
	Pitched bool
}

// RestEndListener hears a camp rest that finished with its rewards given
// (Phase 60: a camp trigger for story events). It runs on the game loop,
// outside the camping module's lock.
type RestEndListener func(leaderUserID, roomID int)

var restEndListeners []RestEndListener

// AddRestEndListener registers a listener for finished camp rests.
func AddRestEndListener(fn RestEndListener) {
	providerMu.Lock()
	defer providerMu.Unlock()
	restEndListeners = append(restEndListeners, fn)
}

// RestEnded reports a finished camp rest to the listeners.
func RestEnded(leaderUserID, roomID int) {
	providerMu.RLock()
	fns := append([]RestEndListener(nil), restEndListeners...)
	providerMu.RUnlock()
	for _, fn := range fns {
		fn(leaderUserID, roomID)
	}
}
