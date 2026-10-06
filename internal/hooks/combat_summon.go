package hooks

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/summons"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 38b: a Hierarch's Angel and a Demonologist's Demon. A summon fights
// as a company member (it is charmed to the leader and found through
// company.LeaderAndKeyForInstance) and lives for one battle. This pass gives
// its gifts each round, before any blow: Mercy and Cleansing light for the
// Angel; Dread, Hellfire and the broken binding for the Demon.

// summonInfo is a character's summon state, nil for anything but a summon.
func summonInfo(c *characters.Character) *characters.SummonInfo {
	if c == nil || c.RT == nil {
		return nil
	}
	return c.RT.Summon
}

// ownerRow is the formation row of a summon's owner, -1 when it holds none.
func ownerRow(sm *characters.SummonInfo, f company.Formation) int {
	key := company.LeaderMemberKey
	if sm.OwnerUser == 0 {
		_, k, ok := company.LeaderAndKeyForInstance(sm.OwnerMob)
		if !ok {
			return -1
		}
		key = k
	}
	if r, _, placed := f.Find(key); placed {
		return r
	}
	return -1
}

// summonOwner is the character that called a summon, nil when it is gone.
func summonOwner(sm *characters.SummonInfo) *characters.Character {
	if sm.OwnerUser > 0 {
		if u := users.GetByUserId(sm.OwnerUser); u != nil {
			return u.Character
		}
		return nil
	}
	if m := mobs.GetInstance(sm.OwnerMob); m != nil {
		return &m.Character
	}
	return nil
}

// summonPass runs every live summon's round, and dismisses the ones whose
// battle is over.
func summonPass() {
	live := map[int]bool{}
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
		side := sideActors(u, room)
		for _, a := range side {
			sm := summonInfo(a.char)
			if sm == nil || a.who.mobId == 0 {
				continue
			}
			live[a.who.mobId] = true
			summonRound(u, a, side, room)
		}
	}
	for _, id := range company.SummonInstances() {
		if !live[id] {
			summons.Dismiss(id)
		}
	}
}

// dismissSummons removes a leader's summons: its battle ended.
func dismissSummons(leaderID int) {
	for _, id := range company.SummonInstances() {
		if l, _, ok := company.SummonOf(id); ok && l == leaderID {
			summons.Dismiss(id)
		}
	}
}

func summonRound(u *users.UserRecord, a actor, side []actor, room *rooms.Room) {
	sm := summonInfo(a.char)
	owner := summonOwner(sm)
	if owner == nil {
		summons.Dismiss(a.who.mobId)
		return
	}
	if sm.Kind == summons.Demon && owner.Health < 1 {
		brokenBinding(a, sm, side)
		return
	}
	if owner.Health < 1 { // the Angel departs when its Hierarch falls
		a.holder.say("", "%s bows its head and rises away in a column of light.", "")
		summons.Dismiss(a.who.mobId)
		return
	}
	if !canFight(a.char) {
		return
	}
	if !sm.Arrived {
		sm.Arrived = true
		arrival(u, a, sm, owner, side, room)
	}
	switch sm.Kind {
	case summons.Angel:
		mercy(a, sm, side)
	case summons.Demon:
		hellfire(a, sm, room)
	}
}

// arrival gives the gifts a summon brings with it.
func arrival(u *users.UserRecord, a actor, sm *characters.SummonInfo, owner *characters.Character, side []actor, room *rooms.Room) {
	if sm.Cleanse {
		var names []string
		for _, t := range side {
			if t.char.Health < 1 {
				continue
			}
			if word := status.CleanseOne(t.char); word != "" {
				names = append(names, verbatim(t.holder.tag())+" ("+word+")")
			}
		}
		if len(names) > 0 {
			a.holder.say("", "%s cleanses the company with a flash of light. (cleansing light: "+verbatim(joinNames(names))+")", "")
		}
	}
	if sm.Dread > 0 {
		dread(u, a, sm)
	}
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += " · "
		}
		out += n
	}
	return out
}

// dread lands Dread on the foes the Demon frightens: its target's group
// (Bind the Fiend), or every enemy group (Terror).
func dread(u *users.UserRecord, a actor, sm *characters.SummonInfo) {
	caster := scripting.GetActor(sm.OwnerUser, sm.OwnerMob)
	if caster == nil {
		return
	}
	room := rooms.LoadRoom(a.char.RoomId)
	if room == nil {
		return
	}
	var targets []int
	if sm.Dread >= 2 {
		for _, p := range enemyparty.Parties(room) {
			targets = append(targets, standingFoes(enemyparty.Group{Party: p}, room)...)
		}
	} else if b, ok := battle.Current(u.UserId); ok {
		if p, found := battleParty(b, enemyparty.Parties(room)); found {
			targets = standingFoes(enemyparty.Group{Party: p}, room)
		}
	}
	scared := 0
	for _, id := range targets {
		foe := scripting.GetActor(0, id)
		if foe == nil {
			continue
		}
		if res := caster.CastHex("dread", *foe); res["landed"] == true {
			scared++
		}
	}
	if scared > 0 {
		a.holder.say("", "%s bares its fangs, and dread falls over the foe. (dread, %d frightened)", "")
	}
}

