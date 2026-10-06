package archetype

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 38b: a player's class and talents. The registry holds the durable
// choices; everything they give is derived from them and the player's level
// (internal/classes), so nothing else is saved.

// ClassRecord is one player's promoted class and picked talents.
type ClassRecord struct {
	Class   string   `yaml:"class,omitempty"`
	Talents []string `yaml:"talents,omitempty"`
}

func (c ClassRecord) clone() ClassRecord {
	c.Talents = slices.Clone(c.Talents)
	return c
}

var (
	_ classes.PlayerProvider = (*ArchetypeModule)(nil)
	_ classes.PlayerWriter   = (*ArchetypeModule)(nil)
)

// PlayerClass implements classes.PlayerProvider.
func (m *ArchetypeModule) PlayerClass(userID int) classes.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.registry.Classes[userID]
	return classes.State{Class: rec.Class, Talents: slices.Clone(rec.Talents)}
}

// ErrNotPromotable is returned for a player with no known archetype.
var ErrNotPromotable = errors.New("choose an archetype first")

// PromotePlayer implements classes.PlayerWriter: it records the class and
// saves, rolling back if the save fails. The class's own rules were checked
// by the caller; this only refuses a player with no archetype, a class from
// another lineage, and a second promotion to the same or a lower tier.
func (m *ArchetypeModule) PromotePlayer(userID int, class string) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	class = strings.ToLower(strings.TrimSpace(class))
	target, ok := classes.Get(class)
	if !ok {
		return classes.ErrNoSuchClass
	}
	m.mu.Lock()
	lineage, chosen := m.registry.Players[userID]
	if !chosen {
		m.mu.Unlock()
		return ErrNotPromotable
	}
	if lineage != target.Lineage {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s", classes.ErrNotAvailable, target.Name)
	}
	prev, had := m.registry.Classes[userID]
	if had {
		if cur, ok := classes.Get(prev.Class); ok && cur.Tier >= target.Tier {
			m.mu.Unlock()
			return nil // already there: a repeated confirmation changes nothing
		}
	}
	rec := prev.clone()
	rec.Class = class
	m.registry.Classes[userID] = rec
	if err := m.saveLocked(); err != nil {
		if had {
			m.registry.Classes[userID] = prev
		} else {
			delete(m.registry.Classes, userID)
		}
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	m.afterClassChange(userID)
	return nil
}

// PickPlayerTalent implements classes.PlayerWriter.
func (m *ArchetypeModule) PickPlayerTalent(userID, level int, talent string) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	talent = strings.ToLower(strings.TrimSpace(talent))
	m.mu.Lock()
	lineage, chosen := m.registry.Players[userID]
	if !chosen {
		m.mu.Unlock()
		return ErrNotPromotable
	}
	prev, had := m.registry.Classes[userID]
	if err := classes.CanPick(lineage, prev.Class, prev.Talents, level, talent); err != nil {
		m.mu.Unlock()
		return err
	}
	rec := prev.clone()
	rec.Talents = append(rec.Talents, talent)
	m.registry.Classes[userID] = rec
	if err := m.saveLocked(); err != nil {
		if had {
			m.registry.Classes[userID] = prev
		} else {
			delete(m.registry.Classes, userID)
		}
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	m.afterClassChange(userID)
	return nil
}

// afterClassChange settles a player's character after the class state moved:
// the spells the class teaches, and the maxima a class percent changes (a
// maximum that falls clamps, one that rises never refills).
func (m *ArchetypeModule) afterClassChange(userID int) {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil {
		return
	}
	user.Character.SetUserId(userID)
	m.grantClassSpells(user)
	user.Character.RecalculateStats()
	user.Character.Health = min(user.Character.Health, user.Character.HealthLimit())
	user.Character.Mana = min(user.Character.Mana, user.Character.ManaMax.Value)
	// Phase 38c1: the web client's character window shows the class.
	events.AddToQueue(events.CharacterTrained{UserId: userID})
}

// grantClassSpells teaches a player the spells their class's ranks have
// reached. Owning the spell is the marker, so it is idempotent and a death
// that costs a level never takes one back.
func (m *ArchetypeModule) grantClassSpells(user *users.UserRecord) []string {
	state := m.PlayerClass(user.UserId)
	if state.Class == "" {
		return nil
	}
	var learned []string
	for _, id := range classes.SpellsAt(state.Class, user.Character.Level) {
		if user.Character.LearnSpell(id) {
			learned = append(learned, id)
		}
	}
	return learned
}
