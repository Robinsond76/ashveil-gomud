// Package encounters is the pure policy and content layer of Phase 37's
// random room encounters: the zone's tables of enemy compositions, how a
// composition is chosen and leveled from the zone's band, the grace that
// follows a battle, and the rules that keep content honest. It knows no
// engine type; the encounters module binds it to rooms, mobs and loot.
package encounters

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Rand returns 0..n-1 (util.Rand's shape), so every roll is deterministic
// in tests.
type Rand func(n int) int

// Contract numbers (owner, 2026-10-05; 35d for the boss shape).
const (
	DefaultEntryChance = 15 // percent per eligible entry
	MinGroup           = 2  // ordinary groups are 2-3, sometimes 4
	MaxGroup           = 4
	MinEscorts         = 2 // a boss brings two or three escorts
	MaxEscorts         = 3
	BossLevelBonus     = 3   // the boss is this many levels over its escorts
	BossHPBonus        = 1.0 // 2x the HP of an ordinary foe of its level
	GraceEntries       = 2   // eligible entries skipped after a battle
	GraceSeconds       = 30  // and at least this long
	HealerShareMax     = 20  // percent of a table's weight a healer group may hold
	RoomGroupLimit     = 4   // unresolved random groups one room holds
	AbandonSeconds     = 120 // an ownerless group disappears after this long
	// OrdinaryHPPercent is the share of its level's HP an ordinary group's
	// foe spawns with for a company at the band's low end or above (owner,
	// 2026-10-07: a company in a zone for its level should win quickly and
	// cheaply). Bosses and their escorts keep full HP: a lair is the
	// set-piece fight. A company under the band meets harder foes (see
	// HPPercent).
	OrdinaryHPPercent = 40
	// UnderBandGap is how many levels under the band's low end a company
	// is before ordinary foes have their full HP again.
	UnderBandGap = 5
	// BossRespawnSeconds is how long a lair stays quiet for a company after
	// it beats the boss (real time, saved: it never moves the world's clock).
	BossRespawnSeconds = 30 * 60
)

// Member is a template and how many of it a composition spawns.
type Member struct {
	MobID int `yaml:"mobid"`
	Count int `yaml:"count"`
}

// Composition is one weighted outcome of a table. For a boss composition
// the first member is the boss (count 1) and the rest are its escorts.
type Composition struct {
	ID      string   `yaml:"id"`
	Weight  int      `yaml:"weight"`
	Text    string   `yaml:"text,omitempty"` // the line that opens the encounter
	Boss    bool     `yaml:"boss,omitempty"`
	Members []Member `yaml:"members"`
}

// Size is how many foes the composition spawns.
func (c Composition) Size() int {
	n := 0
	for _, m := range c.Members {
		n += m.Count
	}
	return n
}

// Band is a zone's recommended level band.
type Band struct {
	Low  int `yaml:"low"`
	High int `yaml:"high"`
}

// Valid reports whether the band can level a group.
func (b Band) Valid() bool { return b.Low >= 1 && b.High >= b.Low }

// ZoneConfig is a zone's encounter content: the default chance, the
// level band, and its named tables. Tables alone enable no room.
type ZoneConfig struct {
	EntryChance *int                     `yaml:"entrychance,omitempty"` // nil inherits the default; 0 means never
	Band        Band                     `yaml:"band,omitempty"`
	Tables      map[string][]Composition `yaml:"tables,omitempty"`
}

// Chance is the zone's entry chance in percent.
func (z ZoneConfig) Chance() int {
	if z.EntryChance == nil {
		return DefaultEntryChance
	}
	return clampPct(*z.EntryChance)
}

// RoomSetting is a room's explicit encounter setting. A room is eligible
// only when Enabled; Chance nil inherits the zone's, and 0 overrides it.
type RoomSetting struct {
	Enabled bool   `yaml:"enabled"`
	Table   string `yaml:"table,omitempty"`
	Chance  *int   `yaml:"chance,omitempty"`
}

// ChanceIn is the room's chance in percent given its zone's.
func (r RoomSetting) ChanceIn(z ZoneConfig) int {
	if r.Chance == nil {
		return z.Chance()
	}
	return clampPct(*r.Chance)
}

func clampPct(n int) int { return min(max(n, 0), 100) }

// Template is what validation needs to know of a mob template.
type Template struct {
	Solitary bool
	Healer   bool
}

// Lookup finds a template; false means it does not exist.
type Lookup func(mobID int) (Template, bool)

// Diagnostic is one reason content was disabled.
type Diagnostic struct {
	Table, Composition string
	Reason             string
}

func (d Diagnostic) String() string {
	if d.Composition != "" {
		return fmt.Sprintf("table %q composition %q: %s", d.Table, d.Composition, d.Reason)
	}
	return fmt.Sprintf("table %q: %s", d.Table, d.Reason)
}

// Validate returns the zone's usable tables and a diagnostic for each
// composition or table it disabled. Unknown content disables only the
// affected composition rather than spawning an incomplete party:
//   - weights and counts are positive, templates exist;
//   - ordinary groups are 2-4 foes, and a solitary template (it stands
//     alone) never joins a group, except as a boss composition's boss;
//   - a boss composition is one boss (count 1, first) and 2-3 escorts, none
//     of them a healer;
//   - a healer is never in a four-foe group, and healer groups hold at most
//     HealerShareMax percent of the table's weight (the owner wants them
//     uncommon).
func (z ZoneConfig) Validate(lookup Lookup) (ZoneConfig, []Diagnostic) {
	out := ZoneConfig{EntryChance: z.EntryChance, Band: z.Band, Tables: map[string][]Composition{}}
	var diags []Diagnostic
	if z.EntryChance != nil && (*z.EntryChance < 0 || *z.EntryChance > 100) {
		diags = append(diags, Diagnostic{Reason: fmt.Sprintf("entry chance %d is outside 0-100", *z.EntryChance)})
		out.EntryChance = nil
	}
	if len(z.Tables) > 0 && !z.Band.Valid() {
		diags = append(diags, Diagnostic{Reason: "the zone has encounter tables but no valid level band (low >= 1, high >= low)"})
		return out, diags
	}
	names := make([]string, 0, len(z.Tables))
	for name := range z.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var kept []Composition
		seen := map[string]bool{}
		for _, c := range z.Tables[name] {
			if reason := c.problem(lookup); reason != "" {
				diags = append(diags, Diagnostic{Table: name, Composition: c.ID, Reason: reason})
				continue
			}
			if seen[c.ID] {
				diags = append(diags, Diagnostic{Table: name, Composition: c.ID, Reason: "duplicate composition id"})
				continue
			}
			seen[c.ID] = true
			kept = append(kept, c)
		}
		kept, diags = capHealers(name, kept, lookup, diags)
		if len(kept) > 0 {
			out.Tables[name] = kept
		}
	}
	return out, diags
}

