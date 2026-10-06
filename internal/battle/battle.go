// Package battle holds Ashveil's battles (Phase 29b2): each player fights
// one enemy group at a time. A battle is a player (with their company, if
// any) against one group. Other groups set on the player wait their turn,
// in the order they turned on the player.
//
// It is runtime state only, never persisted: after a restart or copyover,
// rooms respawn and a fight still in progress resumes as a new battle. It
// is the source of truth for who is fighting whom this round; the combat
// event stream (internal/combatstream) only reports it.
package battle

import (
	"sort"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/sigils"
)

// Battle is one player's current battle.
type Battle struct {
	UserId     int
	RoomId     int
	PartyID    string // the group's party id when the battle began or was last seen
	StartRound uint64
	Enemies    map[int]bool      // the group's mob instance ids seen in this battle
	EnemyNames map[int]EnemyName // immutable enemy narration snapshots
	FightID    uint64            // its fight on the combat event stream

	// Phase 30c: the company focus ordered for this battle only, over the
	// player's saved tactics (FocusSet), and whether the order waits for
	// the next round's upkeep (one order a round).
	Focus        string
	FocusSet     bool
	FocusPending bool

	// Phase 30c2: guardians' guards by member key; one absent has spent
	// none (MaxGuards left).
	Guards map[string]Guard
	// Phase 35b: each guardian's most guards, captured from its level
	// the first time its guards are read in the battle (MaxGuardsFor).
	GuardMax map[string]int

	// Phase 33i2: the enemy group's coordination tier, fixed when the
	// battle began (0: none given, read as a rabble), and the guards its
	// guardians have spent in it, all together.
	Coordination int
	EnemyGuards  int
	Opening      int    // -1: company surprised; +1: enemy surprised; opening round only
	EnemyFocus   string // the member key the group's leader last aimed it at

	// Phase 39c: the weather a Shaman has called into this battle.
	Weather Weather

	// Phase 54: the sigil the company stood in when the battle began, and
	// the Unix second it fades.
	Sigil        sigils.Kind
	SigilExpires int64
}

// Has reports whether instanceId is one of the battle's enemies.
func (b Battle) Has(instanceId int) bool { return b.Enemies[instanceId] }

func (b Battle) clone() Battle {
	c := b
	c.Enemies = make(map[int]bool, len(b.Enemies))
	for id := range b.Enemies {
		c.Enemies[id] = true
	}
	c.EnemyNames = cloneEnemyNames(b.EnemyNames)
	if b.GuardMax != nil {
		c.GuardMax = make(map[string]int, len(b.GuardMax))
		for k, n := range b.GuardMax {
			c.GuardMax[k] = n
		}
	}
	if b.Guards != nil {
		c.Guards = make(map[string]Guard, len(b.Guards))
		for k, g := range b.Guards {
			c.Guards[k] = g
		}
	}
	return c
}

var (
	mu       sync.Mutex
	battles  = map[int]*Battle{}
	firstSet = map[int]map[string]uint64{} // user -> party id -> round first set on them
)

// Current returns the player's battle.
func Current(userId int) (Battle, bool) {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return Battle{}, false
	}
	return b.clone(), true
}

// Begin starts the player's battle against a group, replacing any other.
func Begin(userId, roomId int, round uint64, partyID string, enemies []int) Battle {
	mu.Lock()
	defer mu.Unlock()
	b := &Battle{UserId: userId, RoomId: roomId, PartyID: partyID, StartRound: round, Enemies: map[int]bool{}, EnemyNames: map[int]EnemyName{}}
	for _, id := range enemies {
		b.Enemies[id] = true
	}
	// Phase 33i2: a group already fighting another player has already
	// spent its guards there.
	for _, other := range battles {
		if other.PartyID == partyID && partyID != "" {
			b.EnemyGuards = max(b.EnemyGuards, other.EnemyGuards)
		}
	}
	battles[userId] = b
	return b.clone()
}

// SetFight records the battle's fight on the combat event stream.
func SetFight(userId int, fightID uint64) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok {
		b.FightID = fightID
	}
}

// Grow adds the group's members seen this round, and its current party id,
// to the player's battle.
func Grow(userId int, partyID string, enemies []int) {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return
	}
	if partyID != "" {
		b.PartyID = partyID
	}
	for _, id := range enemies {
		b.Enemies[id] = true
	}
}

