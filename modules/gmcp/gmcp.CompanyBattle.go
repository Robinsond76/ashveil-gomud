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
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/assessment"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/morale"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// battleFacts is what the builder needs, read from the live game.
type battleFacts struct {
	Narrow    bool
	Positions map[string]battleCell
	Retreat   *retreatFact
	InBattle  bool
	Dark      bool // too dark to make the enemy out, as scout says
	Group     string
	Placed    bool        // the player stands in their company's formation
	Enemies   []enemyFact // every enemy seen in the battle, in instance order
	Company   []aimFact
	Waiting   []string // the waiting groups' names, in the order they come
	// Phase 30c: the focus the company aims by now, the saved one, and
	// whether an order may be given (none waiting for the next round).
	Focus, SavedFocus string
	FocusReady        bool
	// Phase 35e: the company is on its healers default and an enemy healer
	// stands (it goes for the healer first).
	HealersFirst bool
	// Phase 30c2: each guardian on the player's side.
	Guards []guardFact
	// Phase 33i1: the company's assessment of the battle's group, as scout
	// ends with it; nil when the group can't be seen.
	Outlook *battleOutlook
	// Phase 40g2: the allied companies fighting the same group here, and
	// whether the player's own company is faltering (its nerve is tested).
	Allies    []allyFact
	Faltering bool
	// Phase 39c: the weather a Shaman has called into the battle, if any.
	Weather *battleWeather
	// Phase 39d: the Doll Masters' dolls standing in the player's company.
	Dolls []battleDoll
}

// battleDoll is a Doll Master's doll standing in the battle (Phase 39d): its
// member key (the id its events use, and its key in positions), its name,
// its Master's member key, and its health. A doll is the player's own side,
// so its numbers are shown as a member's are.
type battleDoll struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Master    string `json:"master"`
	Health    int    `json:"hp"`
	HealthMax int    `json:"hp_max"`
}

// battleWeather is a Shaman's battle weather (Phase 39c): its kind ("fog",
// "chill", "rain"), the name the battle's lines use, the combat rounds it
// has left, and what it does, in a few words.
type battleWeather struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Rounds int    `json:"rounds"`
	Effect string `json:"effect"`
}

// allyFact is one allied company in the battle: its leader and the members
// standing in its formation, health in words only.
type allyFact struct {
	Leader  int
	Name    string // the leader's name
	Members []allyMemberFact
}

type allyMemberFact struct {
	Key               string
	Name, Class       string
	Promoted          string // the promoted class id (40s5 art key), if any
	Row, Col          int
	Health, HealthMax int
	Down              bool
}

// battleOutlook is the assessment in words: the risk, whether it could go
// either way, and scout's headline sentence. Never a number.
type battleOutlook struct {
	Risk  string `json:"risk"`
	Close bool   `json:"close"`
	Text  string `json:"text"`
	// Phase 33i2: how the group fights together, as scout says it.
	Coordination string `json:"coordination,omitempty"`
}

// guardFact is one guardian's guards left and its set ward (blank: the
// most hurt).
type retreatFact struct {
	Exit   string `json:"exit"`
	Rounds int    `json:"rounds"`
}

type guardFact struct {
	Key  string `json:"key"`
	Left int    `json:"left"`
	Ward string `json:"ward"`
}

