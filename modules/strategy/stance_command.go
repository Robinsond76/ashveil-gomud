package strategy

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 69: weapon stances. `stance` shows every member's, `stance [who]`
// one's, and `stance [who] [name]` or `stance [who] off` sets it, outside a
// battle only. A stance is kept even while its member lacks the weapon; it
// does nothing then, and the lines say so.

const stanceUsage = `Set one with <ansi fg="command">stance [who] [name]</ansi> (heavy, wall, quick or keen) or take it off with <ansi fg="command">stance [who] off</ansi>. See <ansi fg="command">help stances</ansi>.`

func (m *StrategyModule) stanceCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.runStance(user, strings.Fields(strings.ToLower(rest))))
	return true, nil
}

func (m *StrategyModule) runStance(user *users.UserRecord, args []string) string {
	members, ok := m.env.members(user)
	if ok {
		keep := map[string]bool{}
		for _, mb := range members {
			keep[mb.key] = true
		}
		m.prune(user.UserId, keep)
	}
	if len(args) == 0 {
		return m.stanceList(user.UserId, members)
	}
	mb, found := resolve(members, args[0])
	if !found {
		// `stance heavy` (or `stance off`) with no one named is the
		// player's own.
		if _, isStance := stance.Parse(strings.Join(args, " ")); (isStance || isStanceOff(strings.Join(args, " "))) && len(members) > 0 {
			mb, found, args = members[0], true, append([]string{"me"}, args...)
		}
	}
	if !found {
		return fmt.Sprintf(`No one in your company answers to "%s". Type <ansi fg="command">stance</ansi> to see them.`, args[0])
	}
	if len(args) == 1 {
		return m.stanceOf(user.UserId, mb)
	}
	if m.env.inBattle(user) {
		return usercommands.BattleUnderWay
	}
	word := strings.Join(args[1:], " ")
	switch {
	case isStanceOff(word):
		if m.StoredStance(user.UserId, mb.key) == stance.None {
			return fmt.Sprintf("%s %s in no stance.", mb.name, verb(mb, "are", "is"))
		}
		if err := m.setStance(user.UserId, mb.key, stance.None); err != nil {
			return err.Error()
		}
		return fmt.Sprintf("%s %s out of any stance.", mb.name, verb(mb, "are", "is"))
	}
	st, ok := stance.Parse(word)
	if !ok {
		return fmt.Sprintf(`"%s" is not a stance. %s`, word, stanceMenu())
	}
	if m.StoredStance(user.UserId, mb.key) == st {
		return fmt.Sprintf("%s %s already in that stance.", mb.name, verb(mb, "are", "is"))
	}
	if err := m.setStance(user.UserId, mb.key, st); err != nil {
		return err.Error()
	}
	d, _ := stance.Lookup(st)
	out := fmt.Sprintf("%s %s now in the %s stance: %s, but %s.", mb.name, verb(mb, "are", "is"), d.Name, d.Gain, d.Cost)
	if note := stanceFitNote(mb, st); note != "" {
		out += "\n" + note
	}
	return out
}

// stanceFitNote says when a member's gear can't use the stance; blank when
// it can, or when the gear can't be read (a companion away).
func stanceFitNote(mb member, st stance.Stance) string {
	if st == stance.None || !mb.gearKnown || stance.Fits(st, mb.gear) {
		return ""
	}
	d, _ := stance.Lookup(st)
	return fmt.Sprintf("It does nothing until %s %s.", verb(mb, "you hold", mb.name+" holds"), d.Needs)
}

func stanceMenu() string {
	var b strings.Builder
	b.WriteString("Stances:")
	for _, d := range stance.Defs {
		fmt.Fprintf(&b, "\n  %s (%s, needs %s): %s, but %s.", d.Key, d.Family, d.Needs, d.Gain, d.Cost)
	}
	return b.String()
}

// stanceLine is a member's stance in one line, with whether it works now.
func (m *StrategyModule) stanceLine(userID int, mb member) string {
	st := m.StoredStance(userID, mb.key)
	if st == stance.None {
		avail := ""
		if mb.gearKnown {
			if list := stance.Available(mb.gear); len(list) > 0 {
				names := make([]string, len(list))
				for i, s := range list {
					names[i] = string(s)
				}
				avail = " (could use: " + strings.Join(names, ", ") + ")"
			}
		}
		return "no stance" + avail
	}
	d, _ := stance.Lookup(st)
	line := d.Name
	switch {
	case !mb.gearKnown:
		line += " (not here to check)"
	case stance.Fits(st, mb.gear):
		line += " (ready)"
	default:
		line += " (idle: needs " + d.Needs + ")"
	}
	return line
}

func (m *StrategyModule) stanceList(userID int, members []member) string {
	var b strings.Builder
	b.WriteString("Weapon stances (set between battles; each trades one strength for another):\n")
	for _, mb := range members {
		fmt.Fprintf(&b, "  %s: %s\n", mb.name, m.stanceLine(userID, mb))
	}
	b.WriteString(stanceUsage)
	return b.String()
}

func (m *StrategyModule) stanceOf(userID int, mb member) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", mb.name, m.stanceLine(userID, mb))
	b.WriteString(stanceMenu() + "\n")
	b.WriteString(stanceUsage)
	return b.String()
}

// StoredStance implements stance.Provider.
func (m *StrategyModule) StoredStance(userID int, key string) stance.Stance {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Stances[userID][key]
}

// setStance stores a member's stance (none clears it) and saves, rolling
// back if the save fails.
func (m *StrategyModule) setStance(userID int, key string, st stance.Stance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailableLocked(); err != nil {
		return err
	}
	before := m.registry.Stances[userID][key]
	m.putStance(userID, key, st)
	if err := m.store.Save(m.registry.Clone()); err != nil {
		m.putStance(userID, key, before)
		return fmt.Errorf("the stance couldn't be saved; please try again: %w", err)
	}
	return nil
}

func (m *StrategyModule) putStance(userID int, key string, st stance.Stance) {
	if st == stance.None {
		delete(m.registry.Stances[userID], key)
		if len(m.registry.Stances[userID]) == 0 {
			delete(m.registry.Stances, userID)
		}
		return
	}
	if m.registry.Stances == nil {
		m.registry.Stances = map[int]map[string]stance.Stance{}
	}
	if m.registry.Stances[userID] == nil {
		m.registry.Stances[userID] = map[string]stance.Stance{}
	}
	m.registry.Stances[userID][key] = st
}

// isStanceOff reports whether the word takes a stance off.
func isStanceOff(word string) bool {
	switch word {
	case "off", "none", "clear", "reset", "default":
		return true
	}
	return false
}
