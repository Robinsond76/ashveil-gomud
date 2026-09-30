package gmcp

// Phase 32g2: "Company.Battle", the web client's Battle view in the Combat
// tab: the player's battle as they can see it. The enemy group's living,
// visible members with their cells, how hurt each looks (scout's words,
// never numbers), whether the player can reach it, and whom it strikes;
// the enemies fallen or gone; whom each company member strikes; anyone
// outside the company an enemy strikes, by name; the groups waiting their
// turn. The company's own cells and health are in Company and
// Company.Vitals. {} when the player is in no battle.
//
// Built on the game loop by the company feed, from internal/battle and the
// live room (the combat event stream only reports), and sent only when it
// changes. Nothing is stored: a restart or copyover starts a new battle,
// and the feed forgets the user, so the view is rebuilt.

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// battleFacts is what the builder needs, read from the live game.
type battleFacts struct {
	InBattle bool
	Dark     bool // too dark to make the enemy out, as scout says
	Group    string
	Placed   bool        // the player stands in their company's formation
	Enemies  []enemyFact // every enemy seen in the battle, in instance order
	Company  []aimFact
	Waiting  []string // the waiting groups' names, in the order they come
	// Phase 30c: the focus the company aims by now, the saved one, and
	// whether an order may be given (none waiting for the next round).
	Focus, SavedFocus string
	FocusReady        bool
}

// enemyFact is one enemy of the battle.
type enemyFact struct {
	Id                int
	Label             string // its 29d battle label
	Standing          bool   // alive, here, and still in the battle's group
	Hidden            bool
	Seen              bool // a fallen one may be named: not last seen hidden
	Row, Col          int
	Health, HealthMax int
	Reach             bool // the player can strike it from their cell
	Target            targetFact
}

// targetFact is whom an enemy strikes: a company member (Key) or someone
// else (UserId or MobInstanceId, with a Name). Zero: no one.
type targetFact struct {
	Key           string
	UserId        int
	MobInstanceId int
	Name          string
}

// aimFact is whom a company member strikes (a mob instance id; 0: no one).
type aimFact struct {
	Key    string
	Target int
}

type battleCell struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type battleEnemy struct {
	ID     string     `json:"id"`
	Label  string     `json:"label"`
	Cell   battleCell `json:"cell"`
	Health string     `json:"health"`
	Reach  *bool      `json:"reach,omitempty"`
	Target string     `json:"target,omitempty"`
}

type battleFallen struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type battleAim struct {
	Key    string `json:"key"`
	Target string `json:"target"`
}

type battleOther struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type battlePayload struct {
	Group   string         `json:"group"`
	Dark    bool           `json:"dark,omitempty"`
	Enemies []battleEnemy  `json:"enemies"`
	Fallen  []battleFallen `json:"fallen,omitempty"`
	Company []battleAim    `json:"company,omitempty"`
	Others  []battleOther  `json:"others,omitempty"`
	Waiting []string       `json:"waiting,omitempty"`
	// Phase 30c: the company focus (the Combat tab's focus buttons).
	Focus      string `json:"focus"`
	SavedFocus string `json:"saved_focus"`
	FocusReady bool   `json:"focus_ready"`
}

func mobID(instanceId int) string { return "m:" + strconv.Itoa(instanceId) }

