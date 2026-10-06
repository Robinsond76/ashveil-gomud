// Package summons calls and dismisses a Hierarch's Angel and a
// Demonologist's Demon (Phase 38b). A summon is an ordinary mob made at its
// caller's level, charmed to the company's leader and registered with
// internal/company as a member no formation holds, so the combat code finds
// it by the paths it finds companions. It lives for one battle: nothing
// about it is saved, and the fight's end (or a sweep) dismisses it.
package summons

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

// Kinds of summon.
const (
	Angel = "angel"
	Demon = "demon"
)

// Mob templates of the summons (_datafiles/world/default/mobs/summons).
const (
	AngelMobID = 95
	DemonMobID = 96
)

// Why a call fails.
var (
	ErrAlready  = fmt.Errorf("summons: already called this battle")
	ErrNoRoom   = fmt.Errorf("summons: no room to call it")
	ErrNoLead   = fmt.Errorf("summons: no company to call it for")
	ErrUnknown  = fmt.Errorf("summons: no such summon")
	ErrNoBattle = fmt.Errorf("summons: only in battle")
)

// Caller is whoever casts the call: a player, or a company mob.
type Caller struct {
	UserID, MobID int
}

func (c Caller) character() *characters.Character {
	if c.UserID > 0 {
		if u := users.GetByUserId(c.UserID); u != nil {
			return u.Character
		}
		return nil
	}
	if m := mobs.GetInstance(c.MobID); m != nil {
		return &m.Character
	}
	return nil
}

// Info is the summon a caster's route has earned at its level: its share of
// a warrior's health, its weapon and every gift the route's ranks have
// added. symbol is whether the caster holds a holy symbol (+25% health).
func Info(kind string, fx classes.Effects, symbol bool) (characters.SummonInfo, bool) {
	info := characters.SummonInfo{Kind: kind, HPPct: 100}
	switch kind {
	case Angel:
		info.Dice, info.Sides = 1, 8
		if s := fx.Int(classes.AngelBlade); s > 0 {
			info.Sides = s
		}
		if info.Sides >= 10 {
			info.Smite = 50 // Sword of the Host
		}
		info.Guards = fx.Int(classes.AngelGuards)
		info.MercyEvery = 3
		if n := fx.Int(classes.AngelMercy); n > 0 {
			info.MercyEvery = n
		}
		info.MercyFull = fx.Has(classes.AngelMercyFull)
		info.MercyTwo = fx.Has(classes.AngelMercyTwo)
		info.Wings = fx.Int(classes.AngelWings)
		info.Cleanse = fx.Has(classes.AngelCleanse)
	case Demon:
		info.HPPct = 80
		info.Dice, info.Sides = 2, 6
		if s := fx.Int(classes.DemonClaws); s > 0 {
			info.Sides = s
		}
		info.Rend = fx.Int(classes.RendHoly)
		info.Hellfire = fx.Int(classes.DemonHellfire)
		info.Feast = fx.Has(classes.DemonFeast)
		info.Dread = fx.Int(classes.DemonDread)
		info.Mastered = fx.Has(classes.DemonMastered)
	default:
		return info, false
	}
	if symbol {
		info.HPPct = info.HPPct * 125 / 100
	}
	return info, true
}

// Call makes the caller's summon in its room, once a battle. It returns the
// summon's mob.
func Call(by Caller, kind string) (*mobs.Mob, error) {
	c := by.character()
	if c == nil {
		return nil, ErrNoLead
	}
	rt := c.RTState()
	if rt.Summoned {
		return nil, ErrAlready
	}
	leader := by.UserID
	if leader == 0 {
		leader = c.GetCharmedUserId()
	}
	lu := users.GetByUserId(leader)
	if lu == nil || lu.Character == nil {
		return nil, ErrNoLead
	}
	// Battle only (Phase 38b review): a call in peace would be dismissed at
	// once and spend the next battle's call.
	if b, ok := battle.Current(leader); !ok || b.RoomId != c.RoomId {
		return nil, ErrNoBattle
	}
	room := rooms.LoadRoom(c.RoomId)
	if room == nil {
		return nil, ErrNoRoom
	}
	templateID := AngelMobID
	if kind == Demon {
		templateID = DemonMobID
	} else if kind != Angel {
		return nil, ErrUnknown
	}
	symbol := c.Equipment.Offhand.ItemId != 0 && c.Equipment.Offhand.StatMod("healing") > 0
	info, _ := Info(kind, c.ClassEffects(), symbol)
	info.OwnerUser, info.OwnerMob = by.UserID, by.MobID
	info.MercyNext = 0

	mob := mobs.NewMobByIdNoElite(mobs.MobId(templateID), c.RoomId, max(c.Level, 1))
	if mob == nil {
		return nil, ErrUnknown
	}
	mob.Character.HPArchetype = "warrior" // its Attack, Evasion and health gain
	mob.Character.RTState().Summon = &info
	if armor := c.ClassEffects().Int(classes.SummonArmor); armor > 0 {
		mob.Character.RT.Bark = armor
	}
	mob.Character.RecalculateStats()
	mob.Character.Health, mob.Character.Mana = mob.Character.HealthMax.Value, mob.Character.ManaMax.Value

	mob.Character.CharmAsCompanion(leader, -2, characters.CharmExpiredRevert)
	lu.Character.TrackCharmed(mob.InstanceId, true)
	room.AddMob(mob.InstanceId)
	key := company.RegisterSummon(mob.InstanceId, leader, kind)
	mob.Character.RT.Summon.OwnerKey = string(key)
	rt.Summoned = true
	return mob, nil
}

// Dismiss removes a summon: from its room, its leader's charmed list, the
// registry, and the world. It is safe to call twice.
func Dismiss(instanceID int) {
	if leader, _, ok := company.SummonOf(instanceID); ok {
		if lu := users.GetByUserId(leader); lu != nil && lu.Character != nil {
			lu.Character.TrackCharmed(instanceID, false)
		}
	}
	company.ForgetSummon(instanceID)
	m := mobs.GetInstance(instanceID)
	if m == nil {
		return
	}
	m.Character.RemoveCharm()
	if room := rooms.LoadRoom(m.Character.RoomId); room != nil {
		room.RemoveMob(instanceID)
	}
	mobs.DestroyInstance(instanceID)
}

// IsSummon reports whether a mob is a live summon.
func IsSummon(m *mobs.Mob) bool {
	return m != nil && m.Character.RT != nil && m.Character.RT.Summon != nil
}
