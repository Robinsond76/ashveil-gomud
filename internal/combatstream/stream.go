package combatstream

import (
	"sort"
	"sync"
)

// Sink receives every emitted event, synchronously, on the caller's
// goroutine (the game loop). It must be cheap and must not block. It may
// emit to or read from the stream: Emit calls sinks after releasing its
// lock.
type Sink func(Event)

// MemberHealth is one company member's health when a fight ends, read from
// the live world by the caller.
type MemberHealth struct {
	Ref    Ref
	Health int
	Max    int
	Fallen bool // dead (a companion's body is gone, or a player slain)
}

// FightInfo is a read-only view of an open fight.
type FightInfo struct {
	ID           uint64
	LeaderUserId int
	RoomId       int
	PartyID      string // the first enemy party engaged
	GroupName    string // the enemy group's name (Phase 32c), "" if unnamed
	StartRound   uint64
	Company      []Ref
	Enemies      []Ref
}

// Final is what the caller reads from the world when a fight ends: the
// company's health, and which enemies are no longer in the fight's room
// (neither slain nor fled, they left it).
type Final struct {
	Company []MemberHealth
	Gone    []Ref
}

type fight struct {
	id           uint64
	leaderUserId int
	roomId       int
	partyID      string
	groupName    string
	startRound   uint64
	company      map[string]Ref
	companyOrder []string
	enemies      map[string]Ref
	enemyOrder   []string
	enemyParty   map[string]string // enemy key -> its party id
	lastDamager  map[string]Ref
	down         map[string]string // key -> death outcome recorded
	fled         map[string]bool
	tally        tally
}

func (f *fight) addCompany(r Ref) {
	k := r.Key()
	if k == "" {
		return
	}
	if _, ok := f.company[k]; !ok {
		f.companyOrder = append(f.companyOrder, k)
	}
	f.company[k] = r
}

func (f *fight) addEnemy(r Ref, partyID string) {
	k := r.Key()
	if k == "" {
		return
	}
	if partyID != "" {
		f.enemyParty[k] = partyID
	}
	if _, ok := f.enemies[k]; !ok {
		f.enemyOrder = append(f.enemyOrder, k)
	}
	f.enemies[k] = r
}

func (f *fight) has(r Ref) bool {
	k := r.Key()
	if k == "" {
		return false
	}
	_, c := f.company[k]
	_, e := f.enemies[k]
	return c || e
}

func (f *fight) info() FightInfo {
	fi := FightInfo{ID: f.id, LeaderUserId: f.leaderUserId, RoomId: f.roomId, PartyID: f.partyID, GroupName: f.groupName, StartRound: f.startRound}
	for _, k := range f.companyOrder {
		fi.Company = append(fi.Company, f.company[k])
	}
	for _, k := range f.enemyOrder {
		fi.Enemies = append(fi.Enemies, f.enemies[k])
	}
	return fi
}

// Stream is an ordered combat event stream with its open fights. The zero
// value is not usable; use New.
type Stream struct {
	mu        sync.Mutex
	seq       uint64
	nextFight uint64
	fights    map[uint64]*fight
	sinks     map[int]Sink
	nextSink  int
}

// New returns an empty stream.
func New() *Stream {
	return &Stream{fights: map[uint64]*fight{}, sinks: map[int]Sink{}}
}

var (
	defaultMu     sync.RWMutex
	defaultStream = New()
)

// Default returns the process-wide stream the combat loop emits to.
func Default() *Stream {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultStream
}

// UseForTest makes s the default stream and returns a func restoring the
// previous one.
func UseForTest(s *Stream) (restore func()) {
	defaultMu.Lock()
	previous := defaultStream
	defaultStream = s
	defaultMu.Unlock()
	return func() {
		defaultMu.Lock()
		defaultStream = previous
		defaultMu.Unlock()
	}
}

// Subscribe adds a sink and returns a func removing it.
func (s *Stream) Subscribe(sink Sink) (unsubscribe func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSink++
	id := s.nextSink
	s.sinks[id] = sink
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.sinks, id)
	}
}

// Emit stamps e with the next sequence number, places it in its fight (the
// open fight both actors are in, or the one the only actor is in), folds
// it into that fight's totals, and calls every sink. It returns the
// stamped event, and false when e was dropped: a repeated death for an
// actor already recorded down in the fight.
func (s *Stream) Emit(e Event) (Event, bool) {
	s.mu.Lock()
	if f := s.fightFor(e); f != nil {
		if !s.place(f, &e) {
			s.mu.Unlock()
			return e, false
		}
	} else {
		e.FightID, e.PartyID = 0, ""
		s.noteOutsideDamage(e)
	}
	e, sinks := s.stampLocked(e)
	s.mu.Unlock()
	call(sinks, e)
	return e, true
}