// buildBattle turns the facts into the payload: {} out of battle.
func buildBattle(f battleFacts) any {
	if !f.InBattle {
		return struct{}{}
	}
	focus, saved := f.Focus, f.SavedFocus
	if focus == "" {
		focus = "none"
	}
	if saved == "" {
		saved = "none"
	}
	if f.Dark {
		return battlePayload{Group: "the enemy", Dark: true, Enemies: []battleEnemy{}, Focus: focus, SavedFocus: saved, FocusReady: f.FocusReady}
	}
	p := battlePayload{Group: f.Group, Enemies: []battleEnemy{}, Waiting: f.Waiting, Focus: focus, SavedFocus: saved, FocusReady: f.FocusReady}
	if p.Group == "" {
		p.Group = "the enemy"
	}
	listed := map[int]bool{}
	others := map[string]bool{}
	for _, e := range f.Enemies {
		if !e.Standing {
			if e.Seen {
				p.Fallen = append(p.Fallen, battleFallen{ID: mobID(e.Id), Label: e.Label})
			}
			continue
		}
		if e.Hidden {
			continue
		}
		listed[e.Id] = true
		be := battleEnemy{ID: mobID(e.Id), Label: e.Label, Cell: battleCell{Row: e.Row, Col: e.Col},
			Health: enemyparty.HealthWord(e.Health, e.HealthMax)}
		if f.Placed {
			reach := e.Reach
			be.Reach = &reach
		}
		switch t := e.Target; {
		case t.Key != "":
			be.Target = t.Key
		case t.UserId > 0 || t.MobInstanceId > 0:
			id := mobID(t.MobInstanceId)
			if t.UserId > 0 {
				id = "u:" + strconv.Itoa(t.UserId)
			}
			be.Target = id
			if !others[id] {
				others[id] = true
				p.Others = append(p.Others, battleOther{ID: id, Name: t.Name})
			}
		}
		p.Enemies = append(p.Enemies, be)
	}
	for _, a := range f.Company {
		if a.Target > 0 && listed[a.Target] {
			p.Company = append(p.Company, battleAim{Key: a.Key, Target: mobID(a.Target)})
		}
	}
	return p
}

// battleExtra is the feed's Company.Battle message.
func battleExtra(gather func(user *users.UserRecord) battleFacts) companyExtra {
	return companyExtra{module: "Company.Battle", build: func(user *users.UserRecord) []byte {
		data, err := json.Marshal(buildBattle(gather(user)))
		if err != nil {
			return nil
		}
		return data
	}}
}

// gatherBattle reads the player's battle from the live game, as scout
// reads a group. battle.Current returns a copy, so no battle lock is held
// while mobs and rooms are read.
func gatherBattle(user *users.UserRecord) battleFacts {
	if user == nil || user.Character == nil {
		return battleFacts{}
	}
	b, ok := battle.Current(user.UserId)
	if !ok || user.Character.RoomId != b.RoomId {
		battleSeen.forget(user.UserId)
		return battleFacts{}
	}
	room := rooms.LoadRoom(b.RoomId)
	if room == nil {
		return battleFacts{}
	}
	f := battleFacts{InBattle: true, SavedFocus: string(strategy.TacticsFor(user.UserId).Focus), FocusReady: battle.FocusReady(user.UserId)}
	if rule, ok := enemyparty.Focus(user.UserId); ok {
		f.Focus = string(rule)
	}
	if room.VisibilityForUser(user) < 1 && !user.Character.HasBuffFlag("nightvision") {
		f.Dark = true
		return f
	}
	fight := battleSeenKey{start: b.StartRound, fight: b.FightID, party: b.PartyID}

	groups := enemyparty.Groups(room)
	var group enemyparty.Group
	found := false
	for _, g := range groups {
		if _, ok := enemyparty.BattleParty(b, []mobparty.Party{g.Party}); ok {
			group, found = g, true
			if len(g.Visible()) > 0 {
				f.Group = g.Name // a group wholly hidden goes unnamed, as scout lists it
			}
			break
		}
	}

	col := 0
	if form, ok := company.FormationFor(user.UserId); ok {
		_, col, f.Placed = form.Find(company.LeaderMemberKey)
	}
	reach := combat.ResolveReach(user.Character, false)
	alive := enemyparty.Alive(group.Party)
	inGroup := map[int]bool{}
	if found {
		for _, id := range group.Party.Members {
			inGroup[id] = true
		}
	}

	ids := make([]int, 0, len(b.Enemies))
	for id := range b.Enemies {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		m := mobs.GetInstance(id)
		label := ""
		if n, ok := b.EnemyNames[id]; ok {
			label = n.DisplayName
		} else if m != nil {
			label = m.Character.Name
		}
		if label == "" {
			continue
		}
		e := enemyFact{Id: id, Label: label}
		if m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId && inGroup[id] {
			key := mobparty.MemberKeyFor(id)
			e.Standing = true
			e.Hidden = m.Character.HasBuffFlag("hidden")
			e.Row, e.Col, _ = group.Party.Formation.Find(key)
			e.Health, e.HealthMax = m.Character.Health, m.Character.HealthMax.Value
			e.Reach = f.Placed && formationcombat.Legal(col, group.Party.Formation, key, alive, reach)
			e.Target = targetOf(user.UserId, room.RoomId, m)
			battleSeen.note(user.UserId, fight, id, e.Hidden)
		} else if m != nil {
			// A body not yet taken away, or one that walked off.
			battleSeen.note(user.UserId, fight, id, m.Character.HasBuffFlag("hidden"))
		}
		e.Seen = !battleSeen.hiddenLast(user.UserId, fight, id)
		f.Enemies = append(f.Enemies, e)
	}

	if a := user.Character.Aggro; a != nil && a.MobInstanceId > 0 {
		f.Company = append(f.Company, aimFact{Key: string(company.LeaderMemberKey), Target: a.MobInstanceId})
	}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		leaderId, key, ok := company.LeaderAndKeyForInstance(instanceId)
		m := mobs.GetInstance(instanceId)
		if !ok || leaderId != user.UserId || m == nil || m.Character.Health < 1 {
			continue
		}
		if a := m.Character.Aggro; a != nil && a.MobInstanceId > 0 {
			f.Company = append(f.Company, aimFact{Key: string(key), Target: a.MobInstanceId})
		}
	}

	byParty := map[string]string{}
	for _, g := range groups {
		if len(g.Visible()) > 0 {
			byParty[g.Party.ID] = g.Name // a wholly hidden group isn't named
		}
	}
	for _, id := range battle.Waiting(user.UserId) {
		if name, ok := byParty[id]; ok {
			f.Waiting = append(f.Waiting, name)
		}
	}
	return f
}