// End ends the player's battle, and forgets when its group was set on
// them.
func End(userId int) (Battle, bool) {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return Battle{}, false
	}
	delete(battles, userId)
	if set, ok := firstSet[userId]; ok {
		delete(set, b.PartyID)
	}
	return b.clone(), true
}

// NoteSet records that a group is set on the player, the first time only,
// and returns the round it was first set.
func NoteSet(userId int, partyID string, round uint64) uint64 {
	mu.Lock()
	defer mu.Unlock()
	set, ok := firstSet[userId]
	if !ok {
		set = map[string]uint64{}
		firstSet[userId] = set
	}
	if r, ok := set[partyID]; ok {
		return r
	}
	set[partyID] = round
	return round
}

// Waiting lists the groups set on a player in a battle, other than the
// battle's own, in the order they were first set (ties by party id): the
// groups waiting their turn (32g2's battle view). Nothing waits on a
// player who isn't in a battle.
func Waiting(userId int) []string {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return nil
	}
	var out []string
	for id := range firstSet[userId] {
		if id != b.PartyID {
			out = append(out, id)
		}
	}
	set := firstSet[userId]
	sort.Slice(out, func(i, j int) bool {
		if set[out[i]] != set[out[j]] {
			return set[out[i]] < set[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// KeepSet forgets every group set on the player except those named: a
// group no longer set on them waits in line no longer.
func KeepSet(userId int, partyIDs []string) {
	mu.Lock()
	defer mu.Unlock()
	set, ok := firstSet[userId]
	if !ok {
		return
	}
	keep := map[string]bool{}
	for _, id := range partyIDs {
		keep[id] = true
	}
	for id := range set {
		if !keep[id] {
			delete(set, id)
		}
	}
	if len(set) == 0 {
		delete(firstSet, userId)
	}
}

// Forget drops everything held for a player (they left the game).
func Forget(userId int) {
	mu.Lock()
	defer mu.Unlock()
	delete(battles, userId)
	delete(firstSet, userId)
}

// Players lists the players with a battle, ascending.
func Players() []int {
	mu.Lock()
	defer mu.Unlock()
	out := make([]int, 0, len(battles))
	for id := range battles {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// Reset clears everything, for tests.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	battles = map[int]*Battle{}
	firstSet = map[int]map[string]uint64{}
}

// Candidate is a group set on a player, waiting for a battle.
type Candidate struct {
	PartyID  string
	FirstSet uint64 // the round it was first set on the player
	Order    int    // its order in the room, for ties
}

// Next picks the group a player fights next: the one set on them first,
// ties by room order. ok is false when there is none.
func Next(candidates []Candidate) (int, bool) {
	best := -1
	for i, c := range candidates {
		if best < 0 || c.FirstSet < candidates[best].FirstSet ||
			(c.FirstSet == candidates[best].FirstSet && c.Order < candidates[best].Order) {
			best = i
		}
	}
	return best, best >= 0
}

// Engaged reports whether a mob is one of any player's battle enemies.
func Engaged(instanceId int) bool {
	mu.Lock()
	defer mu.Unlock()
	for _, b := range battles {
		if b.Enemies[instanceId] {
			return true
		}
	}
	return false
}

// Involving lists the players whose battles count instanceId among their
// enemies, without copying any battle.
func Involving(instanceId int) []int {
	mu.Lock()
	defer mu.Unlock()
	var out []int
	for id, b := range battles {
		if b.Enemies[instanceId] {
			out = append(out, id)
		}
	}
	return out
}

// RoomOf returns the room of the player's battle, if they have one.
func RoomOf(userId int) (int, bool) {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return 0, false
	}
	return b.RoomId, true
}

// Retain drops everything held for players not named: those who left the
// game.
func Retain(userIds map[int]bool) {
	mu.Lock()
	defer mu.Unlock()
	for id := range battles {
		if !userIds[id] {
			delete(battles, id)
		}
	}
	for id := range firstSet {
		if !userIds[id] {
			delete(firstSet, id)
		}
	}
}

// Allows reports whether the player userId may fight the mob instanceId
// now: they have no battle, or the mob is one of its enemies.
func Allows(userId, instanceId int) bool {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	return !ok || b.Enemies[instanceId]
}

// SetOpening records a consumed encounter advantage, never durable state.
func SetOpening(uid, advantage int) {
	mu.Lock()
	defer mu.Unlock()
	if b := battles[uid]; b != nil {
		b.Opening = advantage
	}
}
