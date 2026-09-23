// Package expedition defines the durable, GoMud-free travel domain: terrain
// travel profiles, the travel-session state machine, real-time UTC progress and
// checkpoint math, and the provider seams native commands use to start and view
// travel.
//
// The package imports internal/survival only for the Exertion value type. It
// imports no rooms, users, commands, plugins, timers, or clocks, and it never
// advances GoMud's global time or round count.
package expedition

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/survival"
)

// CheckpointCount is the fixed number of durable progress checkpoints on a
// route. Survival exertion is accrued at these boundaries, never up front.
const CheckpointCount = 10

// Errors returned by the expedition domain and provider seams.
var (
	ErrInvalidProfileName     = errors.New("expedition: profile name is required")
	ErrInvalidProfileDuration = errors.New("expedition: profile duration must be positive")
	ErrInvalidProfileExertion = errors.New("expedition: profile exertion must be non-negative")
	ErrInvalidInterruption    = errors.New("expedition: invalid travel interruption")
	ErrInvalidSession         = errors.New("expedition: invalid travel session")
	ErrInvalidTransition      = errors.New("expedition: invalid session transition")
)

// InterruptionKind names a scripted, durable event that halts a route until the
// leader chooses to resume or turn back.
type InterruptionKind string

// Interruption kinds the domain ships. Adding a kind means extending Valid
// and InterruptionText; persisted payloads of unknown kinds are corrupt.
const (
	FallenTree InterruptionKind = "fallen-tree"
	Discovery  InterruptionKind = "discovery"
	Tracks     InterruptionKind = "tracks"
)

// Valid reports whether the kind is a known interruption.
func (k InterruptionKind) Valid() bool {
	switch k {
	case FallenTree, Discovery, Tracks:
		return true
	default:
		return false
	}
}

// InterruptionText returns the player-facing description for a fired
// interruption's kind, naming the route it fired on. Every valid kind has an
// entry; an unknown kind (already rejected everywhere a session is
// constructed) falls back to a generic message rather than an empty string.
func InterruptionText(kind InterruptionKind, profileName string) string {
	switch kind {
	case FallenTree:
		return fmt.Sprintf("A fallen tree blocks the %s route. Your company pauses before the obstruction.\nUse \"travel resume\" when you are ready to continue, or \"travel return\" to head back.", profileName)
	case Discovery:
		return fmt.Sprintf("Something catches your eye off the %s route. Your company pauses to take a closer look.\nUse \"travel resume\" when you are ready to continue, or \"travel return\" to head back.", profileName)
	case Tracks:
		return fmt.Sprintf("Fresh tracks cross the %s route ahead. Your company pauses to consider them.\nUse \"travel resume\" when you are ready to continue, or \"travel return\" to head back.", profileName)
	default:
		return fmt.Sprintf("Something gives your company pause on the %s route.\nUse \"travel resume\" when you are ready to continue, or \"travel return\" to head back.", profileName)
	}
}

// WeightedInterruptionKind is one entry in a weighted interruption roll:
// Kind fires with probability proportional to Weight among its table.
type WeightedInterruptionKind struct {
	Kind   InterruptionKind `yaml:"kind"`
	Weight uint             `yaml:"weight"`
}

// InterruptionProfile configures the checkpoint on a route where an
// interruption fires, and which kind fires there: either a single Kind
// (today's form) or a weighted Kinds table (12b) rolled once at fire time.
// Exactly one of Kind or Kinds must be set. Checkpoint is 1..CheckpointCount-1:
// the route end is not a legal interruption point, because a route that is
// already over cannot be interrupted.
type InterruptionProfile struct {
	Kind       InterruptionKind           `yaml:"kind,omitempty"`
	Kinds      []WeightedInterruptionKind `yaml:"kinds,omitempty"`
	Checkpoint uint8                      `yaml:"checkpoint"`
}

// Validate rejects an unusable interruption configuration.
func (i InterruptionProfile) Validate() error {
	switch {
	case i.Kind != "" && len(i.Kinds) > 0:
		return ErrInvalidInterruption
	case len(i.Kinds) > 0:
		for _, wk := range i.Kinds {
			if !wk.Kind.Valid() || wk.Weight == 0 {
				return ErrInvalidInterruption
			}
		}
	default:
		if !i.Kind.Valid() {
			return ErrInvalidInterruption
		}
	}
	if i.Checkpoint == 0 || i.Checkpoint >= CheckpointCount {
		return ErrInvalidInterruption
	}
	return nil
}

