package hooks

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 29b2: one battle at a time. Each player (with their company) fights
// one enemy group at a time. Other groups set on the player hold back until
// that battle is over, then the one set on them first begins the next
// battle; a waiting group turns on another player in the room with no
// battle of their own. internal/battle holds the battles; this file decides
// them each round and keeps both sides to them. The combat event stream
// opens a fight for each battle and ends it with the battle.

// side is a player and the mobs fighting for them (their companions and
// any other mob they have charmed) in the player's room.
type side struct {
	user   *users.UserRecord
	allies map[int]bool
}

func loadSide(u *users.UserRecord, room *rooms.Room) side {
	sd := side{user: u, allies: map[int]bool{}}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		if owner, ok := allyOwner(instanceId); ok && owner == u.UserId {
			sd.allies[instanceId] = true
		}
	}
	return sd
}

// allyOwner is the player a mob fights for: its company leader, or the
// player who charmed it.
func allyOwner(instanceId int) (int, bool) {
	if leaderId, _, ok := company.LeaderAndKeyForInstance(instanceId); ok {
		return leaderId, true
	}
	if m := mobs.GetInstance(instanceId); m != nil && m.Character.Charmed != nil && m.Character.Charmed.UserId > 0 {
		return m.Character.Charmed.UserId, true
	}
	return 0, false
}

// setOn reports whether an enemy group is set on the player: a living
// member here aims at the player or an ally, or the player or an ally aims
// at one of its members.
func (sd side) setOn(party mobparty.Party, room *rooms.Room) bool {
	// Only living members here count: a foe just slain is not yet removed
	// from the room, and an aim at its body sets nothing.
	members := map[int]bool{}
	for _, instanceId := range party.Members {
		if m := mobs.GetInstance(instanceId); m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId {
			members[instanceId] = true
		}
	}
	if len(members) == 0 {
		return false
	}
	if sd.user.Character.Health > 0 && aimsAt(sd.user.Character.Aggro, members) {
		return true
	}
	for instanceId := range sd.allies {
		if m := mobs.GetInstance(instanceId); m != nil && m.Character.Health > 0 && aimsAt(m.Character.Aggro, members) {
			return true
		}
	}
	for _, instanceId := range party.Members {
		m := mobs.GetInstance(instanceId)
		if m == nil || m.Character.Health < 1 || m.Character.RoomId != room.RoomId || m.Character.Aggro == nil {
			continue
		}
		// An aim at a downed player or ally sets nothing: a player who
		// falls isn't drawn into battle after battle while they lie there.
		if m.Character.Aggro.UserId == sd.user.UserId && sd.user.Character.Health > 0 {
			return true
		}
		if ally := m.Character.Aggro.MobInstanceId; sd.allies[ally] {
			if a := mobs.GetInstance(ally); a != nil && a.Character.Health > 0 {
				return true
			}
		}
	}
	return false
}

// setParties lists the groups set on the player, in room order, and
// notes (once) the round each was first set on them.
func (sd side) setParties(parties []mobparty.Party, room *rooms.Room, round uint64) []mobparty.Party {
	var set []mobparty.Party
	var ids []string
	for _, p := range parties {
		if sd.setOn(p, room) {
			set = append(set, p)
			ids = append(ids, p.ID)
			battle.NoteSet(sd.user.UserId, p.ID, round)
		}
	}
	battle.KeepSet(sd.user.UserId, ids)
	return set
}

// battleParty finds the battle's group among the room's parties: the one
// sharing an enemy with it.
func battleParty(b battle.Battle, parties []mobparty.Party) (mobparty.Party, bool) {
	for _, p := range parties {
		for _, instanceId := range p.Members {
			if b.Has(instanceId) {
				return p, true
			}
		}
	}
	return mobparty.Party{}, false
}

func (sd side) allyRefs() []combatstream.Ref {
	var out []combatstream.Ref
	for _, instanceId := range sortedKeys(sd.allies) {
		if m := mobs.GetInstance(instanceId); m != nil {
			out = append(out, mobRef(m))
		}
	}
	return out
}

func partyRefs(p mobparty.Party) []combatstream.Ref {
	var out []combatstream.Ref
	for _, instanceId := range p.Members {
		if m := mobs.GetInstance(instanceId); m != nil {
			out = append(out, mobRef(m))
		}
	}
	return out
}

