package hooks

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 39d: the Doll Master. A Master's dolls stand in the formation for
// one battle (internal/dolls makes and keeps them). They have no turn of
// their own: the Master's turn is their strike (Puppet Strike), resolved in
// the ability pass, so the company's number of actions never grows. Guard
// String, Tangle and Emergency Splice are resolved here too. All of it is
// runtime only, on the game loop.

// tangledUntil is the combat round through which a foe can't be tangled
// again.
var tangledUntil = map[int]int{}

// dollTurns are the Masters whose turn this round was their dolls' strike:
// they don't swing in either pass.
var dollTurns = map[caster]bool{}

// ResetDollsForTest forgets the pass's runtime state.
func ResetDollsForTest() {
	clear(tangledUntil)
	clear(dollTurns)
}

// dollMasters are the Doll Masters among a leader's side in a battle.
func dollMasters(side []actor) []actor {
	var out []actor
	for _, a := range side {
		if dolls.IsMaster(a.char) && a.char.Health > 0 {
			out = append(out, a)
		}
	}
	return out
}

// dollPass stands every Master's dolls in its battle's room and takes away
// the dolls of a battle that is over (or whose Master has fallen: a doll
// goes limp). It runs at the round's start, right after the battles are
// decided.
func dollPass() {
	clear(dollTurns)
	for foe, round := range tangledUntil {
		if round < abilityRounds {
			delete(tangledUntil, foe)
		}
	}
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
		for _, a := range dollMasters(sideActors(u, room)) {
			master, found := dolls.Of(uid, a.key)
			if !found {
				continue
			}
			if _, err := dolls.Spawn(master); err != nil && err != dolls.ErrNoBattle {
				continue
			}
			for i := range a.char.Dolls {
				if mob, standing := dolls.Live(master, i); standing {
					live[mob.InstanceId] = true
				}
			}
		}
	}
	for _, id := range company.DollInstances() {
		if !live[id] {
			dollGoesLimp(id)
			dolls.Dismiss(id)
		}
	}
	// A doll that fell outside a blow's own round handling (a spell, a
	// burning) breaks or splices here.
	for _, id := range company.DollInstances() {
		if d := mobs.GetInstance(id); d != nil && d.Character.Health < 1 {
			dollFalls(d)
		}
	}
	for _, id := range company.DollInstances() {
		dolls.Sync(id)
	}
}

// dollGoesLimp tells the room when a Master has fallen while its doll still
// stands (Phase 47): the strings it hung from are cut. A doll removed because
// its battle ended, or because it was already broken, says nothing.
func dollGoesLimp(id int) {
	d := mobs.GetInstance(id)
	e, ok := company.DollOf(id)
	if d == nil || !ok || d.Character.Health < 1 {
		return
	}
	master, found := dolls.Of(e.Leader, e.Owner)
	if !found || master.Char.Health > 0 {
		return
	}
	if room := rooms.LoadRoom(d.Character.RoomId); room != nil {
		room.SendText(fmt.Sprintf(`%s goes limp as its Master falls.`, named(mobTag(mobName(id)))))
	}
	emitCombat(combatstream.Event{Kind: combatstream.Death, RoomId: d.Character.RoomId, Target: mobRef(d), Outcome: combatstream.OutcomeIncapacitated})
}

// dismissDolls takes a leader's dolls away: its battle ended.
func dismissDolls(leaderID int) {
	dolls.DismissAll(leaderID)
}

// dollBody is a Master's standing dolls, in formation order, those able to
// strike this turn first.
func dollBody(master dolls.Master) []*mobs.Mob {
	f, _ := company.FormationFor(master.Leader)
	var out []*mobs.Mob
	for i := range master.Char.Dolls {
		mob, ok := dolls.Live(master, i)
		if !ok || mob.Character.Health < 1 {
			continue
		}
		out = append(out, mob)
	}
	order := func(m *mobs.Mob) int {
		if e, ok := company.DollOf(m.InstanceId); ok {
			if r, c, placed := f.Find(company.DollMemberKey(e.Owner, e.Index)); placed {
				return r*company.FormationCols + c
			}
		}
		return company.FormationRows * company.FormationCols
	}
	sort.SliceStable(out, func(i, j int) bool { return order(out[i]) < order(out[j]) })
	return out
}

