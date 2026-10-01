// Package effecttargets resolves helpful effects from authoritative membership.
// Casts capture a starting set and can only lose targets while they chant.
package effecttargets

import (
	"slices"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var alliedMu sync.RWMutex
var alliedLeaders func(int) []int

// SetAlliedLeaders installs 33d's consent provider; absent consent never
// broadens a company. It returns the previous provider for test restoration.
func SetAlliedLeaders(resolve func(int) []int) func(int) []int {
	alliedMu.Lock()
	defer alliedMu.Unlock()
	previous := alliedLeaders
	alliedLeaders = resolve
	return previous
}

func Helpful(sp *spells.SpellData) bool {
	return sp != nil && (sp.Type == spells.HelpSingle || sp.Type == spells.HelpMulti || sp.Type == spells.HelpArea)
}

func identity(id int) characters.FriendlyMobIdentity {
	m := mobs.GetInstance(id)
	if m == nil {
		return characters.FriendlyMobIdentity{}
	}
	owner, key, _ := company.LeaderAndKeyForInstance(id)
	return characters.FriendlyMobIdentity{OwnerID: owner, MemberKey: string(key), Charm: m.Character.Charmed}
}

func owner(userID, mobID int) int {
	if userID > 0 {
		return userID
	}
	leader, _, member := company.LeaderAndKeyForInstance(mobID)
	m := mobs.GetInstance(mobID)
	if member && m != nil && m.Character.IsCharmed(leader) && m.Character.Charmed.Companion && m.Character.Charmed.RoundsRemaining != 0 {
		return leader
	}
	return 0
}

func source(userID, mobID int) *characters.Character {
	if userID > 0 {
		if u := users.GetByUserId(userID); u != nil {
			return u.Character
		}
		return nil
	}
	if m := mobs.GetInstance(mobID); m != nil {
		return &m.Character
	}
	return nil
}

func leaders(leader int, scope spells.EffectScope) []int {
	if leader <= 0 {
		return nil
	}
	result := []int{leader}
	if scope == spells.ScopeAllied || scope == spells.ScopeArea {
		alliedMu.RLock()
		resolve := alliedLeaders
		alliedMu.RUnlock()
		if resolve != nil {
			for _, id := range resolve(leader) {
				if id > 0 && !slices.Contains(result, id) {
					result = append(result, id)
				}
			}
		}
	}
	return result
}

func eligiblePlayer(c *characters.Character, roomID int, sp *spells.SpellData) bool {
	return c != nil && c.RoomId == roomID && !c.CombatWithdrawn && (c.Health > 0 || (sp.AllowDowned && c.Health > -10))
}

func eligibleMob(m *mobs.Mob, roomID int) bool {
	return m != nil && m.Character.RoomId == roomID && m.Character.Health > 0 && !m.Character.CombatWithdrawn
}

func attached(m *mobs.Mob, leader int) bool {
	u := users.GetByUserId(leader)
	who, _, member := company.LeaderAndKeyForInstance(m.InstanceId)
	return member && who == leader && u != nil && u.Character != nil && m.Character.IsCharmed(leader) && m.Character.Charmed.Companion && m.Character.Charmed.RoundsRemaining != 0 && slices.Contains(u.Character.GetCharmIds(), m.InstanceId)
}

// Resolve is used before spending mana, and before every script callback.
// Initialized casts retain only their original eligible targets.
func Resolve(userID, mobID int, info characters.SpellAggroInfo) characters.SpellAggroInfo {
	sp := spells.GetSpell(info.SpellId)
	if !Helpful(sp) {
		return info
	}
	c := source(userID, mobID)
	if c == nil || c.Health < 1 || c.CombatWithdrawn {
		info.TargetUserIds = nil
		info.TargetMobInstanceIds = nil
		return info
	}
	scope := sp.FriendlyScope()
	leader := owner(userID, mobID)
	starting := info.FriendlyTargets == nil
	if starting {
		info.FriendlyTargets = &characters.FriendlyCastTargets{RoomID: c.RoomId, SourceOwner: leader, SourceMob: identity(mobID), Mobs: map[int]characters.FriendlyMobIdentity{}}
		if scope != spells.ScopeMember {
			info.TargetUserIds = nil
			info.TargetMobInstanceIds = nil
			ls := leaders(leader, scope)
			if leader == 0 && mobID > 0 {
				info.TargetMobInstanceIds = []int{mobID}
			}
			for _, uid := range ls {
				info.TargetUserIds = append(info.TargetUserIds, uid)
				if room := rooms.LoadRoom(c.RoomId); room != nil {
					for _, id := range room.GetMobs() {
						if m := mobs.GetInstance(id); m != nil && attached(m, uid) {
							info.TargetMobInstanceIds = append(info.TargetMobInstanceIds, id)
						}
					}
				}
			}
		}
		for _, id := range info.TargetMobInstanceIds {
			info.FriendlyTargets.Mobs[id] = identity(id)
		}
	}
	snap := info.FriendlyTargets
	if snap.RoomID != c.RoomId || snap.SourceOwner != leader || (mobID > 0 && (snap.SourceMob != identity(mobID) || (snap.SourceMob.Charm != nil && snap.SourceMob.Charm.RoundsRemaining == 0))) {
		info.TargetUserIds = nil
		info.TargetMobInstanceIds = nil
		return info
	}
	ls := leaders(leader, scope)
	var keepUsers, keepMobs []int
	for _, uid := range info.TargetUserIds {
		u := users.GetByUserId(uid)
		if u == nil || !eligiblePlayer(u.Character, c.RoomId, sp) || (sp.ExcludeSelf && uid == userID) || (scope != spells.ScopeMember && !slices.Contains(ls, uid)) || slices.Contains(keepUsers, uid) {
			continue
		}
		keepUsers = append(keepUsers, uid)
	}
	for _, id := range info.TargetMobInstanceIds {
		m := mobs.GetInstance(id)
		if !eligibleMob(m, c.RoomId) || (sp.ExcludeSelf && id == mobID) || slices.Contains(keepMobs, id) {
			continue
		}
		captured, ok := snap.Mobs[id]
		if !ok || captured != identity(id) || (captured.Charm != nil && captured.Charm.RoundsRemaining == 0) {
			continue
		}
		if scope != spells.ScopeMember && !(leader == 0 && id == mobID) {
			valid := false
			for _, uid := range ls {
				if attached(m, uid) {
					valid = true
					break
				}
			}
			if !valid {
				continue
			}
		}
		keepMobs = append(keepMobs, id)
	}
	info.TargetUserIds = keepUsers
	info.TargetMobInstanceIds = keepMobs
	return info
}
