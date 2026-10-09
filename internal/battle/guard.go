package battle

// Phase 30c2: a guardian's guards. Each guardian on a player's side has
// MaxGuards guards in a battle (Phase 35b: MaxGuardsFor its level,
// captured when its guards are first read); a spent guard comes back one
// per GuardRefillRounds combat rounds, up to that most (the owner's
// decision 9). The counts live on the battle, runtime only: a new battle, or one
// rebuilt after a restart or copyover, starts full.

const (
	// MaxGuards is the guards a guardian has in a battle before its level
	// adds more (Phase 35b: MaxGuardsFor).
	MaxGuards = 2
	// GuardRefillRounds is the combat rounds a spent guard takes to come
	// back.
	GuardRefillRounds = 2
)

// Guard is one guardian's guards in a battle: those left, and the combat
// rounds counted toward the next one back.
type Guard struct {
	Left   int
	Charge int
}

// MaxGuardsFor is a guardian's guards a battle at its level (Phase 35b):
// 2, and 1 more each 10 levels.
func MaxGuardsFor(level int) int {
	return MaxGuards + max(level, 0)/10
}

// maxOf is the member's most guards in b: as captured, else MaxGuards.
func maxOf(b *Battle, key string) int {
	if n, ok := b.GuardMax[key]; ok {
		return n
	}
	return MaxGuards
}

// CaptureGuards fixes the member's most guards in the player's battle from
// its level, the first time only: a level gained mid-battle waits for the
// next battle. No battle does nothing.
func CaptureGuards(userId int, key string, level int) {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return
	}
	if _, set := b.GuardMax[key]; set {
		return
	}
	if b.GuardMax == nil {
		b.GuardMax = map[string]int{}
	}
	b.GuardMax[key] = MaxGuardsFor(level)
}

// guardOf is the member's guards in b, full when it has spent none.
func guardOf(b *Battle, key string) Guard {
	if g, ok := b.Guards[key]; ok {
		return g
	}
	return Guard{Left: maxOf(b, key)}
}

// GuardsLeft is the guards the member (by member key) has left in the
// player's battle; 0 with no battle.
func GuardsLeft(userId int, key string) int {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok {
		return 0
	}
	return guardOf(b, key).Left
}

// SpendGuard spends one of the member's guards in the player's battle and
// returns how many are left; ok is false when there is no battle or no
// guard left.
func SpendGuard(userId int, key string) (left int, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	b, found := battles[userId]
	if !found {
		return 0, false
	}
	g := guardOf(b, key)
	if g.Left < 1 {
		return 0, false
	}
	g.Left--
	if b.Guards == nil {
		b.Guards = map[string]Guard{}
	}
	b.Guards[key] = g
	return g.Left, true
}

// SpendBondGuard spends one of the company's bond steps (Phase 65: a friend
// stepping in for a friend) in the player's battle, at most max in all; ok
// is false when there is no battle or the company has made its steps.
func SpendBondGuard(userId, max int) bool {
	mu.Lock()
	defer mu.Unlock()
	b, found := battles[userId]
	if !found || b.BondGuards >= max {
		return false
	}
	b.BondGuards++
	return true
}

// TickGuards counts one combat round toward every spent guard's return,
// in every battle. Called once per combat round.
func TickGuards() {
	mu.Lock()
	defer mu.Unlock()
	for _, b := range battles {
		for key, g := range b.Guards {
			top := maxOf(b, key)
			if g.Left >= top {
				continue
			}
			g.Charge++
			if g.Charge >= GuardRefillRounds {
				g.Left++
				g.Charge = 0
			}
			if g.Left >= top {
				delete(b.Guards, key) // full again: as if none were spent
				continue
			}
			b.Guards[key] = g
		}
	}
}

// SetCoordination records the enemy group's coordination tier on the
// player's battle (Phase 33i2).
func SetCoordination(userId, tier int) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok {
		b.Coordination = tier
	}
}

// SpendEnemyGuard spends one of the enemy group's guards in the player's
// battle, when it has spent fewer than limit (Phase 33i2: a tier's guards
// are the group's for its battle, all told, and never come back). A group
// fighting several players spends from one count: every battle against
// it is charged (33i2 review finding 2). ok is false with no battle or
// none left.
func SpendEnemyGuard(userId, limit int) (left int, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	b, found := battles[userId]
	if !found || b.EnemyGuards >= limit {
		return 0, false
	}
	spent := b.EnemyGuards + 1
	for _, other := range battles {
		if other.PartyID == b.PartyID {
			other.EnemyGuards = max(other.EnemyGuards, spent)
		}
	}
	return limit - spent, true
}

// SetEnemyFocus records the enemy group's focus (a company member key) on
// the player's battle and reports whether it changed (Phase 33i2: a
// coordinated group says its focus aloud once per change).
func SetEnemyFocus(userId int, key string) bool {
	mu.Lock()
	defer mu.Unlock()
	b, ok := battles[userId]
	if !ok || b.EnemyFocus == key {
		return false
	}
	b.EnemyFocus = key
	return true
}