// targetOf is whom an enemy strikes, seen by the player whose battle it
// is: the player or one of their companions by member key, anyone else in
// the room by name.
func targetOf(userId, roomId int, m *mobs.Mob) targetFact {
	a := m.Character.Aggro
	if a == nil {
		return targetFact{}
	}
	if a.UserId > 0 {
		if a.UserId == userId {
			return targetFact{Key: string(company.LeaderMemberKey)}
		}
		if u := users.GetByUserId(a.UserId); u != nil && u.Character != nil && u.Character.RoomId == roomId {
			return targetFact{UserId: a.UserId, Name: u.Character.Name}
		}
		return targetFact{}
	}
	if a.MobInstanceId > 0 {
		if leaderId, key, ok := company.LeaderAndKeyForInstance(a.MobInstanceId); ok && leaderId == userId {
			return targetFact{Key: string(key)}
		}
		if t := mobs.GetInstance(a.MobInstanceId); t != nil && t.Character.RoomId == roomId && t.Character.Health > 0 {
			return targetFact{MobInstanceId: a.MobInstanceId, Name: t.Character.Name}
		}
	}
	return targetFact{}
}

// battleSeenKey names one battle of a player's.
type battleSeenKey struct {
	start uint64
	fight uint64
	party string
}

// seenEnemies remembers, per player, whether each enemy of their current
// battle was hidden when the view last looked, so one that falls or leaves
// while hidden is never named in the view (32g2 review finding 2). One the
// view never looked at (it fell in the round it joined) is named: the
// player saw it fight. Runtime only: after a restart the battle is a new
// one. Built on the game loop; mu keeps tests honest.
type seenEnemies struct {
	mu     sync.Mutex
	byUser map[int]seenFight
}

type seenFight struct {
	key    battleSeenKey
	hidden map[int]bool // instance -> hidden when last seen
}

func newSeenEnemies() *seenEnemies { return &seenEnemies{byUser: map[int]seenFight{}} }

func (s *seenEnemies) note(userId int, key battleSeenKey, id int, hidden bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.byUser[userId]
	if !ok || f.key != key {
		f = seenFight{key: key, hidden: map[int]bool{}}
		s.byUser[userId] = f
	}
	f.hidden[id] = hidden
}

// hiddenLast reports whether the enemy was hidden when last seen in this
// battle; false when it was never seen.
func (s *seenEnemies) hiddenLast(userId int, key battleSeenKey, id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.byUser[userId]
	return ok && f.key == key && f.hidden[id]
}

func (s *seenEnemies) forget(userId int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byUser, userId)
}

// prune drops players no longer online.
func (s *seenEnemies) prune(online []int) {
	live := make(map[int]bool, len(online))
	for _, id := range online {
		live[id] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.byUser {
		if !live[id] {
			delete(s.byUser, id)
		}
	}
}

var battleSeen = newSeenEnemies()
