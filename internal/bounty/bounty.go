// Package bounty is Phase 76's bounty boards: a town board posts bounties on
// lair bosses and named groups from nearby zones, rotated on real time, paid
// when the company's chronicle shows the kill. The package is GoMud-free:
// the posting rule, the pay and the saved shapes. modules/bounties binds it
// to board rooms, the chronicle and the `bounty` command.
//
// Everything is real time (Unix seconds). A bounty names a foe the zone
// already has; it never changes a foe's level or numbers.
package bounty

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
)

// Kinds of target.
const (
	Boss  = "boss"  // a lair's master, proved by a boss deed
	Group = "group" // a named ordinary group, proved by group deeds
)

// Numbers (full autonomy, recorded in the phase 76 plan).
const (
	WindowSeconds    = 6 * 60 * 60  // the board's rotation: a new list every six hours
	TermSeconds      = 24 * 60 * 60 // a taken bounty must be claimed within a day
	PostingsPerBoard = 5            // postings on a board at a time
	MaxBosses        = 2            // at most this many lair bosses among them, when groups exist
	MaxHeld          = 3            // bounties a company may hold at once
	GroupCount       = 3            // groups to break for a group bounty
	BossPerLevel     = 20           // gold per level of the band's middle, for a boss
	GroupPerLevel    = 6            // gold per level of the band's middle, for each group
	// Nearby: a target zone counts when its band reaches no further than this
	// below the board's band or above it. A board in a zone with no band
	// posts only targets whose band starts at or below NoBandCeiling.
	NearbyBelow    = 2
	NearbyAbove    = 3
	NoBandCeiling  = 6
	DoneKeepWindow = 2 // claimed postings are remembered this many terms
)

// Band is a zone's recommended level band.
type Band struct{ Low, High int }

// Valid reports whether the band is usable.
func (b Band) Valid() bool { return b.Low >= 1 && b.High >= b.Low }

// Mid is the band's middle level, at least 1.
func (b Band) Mid() int { return max(1, (b.Low+b.High)/2) }

func (b Band) String() string {
	if b.Low == b.High {
		return fmt.Sprintf("%d", b.Low)
	}
	return fmt.Sprintf("%d-%d", b.Low, b.High)
}

// Target is what a bounty asks the company to kill. Ref is the chronicle
// reference that proves a kill: "mob:<id>" for a boss, "group:<id>" for a
// group. Zone is where the kill must happen.
type Target struct {
	Kind  string `yaml:"kind" json:"kind"`
	Zone  string `yaml:"zone" json:"zone"`
	Ref   string `yaml:"ref" json:"ref"`
	Name  string `yaml:"name" json:"name"`
	Count int    `yaml:"count" json:"count"`
	Low   int    `yaml:"low" json:"low"`
	High  int    `yaml:"high" json:"high"`
}

// Band is the target zone's band.
func (t Target) Band() Band { return Band{t.Low, t.High} }

// Reward is the gold a bounty on t pays: the band's middle level times a
// rate, for a boss once and for a group for each group asked.
func Reward(t Target) int {
	mid := t.Band().Mid()
	if t.Kind == Boss {
		return BossPerLevel * mid
	}
	return GroupPerLevel * mid * max(1, t.Count)
}

// Posting is one bounty on a board's list. ID is stable for the window, so
// a bounty taken from it can be matched and never paid twice.
type Posting struct {
	ID     string `json:"id"`
	Target Target `json:"target"`
	Reward int    `json:"reward"`
	Until  int64  `json:"until"` // when the board stops offering it
}

// Window is the start of the rotation window now falls in.
func Window(now int64) int64 { return now - now%WindowSeconds }

// Nearby reports whether a target zone's band suits a board whose zone has
// board (valid or not).
func Nearby(board Band, target Band) bool {
	if !target.Valid() {
		return false
	}
	if !board.Valid() {
		return target.Low <= NoBandCeiling
	}
	return target.High >= board.Low-NearbyBelow && target.Low <= board.High+NearbyAbove
}

// PostingID names a posting for a window.
func PostingID(window int64, t Target) string {
	return fmt.Sprintf("%d:%s:%s", window, t.Zone, t.Ref)
}