func (c Composition) healers(lookup Lookup) bool {
	for _, m := range c.Members {
		if t, ok := lookup(m.MobID); ok && t.Healer {
			return true
		}
	}
	return false
}

// problem is why the composition cannot be used, or "".
func (c Composition) problem(lookup Lookup) string {
	if strings.TrimSpace(c.ID) == "" {
		return "no id"
	}
	if c.Weight < 1 {
		return "weight must be positive"
	}
	if len(c.Members) == 0 {
		return "no members"
	}
	size := 0
	for i, m := range c.Members {
		if m.Count < 1 {
			return fmt.Sprintf("member %d has a non-positive count", m.MobID)
		}
		t, ok := lookup(m.MobID)
		if !ok {
			return fmt.Sprintf("unknown mob template %d", m.MobID)
		}
		if t.Solitary && !(c.Boss && i == 0) { // a boss may be a solitary template: it leads a group here
			return fmt.Sprintf("mob template %d is solitary and cannot join a group", m.MobID)
		}
		size += m.Count
	}
	if c.Boss {
		if c.Members[0].Count != 1 {
			return "a boss composition's first member is the single boss"
		}
		if escorts := size - 1; escorts < MinEscorts || escorts > MaxEscorts {
			return fmt.Sprintf("a boss brings %d-%d escorts, not %d", MinEscorts, MaxEscorts, escorts)
		}
		for _, m := range c.Members {
			if t, _ := lookup(m.MobID); t.Healer {
				return "a boss group has no healer"
			}
		}
		return ""
	}
	if size < MinGroup || size > MaxGroup {
		return fmt.Sprintf("an ordinary group is %d-%d foes, not %d", MinGroup, MaxGroup, size)
	}
	if size == MaxGroup && c.healers(lookup) {
		return "a four-foe group has no healer"
	}
	return ""
}

