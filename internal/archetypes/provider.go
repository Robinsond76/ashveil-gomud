package archetypes

import "sync"

// Provider is implemented by modules/archetype. It is the read-only seam
// the engine (skill training, scripted spell learning, picklock) and other
// Ashveil modules consult, the same shape as expedition.MovementProvider.
// Every call must be safe on the game loop and must not call back into the
// engine while holding the module's lock.
type Provider interface {
	// CanTrain reports whether the user may train a skill; reason is
	// player-facing when refused.
	CanTrain(userID int, skillID string) (ok bool, reason string)
	// CanLearnSpell reports whether the user may learn a spell.
	CanLearnSpell(userID int, spellID string) (ok bool, reason string)
	// Exists reports whether an archetype id is configured.
	Exists(archetypeID string) bool
	// ArchetypeName is the display name of a configured archetype.
	ArchetypeName(archetypeID string) (string, bool)
	// PlayerArchetype is the user's chosen archetype id, if any.
	PlayerArchetype(userID int) (string, bool)
}

// TrapProvider is optionally implemented by the provider (Phase 17b). A
// trapped lock stays armed unless it reports otherwise.
type TrapProvider interface {
	TrapArmed(lockID string) bool
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the active provider. nil clears it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

func current() Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return provider
}

// CanTrain allows everything without a provider (upstream behaviour).
func CanTrain(userID int, skillID string) (bool, string) {
	if p := current(); p != nil {
		return p.CanTrain(userID, skillID)
	}
	return true, ""
}

// CanLearnSpell allows everything without a provider.
func CanLearnSpell(userID int, spellID string) (bool, string) {
	if p := current(); p != nil {
		return p.CanLearnSpell(userID, spellID)
	}
	return true, ""
}

// Exists is false without a provider.
func Exists(archetypeID string) bool {
	if p := current(); p != nil {
		return p.Exists(archetypeID)
	}
	return false
}

// Name returns an archetype's display name; ok is false without a
// provider or for an unknown id.
func Name(archetypeID string) (string, bool) {
	if p := current(); p != nil {
		return p.ArchetypeName(archetypeID)
	}
	return "", false
}

// PlayerArchetype returns the user's chosen archetype; ok is false when
// unchosen or without a provider.
func PlayerArchetype(userID int) (string, bool) {
	if p := current(); p != nil {
		return p.PlayerArchetype(userID)
	}
	return "", false
}

// TrapArmed reports whether a lock's trap is armed. Without a provider, or
// one that doesn't track disarms, every trap stays armed.
func TrapArmed(lockID string) bool {
	if tp, ok := current().(TrapProvider); ok {
		return tp.TrapArmed(lockID)
	}
	return true
}

// ClaimantNamer is optionally implemented by the provider: the display
// names of the archetypes that claim a skill ("Cleric or Wizard").
type ClaimantNamer interface {
	SkillClaimantNames(skillID string) string
}

// ClaimantNames is "" without a provider, or for an unclaimed skill.
func ClaimantNames(skillID string) string {
	if cn, ok := current().(ClaimantNamer); ok {
		return cn.SkillClaimantNames(skillID)
	}
	return ""
}

// TrapSenser is optionally implemented by the provider (Phase 17b): a free
// trap sense when a player starts picking a trapped lock.
type TrapSenser interface {
	SenseBeforePick(userID, roomID int, lockID string)
}

// SenseBeforePick does nothing without a provider that senses traps.
func SenseBeforePick(userID, roomID int, lockID string) {
	if ts, ok := current().(TrapSenser); ok {
		ts.SenseBeforePick(userID, roomID, lockID)
	}
}

// Choice is one archetype as offered during character creation (Phase 22a).
type Choice struct {
	ID          string
	Name        string
	Description string
	Skills      []string
	// Kit is the display names of the starter kit's items.
	Kit []string
}

// Creator is optionally implemented by the provider (Phase 22a): the
// archetype step of character creation. ChooseAtCreation commits the
// choice and grants its kit on the game loop; text is player-facing either
// way.
type Creator interface {
	CreationChoices() []Choice
	ChooseAtCreation(userID int, archetypeID string) (text string, ok bool)
}

// CreationChoices is the archetypes to offer a character during creation.
// It is nil without a provider that creates, or once the user has a
// configured archetype, so creation skips the step. A stored archetype
// that is no longer configured doesn't count, matching the module's
// choose rule.
func CreationChoices(userID int) []Choice {
	p := current()
	c, ok := p.(Creator)
	if !ok {
		return nil
	}
	if id, chosen := p.PlayerArchetype(userID); chosen && p.Exists(id) {
		return nil
	}
	return c.CreationChoices()
}

// ChooseAtCreation commits a creation-time choice; ok is false without a
// provider that creates.
func ChooseAtCreation(userID int, archetypeID string) (string, bool) {
	if c, ok := current().(Creator); ok {
		return c.ChooseAtCreation(userID, archetypeID)
	}
	return "", false
}