func (s *Stream) stampLocked(e Event) (Event, []Sink) {
	s.seq++
	e.Seq = s.seq
	ids := make([]int, 0, len(s.sinks))
	for id := range s.sinks {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	sinks := make([]Sink, 0, len(ids))
	for _, id := range ids {
		sinks = append(sinks, s.sinks[id])
	}
	return e, sinks
}

func call(sinks []Sink, e Event) {
	for _, sink := range sinks {
		sink(e)
	}
}

// fightFor finds the open fight an event belongs to: an explicit FightID
// that is still open, else the lowest-id fight holding both actors, else
// (for a one-actor event) the lowest-id fight holding that actor.
func (s *Stream) fightFor(e Event) *fight {
	if e.FightID != 0 {
		return s.fights[e.FightID] // nil once it has ended
	}
	for _, id := range s.openIds() {
		f := s.fights[id]
		switch {
		case !e.Source.Zero() && !e.Target.Zero():
			if f.has(e.Source) && f.has(e.Target) {
				return f
			}
		case !e.Source.Zero():
			if f.has(e.Source) {
				return f
			}
		case !e.Target.Zero():
			if f.has(e.Target) {
				return f
			}
		}
	}
	return nil
}

func (s *Stream) openIds() []uint64 {
	ids := make([]uint64, 0, len(s.fights))
	for id := range s.fights {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// noteOutsideDamage records a blow from outside any fight on a fight's
// member as its last damage, so a kill by an outsider is not credited to
// the company.
func (s *Stream) noteOutsideDamage(e Event) {
	if (e.Kind != Attack && e.Kind != SpellHit) || e.Damage <= 0 || e.Source.Zero() {
		return
	}
	for _, f := range s.fights {
		if f.has(e.Target) {
			f.lastDamager[e.Target.Key()] = e.Source
		}
	}
}

// place stamps e with f's identity and folds it in. false drops e.
func (s *Stream) place(f *fight, e *Event) bool {
	e.FightID = f.id
	e.PartyID = f.partyID
	for _, r := range []Ref{e.Target, e.Source} {
		if p, ok := f.enemyParty[r.Key()]; ok {
			e.PartyID = p // the enemy actor's own party
			break
		}
	}
	if e.RoomId == 0 {
		e.RoomId = f.roomId
	}
	switch e.Kind {
	case Death:
		k := e.Target.Key()
		if prior, ok := f.down[k]; ok && (prior != OutcomeIncapacitated || prior == e.Outcome) {
			return false
		}
		f.down[k] = e.Outcome
		if e.Source.Zero() {
			e.Source = f.lastDamager[k]
		}
	case Attack, SpellHit:
		if e.Damage > 0 && !e.Source.Zero() {
			f.lastDamager[e.Target.Key()] = e.Source
		}
	case Flee:
		f.fled[e.Source.Key()] = true
	}
	f.tally.add(f, *e)
	return true
}

// Open opens a fight (and emits FightStart) for one battle (Phase 29b2):
// the player (leader), with any companions, against one enemy group in a
// room. It returns the fight id. Each battle is its own fight, however
// many groups share the room.
func (s *Stream) Open(round uint64, roomId int, partyID string, leader Ref, companions []Ref, enemies []Ref) uint64 {
	s.mu.Lock()
	s.nextFight++
	f := &fight{
		id:           s.nextFight,
		leaderUserId: leader.UserId,
		roomId:       roomId,
		partyID:      partyID,
		startRound:   round,
		company:      map[string]Ref{},
		enemies:      map[string]Ref{},
		enemyParty:   map[string]string{},
		lastDamager:  map[string]Ref{},
		down:         map[string]string{},
		fled:         map[string]bool{},
		tally:        newTally(),
	}
	f.addCompany(leader)
	for _, r := range companions {
		f.addCompany(r)
	}
	for _, r := range enemies {
		f.addEnemy(r, partyID)
	}
	s.fights[f.id] = f
	e, sinks := s.stampLocked(Event{Kind: FightStart, Round: round, FightID: f.id, PartyID: partyID, RoomId: roomId, Source: leader})
	s.mu.Unlock()
	call(sinks, e)
	return f.id
}

// Grow adds members seen this round to an open fight: companions who have
// joined the company, and the group's members (with its current party
// id). It does nothing for a fight that isn't open.
func (s *Stream) Grow(id uint64, partyID string, companions []Ref, enemies []Ref) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.fights[id]
	if !ok {
		return
	}
	for _, r := range companions {
		f.addCompany(r)
	}
	for _, r := range enemies {
		f.addEnemy(r, partyID)
	}
}

// Name names an open fight after the enemy group it is fought with
// (Phase 32c): "a band of ruffians". The summary's heading uses it. It
// does nothing for a fight that isn't open.
func (s *Stream) Name(id uint64, groupName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f, ok := s.fights[id]; ok {
		f.groupName = groupName
	}
}

// Fight returns an open fight.
func (s *Stream) Fight(id uint64) (FightInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.fights[id]
	if !ok {
		return FightInfo{}, false
	}
	return f.info(), true
}

// OpenFights lists the open fights by id.
func (s *Stream) OpenFights() []FightInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []FightInfo
	for _, id := range s.openIds() {
		out = append(out, s.fights[id].info())
	}
	return out
}

// EndFight closes an open fight: it builds the fight's Summary (with what
// the caller read from the world), emits FightEnd carrying it, and forgets
// the fight. ok is false when the fight isn't open.
func (s *Stream) EndFight(id uint64, round uint64, outcome string, final Final) (*Summary, bool) {
	s.mu.Lock()
	f, ok := s.fights[id]
	if !ok {
		s.mu.Unlock()
		return nil, false
	}
	delete(s.fights, id)
	sum := f.summary(round, outcome, final)
	var leader Ref
	if len(f.companyOrder) > 0 {
		leader = f.company[f.companyOrder[0]]
	}
	e, sinks := s.stampLocked(Event{Kind: FightEnd, Round: round, FightID: f.id, PartyID: f.partyID, RoomId: f.roomId, Outcome: outcome, Source: leader, Summary: sum})
	s.mu.Unlock()
	call(sinks, e)
	return sum, true
}
