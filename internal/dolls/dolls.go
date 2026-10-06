// Package dolls makes and keeps a Doll Master's dolls (Phase 39d). A doll's
// record lives on its Master (characters.DollState: name, wear, whether it is
// broken, its gear); this package turns the record into a mob when a battle
// opens, writes the mob's health and gear back as the battle goes, and takes
// the mob away when the battle ends. A doll is charmed to the company's
// leader and registered with internal/company as a member no record holds, so
// the combat code finds it by the paths it finds companions and a formation
// cell holds it, but it takes no company slot and has no turn of its own: the
// Master's turn is its strike (internal/hooks/combat_doll.go).
package dolls

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobID is the doll's mob template (_datafiles/world/default/mobs/summons).
const MobID = 161

// PartsItemID is the doll part: what mends a doll.
const PartsItemID = 70

// Lineage is the archetype id of the Doll Master.
const Lineage = "dollmaster"

// Numbers of the standard doll: its health is a share of a warrior's at its
// Master's level, its Attack a share of the warrior's rate, and it never
// dodges.
const (
	BaseHPPct   = 70
	AttackPct   = 90
	MendBase    = 10 // health a doll part mends, plus MendPerLvl a level of the Master
	MendPerLvl  = 3
	TanglePush  = 50 // action meter points a Tangle pushes a foe back
	BossPush    = 25
	TangleHold  = 2 // rounds a tangled foe can't be tangled again
	SplicePct   = 25
	GuardsLevel = 8 // the level Guard String works a third time
)

// Why a doll can't be made.
var (
	ErrNoMaster = fmt.Errorf("dolls: no Doll Master to drive it")
	ErrNoBattle = fmt.Errorf("dolls: only in battle")
	ErrNoRoom   = fmt.Errorf("dolls: no room to stand in")
)

// Gifts are what a Master's route and talents give its dolls.
type Gifts struct {
	Count  int  // dolls the Master drives
	HPPct  int  // each doll's health as a percent of a warrior's
	Armor  int  // armor on each doll's own body
	Attack int  // Attack added to each doll's blows
	Damage int  // damage added to each doll's blows
	NoWear bool // a golem can't wear armor
}

// GiftsFor reads a Master's gifts from its class effects.
func GiftsFor(fx classes.Effects) Gifts {
	mult := 100
	if v := fx.Int(classes.DollHPPct); v > 0 {
		mult = v
	}
	mult += fx.Int(classes.DollHPBonus)
	return Gifts{
		Count:  1 + max(0, fx.Int(classes.DollCount)),
		HPPct:  max(1, BaseHPPct*mult/100),
		Armor:  fx.Int(classes.DollArmor),
		Attack: fx.Int(classes.DollAttack),
		Damage: fx.Int(classes.DollDamage),
		NoWear: fx.Has(classes.DollNoWear),
	}
}

// IsMaster reports whether a character is a Doll Master.
func IsMaster(c *characters.Character) bool {
	return c != nil && c.ArchetypeID() == Lineage
}

// MendPerPart is the health one doll part mends a doll for a Master of a
// level.
func MendPerPart(level int) int { return MendBase + MendPerLvl*max(level, 1) }

// GuardUses is how many times a battle the Master's dolls can guard (Guard
// String): its effect, 0 before it is learned.
func GuardUses(fx classes.Effects) int { return max(0, fx.Int(classes.DollGuards)) }

// Master is whoever drives dolls: a player, or a company mob.
type Master struct {
	Leader int               // the company's leader
	Key    company.MemberKey // the Master's member key
	Char   *characters.Character
	Room   int
}

// Of finds a Master by its member key in a leader's company, ok false when
// it can't be found (gone, or not a Doll Master).
func Of(leader int, key company.MemberKey) (Master, bool) {
	if key == company.LeaderMemberKey {
		u := users.GetByUserId(leader)
		if u == nil || u.Character == nil {
			return Master{}, false
		}
		return Master{Leader: leader, Key: key, Char: u.Character, Room: u.Character.RoomId}, true
	}
	id, ok := company.InstanceForKey(leader, key)
	if !ok {
		return Master{}, false
	}
	m := mobs.GetInstance(id)
	if m == nil {
		return Master{}, false
	}
	return Master{Leader: leader, Key: key, Char: &m.Character, Room: m.Character.RoomId}, true
}