// enemyFact is one enemy of the battle.
type enemyFact struct {
	Id                int
	Label             string // its 29d battle label
	Surrendered       bool
	Standing          bool // alive, here, and still in the battle's group
	Hidden            bool
	Seen              bool // a fallen one may be named: not last seen hidden
	Row, Col          int
	Health, HealthMax int
	Reach             bool   // the player can strike it from their cell
	Sprite            string // Phase 40f: its battle-screen sprite key
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
	// Phase 40f: the battle screen's sprite key (the mob's own, else its
	// race's unknown-* silhouette).
	Sprite string `json:"sprite,omitempty"`
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
	Narrow      bool                  `json:"narrow,omitempty"`
	Positions   map[string]battleCell `json:"positions,omitempty"`
	Retreat     *retreatFact          `json:"retreat,omitempty"`
	Group       string                `json:"group"`
	Dark        bool                  `json:"dark,omitempty"`
	Enemies     []battleEnemy         `json:"enemies"`
	Fallen      []battleFallen        `json:"fallen,omitempty"`
	Surrendered []battleFallen        `json:"surrendered,omitempty"`
	Company     []battleAim           `json:"company,omitempty"`
	Others      []battleOther         `json:"others,omitempty"`
	Waiting     []string              `json:"waiting,omitempty"`
	// Phase 30c: the company focus (the Combat tab's focus buttons).
	Focus      string `json:"focus"`
	SavedFocus string `json:"saved_focus"`
	FocusReady bool   `json:"focus_ready"`
	// Phase 35e: the default focus is on the enemy healer first.
	HealersFirst bool `json:"healers_first,omitempty"`
	// Phase 30c2: guardians' guards (the battle view's guard counts).
	Guards []guardFact `json:"guards,omitempty"`
	// Phase 33i1: the company's outlook (the battle view's assessment).
	Outlook *battleOutlook `json:"outlook,omitempty"`
	// Phase 40g2: allied companies in the same battle (each its own
	// formation, drawn behind the player's), and "faltering" while the
	// player's company is losing and its nerve is being tested; omitted
	// while steady.
	Allies []battleAlly `json:"allies,omitempty"`
	Nerve  string       `json:"nerve,omitempty"`
	// Phase 39c: the battle's weather (the battle screen's banner and the
	// Battle view's note); omitted while the sky is clear.
	Weather *battleWeather `json:"weather,omitempty"`
	// Phase 39d: the company's standing dolls (the Combat tab's fighters
	// and the battle screen's units); omitted when there are none.
	Dolls []battleDoll `json:"dolls,omitempty"`
}

// battleAlly is an allied company: its leader's ref ("a:<user>") and name,
// and its members, whose ids ("a:<user>:<key>") are the ones the allied
// events use. Another company's health is shown in words, never numbers.
type battleAlly struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Members []battleAllyMember `json:"members"`
}

type battleAllyMember struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Class string `json:"class,omitempty"`
	// Promoted is the member's promoted class id, the art the battle
	// screen draws first (as Company members' `class`).
	Promoted string     `json:"promoted,omitempty"`
	Cell     battleCell `json:"cell"`
	Health   string     `json:"health"`
	Down     bool       `json:"down,omitempty"`
}

// allyRef is an allied company's leader ref, and allyMemberRef a member's.
func allyRef(leaderId int) string { return "a:" + strconv.Itoa(leaderId) }

