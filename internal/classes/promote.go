package classes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/creatures"
)

// Option is one promotion open to a character, or listed with the reason
// it is not open yet.
type Option struct {
	Class    Class
	Eligible bool
	// Waiting is an elite route a character is high enough for but whose
	// alignment gate it does not meet: it keeps its ranks and promotes as
	// soon as its alignment recovers.
	Waiting bool
	Reason  string
}

// Options lists the promotions a character can look at: its lineage's
// advanced classes with none, its class's elite continuation with an
// advanced class, nothing with an elite one or an unknown lineage.
func Options(lineageID, current string, level, alignment int) []Option {
	lineageID, current = normalize(lineageID), normalize(current)
	var classes []Class
	switch cur, _ := Get(current); {
	case current == "":
		classes = Advanced(lineageID)
	case cur.Tier == TierAdvanced:
		if e, ok := Elite(cur.ID); ok {
			classes = []Class{e}
		}
	}
	out := make([]Option, 0, len(classes))
	for _, c := range classes {
		out = append(out, option(c, level, alignment))
	}
	return out
}

func option(c Class, level, alignment int) Option {
	gate := Gate(c.Gate)
	need := AdvancedLevel
	if c.Tier == TierElite {
		need = EliteLevel
		if parent, ok := Get(c.Parent); ok {
			gate = parent.Gate
		}
	}
	o := Option{Class: c}
	switch {
	case c.Planned:
		o.Reason = "this route is not open yet"
	case level < need:
		o.Reason = fmt.Sprintf("needs level %d", need)
	case !gate.Allows(alignment):
		o.Reason = fmt.Sprintf("needs %s; yours is %+d", gate.Label(), alignment)
		o.Waiting = c.Tier == TierElite
	default:
		o.Eligible = true
	}
	return o
}

// Errors a promotion can fail with; each is player-facing.
var (
	ErrNoSuchClass  = errors.New("no such class")
	ErrNotAvailable = errors.New("that class isn't open to you")
	ErrFinalRoute   = errors.New("your route is final; there is nothing further to promote to")
)

// Check decides whether a character may promote to a class now. The error
// reads to the player.
func Check(lineageID, current, target string, level, alignment int) (Class, error) {
	target = normalize(target)
	c, ok := Get(target)
	if !ok {
		return Class{}, fmt.Errorf("%w: %q", ErrNoSuchClass, target)
	}
	if cur, ok := Get(current); ok && cur.Tier == TierElite {
		return Class{}, ErrFinalRoute
	}
	for _, o := range Options(lineageID, current, level, alignment) {
		if o.Class.ID != c.ID {
			continue
		}
		if !o.Eligible {
			return Class{}, fmt.Errorf("%s %s", c.Name, o.Reason)
		}
		return c, nil
	}
	// Phase 38c1: a base character at the elite level takes its advanced
	// class first and may take the elite one in the same visit.
	if normalize(current) == "" && c.Tier == TierElite && c.Lineage == normalize(lineageID) {
		if parent, ok := Get(c.Parent); ok {
			return Class{}, fmt.Errorf("%w: a %s continues from a %s; take the %s first, then the %s in the same visit", ErrNotAvailable, c.Name, parent.Name, parent.Name, c.Name)
		}
	}
	if normalize(current) != "" {
		return Class{}, fmt.Errorf("%w: a %s doesn't continue as a %s (routes are final)", ErrNotAvailable, nameOr(current), c.Name)
	}
	return Class{}, fmt.Errorf("%w: a %s can't become a %s", ErrNotAvailable, normalize(lineageID), c.Name)
}

func nameOr(id string) string {
	if c, ok := Get(id); ok {
		return c.Name
	}
	return id
}

// RanksGained are the ranks a character of a lineage and class earned
// between two levels (above from, up to and including to): its lineage's base
// ranks, then its route's. The level-up report names them (39b review), so a
// player sees what a rank just gave, not only the next one.
func RanksGained(lineageID, classID string, from, to int) []Rank {
	var out []Rank
	for _, r := range BaseRanks(lineageID) {
		if r.Level > from && r.Level <= to {
			out = append(out, r)
		}
	}
	if classID != "" {
		for _, r := range RanksReached(classID, to) {
			if r.Level > from {
				out = append(out, r)
			}
		}
	}
	return out
}

// RankLines are RanksGained as level-up report lines.
func RankLines(lineageID, classID string, from, to int) []string {
	var out []string
	for _, r := range RanksGained(lineageID, classID, from, to) {
		out = append(out, fmt.Sprintf("New rank: %s, %s.", r.Name, r.Text))
	}
	return out
}

// Milestone describes what a character at a level gains next on its way,
// for the level-up report: the next talent, promotion or rank, with
// everything that arrives at the same level.
func Milestone(current string, level int) string { return MilestoneFor("", current, level) }