// ResolveKind picks which kind fires for this interruption. A singular-Kind
// profile returns Kind unchanged and ignores roll entirely — today's exact,
// deterministic behavior. A Kinds table selects proportionally to Weight:
// roll is reduced modulo the total weight, so any caller-supplied uint64
// source works; ResolveKind itself stays pure and never generates randomness.
// The profile must already be Validate-clean, or ResolveKind returns the
// same error Validate would.
func (i InterruptionProfile) ResolveKind(roll uint64) (InterruptionKind, error) {
	if err := i.Validate(); err != nil {
		return "", err
	}
	if len(i.Kinds) == 0 {
		return i.Kind, nil
	}
	var totalWeight uint64
	for _, wk := range i.Kinds {
		totalWeight += uint64(wk.Weight)
	}
	target := roll % totalWeight
	var cumulative uint64
	for _, wk := range i.Kinds {
		cumulative += uint64(wk.Weight)
		if target < cumulative {
			return wk.Kind, nil
		}
	}
	return i.Kinds[len(i.Kinds)-1].Kind, nil
}

// TravelInterruption is the immutable payload a session records when it
// interrupts, so a restart or copyover can render the same choice.
type TravelInterruption struct {
	Kind       InterruptionKind `yaml:"kind"`
	Checkpoint uint8            `yaml:"checkpoint"`
}

// Validate rejects a payload that could not have come from Interrupt. Unlike
// InterruptionProfile, a fired TravelInterruption always carries the single
// already-resolved Kind (never a weighted Kinds table — ResolveKind runs
// before Interrupt persists this payload).
func (i TravelInterruption) Validate() error {
	if !i.Kind.Valid() {
		return ErrInvalidInterruption
	}
	if i.Checkpoint == 0 || i.Checkpoint >= CheckpointCount {
		return ErrInvalidInterruption
	}
	return nil
}

// TravelProfile is a data-driven terrain route definition. Duration is real
// UTC time; Exertion is the total cost charged proportionally over the route.
type TravelProfile struct {
	Name         string
	Duration     time.Duration
	Exertion     survival.Exertion
	Interruption *InterruptionProfile `yaml:"interruption,omitempty"`
}

