package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
)

// Phase 38b: class auras. Each combat round, before any blow, a company
// member who has fallen gives none; a standing holder gives the allies in
// its row its Evasion (Watchful row) and damage reduction (Aura of
// Resolve, Shelter), and the rest of the company its Rallying share.
// Auras of one kind don't stack: a row takes the best.

// auraPass sets every company member's aura for the round.
func auraPass() {
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || u == nil || u.Character == nil || u.Character.RoomId != b.RoomId {
			continue
		}
		room := rooms.LoadRoom(b.RoomId)
		if room == nil {
			continue
		}
		f, _ := company.FormationFor(uid)
		side := sideActors(u, room)
		applyAuras(uid, side, f)
		cry, cryer := battleCry(side, b, combatRound.Load())
		for _, a := range side {
			a.char.Aura.Attack = cry
		}
		announceCry(cryer, cry, b, combatRound.Load())
		for _, a := range side {
			if a.char.ClassEffects() != nil {
				a.char.RTState()
			}
		}
		rejuvPass(side)
		samuraiRound(side)
	}
}

// applyAuras computes the round's auras for a side from its standing
// holders and its formation, replacing last round's.
func applyAuras(uid int, side []actor, f company.Formation) {
	rowOf := func(a actor) int {
		r, _, placed := f.Find(a.key)
		if !placed {
			return -1
		}
		return r
	}
	evade := map[int]int{}
	resolve := map[int]int{}
	rally := 0
	// Phase 39c: a Mistweaver's Veil of mist, while its Fog lasts, covers
	// the whole company.
	fogged := battle.WeatherOf(uid).Kind == stormcraft.Fog
	fogEvade := 0
	for _, a := range side {
		// An Angel's wings cover the owner's row.
		if sm := summonInfo(a.char); sm != nil && sm.Wings > 0 && a.char.Health >= 1 {
			if r := ownerRow(sm, f); r >= 0 {
				evade[r] = max(evade[r], sm.Wings)
			}
		}
		fx := a.char.ClassEffects()
		if fx == nil || a.char.Health < 1 {
			continue
		}
		row := rowOf(a)
		rally = max(rally, fx.Int(classes.AuraCompan))
		if fogged {
			fogEvade = max(fogEvade, fx.Int(classes.FogEvade))
		}
		if row < 0 {
			continue // an unplaced member holds no row
		}
		evade[row] = max(evade[row], fx.Int(classes.AuraEvade))
		resolve[row] = max(resolve[row], fx.Int(classes.AuraResolv))
	}
	for _, a := range side {
		row := rowOf(a)
		a.char.Aura = characters.ClassAura{
			Evasion: max(evade[row], fogEvade),
			Resolve: max(resolve[row], rally),
		}
		// A Knight guarding a ward holds its shield the firmer.
		if fx := a.char.ClassEffects(); fx.Has(classes.FaithBlock) && enemyparty.MemberStrategy(uid, a.key).Role == strategy.Guardian {
			a.char.Aura.Block = fx.Int(classes.FaithBlock)
		}
	}
}

// rejuvPass heals each ally that carries Rejuvenation by one round's share
// of it, and counts the round off (and a round of Bless). Healing stops at the wound limit.
func rejuvPass(side []actor) {
	for _, a := range side {
		rt := a.char.RT
		if rt == nil {
			continue
		}
		if rt.Bless > 0 {
			rt.Bless-- // Phase 38b review: a Bless lasts its rounds, not the battle
		}
		if rt.Rejuv <= 0 {
			continue
		}
		rt.Rejuv--
		if a.char.Health < 1 {
			continue
		}
		healed := a.char.ApplyHealthChange(rt.Per)
		if a.who.userId > 0 {
			events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
		}
		if healed > 0 {
			a.holder.say(fmt.Sprintf("Rejuvenation mends you. (%d healed)", healed),
				"Rejuvenation mends %s. ("+fmt.Sprint(healed)+" healed)", "")
		}
	}
}

// thornsBlow hurts a foe that struck an ally under Barkskin's Thornhide.
func thornsBlow(attacker, defender statusHolder, r combat.AttackResult) {
	if defender.char.RT == nil || defender.char.RT.Thorns <= 0 || !r.Hit || r.DamageToTarget <= 0 || attacker.char.Health < 1 {
		return
	}
	dealt := -attacker.char.ApplyHealthChange(-defender.char.RT.Thorns)
	if dealt <= 0 {
		return
	}
	attacker.say(fmt.Sprintf("Thorns tear at you. (%d damage)", dealt), "Thorns tear at %s. ("+fmt.Sprint(dealt)+" damage)", "")
	if attacker.user != nil {
		roundExtraPlayers = append(roundExtraPlayers, attacker.user.UserId)
		events.AddToQueue(events.CharacterVitalsChanged{UserId: attacker.user.UserId})
	} else {
		roundExtraMobs = append(roundExtraMobs, attacker.mob.InstanceId)
	}
}