// Stats puts a doll's numbers on its mob: its level is its Master's, its
// health a share of a warrior's, its Attack 90% of a warrior's and its
// Evasion none.
func shape(mob *mobs.Mob, level int, g Gifts, info *characters.DollInfo) {
	c := &mob.Character
	c.Level = max(level, 1)
	c.HPArchetype = "warrior"
	rt := c.RTState()
	rt.Doll = info
	rt.Bark = g.Armor
	warriorAttack := levelRating(c.Level, 1.0)
	c.AttackOffset = levelRating(c.Level, float64(AttackPct)/100) - warriorAttack + g.Attack
	c.EvasionOffset = -levelRating(c.Level, 1.0)
	c.RecalculateStats()
}

func levelRating(level int, rate float64) int {
	return int(float64(max(level, 0))*rate + 1e-9)
}

// Live is the doll mob of a Master's record at an index, if it stands.
func Live(m Master, index int) (*mobs.Mob, bool) {
	id, ok := company.DollInstanceFor(m.Leader, company.DollMemberKey(m.Key, index))
	if !ok {
		return nil, false
	}
	mob := mobs.GetInstance(id)
	return mob, mob != nil
}

// Spawn makes every doll of a Master that is whole and not already standing,
// in its room, for the battle its leader is in. It returns the new ones.
func Spawn(m Master) ([]*mobs.Mob, error) {
	if m.Char == nil || !IsMaster(m.Char) {
		return nil, ErrNoMaster
	}
	lu := users.GetByUserId(m.Leader)
	if lu == nil || lu.Character == nil {
		return nil, ErrNoMaster
	}
	if b, ok := battle.Current(m.Leader); !ok || b.RoomId != m.Room {
		return nil, ErrNoBattle
	}
	room := rooms.LoadRoom(m.Room)
	if room == nil {
		return nil, ErrNoRoom
	}
	g := GiftsFor(m.Char.ClassEffects())
	m.Char.EnsureDolls(g.Count)
	var made []*mobs.Mob
	for i := 0; i < g.Count; i++ {
		rec := &m.Char.Dolls[i]
		if rec.Broken {
			continue
		}
		if _, standing := Live(m, i); standing {
			continue
		}
		mob := mobs.NewMobByIdNoElite(mobs.MobId(MobID), m.Room, max(m.Char.Level, 1))
		if mob == nil {
			return made, ErrNoMaster
		}
		mob.Character.Name = rec.Name
		mob.Character.Equipment = characters.Worn{}
		wear := characters.CloneDolls([]characters.DollState{*rec})[0].Equipment
		for _, slot := range characters.AllSlots() {
			itm := wear.Get(slot)
			if itm == nil || itm.ItemId == 0 {
				continue
			}
			if g.NoWear && slot != items.Weapon {
				continue // a golem's body is its armor
			}
			mob.Character.Equipment.Set(slot, *itm)
		}
		info := &characters.DollInfo{OwnerKey: string(m.Key), Index: i, HPPct: g.HPPct, Damage: g.Damage}
		if m.Key == company.LeaderMemberKey {
			info.OwnerUser = m.Leader
		} else if id, ok := company.InstanceForKey(m.Leader, m.Key); ok {
			info.OwnerMob = id
		}
		shape(mob, m.Char.Level, g, info)
		mob.Character.Health = max(1, mob.Character.HealthLimit()-rec.Damage)
		mob.Character.Mana = 0
		mob.Character.CharmAsCompanion(m.Leader, -2, characters.CharmExpiredRevert)
		lu.Character.TrackCharmed(mob.InstanceId, true)
		room.AddMob(mob.InstanceId)
		company.RegisterDoll(mob.InstanceId, m.Leader, m.Key, i)
		made = append(made, mob)
	}
	return made, nil
}

// Sync writes a live doll's health and gear back to its Master's record. A
// doll at 0 health is broken.
func Sync(instanceID int) {
	e, ok := company.DollOf(instanceID)
	if !ok {
		return
	}
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return
	}
	m, ok := Of(e.Leader, e.Owner)
	if !ok || e.Index >= len(m.Char.Dolls) {
		return
	}
	rec := &m.Char.Dolls[e.Index]
	limit := max(1, mob.Character.HealthLimit())
	if mob.Character.Health < 1 {
		rec.Broken, rec.Damage = true, limit
	} else {
		rec.Damage = max(0, limit-mob.Character.Health)
	}
	// Armor a golem can't wear stays in the record: only the slots a doll
	// can use are copied back.
	noWear := GiftsFor(m.Char.ClassEffects()).NoWear
	worn := characters.CloneDolls([]characters.DollState{{Equipment: mob.Character.Equipment}})[0].Equipment
	for _, slot := range characters.AllSlots() {
		if noWear && slot != items.Weapon {
			continue
		}
		if itm := worn.Get(slot); itm != nil && itm.ItemId != 0 {
			rec.Equipment.Set(slot, *itm)
		} else {
			rec.Equipment.Set(slot, items.Item{})
		}
	}
}

