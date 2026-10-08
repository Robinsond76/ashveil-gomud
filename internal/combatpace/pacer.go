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
	Instant              // no wait: a block that goes out whole (the battle summary)
)

// Release is a held entry now due, for its player: a line of text, or (when
// IsData) a data entry the caller delivers some other way (Phase 40e: the
// web client's structured combat events), carried in Data.
type Release struct {
	UserId int
	Text   string
	Data   any
	IsData bool
}

// held is one entry of a round's queue. A data entry takes no beat of its
// own: it goes out with the next text line, or at the round's end.
type held struct {
	text   string
	beat   Beat
	slot   int // the fighter's turn the line belongs to (Phase 82c), 0 for upkeep
	data   any
	isData bool
}

func (h held) release(userId int) Release {
	return Release{UserId: userId, Text: h.text, Data: h.data, IsData: h.isData}
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
	// report are players whose later lines this round go out without a wait.
	report map[int]struct{}
	// open are players in a combat round whose lines haven't all gone out:
	// from the round's start (before its lines are held) until they drain.
	open map[int]struct{}
}

func New() *Pacer {
	return &Pacer{queues: map[int]*queue{}, marks: map[string]struct{}{}, report: map[int]struct{}{}, open: map[int]struct{}{}}
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

// StartReport makes the player's lines held from now on in this round go
// out with no wait of their own: the fight's report (its summary and what
// the end of the fight causes) arrives whole instead of a beat a line.
func (p *Pacer) StartReport(userId int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.report[userId] = struct{}{}
}

// EndReport stops StartReport: the player's lines held from now on take
// their beats again (their next battle began in the same round).
func (p *Pacer) EndReport(userId int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.report, userId)
}

// Marked reports whether a line is marked dramatic this round.
func (p *Pacer) Marked(text string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.marks[key(text)]
	return ok
}

// StartRound forgets the previous round's marks and opens the round for
// the players who pace combat: they are Busy until Due finds their lines
// all gone out (or finds they had none). It is called as each combat round
// begins, after FlushAll.
func (p *Pacer) StartRound(userIds ...int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.marks = map[string]struct{}{}
	p.report = map[int]struct{}{}
	p.open = make(map[int]struct{}, len(userIds))
	for _, id := range userIds {
		p.open[id] = struct{}{}
	}
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

// Hold queues a line of round's text for a player. Entries of an older round
// still held are returned first, for the caller to send at once: a round's
// lines never run into the next round's. A late line of an older round (a
// caused event requeued past the round's end) joins the newer round's
// lines rather than cutting them short. slot is the fighter's turn the
// line belongs to (0 for the round's upkeep), which paces by action under a
// Beats spec.
func (p *Pacer) Hold(userId int, round uint64, slot int, text string, spec Spec, now time.Time) (flushed []Release) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.holdLocked(userId, round, held{text: text, beat: p.beatOf(text), slot: slot}, spec, now)
}

// HoldData queues a data entry among a player's held lines, in order. It
// takes no beat of its own: it is released with the next text line that
// follows it, or when the last line goes out. Older rounds' entries come
// back as Hold's do.
func (p *Pacer) HoldData(userId int, round uint64, data any, spec Spec, now time.Time) (flushed []Release) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.holdLocked(userId, round, held{data: data, isData: true}, spec, now)
}

func (p *Pacer) holdLocked(userId int, round uint64, h held, spec Spec, now time.Time) (flushed []Release) {
	if _, ok := p.report[userId]; ok && !h.isData {
		h.beat = Instant
	}
	q := p.queues[userId]
	if q != nil && round < q.round {
		q.lines = append(q.lines, h)
		return nil
	}
	if q != nil && q.round != round {
		flushed = q.remaining(userId)
		q = nil
	}
	if q == nil {
		q = &queue{round: round, start: now, spec: spec}
		p.queues[userId] = q
	}
	q.lines = append(q.lines, h)
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
	h := held{text: text, beat: p.beatOf(text), slot: q.lastSlot()}
	if _, ok := p.report[userId]; ok {
		h.beat = Instant
	}
	q.lines = append(q.lines, h)
	return true
}

// lastSlot is the slot of the last text line held, so a line that follows
// the round joins its last turn's beat.
func (q *queue) lastSlot() int {
	for i := len(q.lines) - 1; i >= 0; i-- {
		if !q.lines[i].isData {
			return q.lines[i].slot
		}
	}
	return 0
}

// FollowData is Follow for a data entry.
func (p *Pacer) FollowData(userId int, data any) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	q := p.queues[userId]
	if q == nil || q.next >= len(q.lines) {
		return false
	}
	q.lines = append(q.lines, held{data: data, isData: true})
	return true
}