// oathBlow is a Blackguard's Blood Oath: a melee blow that lands heals the
// most hurt ally (itself when no one is hurt) for a share of the damage,
// for so many blows a battle, and leaves the foe cowed (Intimidation).
func oathBlow(attacker, defender statusHolder, r combat.AttackResult) {
	fx := attacker.char.ClassEffects()
	if !fx.Has(classes.BloodOath) || !r.Hit || r.DamageToTarget <= 0 || attacker.char.Health < 1 {
		return
	}
	if weaponType(attacker.char) == string(items.Shooting) {
		return
	}
	rt := attacker.char.RTState()
	// The wounded foe turns from the Blackguard's allies.
	if n := fx.Int(classes.Intimidate); n > 0 && defender.char.Health >= 1 {
		foe := defender.char.RTState()
		foe.Intim, foe.IntimOwner, foe.IntimRound = n, rt, combatRound.Load()
		intimidated = append(intimidated, foe)
	}
	if rt.OathUsed >= fx.Int(classes.BloodOath) {
		return
	}
	self := scripting.GetActor(attacker.ref.UserId, attacker.ref.MobInstanceId)
	if self == nil {
		return
	}
	heal := max(1, r.DamageToTarget*max(fx.Int(classes.OathPct), 50)/100)
	ally := self.MostHurtAlly(false)
	if ally == nil {
		ally = self
	}
	healed := ally.AddHealth(heal)
	if healed <= 0 {
		return
	}
	rt.OathUsed++
	line := fmt.Sprintf(" (blood oath, %d healed, %d left)", healed, fx.Int(classes.BloodOath)-rt.OathUsed)
	attacker.say("Your oath drinks the blood and gives it to "+ally.GetCombatName(false)+"."+line, "%s's oath drinks the blood and gives it to "+verbatim(ally.GetCombatName(false))+"."+line, "")
	if fx.Has(classes.OathSecond) {
		for _, other := range self.HurtAllies(false) {
			if other.UserId() != ally.UserId() || other.InstanceId() != ally.InstanceId() {
				other.AddHealth(max(1, healed/2))
				break
			}
		}
	}
}

// intimidated are the foes a Blackguard has cowed; expireIntimidation
// lifts each once the round after its wound is over.
var intimidated []*characters.ClassRT

// expireIntimidation runs at a round's start: Intimidation lasts the round
// the foe was wounded in and the next (so the foe's next turn always feels
// it, whichever side moves first), not the whole battle.
func expireIntimidation(round uint64) {
	keep := intimidated[:0]
	for _, rt := range intimidated {
		if rt.Intim > 0 && round >= rt.IntimRound && round <= rt.IntimRound+1 {
			keep = append(keep, rt)
			continue
		}
		rt.Intim, rt.IntimOwner = 0, nil
	}
	clear(intimidated[len(keep):])
	intimidated = keep
}

// layHands is a Knight's Lay on Hands, taking its whole turn: it heals the
// most hurt of itself and the allies next to it (any ally with Far hands)
// below the company's healing threshold for half a Minor Heal of its level
// (a full one for a Paladin), with no chant and no mana, so many times
// between rests. It reports whether it used the turn.
func layHands(a actor, side []actor, u *users.UserRecord, f company.Formation) bool {
	fx := a.char.ClassEffects()
	uses := fx.Int(classes.LayHands)
	if uses < 1 || a.char.Health < 1 || tempoActive && tempoTurns[a.who] == 0 {
		return false
	}
	if enemyparty.MemberStrategy(u.UserId, a.key).NoAbilities || !readyToCast(a, u) {
		return false
	}
	if agg := a.char.Aggro; agg != nil && agg.Type == characters.Retreat {
		return false
	}
	rt := a.char.RTState()
	if rt.Hands >= uses {
		return false
	}
	below := strategy.TacticsFor(u.UserId).Healing
	if below <= 0 {
		below = strategy.DefaultHealing
	}
	myRow, myCol, placed := f.Find(a.key)
	best, bestFrac := -1, 1000
	for i, t := range side {
		if t.char.Health < 1 && !(t.who.userId > 0 && t.char.Health < 1) {
			continue
		}
		limit := max(1, t.char.HealthLimit())
		if t.char.Health*100 >= below*limit || t.char.Health >= limit {
			continue
		}
		if t.who != a.who && !fx.Has(classes.LayReach) && placed {
			row, col, ok := f.Find(t.key)
			if ok && (abs(row-myRow) > 1 || abs(col-myCol) > 1) {
				continue
			}
		}
		if frac := t.char.Health * 1000 / limit; frac < bestFrac {
			best, bestFrac = i, frac
		}
	}
	if best < 0 {
		return false
	}
	patient := side[best]
	amount := minorHealRoll(a.char, fx.Has(classes.LayFull))
	healed := patient.char.ApplyHealthChange(amount)
	rt.Hands++
	abilityTurns[a.who] = true
	if patient.who.userId > 0 {
		events.AddToQueue(events.CharacterVitalsChanged{UserId: patient.who.userId})
	}
	extra := ""
	if fx.Has(classes.LayFull) {
		if word := status.CleanseOne(patient.char); word != "" {
			extra = ", " + word + " removed"
		}
	}
	if fx.Has(classes.LayBleed) && status.Live(patient.char, status.Bleeding) {
		patient.char.RemoveBuff(status.Bleeding)
		extra += ", bleeding stopped"
	}
	suffix := fmt.Sprintf(" (lay on hands, %d healed%s, %d left)", healed, extra, uses-rt.Hands)
	you, other := "You lay your hands on "+patient.holder.tag()+", and the wounds close a little."+suffix, "%s lays hands on "+verbatim(patient.holder.tag())+", and the wounds close a little."+suffix
	if patient.who == a.who {
		you, other = "You lay your hands on your own wounds."+suffix, "%s lays hands on "+"their own wounds."+suffix
	}
	a.holder.say(you, other, "")
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Target: patient.ref, Status: "Lay on Hands", Outcome: combatstream.OutcomeSucceeded})
	return true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// minorHealRoll is one Minor Heal's worth for a healer of the character's
// level and Mysticism (half of it unless full), with its healing bonus.
func minorHealRoll(c *characters.Character, full bool) int {
	amount := 1
	if sp := spells.GetSpell("heal"); sp != nil && sp.Power != nil {
		amount = int(sp.Power.Raw(c.Level, c.Stats.Mysticism.ValueAdj, util.Rand))
	}
	if !full {
		amount /= 2
	}
	return max(1, amount*(100+c.HealingBonusPct())/100)
}
