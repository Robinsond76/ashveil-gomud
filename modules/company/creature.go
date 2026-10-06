package company

// Phase 38e: creature recruits (a hound and a stone golem). The family rules
// are internal/creatures; this file is where the company module applies them:
// what a creature's recruit listing says, how it is repaired, and the helpers
// the other seams (carry, drift, morale, camp) ask. Biological creatures
// follow every ordinary companion rule; a construct is the exception in each
// place the rules below name.

import (
	"fmt"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// familyOf is the creature family of a companion, if it is a creature.
func familyOf(c domain.Companion) (creatures.Family, bool) {
	return creatures.ForArchetype(c.Archetype)
}

// bound reports whether a companion is a construct, held by bond rather than
// loyalty: it never drifts, loses heart or deserts.
func bound(c domain.Companion) bool {
	f, ok := familyOf(c)
	return ok && f.Bound()
}

// familyLine is the recruit listing's and `company inspect`'s line about a
// creature species: what it is and how it is kept.
func familyLine(f creatures.Family) string {
	switch f.Kind {
	case creatures.Construct:
		return fmt.Sprintf("%s, a construct: %s. It needs no food, drink or rest and never loses heart; it does not mend on its own and is repaired with stone mortar (company repair). It carries nothing and wears %s; its attack is %s.", f.Name, f.Role, f.Gear, f.Innate)
	default:
		return fmt.Sprintf("%s, a biological creature: %s. It eats, drinks and tires like the rest of the company, mends in camp and by rest, and can lose faith in a company it disagrees with. It carries nothing and wears %s; its attack is %s.", f.Name, f.Role, f.Gear, f.Innate)
	}
}

// familyShort is the recruit listing's one-line form of familyLine.
func familyShort(f creatures.Family) string {
	if f.Bound() {
		return f.Name + ", a construct: needs no food, drink or rest, never loses heart, repaired with stone mortar (help stone-golem)"
	}
	return f.Name + ", a biological creature: eats, drinks and rests like the company (help hound)"
}

// templateFamily is the creature family a mob template recruits as.
func (m *CompanyModule) templateFamily(templateID int) (creatures.Family, bool) {
	return creatures.ForArchetype(m.companionArchetypes()[templateID])
}

const repairUsage = `Usage: <ansi fg="command">company repair</ansi> (who needs it) or <ansi fg="command">company repair [golem]</ansi>. See <ansi fg="command">help repair</ansi>.`

// repairNeed is a construct that can be repaired now: out with the leader,
// alive, and below its health limit.
type repairNeed struct {
	c        domain.Companion
	instance int
	mob      *mobs.Mob
}

// constructs lists the leader's construct companions out with them.
func (m *CompanyModule) constructs(leaderUserID int) []repairNeed {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	var out []repairNeed
	for _, c := range record.Companions {
		if !bound(c) || c.Dead() || c.Separated() {
			continue
		}
		instanceID, tracked := m.instance(leaderUserID, c.ID)
		if !tracked || !m.runtime.IsLive(instanceID) || !m.runtime.IsAttached(leaderUserID, instanceID) || !m.runtime.WithLeader(leaderUserID, instanceID) {
			continue
		}
		mob := mobs.GetInstance(instanceID)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		out = append(out, repairNeed{c: c, instance: instanceID, mob: mob})
	}
	return out
}

// repair is `company repair [golem]`: with no word, who is damaged and how
// much mortar the company has; with a name, mortar is spent one at a time,
// each mending half the golem's health, until it is whole or the mortar is
// gone. Nothing runs in a fight, and the clock never moves.
func (m *CompanyModule) repair(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if usercommands.InBattle(user) {
		return usercommands.BattleUnderWay
	}
	members := m.woundMembers(user)
	if companyFighting(user, members) {
		return "You can't make repairs in the middle of a fight."
	}
	stock := len(m.supplies(user, members, creatures.RepairItemID, 0))
	selector := strings.TrimSpace(strings.Join(args, " "))
	if selector == "" || selector == "list" {
		return m.repairView(user, stock)
	}
	record, _ := m.registry.Get(user.UserId)
	c, found := resolveCompanion(record, selector)
	if !found {
		return "You have no companion like that."
	}
	name := nameOf(c, "that companion")
	f, creature := familyOf(c)
	switch {
	case !creature || !f.Repaired():
		return fmt.Sprintf("%s isn't a construct. Repair is for stone and iron; the living heal (help heal).", name)
	case c.Dead():
		return fmt.Sprintf("%s has fallen. See help resurrect.", name)
	}
	var need *repairNeed
	for _, n := range m.constructs(user.UserId) {
		if n.c.ID == c.ID {
			n := n
			need = &n
		}
	}
	if need == nil {
		return fmt.Sprintf("%s isn't here with you to repair.", name)
	}
	ch := &need.mob.Character
	if ch.Health >= ch.HealthLimit() {
		return fmt.Sprintf("%s is whole; there is nothing to repair.", name)
	}
	if stock == 0 {
		return "You have no stone mortar. Markets sell it for a few gold (help repair)."
	}
	used := 0
	startHealth := ch.Health
	for ch.Health < ch.HealthLimit() {
		if !m.spendItem(user, members, creatures.RepairItemID, 0) {
			break
		}
		used++
		ch.Health = creatures.RepairedHealth(ch.Health, ch.HealthLimit(), ch.HealthMax.Value)
	}
	if used == 0 {
		return fmt.Sprintf("You couldn't get at any stone mortar for %s.", name)
	}
	m.refreshSnapshot(user.UserId, c.ID)
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	left := ""
	if ch.Health < ch.HealthLimit() {
		left = fmt.Sprintf(" It still needs more: %d/%d.", ch.Health, ch.HealthLimit())
	}
	return fmt.Sprintf("You pack stone mortar into the cracks of %s, %d use%s. It is mended from %d to %d of %d.%s", name, used, plural(used), startHealth, ch.Health, ch.HealthMax.Value, left)
}

// repairView lists each construct and what it needs.
func (m *CompanyModule) repairView(user *users.UserRecord, stock int) string {
	all := m.constructs(user.UserId)
	if len(all) == 0 {
		return "No stone golem is with you. Repair is for constructs (help repair)."
	}
	lines := []string{fmt.Sprintf("Stone mortar in reach: %d (each mends half a golem's health).", stock)}
	for _, n := range all {
		ch := &n.mob.Character
		state := "whole"
		if ch.Health < ch.HealthLimit() {
			state = "needs repair"
		}
		lines = append(lines, fmt.Sprintf("  #%d %s: %d/%d, %s", n.c.ID, nameOf(n.c, "golem"), ch.Health, ch.HealthMax.Value, state))
	}
	return strings.Join(lines, "\n")
}