// beginBattle starts the player's battle against a group, opening its
// fight on the stream.
func (sd side) beginBattle(p mobparty.Party, room *rooms.Room, round uint64) battle.Battle {
	b := battle.Begin(sd.user.UserId, room.RoomId, round, p.ID, p.Members)
	id := combatstream.Default().Open(round, room.RoomId, p.ID, userRef(sd.user), sd.allyRefs(), partyRefs(p))
	battle.SetFight(sd.user.UserId, id)
	b.FightID = id
	sd.keepOnBattle(b, room)
	return b
}

// beginNext starts the player's next battle, with the group set on them
// first, if any.
func (sd side) beginNext(set []mobparty.Party, room *rooms.Room, round uint64) {
	cands := make([]battle.Candidate, len(set))
	for i, p := range set {
		cands[i] = battle.Candidate{PartyID: p.ID, FirstSet: battle.NoteSet(sd.user.UserId, p.ID, round), Order: i}
	}
	if i, ok := battle.Next(cands); ok {
		sd.beginBattle(set[i], room, round)
	}
}

// keepOnBattle turns the player and their allies off a foe outside their
// battle (a group that is waiting its turn) onto one in it.
func (sd side) keepOnBattle(b battle.Battle, room *rooms.Room) {
	foe := 0
	for _, instanceId := range room.GetMobs() {
		if m := mobs.GetInstance(instanceId); m != nil && b.Has(instanceId) && m.Character.Health > 0 && !m.Character.HasBuffFlag("hidden") {
			foe = instanceId
			break
		}
	}
	if foe == 0 {
		return
	}
	offBattle := func(a *characters.Aggro) bool {
		if a == nil || a.MobInstanceId <= 0 || !plainAttack(a) || b.Has(a.MobInstanceId) {
			return false
		}
		m := mobs.GetInstance(a.MobInstanceId)
		return m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId
	}
	if u := sd.user; offBattle(u.Character.Aggro) {
		emitTargetChange(userRef(u), mobRefById(u.Character.Aggro.MobInstanceId), mobRefById(foe), room.RoomId)
		u.Character.SetAggro(0, foe, attackType(u.Character.Aggro))
		events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: u.Character.RoomId})
		u.SendText(fmt.Sprintf(`You turn on <ansi fg="mobname">%s</ansi>.`, mobName(foe)))
	}
	for _, instanceId := range sortedKeys(sd.allies) {
		m := mobs.GetInstance(instanceId)
		if m == nil || !offBattle(m.Character.Aggro) {
			continue
		}
		emitTargetChange(mobRef(m), mobRefById(m.Character.Aggro.MobInstanceId), mobRefById(foe), room.RoomId)
		m.Character.SetAggro(0, foe, attackType(m.Character.Aggro))
		events.AddToQueue(events.AggroChanged{MobInstanceId: m.InstanceId, RoomId: m.Character.RoomId})
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> turns on <ansi fg="mobname">%s</ansi>.`, m.Character.Name, mobName(foe)))
	}
}

// battleOutcome is how a battle stands now: defeat when every company
// member is dead, victory when none of its enemies stands in its room,
// else it is still being fought ("").
func battleOutcome(b battle.Battle) string {
	fs := sidesOf(b)
	switch {
	case !fs.companyAlive():
		return combatstream.OutcomeDefeat
	case !fs.enemiesStanding():
		return combatstream.OutcomeVictory
	}
	return ""
}

// sidesOf reads a battle's two sides from its fight, or from the battle
// alone when the fight is gone.
func sidesOf(b battle.Battle) fightSides {
	fi, ok := combatstream.Default().Fight(b.FightID)
	if !ok {
		fi = combatstream.FightInfo{ID: b.FightID, LeaderUserId: b.UserId, RoomId: b.RoomId, PartyID: b.PartyID}
		if u := users.GetByUserId(b.UserId); u != nil {
			fi.Company = []combatstream.Ref{userRef(u)}
		}
		for _, instanceId := range sortedKeys(b.Enemies) {
			fi.Enemies = append(fi.Enemies, mobRefById(instanceId))
		}
	}
	fs := loadFightSides(fi)
	for instanceId := range b.Enemies {
		fs.enemyMobs[instanceId] = true
	}
	return fs
}

// endBattle ends the player's battle and its fight, and sends the summary.
func endBattle(userId int, outcome string) {
	b, ok := battle.End(userId)
	if !ok {
		return
	}
	fs := sidesOf(b)
	if _, open := combatstream.Default().Fight(b.FightID); open {
		fs.end(outcome)
	}
}

// battlePass decides every player's battle at the top of the round,
// before the upkeep and any blow: a battle whose group is gone or no
// longer set on the player ends, the next begins, and a waiting group
// turns on a free player in its room.
func battlePass() {
	round := combatRound.Load()
	online := map[int]bool{}
	for _, uid := range users.GetOnlineUserIds() {
		online[uid] = true
	}
	for _, uid := range battle.Players() {
		if !online[uid] {
			endBattle(uid, combatstream.OutcomeBrokenOff)
		}
	}
	battle.Retain(online) // a player who left waits in no line

	for _, uid := range users.GetOnlineUserIds() {
		u := users.GetByUserId(uid)
		if u == nil || u.Character == nil {
			continue
		}
		room := rooms.LoadRoom(u.Character.RoomId)
		if room == nil {
			continue
		}
		sd := loadSide(u, room)
		parties := enemyparty.Parties(room)
		set := sd.setParties(parties, room, round)
		if b, ok := battle.Current(uid); ok {
			// A battle whose group still stands where the player is goes on;
			// whether anyone is still fighting is judged after the upkeep
			// (closeIdleBattles), which may have turned them back on.
			if p, found := battleParty(b, parties); found && b.RoomId == room.RoomId && battleOutcome(b) == "" {
				battle.Grow(uid, p.ID, p.Members)
				combatstream.Default().Grow(b.FightID, p.ID, sd.allyRefs(), partyRefs(p))
				b, _ = battle.Current(uid)
				sd.keepOnBattle(b, room)
				continue
			}
			outcome := battleOutcome(b)
			if outcome == "" {
				outcome = combatstream.OutcomeBrokenOff // the player left, or the group did
			}
			endBattle(uid, outcome)
			set = sd.setParties(parties, room, round)
		}
		sd.beginNext(set, room, round)
	}

	turnWaitingOntoFreePlayers(round)
}

// closeIdleBattles ends, after the upkeep, each battle nobody is still
// fighting: no one on either side aims at the other (a `break` that held,
// a group that lost interest), and the player isn't casting. The next group
// set on the player, if any, begins at once.
func closeIdleBattles() {
	round := combatRound.Load()
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		if !ok {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil || u.Character == nil {
			continue
		}
		room := rooms.LoadRoom(u.Character.RoomId)
		if room == nil {
			continue
		}
		if a := u.Character.Aggro; a != nil && a.Type == characters.SpellCast {
			continue // mid-cast: still fighting
		}
		sd := loadSide(u, room)
		parties := enemyparty.Parties(room)
		p, found := battleParty(b, parties)
		if found {
			sd.rallyIdleFoes(p, room)
			sd.turnAlone(p, room)
		}
		if found && sd.setOn(p, room) {
			continue
		}
		endBattle(uid, combatstream.OutcomeBrokenOff)
		sd.beginNext(sd.setParties(parties, room, round), room, round)
	}
}

// rallyIdleFoes turns the battle's idle members that would fight the player
// anyway (a hostile mob, or a group made hostile) onto the player, when the
// player stands here and can be seen: a foe whose target just fell doesn't
// leave the battle. With a company present, 29a's upkeep has already done
// this; it matters for a player alone, or one whose companions have all
// fallen.
func (sd side) rallyIdleFoes(p mobparty.Party, room *rooms.Room) {
	u := sd.user
	if u.Character.Health < 1 || u.Character.RoomId != room.RoomId || u.Character.HasBuffFlag("hidden") {
		return
	}
	for _, instanceId := range p.Members {
		m := mobs.GetInstance(instanceId)
		if m == nil || m.Character.Aggro != nil || m.Character.RoomId != room.RoomId || !canFight(&m.Character) || !joinsTheFight(m, u.UserId) {
			continue
		}
		emitTargetChange(mobRef(m), combatstream.Ref{}, userRef(u), room.RoomId)
		m.Character.SetAggro(u.UserId, 0, characters.DefaultAttack)
		m.PlayerAttacked(u.UserId)
		m.PreventIdle = true
		events.AddToQueue(events.AggroChanged{MobInstanceId: m.InstanceId, RoomId: m.Character.RoomId})
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> turns on <ansi fg="username">%s</ansi>.`, m.Character.Name, u.Character.Name))
	}
}