// Post is the board's list for a window: the same board and window always
// give the same list. Candidates are every lair boss and named group the
// zones hold; the board keeps the nearby ones, shuffles them with a seed
// from the board and the window, and takes up to PostingsPerBoard with at
// most MaxBosses lair bosses (more only when no groups fill the board).
func Post(board string, boardBand Band, window int64, candidates []Target) []Posting {
	var bosses, groups []Target
	seen := map[string]bool{}
	for _, c := range candidates {
		key := c.Zone + "|" + c.Ref
		if seen[key] || !Nearby(boardBand, c.Band()) {
			continue
		}
		seen[key] = true
		if c.Kind == Boss {
			bosses = append(bosses, c)
		} else {
			groups = append(groups, c)
		}
	}
	order := func(s []Target) {
		sort.Slice(s, func(i, j int) bool {
			if s[i].Zone != s[j].Zone {
				return s[i].Zone < s[j].Zone
			}
			return s[i].Ref < s[j].Ref
		})
	}
	order(bosses)
	order(groups)
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%d", board, window)
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	rng.Shuffle(len(bosses), func(i, j int) { bosses[i], bosses[j] = bosses[j], bosses[i] })
	rng.Shuffle(len(groups), func(i, j int) { groups[i], groups[j] = groups[j], groups[i] })

	var picked []Target
	nb := min(MaxBosses, len(bosses))
	picked = append(picked, bosses[:nb]...)
	ng := min(PostingsPerBoard-len(picked), len(groups))
	picked = append(picked, groups[:ng]...)
	rest := bosses[nb:]
	for len(picked) < PostingsPerBoard && len(rest) > 0 {
		picked = append(picked, rest[0])
		rest = rest[1:]
	}
	// Lair bosses lead the list; each kind keeps its shuffled order.
	sort.SliceStable(picked, func(i, j int) bool { return picked[i].Kind == Boss && picked[j].Kind != Boss })
	out := make([]Posting, 0, len(picked))
	for _, t := range picked {
		if t.Kind == Group {
			t.Count = GroupCount
		} else {
			t.Count = 1
		}
		out = append(out, Posting{ID: PostingID(window, t), Target: t, Reward: Reward(t), Until: window + WindowSeconds})
	}
	return out
}

// Held is a bounty a company has taken: saved, so it survives restart and
// copyover. AfterSeq is the newest chronicle deed's number when it was taken,
// so only later deeds count as proof.
type Held struct {
	PostID   string `yaml:"post_id" json:"post_id"`
	Target   Target `yaml:"target" json:"target"`
	Reward   int    `yaml:"reward" json:"reward"`
	TakenAt  int64  `yaml:"taken_at" json:"taken_at"`
	Due      int64  `yaml:"due" json:"due"`
	AfterSeq int    `yaml:"after_seq" json:"after_seq"`
}

// State is one company's bounties: those held, and the postings it has
// claimed or let lapse (by posting id, with the time to forget them).
type State struct {
	Held []Held           `yaml:"held,omitempty"`
	Done map[string]int64 `yaml:"done,omitempty"`
}

// Expired removes held bounties past their due time and forgotten postings,
// returning how many bounties lapsed.
func (s *State) Expired(now int64) int {
	kept := s.Held[:0]
	lapsed := 0
	for _, h := range s.Held {
		if now >= h.Due {
			lapsed++
			continue
		}
		kept = append(kept, h)
	}
	s.Held = kept
	for id, forget := range s.Done {
		if now >= forget {
			delete(s.Done, id)
		}
	}
	return lapsed
}

// Clone shares nothing with s.
func (s State) Clone() State {
	out := State{Held: append([]Held(nil), s.Held...)}
	if s.Done != nil {
		out.Done = make(map[string]int64, len(s.Done))
		for k, v := range s.Done {
			out.Done[k] = v
		}
	}
	return out
}

// Empty reports whether the state holds nothing worth saving.
func (s State) Empty() bool { return len(s.Held) == 0 && len(s.Done) == 0 }

// Holds reports whether a bounty on the same target is already held.
func (s State) Holds(t Target) bool {
	for _, h := range s.Held {
		if h.Target.Zone == t.Zone && h.Target.Ref == t.Ref {
			return true
		}
	}
	return false
}

// Take problems.
var (
	ErrFull      = fmt.Errorf("you hold %d bounties already; claim or drop one first", MaxHeld)
	ErrHeld      = fmt.Errorf("you already hold a bounty on that mark")
	ErrDone      = fmt.Errorf("your company has already settled that posting")
	ErrNoPosting = fmt.Errorf("the board has no such posting")
)

// Take adds the posting as a held bounty, or says why not. afterSeq is the
// newest chronicle deed's number now.
func (s *State) Take(p Posting, now int64, afterSeq int) error {
	if s.Holds(p.Target) {
		return ErrHeld
	}
	if _, done := s.Done[p.ID]; done {
		return ErrDone
	}
	if len(s.Held) >= MaxHeld {
		return ErrFull
	}
	s.Held = append(s.Held, Held{PostID: p.ID, Target: p.Target, Reward: p.Reward, TakenAt: now, Due: now + TermSeconds, AfterSeq: afterSeq})
	return nil
}

// Settle removes held bounty i and remembers its posting so it cannot be
// taken again from the same list.
func (s *State) Settle(i int, now int64) Held {
	h := s.Held[i]
	s.Held = append(s.Held[:i:i], s.Held[i+1:]...)
	if s.Done == nil {
		s.Done = map[string]int64{}
	}
	s.Done[h.PostID] = now + DoneKeepWindow*TermSeconds
	return h
}

// hunting answers whether a company holds a live bounty on a target. The
// bounties module sets it; nil (no module) is never hunting.
var hunting func(userID int, ref, zone string) bool

// SetHunting installs the held-bounty check (modules/bounties).
func SetHunting(fn func(userID int, ref, zone string) bool) { hunting = fn }

// Hunting reports whether the company holds a live bounty on the target
// with this chronicle reference in this zone. The encounters module writes a
// group deed only then, so ordinary fights do not crowd the chronicle out of
// the boss deeds, rites and choices towns and relics read.
func Hunting(userID int, ref, zone string) bool {
	return hunting != nil && hunting(userID, ref, zone)
}
