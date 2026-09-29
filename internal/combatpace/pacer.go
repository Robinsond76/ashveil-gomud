package combatpace

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Beat is how long a line waits after the line before it.
type Beat int

const (
	Plain    Beat = iota // the pace's ordinary gap
	Dramatic             // a critical hit's pain or death line
	Quick                // an indented follow-up line
)

// Release is a held line now due, for its player.
type Release struct {
	UserId int
	Text   string
}

type held struct {
	text string
	beat Beat
}

// queue is one player's held lines of one combat round.
type queue struct {
	round uint64
	start time.Time
	spec  Spec
	lines []held
	next  int // index of the next line to release
}

// Pacer holds each player's combat lines. It is safe for concurrent use,
// takes no other lock, and never calls out while holding its own.
type Pacer struct {
	mu     sync.Mutex
	queues map[int]*queue
	marks  map[string]struct{}
}

func New() *Pacer {
	return &Pacer{queues: map[int]*queue{}, marks: map[string]struct{}{}}
}

var (
	defaultMu    sync.RWMutex
	defaultPacer = New()
)

// Default is the game's pacer.
func Default() *Pacer {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultPacer
}

// UseForTest makes p the default pacer and returns a func restoring the
// previous one.
func UseForTest(p *Pacer) (restore func()) {
	defaultMu.Lock()
	prev := defaultPacer
	defaultPacer = p
	defaultMu.Unlock()
	return func() {
		defaultMu.Lock()
		defaultPacer = prev
		defaultMu.Unlock()
	}
}

func key(text string) string { return strings.TrimRight(text, "\r\n") }

// Mark flags lines as dramatic for this round: when one is held, it waits
// the longer gap. Narration marks a critical hit's pain and death lines.
func (p *Pacer) Mark(texts ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range texts {
		p.marks[key(t)] = struct{}{}
	}
}

// Marked reports whether a line is marked dramatic this round.
func (p *Pacer) Marked(text string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.marks[key(text)]
	return ok
}

// StartRound forgets the previous round's marks. It is called as each
// combat round begins, after FlushAll.
func (p *Pacer) StartRound() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.marks = map[string]struct{}{}
}

func (p *Pacer) beatOf(text string) Beat {
	if _, ok := p.marks[key(text)]; ok {
		return Dramatic
	}
	if strings.HasPrefix(text, " ") || strings.HasPrefix(text, "\t") {
		return Quick
	}
	return Plain
}

// Hold queues a line of round's text for a player. A line of an older round
// still held is returned first, for the caller to send at once: a round's
// lines never run into the next round's.
func (p *Pacer) Hold(userId int, round uint64, text string, spec Spec, now time.Time) (flushed []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	q := p.queues[userId]
	if q != nil && q.round != round {
		flushed = q.remaining()
		q = nil
	}
	if q == nil {
		q = &queue{round: round, start: now, spec: spec}
		p.queues[userId] = q
	}
	q.lines = append(q.lines, held{text: text, beat: p.beatOf(text)})
	return flushed
}

// Follow adds a line to the end of a player's held lines, if they have any,
// so it can't jump ahead of the round it follows. It reports whether it
// did; with nothing held, the caller sends the line at once.
func (p *Pacer) Follow(userId int, text string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	q := p.queues[userId]
	if q == nil || q.next >= len(q.lines) {
		return false
	}
	q.lines = append(q.lines, held{text: text, beat: p.beatOf(text)})
	return true
}

func (q *queue) remaining() []string {
	out := make([]string, 0, len(q.lines)-q.next)
	for _, l := range q.lines[q.next:] {
		out = append(out, l.text)
	}
	q.next = len(q.lines)
	return out
}

func (s Spec) gap(b Beat) time.Duration {
	switch b {
	case Dramatic:
		return s.Dramatic
	case Quick:
		return s.Quick
	}
	return s.Gap
}

// offsets is each line's time after the round's first line: its gaps,
// scaled down together when they would overrun the window.
func (q *queue) offsets() []time.Duration {
	out := make([]time.Duration, len(q.lines))
	var total time.Duration
	for i := 1; i < len(q.lines); i++ {
		total += q.spec.gap(q.lines[i].beat)
		out[i] = total
	}
	if total > q.spec.Window && total > 0 {
		for i := range out {
			out[i] = time.Duration(int64(out[i]) * int64(q.spec.Window) / int64(total))
		}
	}
	return out
}

// Due releases every line due by now, in order, player by player, and
// reports the players whose lines have all gone out.
func (p *Pacer) Due(now time.Time) (out []Release, drained []int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, userId := range p.userIdsLocked() {
		q := p.queues[userId]
		offs := q.offsets()
		for q.next < len(q.lines) && !now.Before(q.start.Add(offs[q.next])) {
			out = append(out, Release{UserId: userId, Text: q.lines[q.next].text})
			q.next++
		}
		if q.next >= len(q.lines) {
			delete(p.queues, userId)
			drained = append(drained, userId)
		}
	}
	return out, drained
}

func (p *Pacer) userIdsLocked() []int {
	ids := make([]int, 0, len(p.queues))
	for id := range p.queues {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// Flush returns a player's held lines, in order, and holds nothing more
// for them.
func (p *Pacer) Flush(userId int) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	q := p.queues[userId]
	if q == nil {
		return nil
	}
	delete(p.queues, userId)
	return q.remaining()
}

// FlushAll returns every held line, player by player and in order, and the
// players who had any.
func (p *Pacer) FlushAll() (out []Release, drained []int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, userId := range p.userIdsLocked() {
		for _, text := range p.queues[userId].remaining() {
			out = append(out, Release{UserId: userId, Text: text})
		}
		delete(p.queues, userId)
		drained = append(drained, userId)
	}
	return out, drained
}

// Busy reports whether a player has lines still held.
func (p *Pacer) Busy(userId int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	q := p.queues[userId]
	return q != nil && q.next < len(q.lines)
}