// turnAlone gives a player fighting with no companion beside them their
// next foe from the battle's group when their target has fallen or gone
// (Phase 32c: a battle plays out on its own, and a bare attack no longer
// picks the next foe). With a companion present, 29a's upkeep does this.
// A player who used break stays out.
func (sd side) turnAlone(p mobparty.Party, room *rooms.Room) {
	u := sd.user
	if u.Character.Health < 1 || u.Character.RoomId != room.RoomId || engagement.StoodDown(u.UserId) {
		return
	}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		if leaderId, _, ok := company.LeaderAndKeyForInstance(instanceId); ok && leaderId == u.UserId {
			return // the upkeep's
		}
	}
	previous := 0
	if a := u.Character.Aggro; a != nil {
		if !plainAttack(a) || a.MobInstanceId <= 0 {
			return
		}
		if m := mobs.GetInstance(a.MobInstanceId); m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId && !m.Character.HasBuffFlag("hidden") {
			return // still has a foe
		}
		previous = a.MobInstanceId
	}
	next, ok := enemyparty.FirstAim(enemyparty.Group{Party: p}, u.UserId, u.Character)
	if !ok || next == previous {
		return
	}
	emitTargetChange(userRef(u), mobRefById(previous), mobRefById(next), room.RoomId)
	u.Character.SetAggro(0, next, attackType(u.Character.Aggro))
	events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: u.Character.RoomId})
	u.SendText(fmt.Sprintf(`You turn on <ansi fg="mobname">%s</ansi>.`, mobName(next)))
}