// Dismiss takes a doll away: its record is brought up to date first, then its
// mob leaves its room, its leader's charmed list, the registry and the world.
// It is safe to call twice.
func Dismiss(instanceID int) {
	Sync(instanceID)
	if e, ok := company.DollOf(instanceID); ok {
		if lu := users.GetByUserId(e.Leader); lu != nil && lu.Character != nil {
			lu.Character.TrackCharmed(instanceID, false)
		}
	}
	company.ForgetDoll(instanceID)
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return
	}
	mob.Character.RemoveCharm()
	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		room.RemoveMob(instanceID)
	}
	mobs.DestroyInstance(instanceID)
}

// DismissAll removes every doll a leader has standing.
func DismissAll(leader int) {
	for _, id := range company.DollInstances() {
		if e, ok := company.DollOf(id); ok && e.Leader == leader {
			Dismiss(id)
		}
	}
}

// IsDoll reports whether a mob is a live doll.
func IsDoll(m *mobs.Mob) bool {
	return m != nil && m.Character.RT != nil && m.Character.RT.Doll != nil
}

// Mend repairs a Master's broken or worn dolls with doll parts from the
// parts it is given, each part mending MendPerPart health, broken dolls
// first and the most worn next. It returns the parts used and the health
// mended.
func Mend(c *characters.Character, parts int) (used, mended int) {
	if c == nil || parts < 1 {
		return 0, 0
	}
	per := MendPerPart(c.Level)
	for used < parts {
		idx := -1
		for i := range c.Dolls {
			d := c.Dolls[i]
			if d.Damage < 1 && !d.Broken {
				continue
			}
			if idx < 0 || d.Broken && !c.Dolls[idx].Broken || d.Broken == c.Dolls[idx].Broken && d.Damage > c.Dolls[idx].Damage {
				idx = i
			}
		}
		if idx < 0 {
			break
		}
		d := &c.Dolls[idx]
		before := d.Damage
		d.Broken = false // the first part brings a broken doll back on its feet
		d.Damage = max(0, d.Damage-per)
		used++
		mended += min(per, max(before, 1))
	}
	return used, mended
}

// Parts counts the doll parts a character carries.
func Parts(c *characters.Character) int {
	n := 0
	for _, itm := range c.Items {
		if itm.ItemId == PartsItemID {
			n++
		}
	}
	return n
}

// TakeParts removes up to n doll parts from a character's pack and returns
// the ones taken.
func TakeParts(c *characters.Character, n int) []items.Item {
	var taken []items.Item
	for i := len(c.Items) - 1; i >= 0 && len(taken) < n; i-- {
		if c.Items[i].ItemId == PartsItemID {
			taken = append(taken, c.Items[i])
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
		}
	}
	return taken
}

// Worn is a doll's state in a short line: whole, worn down or broken.
func Condition(d characters.DollState, limit int) string {
	switch {
	case d.Broken:
		return "broken"
	case d.Damage < 1:
		return "whole"
	case d.Damage*2 >= limit:
		return "badly worn"
	}
	return "worn"
}

// HealthLimit is the health a doll of a Master would have, whole: a share of
// a warrior's health at the Master's level.
func HealthLimit(m *characters.Character) int {
	g := GiftsFor(m.ClassEffects())
	probe := characters.Character{Level: max(m.Level, 1), HPArchetype: "warrior"}
	probe.RT = &characters.ClassRT{Doll: &characters.DollInfo{HPPct: g.HPPct}}
	probe.RecalculateStats()
	return max(1, probe.HealthLimit())
}

// Worn reports whether any of a Master's dolls is broken or worn down.
func NeedsMending(c *characters.Character) bool {
	for _, d := range c.Dolls {
		if d.Broken || d.Damage > 0 {
			return true
		}
	}
	return false
}

// MendCompany mends each Master's dolls with doll parts from the leader's
// pack, in the order given (the leader first), until the parts run out. It
// returns the parts taken from the pack and how many each Master used.
func MendCompany(leader *characters.Character, masters []*characters.Character) (taken []items.Item, used []int) {
	used = make([]int, len(masters))
	for i, c := range masters {
		if c == nil || !NeedsMending(c) {
			continue
		}
		n, _ := Mend(c, Parts(leader))
		if n > 0 {
			taken = append(taken, TakeParts(leader, n)...)
			used[i] = n
		}
	}
	return taken, used
}