func (q *queue) remaining(userId int) []Release {
	out := make([]Release, 0, len(q.lines)-q.next)
	for _, l := range q.lines[q.next:] {
		out = append(out, l.release(userId))
	}
	q.next = len(q.lines)
	return out
}

// gap is the wait before a line: by line from its beat; by action from
// whether it opens a new turn (newSlot), with the extra before a pain or
// death line.
func (s Spec) gap(b Beat, newSlot bool) time.Duration {
	if b == Instant {
		return 0
	}
	if s.Beats {
		d := s.Follow
		if newSlot {
			d = s.Beat
		}
		if b == Dramatic {
			d += s.Extra
		}
		return d
	}
	switch b {
	case Dramatic:
		return s.Dramatic
	case Quick:
		return s.Quick
	}
	return s.Gap
}

// offsets is each entry's time after the round's first line: its gaps,
// scaled down together when they would overrun the window. A data entry
// takes the offset of the next text line, or of the last one.
func (q *queue) offsets() []time.Duration {
	out := make([]time.Duration, len(q.lines))
	var total time.Duration
	first := true
	prevSlot := 0
	for i, l := range q.lines {
		if l.isData {
			continue
		}
		if !first {
			total += q.spec.gap(l.beat, l.slot != prevSlot)
		}
		first = false
		prevSlot = l.slot
		out[i] = total
	}
	scale := 1.0
	if q.spec.Window > 0 && total > q.spec.Window && total > 0 {
		// Scale in floating point: offset × window overflows int64
		// nanoseconds once a round runs past a dozen seconds of gaps.
		scale = float64(q.spec.Window) / float64(total)
		for i, l := range q.lines {
			if !l.isData {
				out[i] = time.Duration(float64(out[i]) * scale)
			}
		}
		total = time.Duration(float64(total) * scale)
	}
	next := total
	for i := len(q.lines) - 1; i >= 0; i-- {
		if q.lines[i].isData {
			out[i] = next
		} else {
			next = out[i]
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
			out = append(out, q.lines[q.next].release(userId))
			q.next++
		}
		if q.next >= len(q.lines) {
			delete(p.queues, userId)
			delete(p.open, userId)
			delete(p.report, userId)
			drained = append(drained, userId)
		}
	}
	// Players the round sent nothing: their round is over.
	var idle []int
	for userId := range p.open {
		if _, held := p.queues[userId]; !held {
			idle = append(idle, userId)
		}
	}
	sort.Ints(idle)
	for _, userId := range idle {
		delete(p.open, userId)
		drained = append(drained, userId)
	}
	sort.Ints(drained)
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

// Flush returns a player's held lines, in order, and ends their round:
// ended is true when they were Busy, so views held back with them can
// catch up.
func (p *Pacer) Flush(userId int) (lines []Release, ended bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ended = p.open[userId]
	delete(p.open, userId)
	delete(p.report, userId)
	if q := p.queues[userId]; q != nil {
		delete(p.queues, userId)
		lines = q.remaining(userId)
		ended = true
	}
	return lines, ended
}

// FlushAll returns every held line, player by player and in order, and the
// players who had any.
func (p *Pacer) FlushAll() (out []Release, drained []int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, userId := range p.userIdsLocked() {
		out = append(out, p.queues[userId].remaining(userId)...)
		delete(p.queues, userId)
		delete(p.open, userId)
		delete(p.report, userId)
		drained = append(drained, userId)
	}
	for userId := range p.open {
		delete(p.open, userId)
		drained = append(drained, userId)
	}
	sort.Ints(drained)
	return out, drained
}

// PlaybackEnd is when the last held line of any player goes out, or the
// zero time when nothing is held: the battle clock (Phase 82c) starts the
// next round after it.
func (p *Pacer) PlaybackEnd() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	var end time.Time
	for _, q := range p.queues {
		offs := q.offsets()
		for i := len(q.lines) - 1; i >= 0; i-- {
			if !q.lines[i].isData {
				if t := q.start.Add(offs[i]); t.After(end) {
					end = t
				}
				break
			}
		}
	}
	return end
}

// Holding reports whether any player still has lines held: a round is still
// playing out to someone. The fixed cadence (Phase 82c) waits for it, so a
// fight's last round is never cut short by the next cadence round's flush.
func (p *Pacer) Holding() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, q := range p.queues {
		if q.next < len(q.lines) {
			return true
		}
	}
	return false
}

// Busy reports whether a player is in a combat round whose lines haven't
// all gone out: views that must not run ahead of the narration (the
// prompt, the web client's vitals and battle view) wait while it is true.
func (p *Pacer) Busy(userId int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.open[userId]; ok {
		return true
	}
	q := p.queues[userId]
	return q != nil && q.next < len(q.lines)
}
