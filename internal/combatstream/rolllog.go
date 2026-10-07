package combatstream

import (
	"fmt"
	"strings"
	"sync"
)

// Phase 62: the roll log keeps each company's recent weapon rounds with the
// parts of their strikes, so `why` can explain a line after it scrolled
// by. It is read-only reporting like the rest of the stream: runtime
// only, never saved, and never a source of truth for the fight.

// RollLogCap is how many rounds a leader's log keeps.
const RollLogCap = 40

// Roll is one weapon round the leader's company fought, as the stream saw it.
type Roll struct {
	Round    uint64
	FightID  uint64
	Source   Ref
	Target   Ref
	Outcome  string
	Damage   int
	Defenses []string
	Quality  string
	Crit     bool
	Strikes  []Strike
	// Ours is true when a member of the company swung.
	Ours bool
}

// RollLog is the rolls of each leader's latest fight, newest last.
type RollLog struct {
	mu    sync.Mutex
	rolls map[int][]Roll
}

// NewRollLog returns an empty log.
func NewRollLog() *RollLog { return &RollLog{rolls: map[int][]Roll{}} }

var defaultRollLog = NewRollLog()

// DefaultRollLog is the process-wide roll log.
func DefaultRollLog() *RollLog { return defaultRollLog }

// Clear forgets a leader's rolls (a new fight began).
func (l *RollLog) Clear(leaderUserId int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.rolls, leaderUserId)
}

// Add records a leader's roll, dropping the oldest past RollLogCap.
func (l *RollLog) Add(leaderUserId int, r Roll) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rolls := append(l.rolls[leaderUserId], r)
	if len(rolls) > RollLogCap {
		rolls = append([]Roll(nil), rolls[len(rolls)-RollLogCap:]...)
	}
	l.rolls[leaderUserId] = rolls
}

// Recent returns up to n of a leader's rolls, newest first. n <= 0 means all.
func (l *RollLog) Recent(leaderUserId, n int) []Roll {
	l.mu.Lock()
	defer l.mu.Unlock()
	rolls := l.rolls[leaderUserId]
	if n <= 0 || n > len(rolls) {
		n = len(rolls)
	}
	out := make([]Roll, 0, n)
	for i := len(rolls) - 1; i >= len(rolls)-n; i-- {
		out = append(out, rolls[i])
	}
	return out
}

// RollFor turns an attack event into the log's entry; ok is false for an
// event with nothing to explain.
func RollFor(e Event, ours bool) (Roll, bool) {
	if e.Kind != Attack || len(e.Strikes) == 0 {
		return Roll{}, false
	}
	return Roll{Round: e.Round, FightID: e.FightID, Source: e.Source, Target: e.Target, Outcome: e.Outcome,
		Damage: e.Damage, Defenses: e.Defenses, Quality: e.Quality, Crit: e.Crit, Strikes: e.Strikes, Ours: ours}, true
}

// Describe lays a roll out for the player viewerUserId, who reads their own
// name as "You": a heading line, then each strike's plain breakdown.
func (r Roll) Describe(viewerUserId int) []string {
	name := func(x Ref) string {
		switch {
		case x.UserId != 0 && x.UserId == viewerUserId:
			return "You"
		case x.Name == "":
			return "someone"
		}
		return x.Name
	}
	result := "missed"
	switch {
	case r.Damage > 0 && r.Crit:
		result = fmt.Sprintf("landed a critical hit for %d", r.Damage)
	case r.Damage > 0:
		result = fmt.Sprintf("hit for %d", r.Damage)
	case len(r.Defenses) > 0:
		result = "was turned aside"
	case r.Outcome == OutcomeHit || r.Outcome == OutcomeCrit:
		result = "hit but did no damage"
	}
	out := []string{fmt.Sprintf("Round %d: %s -> %s, %s.", r.Round, name(r.Source), name(r.Target), result)}
	for _, line := range r.Breakdown() {
		out = append(out, "  "+line)
	}
	return out
}

// Breakdown is the round's strikes in plain lines: each strike's
// explanation, led by "Strike n:" when the round threw more than one.
func Breakdown(strikes []Strike) []string {
	var out []string
	for i, s := range strikes {
		prefix := ""
		if len(strikes) > 1 {
			prefix = fmt.Sprintf("Strike %d: ", i+1)
		}
		for j, line := range s.Explain() {
			if j == 0 {
				line = prefix + line
			} else if prefix != "" {
				line = strings.Repeat(" ", len(prefix)) + line
			}
			out = append(out, line)
		}
	}
	return out
}

// Breakdown is the roll's strikes in plain lines.
func (r Roll) Breakdown() []string { return Breakdown(r.Strikes) }
