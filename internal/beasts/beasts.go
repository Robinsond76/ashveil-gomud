// Package beasts raises a Beast Tamer's bonded beast (Phase 39e). The beast's
// record lives on its Tamer (characters.BeastState: name, health lost,
// whether it fell); this package turns the record into a mob when a battle
// opens, writes the mob's health back as the battle goes, and takes the mob
// away when the battle ends. The beast is charmed to the company's leader
// and registered with internal/company as a member no record holds, so the
// combat code finds it by the paths it finds companions, and a formation cell
// holds it, but it takes no company slot. Unlike a doll it is alive: it takes
// its own turn (internal/hooks/combat_beast.go) and a fall leaves it wounded
// until the company rests, never dead.
package beasts

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Lineage is the archetype id of the Beast Tamer.
const Lineage = "beasttamer"

// Kinds of beast.
const (
	Wolf     = "wolf"
	Warhound = "warhound"
	Bear     = "bear"
	Drake    = "drake"
)

// RaceID is the beasts' race (_datafiles/world/default/races/26-beast.yaml).
const RaceID = 26

// Mob templates of the beasts (_datafiles/world/default/mobs/summons).
const (
	WolfMobID     = 200
	WarhoundMobID = 201
	BearMobID     = 202
	DrakeMobID    = 203
)

// Numbers of the standard beast: its health is a share of a warrior's at its
// Tamer's level, it takes 70% of a normal tempo, and its Attack and Evasion
// run at 90% of the warrior's rate.
const (
	BaseHPPct    = 2 // the beast's health share of a warrior's starts here...
	MinHPPct     = 8 // ...never below this...
	HPPctPer4    = 7 // ...and grows this many points every 4 of the Tamer's levels
	TempoPct     = 70
	RatePct      = 90
	HobbleBelow  = 50 // percent health a foe is hobbled below
	BreathDice   = 6
	BiteSides    = 3 // a bite is 1d3 (the drake's 1d2) plus BiteBonus
	BitePerLevel = 3 // a bite adds 1 damage for every this many of the Tamer's levels
	BreathHalf   = 2 // the Breath adds this share of the Tamer's level: level/BreathHalf
	RallyDefault = 2
)

// Why a beast can't stand.
var (
	ErrNoTamer  = fmt.Errorf("beasts: no Beast Tamer to raise it")
	ErrNoBattle = fmt.Errorf("beasts: only in battle")
	ErrNoRoom   = fmt.Errorf("beasts: no room to stand in")
	ErrWounded  = fmt.Errorf("beasts: the beast is wounded")
)

// Gifts are what a Tamer's route and talents give its beast.
type Gifts struct {
	Kind        string
	MobID       int
	HPMult      int // the route's and talents' percent of the standard beast's health
	Dice, Sides int
	Attack      int
	Damage      int
	Hobble      bool
	Guards      int
	BreathEvery int // rounds between Breaths, 0 for none
}

// KindOf is the beast a Tamer's route raises.
func KindOf(fx classes.Effects) string {
	switch fx.Int(classes.BeastKind) {
	case classes.KindWarhound:
		return Warhound
	case classes.KindBear:
		return Bear
	case classes.KindDrake:
		return Drake
	}
	return Wolf
}

// KindName is a beast kind in words.
func KindName(kind string) string {
	switch kind {
	case Warhound:
		return "warhound"
	case Bear:
		return "war bear"
	case Drake:
		return "drake hatchling"
	}
	return "wolf"
}

// GiftsFor reads a Tamer's gifts from its class effects.
func GiftsFor(fx classes.Effects) Gifts {
	kind := KindOf(fx)
	mult := 100
	if v := fx.Int(classes.BeastHPPct); v > 0 {
		mult = v
	}
	mult += fx.Int(classes.BeastHPBonus)
	g := Gifts{
		Kind:   kind,
		MobID:  WolfMobID,
		HPMult: mult,
		Dice:   1,
		Sides:  BiteSides,
		Attack: fx.Int(classes.BeastAttack),
		Damage: fx.Int(classes.BeastDamage),
		Hobble: fx.Has(classes.BeastHobble),
		Guards: fx.Int(classes.BeastGuards),
	}
	switch kind {
	case Warhound:
		g.MobID = WarhoundMobID
	case Bear:
		g.MobID = BearMobID
	case Drake:
		g.MobID, g.Sides = DrakeMobID, BiteSides-1
		if every := fx.Int(classes.BeastBreath); every > 0 {
			g.BreathEvery = max(1, every-fx.Int(classes.BeastBreathCut))
		}
	}
	return g
}

