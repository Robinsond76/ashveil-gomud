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

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// battleFacts is what the builder needs, read from the live game.
type battleFacts struct {
	InBattle bool
	Group    string
	Placed   bool        // the player stands in their company's formation
	Enemies  []enemyFact // every enemy seen in the battle, in instance order
	Company  []aimFact
	Waiting  []string // the waiting groups' names, in the order they come
}

// enemyFact is one enemy of the battle.
type enemyFact struct {
	Id                int
	Label             string // its 29d battle label
	Standing          bool   // alive, here, and still in the battle's group
	Hidden            bool
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
	Enemies []battleEnemy  `json:"enemies"`
	Fallen  []battleFallen `json:"fallen,omitempty"`
	Company []battleAim    `json:"company,omitempty"`
	Others  []battleOther  `json:"others,omitempty"`
	Waiting []string       `json:"waiting,omitempty"`
}

func mobID(instanceId int) string { return "m:" + strconv.Itoa(instanceId) }

// buildBattle turns the facts into the payload: {} out of battle.
func buildBattle(f battleFacts) any {
	if !f.InBattle {
		return struct{}{}
	}
	p := battlePayload{Group: f.Group, Enemies: []battleEnemy{}, Waiting: f.Waiting}
	if p.Group == "" {
		p.Group = "the enemy"
	}
	listed := map[int]bool{}
	others := map[string]bool{}
	for _, e := range f.Enemies {
		if !e.Standing {
			p.Fallen = append(p.Fallen, battleFallen{ID: mobID(e.Id), Label: e.Label})
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
		return battleFacts{}
	}
	room := rooms.LoadRoom(b.RoomId)
	if room == nil {
		return battleFacts{}
	}
	f := battleFacts{InBattle: true}

	groups := enemyparty.Groups(room)
	var group enemyparty.Group
	found := false
	for _, g := range groups {
		if _, ok := enemyparty.BattleParty(b, []mobparty.Party{g.Party}); ok {
			group, found = g, true
			f.Group = g.Name
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
		}
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
		byParty[g.Party.ID] = g.Name
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