// turnWaitingOntoFreePlayers turns each group waiting on a busy player onto
// a player in the room with no battle, who then fights it (the owner's
// rule: another player walking in takes the next group).
func turnWaitingOntoFreePlayers(round uint64) {
	for _, uid := range users.GetOnlineUserIds() {
		busy := users.GetByUserId(uid)
		if busy == nil || busy.Character == nil {
			continue
		}
		b, ok := battle.Current(uid)
		if !ok {
			continue
		}
		room := rooms.LoadRoom(busy.Character.RoomId)
		if room == nil {
			continue
		}
		free := freePlayer(room, uid)
		if free == nil {
			continue
		}
		sd := loadSide(busy, room)
		// The waiting groups turn in the order they set on the busy player.
		var waiting []battle.Candidate
		parties := enemyparty.Parties(room)
		for i, p := range parties {
			if _, current := battleParty(b, []mobparty.Party{p}); current || !sd.setOn(p, room) {
				continue
			}
			waiting = append(waiting, battle.Candidate{PartyID: p.ID, FirstSet: battle.NoteSet(uid, p.ID, round), Order: i})
		}
		sort.SliceStable(waiting, func(i, j int) bool {
			if waiting[i].FirstSet != waiting[j].FirstSet {
				return waiting[i].FirstSet < waiting[j].FirstSet
			}
			return waiting[i].Order < waiting[j].Order
		})
		for _, w := range waiting {
			if free == nil {
				break
			}
			p := parties[w.Order]
			turned := false
			for _, instanceId := range p.Members {
				m := mobs.GetInstance(instanceId)
				if m == nil || m.Character.Health < 1 || m.Character.Aggro == nil {
					continue
				}
				if m.Character.Aggro.UserId == uid || sd.allies[m.Character.Aggro.MobInstanceId] {
					previous := userRef(busy)
					if m.Character.Aggro.UserId != uid {
						previous = mobRefById(m.Character.Aggro.MobInstanceId)
					}
					emitTargetChange(mobRef(m), previous, userRef(free), room.RoomId)
					m.Character.SetAggro(free.UserId, 0, attackType(m.Character.Aggro))
					m.PlayerAttacked(free.UserId)
					events.AddToQueue(events.AggroChanged{MobInstanceId: m.InstanceId, RoomId: m.Character.RoomId})
					turned = true
				}
			}
			if !turned {
				continue
			}
			room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="username">%s</ansi>.`, groupTurnsOn(p), free.Character.Name))
			freeSide := loadSide(free, room)
			battle.NoteSet(free.UserId, p.ID, round)
			freeSide.beginBattle(p, room, round)
			free = freePlayer(room, uid)
		}
	}
}

// freePlayer is a player in the room, other than busyId, standing, not
// hidden, and with no battle: the lowest user id.
func freePlayer(room *rooms.Room, busyId int) *users.UserRecord {
	var best *users.UserRecord
	for _, uid := range room.GetPlayers() {
		if uid == busyId {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil || u.Character.Health < 1 || u.Character.HasBuffFlag("hidden") {
			continue
		}
		if _, busy := battle.Current(uid); busy {
			continue
		}
		if best == nil || uid < best.UserId {
			best = u
		}
	}
	return best
}

// groupTurnsOn names a group turning on someone, for a message: "The
// ruffian turns on", or "The ruffian and the others turn on".
func groupTurnsOn(p mobparty.Party) string {
	var names []string
	for _, instanceId := range p.Members {
		if m := mobs.GetInstance(instanceId); m != nil && m.Character.Health > 0 {
			names = append(names, m.Character.Name)
		}
	}
	switch len(names) {
	case 0:
		return `They turn on`
	case 1:
		return `The ` + names[0] + ` turns on`
	}
	return fmt.Sprintf(`The %s and the others turn on`, names[0])
}

// settleBattles ends, at the end of the round and after its deaths are
// reported, each battle one side of which has fallen, and begins the
// player's next battle at once, so the next group set on them strikes
// next round.
func settleBattles() {
	round := combatRound.Load()
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		if !ok {
			continue
		}
		outcome := battleOutcome(b)
		if outcome == "" {
			continue
		}
		endBattle(uid, outcome)
		u := users.GetByUserId(uid)
		if u == nil || u.Character == nil || u.Character.Health < 1 {
			continue
		}
		room := rooms.LoadRoom(u.Character.RoomId)
		if room == nil {
			continue
		}
		sd := loadSide(u, room)
		sd.beginNext(sd.setParties(enemyparty.Parties(room), room, round), room, round)
	}
}

// Hold checks: a blow outside a player's battle waits its turn.

// holdsAgainstPlayer reports whether mob, striking at the player userId
// (or one of their allies), must hold back: the player is in a battle with
// another group.
func holdsAgainstPlayer(mob *mobs.Mob, userId int) bool {
	b, ok := battle.Current(userId)
	return ok && !b.Has(mob.InstanceId)
}

// mobHolds reports whether a mob's blow at another mob must hold back: it
// strikes a player's ally outside that player's battle, or it is a
// player's ally striking a foe outside its player's battle.
func mobHolds(attacker, defender *mobs.Mob) bool {
	if owner, ok := allyOwner(defender.InstanceId); ok && holdsAgainstPlayer(attacker, owner) {
		return true
	}
	if owner, ok := allyOwner(attacker.InstanceId); ok {
		if b, inBattle := battle.Current(owner); inBattle && !b.Has(defender.InstanceId) {
			if _, defenderAlly := allyOwner(defender.InstanceId); !defenderAlly {
				return true
			}
		}
	}
	return false
}

// playerHolds reports whether a player's blow at a mob must hold back: it
// is outside the player's battle.
func playerHolds(userId int, target *mobs.Mob) bool {
	b, ok := battle.Current(userId)
	return ok && !b.Has(target.InstanceId)
}

// inBattleWith reports whether a party is the leader's battle's group.
func inBattleWith(leaderId int, party mobparty.Party) bool {
	b, ok := battle.Current(leaderId)
	if !ok {
		return false
	}
	_, found := battleParty(b, []mobparty.Party{party})
	return found
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// harmful reports whether a spell harms its targets.
func harmful(spellId string) bool {
	s := spells.GetSpell(spellId)
	return s != nil && (s.Type == spells.HarmSingle || s.Type == spells.HarmMulti || s.Type == spells.HarmArea)
}

// holdPlayerSpell keeps a player's harmful spell to their battle: targets
// outside it are dropped. held is true when it had targets and none are
// left.
func holdPlayerSpell(userId int, info *characters.SpellAggroInfo) (held bool) {
	if info == nil || len(info.TargetMobInstanceIds) == 0 || !harmful(info.SpellId) {
		return false
	}
	var keep []int
	for _, id := range info.TargetMobInstanceIds {
		if m := mobs.GetInstance(id); m != nil && playerHolds(userId, m) {
			continue
		}
		keep = append(keep, id)
	}
	info.TargetMobInstanceIds = keep
	return len(keep) == 0
}

// holdMobSpell keeps a mob's harmful spell off players (and their allies)
// who are fighting another group. held is true when it had targets and none
// are left; waitOn is then a player it held back from, whose line it keeps
// its place in (0 when none).
func holdMobSpell(mob *mobs.Mob, info *characters.SpellAggroInfo) (held bool, waitOn int) {
	if info == nil || !harmful(info.SpellId) {
		return false, 0
	}
	if len(info.TargetUserIds)+len(info.TargetMobInstanceIds) == 0 {
		return false, 0
	}
	var users []int
	for _, id := range info.TargetUserIds {
		if holdsAgainstPlayer(mob, id) {
			if waitOn == 0 {
				waitOn = id
			}
			continue
		}
		users = append(users, id)
	}
	var foes []int
	for _, id := range info.TargetMobInstanceIds {
		if m := mobs.GetInstance(id); m != nil && mobHolds(mob, m) {
			if owner, ok := allyOwner(id); ok && waitOn == 0 {
				waitOn = owner
			}
			continue
		}
		foes = append(foes, id)
	}
	info.TargetUserIds, info.TargetMobInstanceIds = users, foes
	if len(users)+len(foes) > 0 {
		return false, 0
	}
	return true, waitOn
}