// MilestoneFor is Milestone for a character of a lineage, whose base ranks
// (a Samurai's Focus and Zanshin) count among what comes next.
func MilestoneFor(lineageID, current string, level int) string {
	at := map[int][]string{}
	// Phase 38e: a creature has ranks, but no talents and no promotion.
	creature := creatures.Is(lineageID)
	if l, ok := NextTalentLevel(level); ok && !creature {
		at[l] = append(at[l], "a talent")
	}
	cur, has := Get(current)
	switch {
	case creature:
	case !has:
		at[AdvancedLevel] = append(at[AdvancedLevel], "your class promotion")
	case cur.Tier == TierAdvanced:
		if e, ok := Elite(cur.ID); ok && !e.Planned {
			at[EliteLevel] = append(at[EliteLevel], "your elite promotion")
		}
	}
	if has {
		if r, ok := NextRank(current, level); ok {
			at[r.Level] = append(at[r.Level], "a rank ("+r.Name+")")
		}
	}
	for _, r := range BaseRanks(lineageID) {
		if r.Level > level {
			at[r.Level] = append(at[r.Level], "a rank ("+r.Name+")")
			break
		}
	}
	next := 0
	for l := range at {
		if l > level && (next == 0 || l < next) {
			next = l
		}
	}
	if next == 0 {
		return ""
	}
	return fmt.Sprintf("Next: %s at level %d.", strings.Join(at[next], " and "), next)
}

// PromotionState is what a character can do about its next promotion, for
// the roster, the web panel and GMCP: "ready" when it may promote now,
// "waiting-gate" when it has the level but not the alignment (an elite
// route keeps its advanced ranks and waits), "" otherwise.
func PromotionState(lineageID, current string, level, alignment int) string {
	for _, o := range Options(lineageID, current, level, alignment) {
		if o.Eligible {
			return "ready"
		}
	}
	for _, o := range Options(lineageID, current, level, alignment) {
		if o.Waiting {
			return "waiting-gate"
		}
	}
	return ""
}

// RankLevel is the highest rank level a character of a class has reached
// at its level (0 before any): the "rank" a surface shows.
func RankLevel(classID string, level int) int {
	n := 0
	for _, r := range RanksReached(classID, level) {
		n = max(n, r.Level)
	}
	return n
}

// ReadinessNote is the level-up line about an elite promotion a character
// at the elite level can take or is waiting on (Phase 38c1); "" when it has
// no elite step open to it. who is the subject's selector for the command
// ("" for the player, "#2" for a companion); noun is "You" or the name.
func ReadinessNote(lineageID, current string, level, alignment int, who, name string) string {
	cur, ok := Get(current)
	if !ok || cur.Tier != TierAdvanced {
		return ""
	}
	for _, o := range Options(lineageID, current, level, alignment) {
		switch {
		case o.Eligible:
			cmd := "class promote"
			if who != "" {
				cmd += " " + who
			}
			cmd += " " + o.Class.ID
			where := "Visit a camp or town and type " + cmd + "."
			return fmt.Sprintf("Elite promotion ready: %s -> %s. %s", cur.Name, o.Class.Name, where)
		case o.Waiting:
			gate := o.Class.Gate
			if p, ok := Get(o.Class.Parent); ok {
				gate = p.Gate
			}
			own, keeps := "yours", "You keep your"
			if name != "" {
				own, keeps = "theirs", name+" keeps their"
			}
			return fmt.Sprintf("%s needs %s (%s: %+d). %s %s ranks and can promote once it rises.", o.Class.Name, gateShort(gate), own, alignment, keeps, cur.Name)
		}
	}
	return ""
}

// gateShort is a gate as the level-up line says it ("alignment +30").
func gateShort(g Gate) string {
	switch g {
	case GateGood:
		return fmt.Sprintf("alignment +%d", GateAlignment)
	case GateEvil:
		return fmt.Sprintf("alignment -%d", GateAlignment)
	}
	return "any alignment"
}

// LevelNotes are the promotion lines a level-up report adds (Phase 38c1):
// from the elite level on, the elite promotion the character can take or is
// waiting on. The ranks earned come from RankLines (39b). who and name are
// the subject's selector and name for a companion ("" for the player).
func LevelNotes(lineageID, current string, to, alignment int, who, name string) []string {
	var out []string
	if to >= EliteLevel {
		if note := ReadinessNote(lineageID, current, to, alignment, who, name); note != "" {
			out = append(out, note)
		}
	}
	return out
}

// Info is a character's class as a surface shows it (the web panels, GMCP):
// derived from its lineage, class, level and alignment, never saved.
type Info struct {
	ID, Name string
	Tier     string // "advanced", "elite" or ""
	Rank     int    // the highest rank level reached, 0 before any
	// Promotion is "ready", "waiting-gate" or "".
	Promotion string
}

// Describe is the Info of a character.
func Describe(lineageID, classID string, level, alignment int) Info {
	info := Info{Promotion: PromotionState(lineageID, classID, level, alignment)}
	if c, ok := Get(classID); ok {
		info.ID, info.Name, info.Tier = c.ID, c.Name, "advanced"
		if c.Tier == TierElite {
			info.Tier = "elite"
		}
		info.Rank = RankLevel(c.ID, level)
	}
	return info
}