// dollAble reports whether a doll can strike now.
func dollAble(m *mobs.Mob) bool {
	if m.Character.Health < 1 || !canFight(&m.Character) {
		return false
	}
	if _, lost := status.LostAction(&m.Character); lost {
		return false
	}
	return !status.Grounded(&m.Character)
}

// dollStrike is a Doll Master's turn: its dolls strike in its place. It
// reports whether the turn was taken (the Master then does not swing); a
// Master with no doll able to strike swings itself.
func dollStrike(a actor, u *users.UserRecord, foes map[int]bool, room *rooms.Room) bool {
	if !dolls.IsMaster(a.char) {
		return false
	}
	if tempoActive && tempoTurns[a.who] == 0 {
		return false
	}
	foe, ok := abilityFoe(a, u, foes)
	if !ok {
		return false
	}
	master, found := dolls.Of(u.UserId, a.key)
	if !found {
		return false
	}
	body := dollBody(master)
	able := body[:0:0]
	for _, m := range body {
		if dollAble(m) {
			able = append(able, m)
		}
	}
	if len(able) == 0 {
		return false
	}
	rt := a.char.RTState()
	if rt.SpliceTurn { // Emergency Splice: the Master spends this turn mending
		rt.SpliceTurn = false
		dollTurns[a.who] = true
		a.holder.say(`You spend your turn knitting the splice tight.`, `%s works the splice tight.`, ` (splice)`)
		return true
	}
	turns := 1
	if tempoActive && tempoTurns[a.who] > 1 {
		turns = tempoTurns[a.who]
	}
	dollTurns[a.who] = true
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(foe), Status: `Puppet Strike`})
	tangled := false
	for t := 0; t < turns; t++ {
		for _, d := range able {
			if !dollAble(d) {
				continue
			}
			target := dollTarget(d, master, foe, foes, room)
			if target == nil {
				continue
			}
			r := extraBlow(dollActor(d, master), target, room, 100)
			if r.Hit && !tangled && target.Character.Health > 0 {
				tangled = tangleFoes(a, d, target, foes, room)
			}
			if r.Hit && target.Character.Health > 0 {
				golemKnock(a, d, target)
			}
		}
	}
	return true
}

// dollActor is a doll as the blow code's actor.
func dollActor(d *mobs.Mob, master dolls.Master) actor {
	key := company.MemberKey("")
	if e, ok := company.DollOf(d.InstanceId); ok {
		key = company.DollMemberKey(e.Owner, e.Index)
	}
	return actor{
		who:    caster{mobId: d.InstanceId},
		char:   &d.Character,
		ref:    mobRef(d),
		key:    key,
		holder: mobHolder(d),
	}
}

// dollTarget is the foe a doll's blow lands on: the Master's foe when the
// doll's reach gets there, else the foe the front row puts in the way, else
// the first standing foe it can reach. nil when it reaches none.
func dollTarget(d *mobs.Mob, master dolls.Master, aim *mobs.Mob, foes map[int]bool, room *rooms.Room) *mobs.Mob {
	col := 1
	f, _ := company.FormationFor(master.Leader)
	if e, ok := company.DollOf(d.InstanceId); ok {
		if _, c, placed := f.Find(company.DollMemberKey(e.Owner, e.Index)); placed {
			col = c
		}
	}
	reach := combat.ResolveReach(&d.Character, false)
	ids := make([]int, 0, len(foes))
	for id := range foes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	try := func(id int) *mobs.Mob {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn || m.Character.HasBuffFlag("hidden") {
			return nil
		}
		t, ok := resolveEnemyAttack(col, id, room, reach)
		if !ok || t == nil || t.Character.Health < 1 {
			return nil
		}
		return t
	}
	if aim != nil && aim.Character.Health > 0 {
		if t := try(aim.InstanceId); t != nil {
			return t
		}
	}
	for _, id := range ids {
		if t := try(id); t != nil {
			return t
		}
	}
	return nil
}

