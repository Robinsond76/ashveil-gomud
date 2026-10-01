// Package effecttargets resolves helpful effects from authoritative membership.
// Casts capture a starting set and can only lose targets while they chant.
package effecttargets

import (
	"slices"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/battle"
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

// OtherBattle reports whether a patient (a player when targetUserID > 0,
// otherwise the mob targetMobID) is fighting in a player's battle the caster
// is not part of. A battle's sides are its foes, its player, and the mobs
// fighting for that player in the battle room (companions or charmed pets,
// as the battle itself counts them). A bystander helps neither side.
func OtherBattle(userID, mobID, targetUserID, targetMobID int) bool {
	var patientIn []int
	if targetUserID > 0 {
		if _, ok := battle.RoomOf(targetUserID); ok {
			patientIn = append(patientIn, targetUserID)
		}
	} else if targetMobID > 0 {
		patientIn = append(patientIn, battle.Involving(targetMobID)...)
		if uid, ok := fightsFor(targetMobID); ok && !slices.Contains(patientIn, uid) {
			patientIn = append(patientIn, uid)
		}
	}
	if len(patientIn) == 0 {
		return false
	}
	var casterIn []int
	if userID > 0 {
		casterIn = append(casterIn, userID)
	}
	if mobID > 0 {
		casterIn = append(casterIn, battle.Involving(mobID)...)
		if uid, ok := fightsFor(mobID); ok {
			casterIn = append(casterIn, uid)
		}
	}
	for _, uid := range patientIn {
		if !slices.Contains(casterIn, uid) && !consentingSharedBattle(owner(userID, mobID), uid) {
			return true
		}
	}
	return false
}

// consentingSharedBattle expands participation only for mutual allies actually
// fighting a common living enemy in the same room, never idle alliance members.
func consentingSharedBattle(caster, patient int) bool {
	if caster <= 0 || !slices.Contains(leaders(caster, spells.ScopeAllied), patient) {
		return false
	}
	a, ok := battle.Current(caster)
	if !ok {
		return false
	}
	b, ok := battle.Current(patient)
	if !ok || a.RoomId != b.RoomId {
		return false
	}
	cu, pu := users.GetByUserId(caster), users.GetByUserId(patient)
	if cu == nil || pu == nil || cu.Character == nil || pu.Character == nil || cu.Character.RoomId != a.RoomId || pu.Character.RoomId != a.RoomId || cu.Character.Health <= 0 || pu.Character.Health <= 0 || cu.Character.CombatWithdrawn || pu.Character.CombatWithdrawn {
		return false
	}
	for id := range a.Enemies {
		if b.Has(id) {
			m := mobs.GetInstance(id)
			if eligibleMob(m, a.RoomId) {
				return true
			}
		}
	}
	return false
}

// fightsFor is the player whose battle a mob fights in as an ally: its
// company leader or charmer, while that player has a battle in its room.
func fightsFor(instanceID int) (int, bool) {
	m := mobs.GetInstance(instanceID)
	if m == nil {
		return 0, false
	}
	uid, _, member := company.LeaderAndKeyForInstance(instanceID)
	if !member {
		if m.Character.Charmed == nil || m.Character.Charmed.UserId <= 0 {
			return 0, false
		}
		uid = m.Character.Charmed.UserId
	}
	room, ok := battle.RoomOf(uid)
	return uid, ok && room == m.Character.RoomId
}

// Unaided reports a helpful cast left with no one to help.
func Unaided(info characters.SpellAggroInfo) bool {
	return Helpful(spells.GetSpell(info.SpellId)) && len(info.TargetUserIds)+len(info.TargetMobInstanceIds) == 0
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
		if u == nil || !eligiblePlayer(u.Character, c.RoomId, sp) || (sp.ExcludeSelf && uid == userID) || (scope != spells.ScopeMember && !slices.Contains(ls, uid)) || slices.Contains(keepUsers, uid) || OtherBattle(userID, mobID, uid, 0) {
			continue
		}
		keepUsers = append(keepUsers, uid)
	}
	for _, id := range info.TargetMobInstanceIds {
		m := mobs.GetInstance(id)
		if !eligibleMob(m, c.RoomId) || (sp.ExcludeSelf && id == mobID) || slices.Contains(keepMobs, id) || OtherBattle(userID, mobID, 0, id) {
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
