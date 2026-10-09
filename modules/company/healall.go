package company

// The admin `healcompany` command: fully restore everyone in the leader's
// company and raise the fallen. Unlike the player paths (heal wounds, camp
// rest, resurrection) it charges nothing, costs no level, and wakes the dead
// at full health and mana. It never touches the clock.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const healCompanyBattle = `The company is in a battle. Finish it or retreat, then <ansi fg="command">healcompany</ansi>.`

func (m *CompanyModule) healCompanyCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	if user.Role != users.RoleAdmin && !user.HasRolePermission("healcompany") {
		return false, nil // no hint that the command exists
	}
	user.SendText(m.healCompany(user, room))
	return true, nil
}

// healCompany restores the leader and every companion and returns the report.
func (m *CompanyModule) healCompany(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	members := m.woundMembers(user)
	if companyFighting(user, members) {
		return healCompanyBattle
	}
	var restored []string
	// The leader, and the living companions the game is running.
	for _, w := range members {
		c := w.char
		c.Wounds = nil
		c.RecalculateStats()
		c.Health, c.Mana = c.HealthMax.Value, c.ManaMax.Value
		if !w.leader() {
			m.refreshSnapshot(user.UserId, w.companionID)
		}
		restored = append(restored, w.name)
	}
	// The record: the dead rise, and any companion not running (away, fled,
	// awaiting restoration) is saved whole.
	record, _ := m.registry.Get(user.UserId)
	var raised []string
	for _, c := range record.Companions {
		if id, tracked := m.instance(user.UserId, c.ID); tracked && !c.Dead() && m.runtime.IsLive(id) {
			continue // restored above
		}
		if c.State != nil {
			state := c.State.Clone()
			state.Wounds, state.Vitals = nil, nil
			if err := m.registry.SetState(user.UserId, c.ID, state); err != nil {
				mudlog.Error("company: healcompany: set state", "leader", user.UserId, "companion", c.ID, "error", err)
			}
		}
		if c.Dead() {
			if err := m.registry.Revive(user.UserId, c.ID); err != nil {
				mudlog.Error("company: healcompany: revive", "leader", user.UserId, "companion", c.ID, "error", err)
				continue
			}
			raised = append(raised, companionName(c))
		}
	}
	if err := m.save(); err != nil {
		mudlog.Error("company: healcompany: save", "leader", user.UserId, "error", err)
	}
	if len(raised) > 0 {
		if err := m.restoreForLeader(user.UserId, user.Character.RoomId); err != nil {
			mudlog.Error("company: healcompany: restore", "leader", user.UserId, "error", err)
		}
	}
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	mudlog.Info("company: healcompany", "leader", user.UserId, "raised", len(raised))
	out := fmt.Sprintf("Your company is whole again: %s are at full health and mana, every wound mended.", andList(append(restored, raised...)))
	if len(raised) == 1 {
		out += " " + raised[0] + " stands again."
	} else if len(raised) > 1 {
		out += " " + andList(raised) + " stand again."
	}
	return out
}

// andList joins names as "a", "a and b" or "a, b and c".
func andList(names []string) string {
	if len(names) <= 2 {
		return strings.Join(names, " and ")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
