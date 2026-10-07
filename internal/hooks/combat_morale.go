package hooks

import (
	cryptorand "crypto/rand"
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type moraleFight struct {
	rules morale.Group
	ids   []int
	owner int
	room  int
	// Actual resolved deaths, not disappearance or flight.
	dead         map[int]bool
	startCompany []combatstream.Ref
	nerveChecked bool
	shared       bool
}
type yieldedEnemy struct {
	owner, room int
	fight       uint64
	ref         combatstream.Ref
}
type mercyQueue struct {
	uid                            int
	room                           int
	fight                          uint64
	ids                            []int
	witnesses                      []int
	token                          string
	deadline                       time.Time
	spare                          bool
	deciding                       bool
	reactionsSaved, alignmentSaved bool
	lines                          []string
}

var (
	moraleFights  = map[uint64]*moraleFight{}
	yielded       = map[int]yieldedEnemy{}
	mercyQueues   = map[int]*mercyQueue{}
	acceptedMercy = map[string]*mercyQueue{}
	moraleRoll    = util.Rand
	mercyNow      = time.Now
	mercySaveUser = func(u *users.UserRecord) error { return users.SaveUser(*u) }
)

func init() { morale.AnswerMercy = answerMercy }
func UseMoraleRollForTest(f func(int) int) func() {
	p := moraleRoll
	moraleRoll = f
	return func() { moraleRoll = p }
}
func enemyTemperament(m *mobs.Mob) string {
	if m == nil || m.Practice || m.NeverBreak || m.Character.IsCharmed() {
		return "unbreakable"
	}
	r := races.GetRace(m.Character.GetRaceId())
	if r != nil && (r.NeverBreak || r.Name == "undead" || r.Name == "construct" || r.Name == "skeleton" || r.Name == "zombie") {
		return "unbreakable"
	}
	if m.Temperament != "" {
		return m.Temperament
	}
	if r != nil {
		return r.Temperament
	}
	return "unbreakable"
}
func startMorale(b battle.Battle) {
	abandonMercy(b.UserId)
	shared := false
	for _, f := range moraleFights {
		if f.room == b.RoomId {
			for _, id := range f.ids {
				if b.Has(id) {
					shared = true
				}
			}
		}
	}
	ids := sortedKeys(b.Enemies)
	f := &moraleFight{ids: ids, owner: b.UserId, room: b.RoomId, dead: map[int]bool{}, shared: shared}
	f.rules.Size = len(ids)
	if room := rooms.LoadRoom(b.RoomId); room != nil {
		if p, ok := battleParty(b, enemyparty.Parties(room)); ok {
			if len(p.Members) > 0 {
				f.rules.Leader = p.Members[0]
			}
		}
	}
	for _, id := range ids {
		m := mobs.GetInstance(id)
		if m == nil {
			continue
		}
		if f.rules.Specialist == 0 && (len(m.Character.SpellBook) > 0 || len(m.CombatCommands) > 0 && containsCast(m.CombatCommands)) {
			f.rules.Specialist = id
		}
	}
	if fi, ok := combatstream.Default().Fight(b.FightID); ok {
		f.startCompany = append([]combatstream.Ref(nil), fi.Company...)
	}
	moraleFights[b.FightID] = f
}
func containsCast(cmds []string) bool {
	for _, s := range cmds {
		if len(s) >= 4 && s[:4] == "cast" {
			return true
		}
	}
	return false
}
func moralePass() {
	fights := make([]uint64, 0, len(moraleFights))
	for id := range moraleFights {
		fights = append(fights, id)
	}
	sort.Slice(fights, func(i, j int) bool { return fights[i] < fights[j] })
	for _, fid := range fights {
		f := moraleFights[fid]
		b, ok := battle.Current(f.owner)
		if !ok || b.FightID != fid || !primaryMoraleFight(fid, f) {
			continue
		}
		var ms []morale.Member
		for _, id := range f.ids {
			m := mobs.GetInstance(id)
			v := morale.Member{ID: id, Dead: f.dead[id]}
			if m != nil {
				v.HP = m.Character.Health
				v.Max = m.Character.HealthMax.Value
				v.Dead = v.Dead || v.HP <= 0
				v.Active = v.HP > 0 && m.Character.RoomId == f.room && !m.Character.CombatWithdrawn
			}
			ms = append(ms, v)
		}
		if !f.rules.Check(ms) {
			continue
		}
		// Snapshot the results first: departures cannot cascade into more checks.
		type result struct {
			id  int
			out morale.Outcome
		}
		var results []result
		for _, v := range ms {
			if !v.Active {
				continue
			}
			m := mobs.GetInstance(v.ID)
			s := enemyTemperament(m)
			if s == "" || s == "unbreakable" {
				continue
			}
			results = append(results, result{v.ID, enemyOutcome(m, s)})
		}
		for _, r := range results {
			m := mobs.GetInstance(r.id)
			if m == nil {
				continue
			}
			applyEnemyMorale(m, r.out, f.owner, f.room, fid)
		}
	}
}

// applyEnemyMorale carries out one enemy's morale outcome: a yield stands
// it aside, a flight sends it running.
func applyEnemyMorale(m *mobs.Mob, out morale.Outcome, owner, room int, fid uint64) {
	switch out {
	case morale.Yield:
		withdrawEnemy(m)
		yielded[m.InstanceId] = yieldedEnemy{owner: owner, room: room, fight: fid, ref: mobRef(m)}
		emitMoraleOutcome(combatstream.Yield, mobRef(m), room)
		moraleSay(m, "surrenders and stands aside, hands raised.")
	case morale.Flee:
		emitMoraleOutcome(combatstream.Flee, mobRef(m), room)
		moraleSay(m, "loses nerve and flees.")
		harrow(m, room)
		escapeEnemy(m.InstanceId)
	}
}

// DreadCheck answers a landed Dread Whisper (Phase 38a): one morale check
// on the enemy, by its temperament, exactly as a leader's fall would roll
// it. An unbreakable enemy holds; one outside the leader's battle is left
// alone.
func DreadCheck(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MoraleCheck)
	if !ok {
		return events.Continue
	}
	b, found := battle.Current(evt.LeaderUserId)
	m := mobs.GetInstance(evt.MobInstanceId)
	if !found || m == nil || m.Character.Health < 1 || !b.Has(m.InstanceId) {
		return events.Continue
	}
	s := enemyTemperament(m)
	if s == "" || s == "unbreakable" {
		return events.Continue
	}
	applyEnemyMorale(m, enemyOutcome(m, s), evt.LeaderUserId, b.RoomId, b.FightID)
	return events.Continue
}
func emitMoraleOutcome(kind combatstream.Kind, ref combatstream.Ref, room int) {
	for _, uid := range battle.Players() {
		b, _ := battle.Current(uid)
		if b.RoomId == room && b.Has(ref.MobInstanceId) {
			combatstream.Default().Emit(combatstream.Event{Kind: kind, FightID: b.FightID, Source: ref, RoomId: room})
		}
	}
}
func MoraleDeath(e events.Event) events.ListenerReturn {
	if v, ok := e.(events.MobDeath); ok {
		for _, f := range moraleFights {
			for _, id := range f.ids {
				if id == v.InstanceId {
					f.dead[id] = true
				}
			}
		}
	}
	return events.Continue
}
func moraleSay(m *mobs.Mob, verb string) {
	if r := rooms.LoadRoom(m.Character.RoomId); r != nil {
		for _, uid := range r.GetPlayers() {
			if u := users.GetByUserId(uid); u != nil {
				label := mobName(m.InstanceId)
				if y, ok := yielded[m.InstanceId]; ok {
					label = y.ref.Name
				}
				if r.VisibilityForUser(u) < 1 && !u.Character.HasBuffFlag("nightvision") || m.Character.HasBuffFlag("hidden") {
					label = "someone"
				}
				u.SendText(util.CapitalizeFirst(named(mobTag(label))) + " " + verb)
			}
		}
	}
}
func withdrawEnemy(m *mobs.Mob) {
	m.Character.CombatWithdrawn = true
	m.Character.Aggro = nil
	status.Clear(&m.Character)
	delete(windUps, m.InstanceId)
	delete(windUpCooldown, m.InstanceId)
	delete(castAims, caster{mobId: m.InstanceId})
	clearAimsAt(m.InstanceId)
}
func clearAimsAt(id int) {
	for _, uid := range users.GetOnlineUserIds() {
		u := users.GetByUserId(uid)
		if u != nil {
			clearWithdrawnAim(u.Character, id)
		}
	}
	for _, mid := range mobs.GetAllMobInstanceIds() {
		if m := mobs.GetInstance(mid); m != nil {
			clearWithdrawnAim(&m.Character, id)
		}
	}
}
func clearWithdrawnAim(c *characters.Character, id int) {
	if a := c.Aggro; a != nil {
		if a.MobInstanceId == id && a.Type != characters.SpellCast {
			c.Aggro = nil
		}
		if a.Type == characters.SpellCast {
			out := a.SpellInfo.TargetMobInstanceIds[:0]
			for _, mid := range a.SpellInfo.TargetMobInstanceIds {
				if mid != id {
					out = append(out, mid)
				}
			}
			a.SpellInfo.TargetMobInstanceIds = out
		}
	}
}
func escapeEnemy(id int) {
	delete(yielded, id)
	if m := mobs.GetInstance(id); m != nil {
		clearAimsAt(id)
		if r := rooms.LoadRoom(m.Character.RoomId); r != nil {
			mobcommands.Suicide("vanish", m, r)
		}
	}
}
func finishMorale(b battle.Battle, outcome string) {
	defer func() {
		f := moraleFights[b.FightID]
		delete(moraleFights, b.FightID)
		if f != nil && !f.shared {
			var next uint64
			for id, other := range moraleFights {
				if other.room != f.room {
					continue
				}
				overlap := false
				for _, mid := range f.ids {
					for _, oid := range other.ids {
						overlap = overlap || mid == oid
					}
				}
				if overlap && (next == 0 || id < next) {
					next = id
				}
			}
			if next > 0 {
				n := moraleFights[next]
				n.shared = false
				n.rules = f.rules
				n.ids = append([]int(nil), f.ids...)
				n.dead = f.dead
			}
		}
	}()
	var ids []int
	for id, y := range yielded {
		if y.owner == b.UserId && y.fight == b.FightID {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	if len(ids) == 0 {
		return
	}
	u := users.GetByUserId(b.UserId)
	if outcome != combatstream.OutcomeVictory || u == nil || u.Character.Health < 1 || u.Character.RoomId != b.RoomId {
		for _, id := range ids {
			escapeEnemy(id)
		}
		return
	}
	q := &mercyQueue{uid: b.UserId, room: b.RoomId, fight: b.FightID, ids: ids}
	for _, m := range company.MoraleMembers(b.UserId) {
		q.witnesses = append(q.witnesses, m.ID)
	}
	mercyQueues[b.UserId] = q
}
func abandonMercy(uid int) {
	if q := mercyQueues[uid]; q != nil {
		if q.deciding {
			acceptedMercy[q.token] = q
			for _, id := range q.ids {
				escapeEnemy(id)
			}
			q.ids = nil
			delete(mercyQueues, uid)
			return
		}
		if u := users.GetByUserId(uid); u != nil {
			if p := u.GetPrompt(); p != nil && p.Command == "mercy" && p.Rest == q.token {
				u.ClearPrompt()
			}
		}
		for _, id := range q.ids {
			escapeEnemy(id)
		}
		delete(mercyQueues, uid)
	}
	// Surrender during a fight also escapes when its owner leaves.
	for id, y := range yielded {
		if y.owner == uid {
			escapeEnemy(id)
		}
	}
}
func MercyTick(e events.Event) events.ListenerReturn {
	retryMercyEffects()
	for uid, q := range mercyQueues {
		u := users.GetByUserId(uid)
		_, busy := battle.Current(uid)
		if u == nil || u.Character == nil || u.Character.Health < 1 || u.Character.RoomId != q.room || busy {
			abandonMercy(uid)
			continue
		}
		if q.deciding {
			if u != nil {
				if err := resolveMercy(uid, q); err != nil {
					u.SendText(err.Error())
				}
			}
			continue
		}
		if u == nil || u.Character == nil || u.Character.Health < 1 || u.Character.RoomId != q.room || busy {
			abandonMercy(uid)
			continue
		}
		if len(q.ids) == 0 {
			delete(mercyQueues, uid)
			continue
		}
		if q.token != "" {
			p := u.GetPrompt()
			if p == nil || p.Command != "mercy" || p.Rest != q.token {
				abandonMercy(uid)
				continue
			}
			if !mercyNow().Before(q.deadline) {
				u.ClearPrompt()
				escapeEnemy(q.ids[0])
				q.ids = q.ids[1:]
				q.token = ""
				u.SendText("The unanswered foe slips away.")
			}
			continue
		}
		if combatpace.Default().Busy(uid) {
			continue
		}
		if u.GetPrompt() != nil {
			u.SendText("The surrendered foes slip away while you are occupied.")
			abandonMercy(uid)
			continue
		}
		m := mobs.GetInstance(q.ids[0])
		if m == nil {
			q.ids = q.ids[1:]
			continue
		}
		q.token = cryptorand.Text()
		q.deadline = mercyNow().Add(30 * time.Second)
		p, _ := u.StartPrompt("mercy", q.token)
		label := yielded[m.InstanceId].ref.Name
		if room := rooms.LoadRoom(q.room); room == nil || room.VisibilityForUser(u) < 1 && !u.Character.HasBuffFlag("nightvision") || m.Character.HasBuffFlag("hidden") {
			label = "a surrendered foe"
		}
		question := p.Ask(util.CapitalizeFirst(named(mobTag(label)))+" kneels, hands raised. Spare "+m.Character.CombatPronouns().Object+"? [yes/no]", []string{"yes", "no"})
		u.SendText(question.Question)
	}
	return events.Continue
}
func answerMercy(uid int, token, answer string) error {
	q := mercyQueues[uid]
	u := users.GetByUserId(uid)
	if q == nil || u == nil || q.token != token || q.deciding || !mercyNow().Before(q.deadline) || u.Character.RoomId != q.room || u.Character.Health < 1 {
		return nil
	}
	if _, busy := battle.Current(uid); busy {
		abandonMercy(uid)
		return nil
	}
	if len(q.ids) == 0 {
		return nil
	}
	y, claimed := yielded[q.ids[0]]
	if !claimed || y.owner != uid || y.fight != q.fight || mobs.GetInstance(q.ids[0]) == nil {
		u.ClearPrompt()
		q.ids = q.ids[1:]
		q.token = ""
		return nil
	}
	if answer != "yes" && answer != "no" {
		return nil
	}
	// Consume the answer before effects; retries use staged state, never reconsume input.
	q.deciding = true
	q.spare = answer == "yes"
	u.ClearPrompt()
	return resolveMercy(uid, q)
}
func resolveMercy(uid int, q *mercyQueue) error {
	u := users.GetByUserId(uid)
	if u == nil {
		var err error
		u, err = users.LoadUserFile(uid)
		if err != nil {
			return err
		}
	}
	if u == nil {
		return nil
	}
	if !q.reactionsSaved {
		lines, err := company.MercyReaction(uid, q.token, q.witnesses, q.spare)
		if err != nil {
			return err
		}
		// Phase 64: the witnesses' personalities answer too (those alignment
		// already moved stay quiet; a retry repeats nothing).
		kind := opinions.Execute
		if q.spare {
			kind = opinions.Spare
		}
		subject := ""
		if m := mobs.GetInstance(q.ids[0]); m != nil {
			subject = m.Character.Name
		}
		said, err := company.Opinion(uid, opinions.Choice{Kind: kind, Op: "mercy:" + q.token, Witnesses: append([]int{}, q.witnesses...), Subject: subject})
		if err != nil {
			return err
		}
		q.lines = append(lines, said...)
		q.reactionsSaved = true
	}
	if !q.alignmentSaved {
		delta := -5
		if q.spare {
			delta = 5
		}
		if err := saveMercyAlignment(u, company.MercyEffect{Token: q.token, Delta: delta}); err != nil {
			return err
		}
		q.alignmentSaved = true
	}
	if err := company.CompleteMercy(uid, q.token); err != nil {
		return err
	}
	u.Character.SetMiscData("mercy-alignment:"+q.token, nil)
	if len(q.ids) == 0 {
		return nil
	}

	id := q.ids[0]
	if m := mobs.GetInstance(id); m != nil {
		outcome := "executed"
		if q.spare {
			outcome = "spared"
			moraleSay(m, "leaves without a weapon.")
			escapeEnemy(id)
		} else {
			moraleSay(m, "is executed.")
			delete(yielded, id)
			if r := rooms.LoadRoom(q.room); r != nil {
				mobcommands.Suicide("mercy", m, r)
			}
		}
		// Phase 63: the answer is the company's deed, whichever way it went.
		deed := chronicle.Executed
		if q.spare {
			deed = chronicle.Spared
		}
		chronicle.Record(uid, chronicle.Entry{Kind: deed, Subject: m.Character.Name, Ref: fmt.Sprintf("mob:%d", m.MobId)})
		combatstream.Default().Emit(combatstream.Event{Kind: combatstream.Mercy, FightID: q.fight, Source: userRef(u), Target: mobRef(m), RoomId: q.room, Outcome: outcome})
	}
	for _, line := range q.lines {
		u.SendText(line)
	}
	q.ids = q.ids[1:]
	q.token = ""
	q.deciding = false
	q.reactionsSaved = false
	q.alignmentSaved = false
	q.lines = nil
	return nil
}
func MercyLeave(e events.Event) events.ListenerReturn {
	switch v := e.(type) {
	case events.RoomChange:
		abandonMercy(v.UserId)
	case events.PlayerDespawn:
		abandonMercy(v.UserId)
	case events.UserPurged:
		abandonMercy(v.UserId)
		for token, q := range acceptedMercy {
			if q.uid == v.UserId {
				delete(acceptedMercy, token)
			}
		}
		delete(mercyQueues, v.UserId)
	}
	return events.Continue
}

// UseMoraleStateForTest isolates transient state and deterministic clocks/saves.
func UseMoraleStateForTest(now func() time.Time, save func(*users.UserRecord) error) func() {
	oldAccepted := acceptedMercy
	acceptedMercy = map[string]*mercyQueue{}
	oldF, oldY, oldQ, oldNow, oldSave := moraleFights, yielded, mercyQueues, mercyNow, mercySaveUser
	moraleFights = map[uint64]*moraleFight{}
	yielded = map[int]yieldedEnemy{}
	mercyQueues = map[int]*mercyQueue{}
	if now != nil {
		mercyNow = now
	}
	if save != nil {
		mercySaveUser = save
	}
	return func() {
		acceptedMercy = oldAccepted
		moraleFights = oldF
		yielded = oldY
		mercyQueues = oldQ
		mercyNow = oldNow
		mercySaveUser = oldSave
		clear(nerveSkip)
	}
}

func saveMercyAlignment(u *users.UserRecord, e company.MercyEffect) error {
	key := "mercy-alignment:" + e.Token
	if u.Character.GetMiscData(key) == true {
		return nil
	}
	before := u.Character.Alignment
	u.Character.Alignment = int8(company.ClampAlignment(int(before) + e.Delta))
	u.Character.SetMiscData(key, true)
	if err := mercySaveUser(u); err != nil {
		u.Character.Alignment = before
		u.Character.SetMiscData(key, nil)
		return fmt.Errorf("Mercy consequences could not be saved; they will be retried: %w", err)
	}
	return nil
}
func retryMercyEffects() {
	for token, q := range acceptedMercy {
		if err := resolveMercy(q.uid, q); err == nil {
			delete(acceptedMercy, token)
		}
	}
	for uid, effects := range company.PendingMercy() {
		u := users.GetByUserId(uid)
		if u == nil {
			var err error
			u, err = users.LoadUserFile(uid)
			if err != nil {
				continue
			}
		}
		for _, e := range effects {
			live := acceptedMercy[e.Token] != nil
			if q := mercyQueues[uid]; q != nil && q.token == e.Token {
				live = true
			}
			if live {
				continue
			}
			if err := saveMercyAlignment(u, e); err != nil {
				continue
			}
			if err := company.CompleteMercy(uid, e.Token); err == nil {
				u.Character.SetMiscData("mercy-alignment:"+e.Token, nil)
			}
		}
	}
}

func SetMercySaveForTest(save func(*users.UserRecord) error) func() {
	old := mercySaveUser
	mercySaveUser = save
	return func() { mercySaveUser = old }
}

// Allied companies keep independent battles over the same enemies. The first
// still-active company evaluates a shared enemy group, owning mercy once;
// allies neither repeat the morale roll nor inherit a yielded enemy's decision.
func primaryMoraleFight(fid uint64, f *moraleFight) bool {
	for otherID, other := range moraleFights {
		if otherID >= fid || other.room != f.room {
			continue
		}
		b, active := battle.Current(other.owner)
		u := users.GetByUserId(other.owner)
		if !active || b.FightID != otherID || u == nil || u.Character.Health <= 0 || u.Character.RoomId != f.room {
			continue
		}
		for _, id := range other.ids {
			for _, mine := range f.ids {
				if id == mine {
					return false
				}
			}
		}
	}
	return true
}