func allyMemberRef(leaderId int, key string) string { return allyRef(leaderId) + ":" + key }

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
		return battlePayload{Group: "the enemy", Dark: true, Enemies: []battleEnemy{}, Focus: focus, SavedFocus: saved, FocusReady: f.FocusReady, Guards: f.Guards, Retreat: f.Retreat, HealersFirst: f.HealersFirst, Weather: f.Weather}
	}
	p := battlePayload{Narrow: f.Narrow, Positions: f.Positions, Group: f.Group, Enemies: []battleEnemy{}, Waiting: f.Waiting, Focus: focus, SavedFocus: saved, FocusReady: f.FocusReady, Guards: f.Guards, Retreat: f.Retreat, Outlook: f.Outlook, HealersFirst: f.HealersFirst, Weather: f.Weather, Dolls: f.Dolls}
	if p.Group == "" {
		p.Group = "the enemy"
	}
	if f.Faltering {
		p.Nerve = "faltering"
	}
	for _, a := range f.Allies {
		ba := battleAlly{ID: allyRef(a.Leader), Name: a.Name, Members: []battleAllyMember{}}
		for _, m := range a.Members {
			ba.Members = append(ba.Members, battleAllyMember{ID: allyMemberRef(a.Leader, m.Key), Name: m.Name, Class: m.Class, Promoted: m.Promoted,
				Cell: battleCell{Row: m.Row, Col: m.Col}, Health: enemyparty.HealthWord(m.Health, m.HealthMax), Down: m.Down})
		}
		p.Allies = append(p.Allies, ba)
	}
	listed := map[int]bool{}
	others := map[string]bool{}
	for _, e := range f.Enemies {
		if e.Surrendered {
			if !e.Hidden {
				p.Surrendered = append(p.Surrendered, battleFallen{ID: mobID(e.Id), Label: e.Label})
			}
			continue
		}
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
			Health: enemyparty.HealthWord(e.Health, e.HealthMax), Sprite: e.Sprite}
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
	f := battleFacts{Narrow: enemyparty.Narrow(room), InBattle: true, SavedFocus: savedFocus(user.UserId), FocusReady: battle.FocusReady(user.UserId)}
	if a := user.Character.Aggro; a != nil && a.Type == characters.Retreat && a.RetreatInfo != nil {
		f.Retreat = &retreatFact{Exit: a.RetreatInfo.ExitName, Rounds: a.RoundsWaiting + 1}
	}
	if rule, ok := enemyparty.Focus(user.UserId); ok {
		f.Focus = string(rule)
	}
	if enemyparty.HealersDefault(user.UserId) {
		for id := range b.Enemies {
			if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 && strategy.Role(m.EnemyRole()) == strategy.Healer {
				f.HealersFirst = true
				break
			}
		}
	}
	f.Guards = gatherGuards(user, room)
	f.Weather = weatherFact(b.Weather)
	f.Faltering = companyFaltering(b)
	f.Allies = gatherAllies(user, b)
	f.Dolls = gatherDolls(user.UserId, room.RoomId)
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

	if found {
		if rep, ok := assessment.Gather(user, room, group); ok {
			f.Outlook = &battleOutlook{Risk: string(rep.Risk), Close: rep.Close, Text: rep.Headline(), Coordination: rep.CoordinationLine()}
		}
	}

	col := 0
	if form, ok := enemyparty.CompanyFormation(user.UserId); ok {
		f.Positions = map[string]battleCell{}
		for r, row := range form {
			for c, key := range row {
				if key != "" {
					f.Positions[string(key)] = battleCell{Row: r, Col: c}
				}
			}
		}
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
		if m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId && m.Character.CombatWithdrawn {
			e.Surrendered = true
			e.Hidden = m.Character.HasBuffFlag("hidden")
		}
		if m != nil && m.Character.Health > 0 && m.Character.RoomId == room.RoomId && inGroup[id] {
			key := mobparty.MemberKeyFor(id)
			e.Standing = true
			e.Sprite = battleSprite(m)
			e.Hidden = m.Character.HasBuffFlag("hidden")
			e.Row, e.Col, _ = group.Party.Formation.Find(key)
			e.Health, e.HealthMax = m.Character.Health, m.Character.HealthMax.Value
			e.Reach = f.Placed && enemyparty.Legal(room, user.UserId, col, group.Party.Formation, key, alive, reach)
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

// gatherDolls lists the player's company's dolls standing in the battle's
// room (Phase 39d), by Master then place.
func gatherDolls(userId, roomId int) []battleDoll {
	var out []battleDoll
	for _, e := range company.DollsOf(userId) {
		m := mobs.GetInstance(e.Instance)
		if m == nil || m.Character.RoomId != roomId || m.Character.Health < 1 {
			continue
		}
		out = append(out, battleDoll{Key: string(e.Key), Name: m.Character.Name, Master: string(e.Owner),
			Health: m.Character.Health, HealthMax: m.Character.HealthMax.Value})
	}
	return out
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

// gatherGuards lists the guardians on the player's side here (Phase 30c2),
// the player first, then companions by instance: guards left in the
// battle, and the set ward.
func gatherGuards(user *users.UserRecord, room *rooms.Room) []guardFact {
	var out []guardFact
	add := func(key company.MemberKey) {
		s := enemyparty.MemberStrategy(user.UserId, key)
		if s.Role == strategy.Guardian {
			out = append(out, guardFact{Key: string(key), Left: battle.GuardsLeft(user.UserId, string(key)), Ward: s.Ward})
		}
	}
	if user.Character.Health > 0 {
		add(company.LeaderMemberKey)
	}
	ids := room.GetMobs(rooms.FindCharmed)
	sort.Ints(ids)
	for _, instanceId := range ids {
		leaderId, key, ok := company.LeaderAndKeyForInstance(instanceId)
		m := mobs.GetInstance(instanceId)
		if ok && leaderId == user.UserId && m != nil && m.Character.Health > 0 {
			add(key)
		}
	}
	return out
}

// savedFocus is the focus the player's saved tactics give their company at
// their level (Phase 35d): "none" when each member goes by its own rule.
func savedFocus(userID int) string {
	if rule, ok := enemyparty.SavedFocus(userID); ok {
		return string(rule)
	}
	return string(strategy.NoFocus)
}

// battleSprite is the sprite key the battle screen draws a mob with (Phase
// 40f): the mob spec's own, else a silhouette by race, so an enemy the
// art doesn't know yet still stands as a shape of the right kind.
func battleSprite(m *mobs.Mob) string {
	if m == nil {
		return "unknown-humanoid"
	}
	if m.Sprite != "" {
		return m.Sprite
	}
	race := races.GetRace(m.Character.GetRaceId())
	if race == nil {
		return "unknown-humanoid"
	}
	return raceSprite(race.Name)
}

// raceSprite maps a race name to its fallback silhouette.
func raceSprite(name string) string {
	switch strings.ToLower(name) {
	case "ogre", "troll", "golem", "tree", "eldritch horror", "giant spider":
		return "unknown-large"
	case "rodent", "canine", "insect", "reptile", "reptilian", "lagomorph", "monkey", "fungus", "orb", "ghostly spirit", "faerie":
		return "unknown-beast"
	}
	return "unknown-humanoid"
}

// companyFaltering reports whether the player's company is losing as the
// nerve check reads it (Phase 30e): half the company down, or a quarter of
// its health left. Companions of weak loyalty may then hesitate or flee.
// The same rule as the check, read from the fight's roster.
func companyFaltering(b battle.Battle) bool {
	fi, ok := combatstream.Default().Fight(b.FightID)
	if !ok || len(fi.Company) == 0 {
		return false
	}
	hp, maximum, down := 0, 0, 0
	for _, r := range fi.Company {
		var c *characters.Character
		if r.UserId > 0 {
			if u := users.GetByUserId(r.UserId); u != nil {
				c = u.Character
			}
		} else if m := mobs.GetInstance(r.MobInstanceId); m != nil {
			c = &m.Character
		}
		if c == nil || c.Health <= 0 {
			down++
			continue
		}
		hp += c.Health
		maximum += c.HealthMax.Value
	}
	return morale.Losing(len(fi.Company), down, hp, maximum)
}

// sharesEnemy reports whether two battles are against some of the same
// mobs.
func sharesEnemy(a, b battle.Battle) bool {
	for id := range a.Enemies {
		if b.Enemies[id] {
			return true
		}
	}
	return false
}

// alliesOf lists the allied leaders fighting the same enemy in the player's
// battle: consenting allies (parties.AlliedLeaders) here, in a battle of
// their own against some of the same mobs.
func alliesOf(user *users.UserRecord, b battle.Battle) []*users.UserRecord {
	ids := parties.AlliedLeaders(user.UserId)
	sort.Ints(ids)
	var out []*users.UserRecord
	for _, id := range ids {
		u := users.GetByUserId(id)
		if u == nil || u.Character == nil {
			continue
		}
		ab, ok := battle.Current(id)
		if !ok || ab.RoomId != b.RoomId || u.Character.RoomId != b.RoomId || !sharesEnemy(b, ab) {
			continue
		}
		out = append(out, u)
	}
	return out
}

// gatherAllies reads each allied company's formation, as it stands: who is
// in which cell, their class and health in words. Each company keeps its
// own formation (33d), so the cells are the ally's own.
func gatherAllies(user *users.UserRecord, b battle.Battle) []allyFact {
	var out []allyFact
	for _, au := range alliesOf(user, b) {
		sum := companyview.For(au)
		if !sum.CompanyKnown {
			continue
		}
		form, ok := enemyparty.CompanyFormation(au.UserId)
		if !ok {
			continue
		}
		cells := map[string]battleCell{}
		for r, row := range form {
			for c, key := range row {
				if key != "" {
					cells[string(key)] = battleCell{Row: r, Col: c}
				}
			}
		}
		fact := allyFact{Leader: au.UserId, Name: au.Character.Name}
		add := func(m companyview.Member) {
			cell, ok := cells[string(m.Key)]
			if !ok || m.Status == company.MemberAwaiting || m.Status == company.MemberFled || m.Status == company.MemberSeparated {
				return
			}
			class := ""
			if !m.Leader || m.ArchetypeKnown {
				class = strings.ToLower(m.Archetype)
			}
			fact.Members = append(fact.Members, allyMemberFact{Key: string(m.Key), Name: m.Name, Class: class, Promoted: m.Class, Row: cell.Row, Col: cell.Col,
				Health: m.HP, HealthMax: m.HPMax, Down: m.Status == company.MemberDead || (m.HasHP && m.HP < 1)})
		}
		add(sum.Leader)
		for _, m := range sum.Companions {
			add(m)
		}
		if len(fact.Members) > 0 {
			out = append(out, fact)
		}
	}
	return out
}

// weatherFact is the battle's weather for the feed: nil while clear. The
// rounds left count the round in progress (a call's last tick ends it).
func weatherFact(w battle.Weather) *battleWeather {
	if w.Kind == stormcraft.None || w.Left < 1 {
		return nil
	}
	return &battleWeather{Kind: string(w.Kind), Name: w.Kind.Name(), Rounds: max(1, w.Left-1), Effect: w.Kind.Effect()}
}