// Validate rejects a malformed profile rather than applying a guess.
func (p TravelProfile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrInvalidProfileName
	}
	if p.Duration <= 0 {
		return ErrInvalidProfileDuration
	}
	if p.Exertion.Hunger < 0 || p.Exertion.Thirst < 0 || p.Exertion.Fatigue < 0 {
		return ErrInvalidProfileExertion
	}
	if p.Interruption != nil {
		if err := p.Interruption.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SessionState is the lifecycle state of a travel session.
type SessionState uint8

const (
	Traveling SessionState = iota
	// Interrupted is reserved for Phase 6; Phase 5 never transitions a live
	// session into it.
	Interrupted
	Completed
	Cancelled
)

func (s SessionState) String() string {
	switch s {
	case Traveling:
		return "traveling"
	case Interrupted:
		return "interrupted"
	case Completed:
		return "completed"
	case Cancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// Valid reports whether the state is a known lifecycle value.
func (s SessionState) Valid() bool {
	return s <= Cancelled
}

// CanTransitionTo reports whether the generic transition is legal. Completed
// and Cancelled are terminal, and Interrupted is not reachable through it:
// entering or leaving Interrupted must go through Interrupt, Resume, or Return,
// which maintain the payload, pause-instant, and one-shot-marker invariants.
func (s SessionState) CanTransitionTo(to SessionState) bool {
	switch s {
	case Traveling:
		return to == Completed || to == Cancelled
	default:
		return false
	}
}

// TravelSession is the durable, leader-owned record of a real-time journey. It
// never stores sockets or live mob instances; progress is always derived from
// StartedAtUTC against the current clock.
type TravelSession struct {
	LeaderUserID           int                 `yaml:"leader_user_id"`
	OriginRoomID           int                 `yaml:"origin_room_id"`
	DestinationRoomID      int                 `yaml:"destination_room_id"`
	ExitName               string              `yaml:"exit_name"`
	ProfileName            string              `yaml:"profile_name"`
	StartedAtUTC           time.Time           `yaml:"started_at_utc"`
	PausedAtUTC            time.Time           `yaml:"paused_at_utc,omitempty"`
	PausedDuration         time.Duration       `yaml:"paused_duration,omitempty"`
	LastExertionCheckpoint uint8               `yaml:"last_exertion_checkpoint"`
	PendingExertion        *PendingExertion    `yaml:"pending_exertion,omitempty"`
	Interruption           *TravelInterruption `yaml:"interruption,omitempty"`
	InterruptionTriggered  bool                `yaml:"interruption_triggered,omitempty"`
	State                  SessionState        `yaml:"state"`
}

// PendingExertion is the durable two-phase record of a survival charge.
type PendingExertion struct {
	OperationID string            `yaml:"operation_id"`
	Checkpoint  uint8             `yaml:"checkpoint"`
	Cost        survival.Exertion `yaml:"cost"`
}

// Validate reports whether the session carries stable, complete identity.
func (s TravelSession) Validate() error {
	if s.LeaderUserID <= 0 || s.OriginRoomID <= 0 || s.DestinationRoomID <= 0 {
		return ErrInvalidSession
	}
	if s.OriginRoomID == s.DestinationRoomID {
		return ErrInvalidSession
	}
	if strings.TrimSpace(s.ExitName) == "" || strings.TrimSpace(s.ProfileName) == "" {
		return ErrInvalidSession
	}
	if s.StartedAtUTC.IsZero() {
		return ErrInvalidSession
	}
	if s.LastExertionCheckpoint > CheckpointCount {
		return ErrInvalidSession
	}
	if p := s.PendingExertion; p != nil && (p.OperationID == "" || p.Checkpoint == 0 || p.Checkpoint > CheckpointCount || p.Cost.Hunger < 0 || p.Cost.Thirst < 0 || p.Cost.Fatigue < 0) {
		return ErrInvalidSession
	}
	if !s.State.Valid() {
		return ErrInvalidSession
	}
	if s.PausedDuration < 0 {
		return ErrInvalidSession
	}
	if i := s.Interruption; i != nil {
		if err := i.Validate(); err != nil {
			return ErrInvalidSession
		}
	}
	if s.State == Interrupted {
		// A halted route must know the immutable event and the instant it
		// halted at, or restart/copyover cannot charge active time correctly. It
		// must also carry the one-shot marker: Interrupt is the only writer of
		// this state and always sets it, so a false marker is corrupt data.
		if s.Interruption == nil || s.PausedAtUTC.IsZero() || !s.InterruptionTriggered {
			return ErrInvalidSession
		}
		return nil
	}
	// Any other state is not paused: a payload or pause instant is corrupt data
	// (Resume clears both and leaves InterruptionTriggered set).
	if s.Interruption != nil || !s.PausedAtUTC.IsZero() {
		return ErrInvalidSession
	}
	return nil
}

// ValidateForProfile validates a persisted session and, when it is paused,
// verifies that its durable interruption is the one configured for the route.
// A well-formed payload from a different profile is still unusable recovery
// data: accepting it could present or resolve the wrong obstruction.
func (s TravelSession) ValidateForProfile(profile TravelProfile) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := profile.Validate(); err != nil {
		return err
	}
	if s.State == Interrupted {
		if profile.Interruption == nil ||
			s.Interruption.Kind != profile.Interruption.Kind ||
			s.Interruption.Checkpoint != profile.Interruption.Checkpoint {
			return ErrInvalidSession
		}
	}
	return nil
}

// PrepareExertion returns the persisted operation required for the next earned
// checkpoint. Repeating it is safe: its ID derives only from session identity.
func (s TravelSession) PrepareExertion(now time.Time, profile TravelProfile) (PendingExertion, bool, error) {
	if err := profile.Validate(); err != nil {
		return PendingExertion{}, false, err
	}
	if s.PendingExertion != nil {
		return *s.PendingExertion, true, nil
	}
	checkpoint := s.CheckpointAt(now, profile.Duration)
	if checkpoint <= s.LastExertionCheckpoint {
		return PendingExertion{}, false, nil
	}
	cost := s.ExertionDue(now, profile)
	if cost.Hunger == 0 && cost.Thirst == 0 && cost.Fatigue == 0 {
		return PendingExertion{}, false, nil
	}
	return PendingExertion{
		OperationID: fmt.Sprintf("expedition/%d/%d/%d/%s/%d/%d", s.LeaderUserID, s.OriginRoomID, s.DestinationRoomID, s.ProfileName, s.StartedAtUTC.UTC().UnixNano(), checkpoint),
		Checkpoint:  checkpoint,
		Cost:        cost,
	}, true, nil
}

// ProgressAt returns the clamped fraction of the route completed, in 0..1.
func (s TravelSession) ProgressAt(now time.Time, duration time.Duration) float64 {
	if duration <= 0 {
		return 1
	}
	return float64(s.ActiveElapsedAt(now, duration)) / float64(duration)
}

// ActiveElapsedAt returns the clamped active travel time at now, in
// 0..duration. Active time is wall time since StartedAtUTC minus every paused
// stretch: durable PausedDuration for completed pauses, plus the open pause
// when the session is currently Interrupted. Interrupting a route therefore
// freezes progress until the leader resumes it, and wall-clock time spent
// paused is never charged against the route.
func (s TravelSession) ActiveElapsedAt(now time.Time, duration time.Duration) time.Duration {
	elapsed := now.Sub(s.StartedAtUTC) - s.PausedDuration
	if s.State == Interrupted && !s.PausedAtUTC.IsZero() {
		elapsed -= now.Sub(s.PausedAtUTC)
	}
	if elapsed < 0 || duration <= 0 {
		return 0
	}
	if elapsed > duration {
		return duration
	}
	return elapsed
}

// RemainingAt returns the clamped active travel time still owed at now, in
// 0..duration.
func (s TravelSession) RemainingAt(now time.Time, duration time.Duration) time.Duration {
	if duration <= 0 {
		return 0
	}
	return duration - s.ActiveElapsedAt(now, duration)
}

// CheckpointAt returns the durable checkpoint attained at now, in 0..10.
func (s TravelSession) CheckpointAt(now time.Time, duration time.Duration) uint8 {
	if duration <= 0 {
		return CheckpointCount
	}
	elapsed := s.ActiveElapsedAt(now, duration)
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= duration {
		return CheckpointCount
	}
	attained := int(int64(elapsed) * CheckpointCount / int64(duration))
	if attained < 0 {
		attained = 0
	}
	if attained > CheckpointCount {
		attained = CheckpointCount
	}
	return uint8(attained)
}

// ExertionDue returns the incremental proportional cost owed between the last
// persisted checkpoint and the checkpoint attained at now. Cumulative rounding
// telescopes so a completed route charges exactly the profile total once.
func (s TravelSession) ExertionDue(now time.Time, profile TravelProfile) survival.Exertion {
	attained := s.CheckpointAt(now, profile.Duration)
	if attained <= s.LastExertionCheckpoint {
		return survival.Exertion{}
	}
	to := cumulativeExertion(profile.Exertion, attained)
	from := cumulativeExertion(profile.Exertion, s.LastExertionCheckpoint)
	return survival.Exertion{
		Hunger:  to.Hunger - from.Hunger,
		Thirst:  to.Thirst - from.Thirst,
		Fatigue: to.Fatigue - from.Fatigue,
	}
}

// cumulativeExertion rounds the total cost up to a checkpoint deterministically.
// At CheckpointCount it equals total exactly.
func cumulativeExertion(total survival.Exertion, checkpoint uint8) survival.Exertion {
	return survival.Exertion{
		Hunger:  scaledCost(total.Hunger, checkpoint),
		Thirst:  scaledCost(total.Thirst, checkpoint),
		Fatigue: scaledCost(total.Fatigue, checkpoint),
	}
}

func scaledCost(total int, checkpoint uint8) int {
	if total <= 0 || checkpoint == 0 {
		return 0
	}
	return (total*int(checkpoint) + CheckpointCount/2) / CheckpointCount
}

// InterruptionDue reports whether the profile's configured interruption has
// come due and has not fired on this session yet. It is derived state: nothing
// is persisted until Interrupt succeeds.
func (s TravelSession) InterruptionDue(now time.Time, p TravelProfile) bool {
	return s.State == Traveling &&
		!s.InterruptionTriggered &&
		p.Interruption != nil &&
		s.CheckpointAt(now, p.Duration) >= p.Interruption.Checkpoint
}

// Interrupt halts a route whose configured interruption has come due, returning
// the interrupted copy. It records the immutable payload and the pause instant,
// and marks the interruption triggered so it fires at most once per session.
// The interruption itself changes no exertion state.
func (s TravelSession) Interrupt(now time.Time, p TravelProfile) (TravelSession, error) {
	if err := s.Validate(); err != nil {
		return s, err
	}
	if err := p.Validate(); err != nil {
		return s, err
	}
	if !s.InterruptionDue(now, p) {
		return s, ErrInvalidInterruption
	}
	s.State = Interrupted
	s.PausedAtUTC = now
	s.Interruption = &TravelInterruption{Kind: p.Interruption.Kind, Checkpoint: p.Interruption.Checkpoint}
	s.InterruptionTriggered = true
	return s, nil
}

// Resume returns an interrupted session to travel, banking the open pause into
// PausedDuration and clearing only the active payload and pause instant. The
// one-shot trigger stays set, so the same interruption never fires again.
func (s TravelSession) Resume(now time.Time) (TravelSession, error) {
	if s.State != Interrupted {
		return s, ErrInvalidTransition
	}
	if err := s.Validate(); err != nil {
		return s, err
	}
	pause := now.Sub(s.PausedAtUTC)
	if pause > 0 {
		s.PausedDuration += pause
	}
	s.State = Traveling
	s.PausedAtUTC = time.Time{}
	s.Interruption = nil
	return s, nil
}

// ResumeForProfile is the profile-aware recovery/resolution form of Resume.
// It rejects a paused record whose interruption does not belong to profile.
func (s TravelSession) ResumeForProfile(now time.Time, profile TravelProfile) (TravelSession, error) {
	if s.State != Interrupted {
		return s, ErrInvalidTransition
	}
	if err := s.ValidateForProfile(profile); err != nil {
		return s, err
	}
	return s.Resume(now)
}

// Return abandons an interrupted route. Only a halted journey can be given up:
// a route that is still running must be cancelled through the ordinary
// transition, and a finished one is already over. It has no clock, so the open
// pause is dropped rather than banked: the record is terminal, and its active
// time no longer drives progress. The one-shot trigger is retained.
func (s TravelSession) Return() (TravelSession, error) {
	if s.State != Interrupted {
		return s, ErrInvalidTransition
	}
	if err := s.Validate(); err != nil {
		return s, err
	}
	s.State = Cancelled
	s.PausedAtUTC = time.Time{}
	s.Interruption = nil
	return s, nil
}

// ReturnForProfile is the profile-aware recovery/resolution form of Return.
func (s TravelSession) ReturnForProfile(profile TravelProfile) (TravelSession, error) {
	if s.State != Interrupted {
		return s, ErrInvalidTransition
	}
	if err := s.ValidateForProfile(profile); err != nil {
		return s, err
	}
	return s.Return()
}

// Transition returns the session in a new state, or ErrInvalidTransition.
func (s TravelSession) Transition(to SessionState) (TravelSession, error) {
	if !s.State.CanTransitionTo(to) {
		return s, ErrInvalidTransition
	}
	s.State = to
	return s, nil
}

// StartRequest describes a profiled exit the native movement layer wants to
// turn into a journey.
type StartRequest struct {
	LeaderUserID      int
	OriginRoomID      int
	DestinationRoomID int
	ExitName          string
	ProfileName       string
}

// StartProvider is implemented by modules/expedition. StartTravel returns
// handled=false when the request is not a travel-enabled exit, and handled=true
// with a nil error once a durable session exists. A handled error leaves the
// caller in place.
type StartProvider interface {
	StartTravel(req StartRequest) (handled bool, err error)
}

// ViewProvider is implemented by modules/expedition. RenderTravelView returns
// handled=true when it rendered an active session view.
type ViewProvider interface {
	RenderTravelView(leaderUserID int) (handled bool, err error)
}

// MovementProvider is implemented by modules/expedition. MovementBlocked
// reports an active journey and the refusal text for ordinary movement.
type MovementProvider interface {
	MovementBlocked(leaderUserID int) (blocked bool, message string)
}

var (
	providerMu       sync.RWMutex
	startProvider    StartProvider
	viewProvider     ViewProvider
	movementProvider MovementProvider
)

// SetStartProvider registers the active start provider. Passing nil clears it.
func SetStartProvider(p StartProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	startProvider = p
}

// SetViewProvider registers the active view provider. Passing nil clears it.
func SetViewProvider(p ViewProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	viewProvider = p
}

// SetMovementProvider registers the active movement-block provider. Passing nil
// clears it.
func SetMovementProvider(p MovementProvider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	movementProvider = p
}

// Start consults the registered start provider. Without one, the exit is simply
// not travel-enabled.
func Start(req StartRequest) (bool, error) {
	providerMu.RLock()
	p := startProvider
	providerMu.RUnlock()
	if p == nil {
		return false, nil
	}
	return p.StartTravel(req)
}

// TravelView consults the registered view provider. Without one, native look
// behavior is unchanged.
func TravelView(leaderUserID int) (bool, error) {
	providerMu.RLock()
	p := viewProvider
	providerMu.RUnlock()
	if p == nil {
		return false, nil
	}
	return p.RenderTravelView(leaderUserID)
}

// MovementBlocked reports whether an active journey must refuse ordinary
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