// tangleFoes is Tangle: strings snag the foe a doll struck, and up to
// TangleFoes - 1 others of the battle, pushing each one's action meter
// back. Once ready it cools down; a foe can't be tangled again for
// TangleHold rounds.
func tangleFoes(a actor, d *mobs.Mob, foe *mobs.Mob, foes map[int]bool, room *rooms.Room) bool {
	fx := a.char.ClassEffects()
	if !fx.Has(classes.DollTangle) {
		return false
	}
	key := abilityKey{who: a.who, id: strategy.Tangle}
	if _, waiting := abilityReady[key]; waiting {
		return false
	}
	if round, held := tangledUntil[foe.InstanceId]; held && round >= abilityRounds {
		return false
	}
	n := max(1, fx.Int(classes.TangleFoes))
	targets := []*mobs.Mob{foe}
	if n > 1 {
		ids := make([]int, 0, len(foes))
		for id := range foes {
			if id != foe.InstanceId {
				ids = append(ids, id)
			}
		}
		sort.Ints(ids)
		for _, id := range ids {
			if len(targets) >= n {
				break
			}
			m := mobs.GetInstance(id)
			if m == nil || m.Character.Health < 1 || m.Character.CombatWithdrawn || m.Character.HasBuffFlag("hidden") {
				continue
			}
			if round, held := tangledUntil[id]; held && round >= abilityRounds {
				continue
			}
			targets = append(targets, m)
		}
	}
	spec, _ := strategy.SpecOf(strategy.Tangle)
	abilityReady[key] = abilityRounds + max(1, spec.Cooldown-fx.Int(classes.TangleCD))
	for _, t := range targets {
		push := dolls.TanglePush + fx.Int(classes.TanglePush)
		if t.Boss {
			push = dolls.BossPush
		}
		tangledUntil[t.InstanceId] = abilityRounds + dolls.TangleHold
		if st := tempoMeters[caster{mobId: t.InstanceId}]; st != nil {
			st.meter.Points -= float64(push)
		}
		suffix := ` (tangled)`
		// A String Sovereign's Cut strings leave the foe's next attack weaker.
		if cut := fx.Int(classes.TangleWeak); cut > 0 {
			t.Character.RTState().Cut = cut
			suffix = fmt.Sprintf(` (tangled, -%d Attack on its next attack)`, cut)
		}
		a.holder.say(fmt.Sprintf(`Strings snag %s and drag at it.`, mobHolder(t).tag()),
			`Strings from %s snag `+verbatim(mobHolder(t).tag())+`.`, suffix)
		emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: room.RoomId, Source: a.ref, Target: mobRef(t), Status: `Tangle`})
	}
	return true
}

// dollKnockRoll rolls a golem's knockdown; tests replace it.
var dollKnockRoll = util.Rand

// UseDollKnockRollForTest replaces the golem's knockdown dice until the
// returned restore is called.
func UseDollKnockRollForTest(roll func(int) int) (restore func()) {
	prev := dollKnockRoll
	dollKnockRoll = roll
	return func() { dollKnockRoll = prev }
}