// mercy: every few rounds the Angel heals the most hurt ally (the two most
// hurt with Swift Host) for half a Minor Heal, or a full one with Mercy.
func mercy(a actor, sm *characters.SummonInfo, side []actor) {
	sm.MercyNext++
	if sm.MercyNext < max(1, sm.MercyEvery) {
		return
	}
	var hurt []actor
	for _, t := range side {
		if t.char.Health >= 1 && t.char.Health < t.char.HealthLimit() {
			hurt = append(hurt, t)
		}
	}
	if len(hurt) == 0 {
		return // Mercy waits for someone to need it
	}
	sort.SliceStable(hurt, func(i, j int) bool {
		return hurt[i].char.Health*1000/max(1, hurt[i].char.HealthLimit()) < hurt[j].char.Health*1000/max(1, hurt[j].char.HealthLimit())
	})
	n := 1
	if sm.MercyTwo {
		n = 2
	}
	sm.MercyNext = 0
	for i := 0; i < n && i < len(hurt); i++ {
		t := hurt[i]
		healed := t.char.ApplyHealthChange(minorHealRoll(a.char, sm.MercyFull))
		if t.who.userId > 0 {
			events.AddToQueue(events.CharacterVitalsChanged{UserId: t.who.userId})
		}
		if healed > 0 {
			t.holder.say(fmt.Sprintf("The Angel's mercy closes your wounds. (mercy, %d healed)", healed),
				"The Angel's mercy closes %s's wounds. ("+fmt.Sprint(healed)+" mercy healed)", "")
		}
	}
}

// hellfire burns every foe standing against the company.
func hellfire(a actor, sm *characters.SummonInfo, room *rooms.Room) {
	if sm.Hellfire < 1 {
		return
	}
	u := users.GetByUserId(sm.OwnerUser)
	if u == nil {
		if l, _, ok := company.LeaderAndKeyForInstance(sm.OwnerMob); ok {
			u = users.GetByUserId(l)
		}
	}
	if u == nil {
		return
	}
	b, ok := battle.Current(u.UserId)
	if !ok {
		return
	}
	p, found := battleParty(b, enemyparty.Parties(room))
	if !found {
		return
	}
	burned := 0
	for _, id := range standingFoes(enemyparty.Group{Party: p}, room) {
		foe := mobs.GetInstance(id)
		if foe == nil {
			continue
		}
		if -foe.Character.ApplyHealthChange(-sm.Hellfire) > 0 {
			burned++
			roundExtraMobs = append(roundExtraMobs, id)
		}
	}
	if burned > 0 {
		a.holder.say("", fmt.Sprintf("Hellfire licks around %%s. (hellfire, %d damage to %d foes)", sm.Hellfire, burned), "")
	}
}

// brokenBinding: the Demon's owner fell and its binding broke. It tears at
// the nearest ally once, then is gone. A mastered binding just lets it go.
func brokenBinding(a actor, sm *characters.SummonInfo, side []actor) {
	if !sm.Mastered {
		for _, t := range side {
			if t.who.mobId == a.who.mobId || t.char.Health < 1 {
				continue
			}
			dmg := max(1, util.RollDice(max(1, sm.Dice), max(1, sm.Sides)))
			dealt := -t.char.ApplyHealthChange(-dmg)
			if t.who.userId > 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: t.who.userId})
			}
			t.holder.say(fmt.Sprintf("The freed Demon rakes you. (broken binding, %d damage)", dealt),
				"The freed Demon rakes %s. (broken binding, "+fmt.Sprint(dealt)+" damage)", "")
			break
		}
	}
	a.holder.say("", "%s snarls and drops back into the dark.", "")
	summons.Dismiss(a.who.mobId)
}

// SoulFeast is the Demonologist's Soul feast: a foe that falls while a
// Demon with the gift fights heals the most hurt ally half a Minor Heal and
// gives its owner 5% of its mana back.
func SoulFeast(e events.Event) events.ListenerReturn {
	d, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}
	if _, _, isCompany := company.LeaderAndKeyForInstance(d.InstanceId); isCompany {
		return events.Continue
	}
	for _, id := range company.SummonInstances() {
		m := mobs.GetInstance(id)
		sm := summonInfo(characterOf(m))
		if sm == nil || !sm.Feast || m.Character.Health < 1 || m.Character.RoomId != d.RoomId {
			continue
		}
		owner := summonOwner(sm)
		if owner == nil {
			continue
		}
		if u := users.GetByUserId(sm.OwnerUser); u != nil || sm.OwnerMob > 0 {
			owner.Mana = min(owner.ManaMax.Value, owner.Mana+max(1, owner.ManaMax.Value*5/100))
			if sm.OwnerUser > 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: sm.OwnerUser})
			}
		}
		if self := scripting.GetActor(0, id); self != nil {
			if ally := self.MostHurtAlly(false); ally != nil {
				ally.AddHealth(max(1, minorHealRoll(&m.Character, false)))
			}
		}
	}
	return events.Continue
}

func characterOf(m *mobs.Mob) *characters.Character {
	if m == nil {
		return nil
	}
	return &m.Character
}
