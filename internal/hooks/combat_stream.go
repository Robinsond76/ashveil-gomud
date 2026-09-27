package hooks

import (
	"sync/atomic"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 29b: the combat loop's producers for the combat event stream
// (internal/combatstream), and the fight tracking around each round.
// Producers emit where the loop already builds its text, and never change
// that text or any state: the stream only reports.

// combatRound is the round DoCombat is resolving, stamped on every event.
var combatRound atomic.Uint64

// BattleSummarySetting is the player setting (`set battlesummary`) that
// sends the battle summary at a fight's end. It is on when unset.
const BattleSummarySetting = `battlesummary`

func emitCombat(e combatstream.Event) (combatstream.Event, bool) {
	e.Round = combatRound.Load()
	return combatstream.Default().Emit(e)
}

func userRef(u *users.UserRecord) combatstream.Ref {
	if u == nil || u.Character == nil {
		return combatstream.Ref{}
	}
	r := combatstream.Ref{UserId: u.UserId, Name: u.Character.Name}
	if _, ok := company.FormationFor(u.UserId); ok {
		r.LeaderUserId = u.UserId
		r.MemberKey = string(company.LeaderMemberKey)
	}
	return r
}

func mobRef(m *mobs.Mob) combatstream.Ref {
	if m == nil {
		return combatstream.Ref{}
	}
	r := combatstream.Ref{MobInstanceId: m.InstanceId, MobId: int(m.MobId), Name: m.Character.Name}
	if leaderId, key, ok := company.LeaderAndKeyForInstance(m.InstanceId); ok {
		r.LeaderUserId = leaderId
		r.MemberKey = string(key)
	}
	return r
}

// mobRefById names a mob instance, alive or gone: a gone one is known only
// by its instance id. 0 names nobody.
func mobRefById(instanceId int) combatstream.Ref {
	if instanceId <= 0 {
		return combatstream.Ref{}
	}
	if m := mobs.GetInstance(instanceId); m != nil {
		return mobRef(m)
	}
	return combatstream.Ref{MobInstanceId: instanceId}
}

// weaponType is the attacker's weapon subtype (stabbing, bludgeoning, ...),
// or "" when it fights unarmed.
func weaponType(c *characters.Character) string {
	if c == nil || c.Equipment.Weapon.ItemId == 0 {
		return ``
	}
	if spec := c.Equipment.Weapon.GetSpec(); spec.ItemId != 0 {
		return string(spec.Subtype)
	}
	return ``
}

// attackEvents turns one resolved attack into its stream events: the
// attack itself, then one status-applied per buff it put on the target and
// on the attacker.
func attackEvents(source, target combatstream.Ref, roomId int, attacker *characters.Character, r combat.AttackResult) []combatstream.Event {
	outcome := combatstream.OutcomeMiss
	switch {
	case r.Hit && r.Crit:
		outcome = combatstream.OutcomeCrit
	case r.Hit:
		outcome = combatstream.OutcomeHit
	}
	out := []combatstream.Event{{
		Kind:       combatstream.Attack,
		RoomId:     roomId,
		Source:     source,
		Target:     target,
		Outcome:    outcome,
		Damage:     r.DamageToTarget,
		Crit:       r.Hit && r.Crit,
		WeaponType: weaponType(attacker),
	}}
	status := func(on combatstream.Ref, buffId int) combatstream.Event {
		name := ``
		if spec := buffs.GetBuffSpec(buffId); spec != nil {
			name = spec.Name
		}
		return combatstream.Event{Kind: combatstream.StatusApplied, RoomId: roomId, Source: source, Target: on, BuffId: buffId, Status: name}
	}
	for _, buffId := range r.BuffTarget {
		out = append(out, status(target, buffId))
	}
	for _, buffId := range r.BuffSource {
		out = append(out, status(source, buffId))
	}
	return out
}

func emitAttack(source, target combatstream.Ref, roomId int, attacker *characters.Character, r combat.AttackResult) {
	for _, e := range attackEvents(source, target, roomId, attacker, r) {
		emitCombat(e)
	}
}

func emitTargetChange(source, previous, next combatstream.Ref, roomId int) {
	emitCombat(combatstream.Event{Kind: combatstream.TargetChange, RoomId: roomId, Source: source, Previous: previous, Target: next})
}

// spellTargets is a cast's targets with their health before the spell's
// script ran, so the damage and healing it did can be read afterwards.
type spellTargets struct {
	refs   []combatstream.Ref
	before []int
}

func snapshotSpellTargets(info characters.SpellAggroInfo) spellTargets {
	var s spellTargets
	for _, userId := range info.TargetUserIds {
		if u := users.GetByUserId(userId); u != nil {
			s.refs = append(s.refs, userRef(u))
			s.before = append(s.before, u.Character.Health)
		}
	}
	for _, instanceId := range info.TargetMobInstanceIds {
		if m := mobs.GetInstance(instanceId); m != nil {
			s.refs = append(s.refs, mobRef(m))
			s.before = append(s.before, m.Character.Health)
		}
	}
	return s
}

// after reads each target's health now; ok=false for one that is gone.
func (s spellTargets) after() (health []int, ok []bool) {
	for _, r := range s.refs {
		switch {
		case r.UserId > 0:
			if u := users.GetByUserId(r.UserId); u != nil {
				health, ok = append(health, u.Character.Health), append(ok, true)
				continue
			}
		case r.MobInstanceId > 0:
			if m := mobs.GetInstance(r.MobInstanceId); m != nil {
				health, ok = append(health, m.Character.Health), append(ok, true)
				continue
			}
		}
		health, ok = append(health, 0), append(ok, false)
	}
	return health, ok
}

// spellDeltaEvents reports what a spell did to each target: a loss of
// health is a spell-hit, a gain a heal. A target that is gone is skipped.
func spellDeltaEvents(caster combatstream.Ref, spellId string, roomId int, refs []combatstream.Ref, before, after []int, present []bool) []combatstream.Event {
	var out []combatstream.Event
	for i, r := range refs {
		if i >= len(after) || !present[i] {
			continue
		}
		delta := after[i] - before[i]
		switch {
		case delta < 0:
			out = append(out, combatstream.Event{Kind: combatstream.SpellHit, RoomId: roomId, Source: caster, Target: r, SpellId: spellId, Damage: -delta})
		case delta > 0:
			out = append(out, combatstream.Event{Kind: combatstream.Heal, RoomId: roomId, Source: caster, Target: r, SpellId: spellId, Amount: delta})
		}
	}
	return out
}

func (s spellTargets) emitResults(caster combatstream.Ref, spellId string, roomId int) {
	after, present := s.after()
	for _, e := range spellDeltaEvents(caster, spellId, roomId, s.refs, s.before, after, present) {
		emitCombat(e)
	}
}

func emitCast(kind combatstream.Kind, caster combatstream.Ref, spellId, outcome string, roomId int) {
	emitCombat(combatstream.Event{Kind: kind, RoomId: roomId, Source: caster, SpellId: spellId, Outcome: outcome})
}

// Round-affected extras: actors an attack struck that the loop's own
// affected lists don't name (the interceptors of 11c's front-row
// interception), so handleAffected ends them in the round they fall.
var (
	roundExtraPlayers []int
	roundExtraMobs    []int
)

func resetRoundExtras() {
	roundExtraPlayers = nil
	roundExtraMobs = nil
}

// engageFight records one engaged company/party pair for the stream,
// opening a fight on the first round.
func engageFight(s companySide, party mobparty.Party) {
	companions := make([]combatstream.Ref, 0, len(s.companionIds))
	for _, instanceId := range s.companionIds {
		if m := mobs.GetInstance(instanceId); m != nil {
			companions = append(companions, mobRef(m))
		}
	}
	enemies := make([]combatstream.Ref, 0, len(party.Members))
	for _, instanceId := range party.Members {
		if m := mobs.GetInstance(instanceId); m != nil {
			enemies = append(enemies, mobRef(m))
		}
	}
	combatstream.Default().Engage(combatRound.Load(), s.leader.Character.RoomId, party.ID, userRef(s.leader), companions, enemies)
}

// fightSides are an open fight's actors as they stand in the world now.
type fightSides struct {
	info        combatstream.FightInfo
	leader      *users.UserRecord // nil when offline
	companyMobs map[int]bool
	enemyMobs   map[int]bool
}

func loadFightSides(fi combatstream.FightInfo) fightSides {
	fs := fightSides{info: fi, leader: users.GetByUserId(fi.LeaderUserId), companyMobs: map[int]bool{}, enemyMobs: map[int]bool{}}
	for _, r := range fi.Company {
		if r.MobInstanceId > 0 {
			fs.companyMobs[r.MobInstanceId] = true
		}
	}
	for _, r := range fi.Enemies {
		fs.enemyMobs[r.MobInstanceId] = true
	}
	return fs
}

// standingMob is a living mob in the fight's room.
func (fs fightSides) standingMob(instanceId int) *mobs.Mob {
	m := mobs.GetInstance(instanceId)
	if m == nil || m.Character.RoomId != fs.info.RoomId || m.Character.Health < 1 {
		return nil
	}
	return m
}

func (fs fightSides) leaderStanding() bool {
	return fs.leader != nil && fs.leader.Character.RoomId == fs.info.RoomId && fs.leader.Character.Health > 0
}

func (fs fightSides) enemiesStanding() bool {
	for instanceId := range fs.enemyMobs {
		if fs.standingMob(instanceId) != nil {
			return true
		}
	}
	return false
}

func (fs fightSides) companyStanding() bool {
	if fs.leaderStanding() {
		return true
	}
	for instanceId := range fs.companyMobs {
		if fs.standingMob(instanceId) != nil {
			return true
		}
	}
	return false
}

// aimsAt reports whether an Aggro is on one of the given mobs (a cast
// counts through its targets).
func aimsAt(a *characters.Aggro, mobSet map[int]bool) bool {
	if a == nil {
		return false
	}
	if mobSet[a.MobInstanceId] {
		return true
	}
	for _, instanceId := range a.SpellInfo.TargetMobInstanceIds {
		if mobSet[instanceId] {
			return true
		}
	}
	return false
}

// engaged reports whether any standing member of either side is still
// fighting the other.
func (fs fightSides) engaged() bool {
	if fs.leaderStanding() && aimsAt(fs.leader.Character.Aggro, fs.enemyMobs) {
		return true
	}
	for instanceId := range fs.companyMobs {
		if m := fs.standingMob(instanceId); m != nil && aimsAt(m.Character.Aggro, fs.enemyMobs) {
			return true
		}
	}
	for instanceId := range fs.enemyMobs {
		m := fs.standingMob(instanceId)
		if m == nil || m.Character.Aggro == nil {
			continue
		}
		if aimsAt(m.Character.Aggro, fs.companyMobs) {
			return true
		}
		if fs.leaderStanding() && m.Character.Aggro.UserId == fs.info.LeaderUserId {
			return true
		}
	}
	return false
}

// companyHealth reads each company member's health from the world.
func (fs fightSides) companyHealth() []combatstream.MemberHealth {
	var out []combatstream.MemberHealth
	for _, r := range fs.info.Company {
		switch {
		case r.UserId > 0:
			u := users.GetByUserId(r.UserId)
			if u == nil {
				continue
			}
			out = append(out, combatstream.MemberHealth{Ref: r, Health: u.Character.Health, Max: u.Character.HealthMax.Value, Fallen: u.Character.Health <= -10})
		default:
			m := mobs.GetInstance(r.MobInstanceId)
			if m == nil || m.Character.Health < 1 {
				out = append(out, combatstream.MemberHealth{Ref: r, Fallen: true})
				continue
			}
			out = append(out, combatstream.MemberHealth{Ref: r, Health: m.Character.Health, Max: m.Character.HealthMax.Value})
		}
	}
	return out
}

// endFight closes a fight and sends its summary to the leader, if online
// and their setting is on.
func (fs fightSides) end(outcome string) {
	sum, ok := combatstream.Default().EndFight(fs.info.ID, combatRound.Load(), outcome, fs.companyHealth())
	if !ok || fs.leader == nil {
		return
	}
	if on := fs.leader.GetConfigOption(BattleSummarySetting); on != nil {
		if enabled, isBool := on.(bool); isBool && !enabled {
			return
		}
	}
	for _, line := range combatstream.Render(*sum, fs.leader.UserId) {
		fs.leader.SendText(line)
	}
}

// closeDisengagedFights ends, after the upkeep, each open fight neither
// side is still fighting (a flee, a `break` that held, a walk out, a
// logout). The upkeep runs first, so a party member it draws in keeps the
// fight open.
func closeDisengagedFights() {
	for _, fi := range combatstream.Default().OpenFights() {
		fs := loadFightSides(fi)
		if fs.leader == nil || !fs.engaged() {
			fs.end(combatstream.OutcomeBrokenOff)
		}
	}
}

// settleFights ends, at the end of the round and after its deaths are
// reported, each open fight one side of which no longer stands.
func settleFights() {
	for _, fi := range combatstream.Default().OpenFights() {
		fs := loadFightSides(fi)
		switch {
		case !fs.enemiesStanding():
			fs.end(combatstream.OutcomeVictory)
		case !fs.companyStanding():
			fs.end(combatstream.OutcomeDefeat)
		}
	}
}