// StandardPct is the standard beast's (the wolf's) health as a percent of a
// warrior's at a Tamer's level: a pup at first that grows with its Tamer
// (review tuning: a flat share made the Tamer far ahead at level 5 and
// behind at 20). 8% to level 4, 10% at 5, 19% at 10, 37% at 20, 54% at 30.
func StandardPct(level int) int {
	return max(MinHPPct, BaseHPPct+max(level, 1)*HPPctPer4/4)
}

// HPPctAt is a Tamer's beast's health as a percent of a warrior's at the
// Tamer's level.
func (g Gifts) HPPctAt(level int) int {
	return max(1, StandardPct(level)*g.HPMult/100)
}

// IsTamer reports whether a character is a Beast Tamer.
func IsTamer(c *characters.Character) bool {
	return c != nil && c.ArchetypeID() == Lineage
}

// Tamer is whoever raises a beast: a player, or a company mob.
type Tamer struct {
	Leader int               // the company's leader
	Key    company.MemberKey // the Tamer's member key
	Char   *characters.Character
	Room   int
}

// Of finds a Tamer by its member key in a leader's company, ok false when it
// can't be found (gone, or not a Tamer).
func Of(leader int, key company.MemberKey) (Tamer, bool) {
	if key == company.LeaderMemberKey {
		u := users.GetByUserId(leader)
		if u == nil || u.Character == nil {
			return Tamer{}, false
		}
		return Tamer{Leader: leader, Key: key, Char: u.Character, Room: u.Character.RoomId}, true
	}
	id, ok := company.InstanceForKey(leader, key)
	if !ok {
		return Tamer{}, false
	}
	m := mobs.GetInstance(id)
	if m == nil {
		return Tamer{}, false
	}
	return Tamer{Leader: leader, Key: key, Char: &m.Character, Room: m.Character.RoomId}, true
}

func levelRating(level int, rate float64) int {
	return int(float64(max(level, 0))*rate + 1e-9)
}

// shape puts a beast's numbers on its mob: its level is its Tamer's, its
// health a share of a warrior's, its Attack and Evasion 90% of a warrior's.
func shape(mob *mobs.Mob, level int, g Gifts, info *characters.BeastInfo) {
	c := &mob.Character
	c.Level = max(level, 1)
	c.HPArchetype = "warrior"
	c.RTState().Beast = info
	rate := float64(RatePct) / 100
	c.AttackOffset = levelRating(c.Level, rate) - levelRating(c.Level, 1.0) + g.Attack
	c.EvasionOffset = levelRating(c.Level, rate) - levelRating(c.Level, 1.0)
	c.RecalculateStats()
}

// Live is the beast mob of a Tamer's record, if it stands.
func Live(t Tamer) (*mobs.Mob, bool) {
	id, ok := company.BeastInstanceFor(t.Leader, company.BeastMemberKey(t.Key))
	if !ok {
		return nil, false
	}
	mob := mobs.GetInstance(id)
	return mob, mob != nil
}

// DefaultName is the name a new beast takes.
func DefaultName(kind string) string {
	switch kind {
	case Warhound:
		return "Fang"
	case Bear:
		return "Bruin"
	case Drake:
		return "Cinder"
	}
	return "Ash"
}

// Spawn makes a Tamer's beast, if it is whole and not already standing, in
// its room, for the battle its leader is in. It returns the new mob.
func Spawn(t Tamer) (*mobs.Mob, error) {
	if t.Char == nil || !IsTamer(t.Char) {
		return nil, ErrNoTamer
	}
	lu := users.GetByUserId(t.Leader)
	if lu == nil || lu.Character == nil {
		return nil, ErrNoTamer
	}
	if b, ok := battle.Current(t.Leader); !ok || b.RoomId != t.Room {
		return nil, ErrNoBattle
	}
	room := rooms.LoadRoom(t.Room)
	if room == nil {
		return nil, ErrNoRoom
	}
	g := GiftsFor(t.Char.ClassEffects())
	rec := t.Char.EnsureBeast(DefaultName(g.Kind))
	if rec.Wounded {
		return nil, ErrWounded
	}
	if _, standing := Live(t); standing {
		return nil, nil
	}
	mob := mobs.NewMobByIdNoElite(mobs.MobId(g.MobID), t.Room, max(t.Char.Level, 1))
	if mob == nil {
		return nil, ErrNoTamer
	}
	mob.Character.Name = rec.Name
	info := &characters.BeastInfo{
		Kind: g.Kind, OwnerKey: string(t.Key), HPPct: g.HPPctAt(t.Char.Level),
		Dice: g.Dice, Sides: g.Sides, Bonus: BiteBonus(t.Char.Level), TempoPct: TempoPct, Damage: g.Damage,
		Hobble: g.Hobble, Guards: g.Guards, BreathEvery: g.BreathEvery,
	}
	if t.Key == company.LeaderMemberKey {
		info.OwnerUser = t.Leader
	} else if id, ok := company.InstanceForKey(t.Leader, t.Key); ok {
		info.OwnerMob = id
	}
	shape(mob, t.Char.Level, g, info)
	mob.Character.Health = max(1, mob.Character.HealthLimit()-rec.Damage)
	mob.Character.Mana = 0
	mob.Character.CharmAsCompanion(t.Leader, -2, characters.CharmExpiredRevert)
	lu.Character.TrackCharmed(mob.InstanceId, true)
	room.AddMob(mob.InstanceId)
	company.RegisterBeast(mob.InstanceId, t.Leader, t.Key)
	if spawnHook != nil {
		spawnHook(mob)
	}
	return mob, nil
}