// capHealers drops healer compositions, heaviest last, until they hold at
// most HealerShareMax percent of the table's weight.
func capHealers(table string, comps []Composition, lookup Lookup, diags []Diagnostic) ([]Composition, []Diagnostic) {
	for {
		total, healer := 0, 0
		for _, c := range comps {
			total += c.Weight
			if c.healers(lookup) {
				healer += c.Weight
			}
		}
		if healer == 0 || healer*100 <= HealerShareMax*total {
			return comps, diags
		}
		drop := -1
		for i, c := range comps {
			if c.healers(lookup) && (drop < 0 || c.Weight >= comps[drop].Weight) {
				drop = i
			}
		}
		diags = append(diags, Diagnostic{Table: table, Composition: comps[drop].ID,
			Reason: fmt.Sprintf("healer groups may hold at most %d%% of a table's weight", HealerShareMax)})
		comps = append(append([]Composition(nil), comps[:drop]...), comps[drop+1:]...)
	}
}

// Pick chooses a composition by weight. It returns false for an empty table.
func Pick(table []Composition, rng Rand) (Composition, bool) {
	total := 0
	for _, c := range table {
		total += max(c.Weight, 0)
	}
	if total < 1 {
		return Composition{}, false
	}
	n := rng(total)
	for _, c := range table {
		if n < c.Weight {
			return c, true
		}
		n -= c.Weight
	}
	return Composition{}, false
}

// Roll is the entry-chance roll: true when an encounter springs.
func Roll(chance int, rng Rand) bool {
	if chance <= 0 {
		return false
	}
	return rng(100) < chance
}

// Foe is one spawn the composition calls for.
type Foe struct {
	MobID int
	Level int
	Boss  bool // the one boss of a boss composition
	// Escort is true for a boss's escorts.
	Escort bool
	// HPPercent is the share of full HP the foe spawns with (Soften); zero
	// means full.
	HPPercent int
}

// Plan expands a composition into the foes to spawn and their levels from
// the zone's band, never from the player:
//   - an ordinary group of 2-3 takes levels from the band's low end to one
//     below its top (the band's low where the band is a single level);
//   - a four-foe group is all at the band's low (35d: four at the top was
//     a near-even fight);
//   - a boss is two levels over the band's low, its escorts at the low.
func Plan(c Composition, band Band, rng Rand) []Foe {
	var foes []Foe
	size := c.Size()
	hi := max(band.Low, band.High-1)
	for i, m := range c.Members {
		for n := 0; n < m.Count; n++ {
			f := Foe{MobID: m.MobID, Level: band.Low}
			switch {
			case c.Boss && i == 0 && n == 0:
				f.Level, f.Boss = band.Low+BossLevelBonus, true
			case c.Boss:
				f.Escort = true
			case size < MaxGroup:
				f.Level = band.Low + rng(hi-band.Low+1)
			}
			foes = append(foes, f)
		}
	}
	return foes
}

// HPPercent is the share of full HP an ordinary group's foe spawns with
// against a company of the given level in the band: OrdinaryHPPercent at or
// above the band's low end, rising evenly to full HP at UnderBandGap levels
// under it. Difficulty comes only from a zone above the company's level
// (owner, 2026-10-06), so the softness fades as the company falls short.
func HPPercent(level int, b Band) int {
	gap := b.Low - level
	if gap <= 0 {
		return OrdinaryHPPercent
	}
	return min(100, OrdinaryHPPercent+gap*(100-OrdinaryHPPercent)/UnderBandGap)
}

// Spread is the aim noise an ordinary foe with hpPercent of its HP gets:
// the percent of its re-aims that take a random member it can reach instead
// of its rule's pick. A fully softened foe (OrdinaryHPPercent) aims at random
// every time, and the noise fades with the softness to none at full HP.
// Without it every foe goes for the weakest member, so one member (the
// leader, or a wizard left in the front row) takes nearly every blow and the
// company rests after a handful of fights while the rest stand untouched.
func Spread(hpPercent int) int {
	if hpPercent <= 0 || hpPercent >= 100 {
		return 0
	}
	return min(100, (100-hpPercent)*100/(100-OrdinaryHPPercent))
}

// Soften sets the HP share of an ordinary group's foes against a company
// of the given level; a boss and its escorts keep full HP.
func Soften(foes []Foe, level int, b Band) []Foe {
	for i := range foes {
		if !foes[i].Boss && !foes[i].Escort {
			foes[i].HPPercent = HPPercent(level, b)
		}
	}
	return foes
}

