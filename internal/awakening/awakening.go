// Package awakening is Phase 67's relic awakenings at runtime. A relic's
// spec lists deeds (items.AwakeningSpec); this package advances them from the
// game's real sources and wakes the power when one is met:
//
//   - slay: Slain, called where a mob's death is credited (internal/mobcommands);
//   - lair: the chronicle's Boss deed, which this package watches through
//     chronicle.OnRecord, so a lair master's fall counts once, from the record
//     the chronicle keeps;
//   - place: Reached and CompanionReached, called by a room-change listener
//     when the leader, or a companion with the leader, crosses into a zone.
//
// Progress counts only for relics worn by a living member of the leader's
// company who stands in the leader's room, and it is saved on the item.
package awakening

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	chronicle.OnRecord(onDeed)
}

// Bearer is a company member who may wear a relic.
type Bearer struct {
	Char *characters.Character
	Key  string // the company member key ("leader", "companion:7")
}

// members is the leader's company that stands with them: the leader and each
// living companion in the leader's room.
var members = func(leaderUserID int) []Bearer {
	u := users.GetByUserId(leaderUserID)
	if u == nil || u.Character == nil {
		return nil
	}
	here := u.Character.RoomId
	var out []Bearer
	add := func(c *characters.Character, key string) {
		if c.Health < 1 || c.CombatWithdrawn || c.RoomId != here {
			return
		}
		out = append(out, Bearer{c, key})
	}
	add(u.Character, string(company.LeaderMemberKey))
	list, _ := company.CompanyMembers(leaderUserID)
	for _, member := range list {
		id, ok := company.InstanceFor(leaderUserID, member.ID)
		if !ok {
			continue
		}
		if m := mobs.GetInstance(id); m != nil {
			add(&m.Character, string(company.CompanionMemberKey(member.ID)))
		}
	}
	return out
}

// UseMembersForTest replaces the member lookup.
func UseMembersForTest(fn func(leaderUserID int) []Bearer) func() {
	prev := members
	members = fn
	return func() { members = prev }
}

// notify says an awakening woke; replaced in tests.
var notify = func(leaderUserID int, text string) {
	if u := users.GetByUserId(leaderUserID); u != nil {
		u.SendText(text)
	}
}

// Slain advances every worn relic's slay awakenings that name the race of a
// foe the leader's company just defeated.
func Slain(leaderUserID int, race string) {
	advance(leaderUserID, items.AwakenSlay, race)
}

// Reached advances the place awakenings that name the zone the leader just
// entered, on the relics the leader wears. Each member is credited by its own
// crossing: companions follow a step behind, so theirs come through
// CompanionReached when they arrive.
func Reached(leaderUserID int, zone string) {
	advanceFor(leaderUserID, items.AwakenPlace, zone, func(b Bearer) bool {
		return b.Key == string(company.LeaderMemberKey)
	})
}

// CompanionReached advances the place awakenings on the relics a companion
// wears when it crosses into zone while its leader is in that zone too (a
// companion that wanders off alone earns nothing). It is credited by its own
// arrival, which comes a moment after the leader's.
func CompanionReached(instanceID int, zone string) {
	leader, b, ok := companionOf(instanceID)
	if !ok || b.Char.Health < 1 || b.Char.CombatWithdrawn || leaderZone(leader) != zone {
		return
	}
	advanceBearers(leader, items.AwakenPlace, zone, []Bearer{b})
}

// companionOf finds the leader of a live company mob and the mob as a
// bearer.
var companionOf = func(instanceID int) (int, Bearer, bool) {
	m := mobs.GetInstance(instanceID)
	if m == nil {
		return 0, Bearer{}, false
	}
	leader, key, ok := company.LeaderAndKeyForInstance(instanceID)
	if !ok || leader < 1 {
		return 0, Bearer{}, false
	}
	return leader, Bearer{&m.Character, string(key)}, true
}

// leaderZone is the zone the leader stands in.
var leaderZone = func(leaderUserID int) string {
	if u := users.GetByUserId(leaderUserID); u != nil && u.Character != nil {
		if r := rooms.LoadRoom(u.Character.RoomId); r != nil {
			return r.Zone
		}
	}
	return ""
}

// onDeed turns the chronicle's boss deeds into lair progress.
func onDeed(leaderUserID int, e chronicle.Entry) {
	if e.Kind != chronicle.Boss {
		return
	}
	id, ok := strings.CutPrefix(e.Ref, "mob:")
	if !ok {
		return
	}
	if _, err := strconv.Atoi(id); err != nil {
		return
	}
	advance(leaderUserID, items.AwakenLair, id)
}

// advance gives one step of a deed to every worn relic awakening it matches.
func advance(leaderUserID int, kind, subject string) {
	advanceFor(leaderUserID, kind, subject, nil)
}

// advanceFor is advance for the members who pass only (all when nil).
func advanceFor(leaderUserID int, kind, subject string, only func(Bearer) bool) {
	var bearers []Bearer
	for _, b := range members(leaderUserID) {
		if only == nil || only(b) {
			bearers = append(bearers, b)
		}
	}
	advanceBearers(leaderUserID, kind, subject, bearers)
}

// advanceBearers gives one step of a deed to each matching awakening on the
// relics these bearers wear.
func advanceBearers(leaderUserID int, kind, subject string, bearers []Bearer) {
	changed := false
	defer func() {
		if changed {
			// The gear window, tooltips and company inventory show progress.
			events.AddToQueue(events.CompanyAssetsChanged{UserId: leaderUserID})
		}
	}()
	for _, b := range bearers {
		for _, slot := range characters.AllSlots() {
			if slot == items.Pack {
				continue
			}
			itm := b.Char.Equipment.Get(slot)
			if itm == nil || itm.ItemId < 1 || !itm.HasAwakenings() {
				continue
			}
			for idx, a := range itm.GetSpec().Relic.Awakenings {
				if !a.Matches(kind, subject) || itm.Awakened(idx) {
					continue
				}
				changed = true
				if itm.AdvanceAwakening(idx, 1) {
					woke(leaderUserID, b, *itm, a)
				}
			}
		}
	}
}

// woke announces an awakening and records it as the company's deed.
func woke(leaderUserID int, b Bearer, itm items.Item, a items.AwakeningSpec) {
	power := strings.Join(classes.DescribeGearEffects(a.Effects), "; ")
	notify(leaderUserID, fmt.Sprintf(`<ansi fg="yellow-bold">%s's %s awakens: %s.</ansi> It now gives %s while worn.`, b.Char.Name, itm.Name(), a.Name, power))
	chronicle.Record(leaderUserID, chronicle.Entry{
		Kind:    chronicle.Awakened,
		Members: []string{b.Char.Name},
		Keys:    []string{b.Key},
		Subject: itm.Name(),
		Detail:  a.Name,
		Ref:     fmt.Sprintf("item:%d", itm.ItemId),
	})
}