// spawnHook lets a test shape a beast as it stands (for example, make it hard
// to kill).
var spawnHook func(*mobs.Mob)

// UseSpawnHookForTest runs fn on every beast as it stands, until the
// returned function restores the old hook.
func UseSpawnHookForTest(fn func(*mobs.Mob)) (restore func()) {
	old := spawnHook
	spawnHook = fn
	return func() { spawnHook = old }
}

// Sync writes a live beast's health back to its Tamer's record. A beast at 0
// health is wounded: out of battles until the company rests.
func Sync(instanceID int) {
	e, ok := company.BeastOf(instanceID)
	if !ok {
		return
	}
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return
	}
	t, ok := Of(e.Leader, e.Owner)
	if !ok || t.Char.Beast == nil {
		return
	}
	rec := t.Char.Beast
	limit := max(1, mob.Character.HealthLimit())
	if mob.Character.Health < 1 {
		rec.Wounded, rec.Damage = true, limit
	} else {
		rec.Damage = max(0, limit-mob.Character.Health)
	}
}

// Dismiss takes a beast away: its record is brought up to date first, then
// its mob leaves its room, its leader's charmed list, the registry and the
// world. It is safe to call twice.
func Dismiss(instanceID int) {
	Sync(instanceID)
	if e, ok := company.BeastOf(instanceID); ok {
		if lu := users.GetByUserId(e.Leader); lu != nil && lu.Character != nil {
			lu.Character.TrackCharmed(instanceID, false)
		}
	}
	company.ForgetBeast(instanceID)
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

// DismissAll removes every beast a leader has standing.
func DismissAll(leader int) {
	for _, id := range company.BeastInstances() {
		if e, ok := company.BeastOf(id); ok && e.Leader == leader {
			Dismiss(id)
		}
	}
}

// IsBeast reports whether a mob is a live bonded beast.
func IsBeast(m *mobs.Mob) bool {
	return m != nil && m.Character.RT != nil && m.Character.RT.Beast != nil
}

// HealthLimit is the health a Tamer's beast would have, whole.
func HealthLimit(t *characters.Character) int {
	g := GiftsFor(t.ClassEffects())
	probe := characters.Character{Level: max(t.Level, 1), HPArchetype: "warrior", RaceId: RaceID} // the beast race's stats, as the live mob has
	probe.RT = &characters.ClassRT{Beast: &characters.BeastInfo{HPPct: g.HPPctAt(t.Level)}}
	probe.RecalculateStats()
	return max(1, probe.HealthLimit())
}

// Condition is a beast's state in a short word: whole, bruised, badly hurt
// or wounded.
func Condition(b characters.BeastState, limit int) string {
	switch {
	case b.Wounded:
		return "wounded"
	case b.Damage < 1:
		return "whole"
	case b.Damage*2 >= limit:
		return "badly hurt"
	}
	return "bruised"
}

// NeedsRest reports whether a Tamer's beast is hurt or wounded.
func NeedsRest(c *characters.Character) bool {
	return c != nil && c.Beast != nil && (c.Beast.Wounded || c.Beast.Damage > 0)
}

// Recover is a rest: the beast's wounds close and its health comes back. It
// reports whether anything changed.
func Recover(c *characters.Character) bool {
	if !NeedsRest(c) {
		return false
	}
	c.Beast.Wounded, c.Beast.Damage = false, 0
	return true
}

// BiteBonus is the damage a Tamer's level adds to its beast's bite: the
// beast grows with its Tamer, so it starts small and keeps up late
// (review tuning: a flat 1d8 bite made the Tamer far ahead at level 5 and
// behind at 20).
func BiteBonus(level int) int { return max(level, 0) / BitePerLevel }

// BreathDamage is the roll of a drake's Breath for a Tamer of a level.
func BreathDamage(level, bonus, roll int) int {
	return max(1, roll+max(level, 1)/BreathHalf+bonus)
}
