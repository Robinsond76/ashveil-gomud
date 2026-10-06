package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/summons"
)

// Phase 38c3: what the wizard and witch elites ask of the spell scripts.

// WardGifts gives the ward just granted to target the extras this caster's
// class adds: Mend Charm, a cleansing break, Hearth's Peace, a Wise One's
// Ward of Life and an Archon's Reflection. A caster with none leaves the
// ward the plain Phase 38b one.
func (a ScriptActor) WardGifts(target *ScriptActor) {
	if a.characterRecord == nil || target == nil || target.characterRecord == nil {
		return
	}
	fx := a.characterRecord.ClassEffects()
	rt := target.characterRecord.RTState()
	rt.WardMend, rt.WardCleanse, rt.WardPeace, rt.WardLifeBy, rt.WardReflectBy = 0, false, 0, nil, nil
	if fx == nil {
		return
	}
	if fx.Has(classes.WardMend) {
		rt.WardMend = max(1, int(a.SpellPower("heal"))/2)
	}
	rt.WardCleanse = fx.Has(classes.WardCleanse)
	rt.WardPeace = fx.Int(classes.WardPeace)
	own := a.characterRecord.RTState()
	if fx.Has(classes.WardLife) && !own.LifeUsed {
		rt.WardLifeBy = own
	}
	if fx.Has(classes.Reflect) && !own.ReflectUsed {
		rt.WardReflectBy = own
	}
}

// WardExtra wards up to ClassEffect(wardextra) more allies than target with
// the same cap and blows, the most hurt without a ward first (an Archon's
// Twin ward), and returns who it warded.
func (a ScriptActor) WardExtra(target *ScriptActor, cap, blows int) []*ScriptActor {
	if a.characterRecord == nil {
		return nil
	}
	n := a.characterRecord.ClassEffects().Int(classes.WardExtra)
	var out []*ScriptActor
	for _, ally := range a.AllAllies(false) {
		if len(out) >= n {
			break
		}
		if target != nil && ally.userId == target.userId && ally.mobInstanceId == target.mobInstanceId {
			continue
		}
		if ally.GrantWard(cap, blows) {
			a.WardGifts(ally)
			out = append(out, ally)
		}
	}
	return out
}

// CastTargets are the foes the actor's chanting spell was aimed at, in
// order (an Archmage's Barrage reaches a second one).
func (a ScriptActor) CastTargets() []*ScriptActor {
	c := a.characterRecord
	if c == nil || c.Aggro == nil {
		return nil
	}
	var out []*ScriptActor
	for _, id := range c.Aggro.SpellInfo.TargetMobInstanceIds {
		if m := GetMob(id); m != nil && m.characterRecord != nil && m.characterRecord.Health > 0 {
			out = append(out, m)
		}
	}
	return out
}

// GiveStatus puts a combat status on the actor for rounds combat rounds
// (Grave Chill's hobble).
func (a ScriptActor) GiveStatus(buffId, rounds int) {
	if a.characterRecord == nil || a.characterRecord.Health < 1 || rounds < 1 {
		return
	}
	events.AddToQueue(events.Buff{UserId: a.userId, MobInstanceId: a.mobInstanceId, BuffId: buffId, Source: `spell`, Triggers: rounds + 1})
}

// PoisonScale is how many times over the actor's poison hurts: 2 when a
// Crone of Ash's Rotting Miasma laid it, else 1.
func (a ScriptActor) PoisonScale() int {
	if a.characterRecord != nil && a.characterRecord.RT != nil && a.characterRecord.RT.PoisonX2 {
		return 2
	}
	return 1
}

// Raise raises the strongest foe that has fallen in the actor's battle as
// its thrall, up to the Raise times its class allows. It returns the
// thrall's name, or an empty string when nothing was raised.
func (a ScriptActor) Raise() string {
	c := a.characterRecord
	if c == nil {
		return ""
	}
	leader := a.userId
	if leader == 0 {
		leader = c.GetCharmedUserId()
	}
	f, ok := summons.TakeFallen(leader)
	if !ok {
		return ""
	}
	mob, err := summons.Raise(summons.Caller{UserID: a.userId, MobID: a.mobInstanceId}, f)
	if err != nil || mob == nil {
		return ""
	}
	return mob.Character.Name
}