// Rating words say how a zone's band compares with a level.
const (
	RatingEasy      = "easy"      // at or above the band's low end
	RatingFair      = "fair"      // one or two levels under it
	RatingRisky     = "risky"     // three or four under: expect losses
	RatingDangerous = "dangerous" // five or more under: prepare carefully
)

// Rating is how hard the band is for a company of the given level (the
// owner's rule: difficulty comes only from entering a zone above the
// company's level, measured in 37b). Balance row: five members at the
// band's low less 2 won about 96%, less 3 about 90%, less 5 about 46%.
func Rating(level int, b Band) string {
	if !b.Valid() {
		return ""
	}
	switch gap := b.Low - level; {
	case gap <= 0:
		return RatingEasy
	case gap <= 2:
		return RatingFair
	case gap <= 4:
		return RatingRisky
	}
	return RatingDangerous
}

// LairQuiet is how much longer the lair in roomID stays quiet for the
// user's company after it beat the boss there (zero when it is not quiet).
// The encounters module sets it; look and scout read it.
var LairQuiet func(userID, roomID int) time.Duration

// Available drops the boss compositions the cooling predicate holds back,
// leaving ordinary ones. It never changes the table's own slice.
func Available(table []Composition, cooling func(id string) bool) []Composition {
	out := make([]Composition, 0, len(table))
	for _, c := range table {
		if c.Boss && cooling(c.ID) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Grace is one leader's quiet after a battle: GraceEntries eligible entries
// skipped and at least GraceSeconds of real time, both. It is saved with
// the leader so crossing a zone, reconnecting or restarting cannot reset
// it. A leader seen for the first time starts with a fresh grace.
type Grace struct {
	Entries int       `yaml:"entries"`
	Until   time.Time `yaml:"until"`
}

// NewGrace is the grace a battle's end (or a fresh character) starts.
func NewGrace(now time.Time) Grace {
	return Grace{Entries: GraceEntries, Until: now.Add(GraceSeconds * time.Second)}
}

// Suppresses is called on an otherwise eligible entry. It consumes one
// allowance and reports whether the entry may not roll.
func (g *Grace) Suppresses(now time.Time) bool {
	if g.Entries > 0 {
		g.Entries--
		return true
	}
	return now.Before(g.Until)
}

// AttemptProvider is implemented by the encounters module (Phase 40a2): a
// company working in place (gathering) rolls the room's encounter chance
// once per attempt, with a bonus for noisy work.
type AttemptProvider interface {
	// Attempt rolls the room's chance plus bonusPct for the user's company
	// and springs the encounter on a hit. It reports whether one sprang.
	Attempt(userID, roomID, bonusPct int) bool
}

var (
	attemptMu       sync.RWMutex
	attemptProvider AttemptProvider
)

// SetAttemptProvider registers the module. nil clears it.
func SetAttemptProvider(p AttemptProvider) {
	attemptMu.Lock()
	defer attemptMu.Unlock()
	attemptProvider = p
}

// Attempt rolls an in-place encounter attempt; false without a provider.
func Attempt(userID, roomID, bonusPct int) bool {
	attemptMu.RLock()
	p := attemptProvider
	attemptMu.RUnlock()
	return p != nil && p.Attempt(userID, roomID, bonusPct)
}

// StartProvider is implemented by the encounters module (Phase 60): a story
// event's battle outcome sets a named group of foes on the company through
// the same spawn, cleanup and grace as a random encounter.
type StartProvider interface {
	// StartGroup spawns the foes in roomID against the leader and keeps
	// them until they are beaten or abandoned. It is an error when the
	// leader already has a group or the foes cannot be spawned.
	StartGroup(userID, roomID int, foes []Foe) error
}

var (
	startMu       sync.RWMutex
	startProvider StartProvider
)

// SetStartProvider registers the module. nil clears it.
func SetStartProvider(p StartProvider) {
	startMu.Lock()
	defer startMu.Unlock()
	startProvider = p
}

// ErrNoStartProvider is returned by StartGroup when no module is loaded.
var ErrNoStartProvider = errors.New("encounters: no module to start a group")

// StartGroup starts a scripted group; ErrNoStartProvider without a module.
func StartGroup(userID, roomID int, foes []Foe) error {
	startMu.RLock()
	p := startProvider
	startMu.RUnlock()
	if p == nil {
		return ErrNoStartProvider
	}
	return p.StartGroup(userID, roomID, foes)
}