// golemKnock is a Golem Lord's Hammering blows: a blow its doll lands may
// knock the foe down (a boss, half as often).
func golemKnock(master actor, d *mobs.Mob, foe *mobs.Mob) {
	chance := master.char.ClassEffects().Int(classes.DollKnock)
	if chance <= 0 || foe.Character.Health < 1 || status.Live(&foe.Character, status.KnockedDown) {
		return
	}
	if foe.Boss {
		chance /= 2
	}
	if dollKnockRoll(100) >= chance {
		return
	}
	events.AddToQueue(events.Buff{MobInstanceId: foe.InstanceId, BuffId: status.KnockedDown, Source: `combat`})
	master.holder.say(fmt.Sprintf(`The doll's blow knocks %s down.`, mobHolder(foe).tag()),
		`The doll's blow knocks `+verbatim(mobHolder(foe).tag())+` down.`, ` (knocked down)`)
}

// dollMasterOf is the character that drives a doll, nil when it is gone.
func dollMasterOf(c *characters.Character) *characters.Character {
	if c == nil || c.RT == nil || c.RT.Doll == nil {
		return nil
	}
	d := c.RT.Doll
	if d.OwnerUser > 0 {
		if u := users.GetByUserId(d.OwnerUser); u != nil {
			return u.Character
		}
		return nil
	}
	if m := mobs.GetInstance(d.OwnerMob); m != nil {
		return &m.Character
	}
	return nil
}

// dollGuardsLeft is the Guard String uses a doll's Master has left this
// battle; 0 for anything but a doll.
func dollGuardsLeft(c *characters.Character) int {
	master := dollMasterOf(c)
	if master == nil || master.Health < 1 {
		return 0
	}
	return max(0, dolls.GuardUses(master.ClassEffects())-master.RTState().DollGuards)
}

// dollFalls is a doll whose health ran out: Emergency Splice stands it back
// up once a battle, else it breaks and leaves the battle (its record keeps
// the break and its gear until a camp mends it).
func dollFalls(d *mobs.Mob) {
	room := rooms.LoadRoom(d.Character.RoomId)
	e, ok := company.DollOf(d.InstanceId)
	var master dolls.Master
	found := false
	if ok {
		master, found = dolls.Of(e.Leader, e.Owner)
	}
	if found && master.Char.Health > 0 {
		fx := master.Char.ClassEffects()
		rt := master.Char.RTState()
		if fx.Has(classes.Splice) && !rt.Spliced {
			rt.Spliced, rt.SpliceTurn = true, true
			d.Character.Health = max(1, d.Character.HealthLimit()*dolls.SplicePct/100)
			if room != nil {
				room.SendText(fmt.Sprintf(`%s's strings snap taut and it lurches back to its feet.`, named(mobTag(mobName(d.InstanceId)))))
			}
			emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: d.Character.RoomId, Target: mobRef(d), Status: `Emergency Splice`, Outcome: combatstream.OutcomeSucceeded})
			return
		}
		// Phase 39i: a Golem Lord's golem stands a second time, at half its
		// health, and the Master spends no turn on it.
		if fx.Has(classes.SplicePlus) && !rt.SplicedAgain {
			rt.SplicedAgain = true
			d.Character.Health = max(1, d.Character.HealthLimit()/2)
			if room != nil {
				room.SendText(fmt.Sprintf(`%s grinds back up from the rubble, whole again.`, named(mobTag(mobName(d.InstanceId)))))
			}
			emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: d.Character.RoomId, Target: mobRef(d), Status: `Rise Again`, Outcome: combatstream.OutcomeSucceeded})
			return
		}
	}
	emitCombat(combatstream.Event{Kind: combatstream.Death, RoomId: d.Character.RoomId, Target: mobRef(d), Outcome: combatstream.OutcomeIncapacitated})
	if room != nil {
		room.SendText(fmt.Sprintf(`%s's strings go slack and it clatters apart.`, named(mobTag(mobName(d.InstanceId)))))
	}
	d.DeathProcessed = true
	dolls.Dismiss(d.InstanceId)
}

// sideHasDoll reports whether an instance is a live doll (for the passes
// that walk a company's mobs and must leave a doll alone).
func isDollInstance(instanceID int) bool {
	_, ok := company.DollOf(instanceID)
	return ok
}
