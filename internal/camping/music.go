package camping

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Camp music (spec: docs/plans/2026-10-07-camp-music.md). A member learns
// the Music skill in one family (strings, winds, drums or voice) and plays
// it in a short song when a camp rest begins. The company's families add
// up to the rest's buffs; the same families can play a timed gig at an inn
// for coin. Everything here is pure: the camping module supplies who plays
// and what the company carries, and keeps the durable state.

// Family is one kind of music.
type Family string

const (
	FamilyStrings Family = "strings"
	FamilyWinds   Family = "winds"
	FamilyDrums   Family = "drums"
	FamilyVoice   Family = "voice"
)

// FamilyInfo describes a family for the commands and help.
type FamilyInfo struct {
	Family Family
	Name   string
	// Needs is what it takes to play it.
	Needs string
}

// Families lists every family in display order.
var Families = []FamilyInfo{
	{FamilyStrings, "Strings", "a lyre, lute or fiddle"},
	{FamilyWinds, "Winds", "a flute, pipes or horn"},
	{FamilyDrums, "Drums", "a drum"},
	{FamilyVoice, "Voice", "nothing: the voice is the instrument"},
}

// FamilyName is a family's display name.
func FamilyName(f Family) string {
	for _, info := range Families {
		if info.Family == f {
			return info.Name
		}
	}
	return string(f)
}

// ValidFamily reports whether f names a family.
func ValidFamily(f Family) bool {
	for _, info := range Families {
		if info.Family == f {
			return true
		}
	}
	return false
}

// ParseFamily reads a command word ("strings", "lute", "sing").
func ParseFamily(word string) (Family, bool) {
	switch strings.ToLower(strings.TrimSpace(word)) {
	case "strings", "string", "lute", "lyre", "fiddle", "harp":
		return FamilyStrings, true
	case "winds", "wind", "flute", "pipes", "pipe", "horn":
		return FamilyWinds, true
	case "drums", "drum":
		return FamilyDrums, true
	case "voice", "sing", "singing", "song":
		return FamilyVoice, true
	}
	return "", false
}

// Music skill levels. Level 1 is taught for gold; 2, 3 and 4 come from
// practice, one song at a time.
const (
	MusicMaxLevel = 4
	// MusicTeachPrice is what a teacher charges for level 1 of a family.
	MusicTeachPrice = 30
)

// PracticeSongs is the songs played that reach each level: index 0 is
// level 1 (taught), then 10, 25 and 50 songs.
var PracticeSongs = [MusicMaxLevel]int{0, 10, 25, 50}

// MusicSkill is one member's Music: a family, and the songs played in it.
// The level is worked out from the songs, so it can never be out of step.
type MusicSkill struct {
	Family Family `yaml:"family"`
	Songs  int    `yaml:"songs,omitempty"`
}

// Valid reports whether the skill names a family and a sane count.
func (s MusicSkill) Valid() bool { return ValidFamily(s.Family) && s.Songs >= 0 }

// Level is 1 to MusicMaxLevel.
func (s MusicSkill) Level() int {
	level := 1
	for i, need := range PracticeSongs {
		if s.Songs >= need {
			level = i + 1
		}
	}
	return level
}

// SongsToNext is how many more songs reach the next level; ok is false at
// the top.
func (s MusicSkill) SongsToNext() (int, bool) {
	level := s.Level()
	if level >= MusicMaxLevel {
		return 0, false
	}
	return PracticeSongs[level] - s.Songs, true
}

// Practised is the skill after n more songs.
func (s MusicSkill) Practised(n int) MusicSkill {
	s.Songs += max(n, 0)
	return s
}

// Label is the skill in a few words: "Strings 2 (7 songs to level 3)".
func (s MusicSkill) Label() string {
	text := fmt.Sprintf("%s %d", FamilyName(s.Family), s.Level())
	if left, ok := s.SongsToNext(); ok {
		text += fmt.Sprintf(" (%d %s to level %d)", left, plural(left, "song"), s.Level()+1)
	} else {
		text += " (mastered)"
	}
	return text
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// Instrument tiers.
const (
	InstrumentCrude      = 1
	InstrumentCommon     = 2
	InstrumentFine       = 3
	InstrumentMasterwork = 4
)

// InstrumentTierName is a tier's name.
func InstrumentTierName(tier int) string {
	switch tier {
	case InstrumentCrude:
		return "crude"
	case InstrumentCommon:
		return "common"
	case InstrumentFine:
		return "fine"
	case InstrumentMasterwork:
		return "masterwork"
	}
	return ""
}

// Quirk is a masterwork instrument's one small oddity.
type Quirk string

const (
	// QuirkRainproof: its strings or reeds don't suffer in rain or snow.
	QuirkRainproof Quirk = "rainproof"
	// QuirkHushed: it adds nothing to the chance that raiders and thieves
	// come.
	QuirkHushed Quirk = "hushed"
	// QuirkMuffled: it carries less far, so it adds half the usual chance.
	QuirkMuffled Quirk = "muffled"
)

// QuirkText says what a quirk does.
func QuirkText(q Quirk) string {
	switch q {
	case QuirkRainproof:
		return "rain and snow don't dull it"
	case QuirkHushed:
		return "it draws no raiders or thieves"
	case QuirkMuffled:
		return "it carries less far, drawing raiders and thieves half as much"
	}
	return ""
}

// Instrument is one instrument item.
type Instrument struct {
	ItemID int
	Family Family
	Tier   int
	Name   string
	Quirk  Quirk
}

// Instruments lists every instrument, by family then tier. The items live
// in _datafiles (their `instrument` and `instrumenttier` spec fields must
// agree with this table; a module test holds that).
var Instruments = []Instrument{
	{ItemID: 3200, Family: FamilyStrings, Tier: InstrumentCrude, Name: "gut-strung lyre"},
	{ItemID: 3201, Family: FamilyStrings, Tier: InstrumentCommon, Name: "travelling lute"},
	{ItemID: 3202, Family: FamilyStrings, Tier: InstrumentFine, Name: "inlaid fiddle"},
	{ItemID: 3210, Family: FamilyStrings, Tier: InstrumentMasterwork, Name: "Wren's rainfiddle", Quirk: QuirkRainproof},
	{ItemID: 3203, Family: FamilyWinds, Tier: InstrumentCrude, Name: "bone flute"},
	{ItemID: 3204, Family: FamilyWinds, Tier: InstrumentCommon, Name: "reed pipes"},
	{ItemID: 3205, Family: FamilyWinds, Tier: InstrumentFine, Name: "silver-keyed flute"},
	{ItemID: 3211, Family: FamilyWinds, Tier: InstrumentMasterwork, Name: "hollow king's horn", Quirk: QuirkHushed},
	{ItemID: 3206, Family: FamilyDrums, Tier: InstrumentCrude, Name: "hide drum"},
	{ItemID: 3207, Family: FamilyDrums, Tier: InstrumentCommon, Name: "frame drum"},
	{ItemID: 3208, Family: FamilyDrums, Tier: InstrumentFine, Name: "tuned kettle drum"},
	{ItemID: 3212, Family: FamilyDrums, Tier: InstrumentMasterwork, Name: "ent-heart drum", Quirk: QuirkMuffled},
}

// InstrumentOf is the instrument an item ID names.
func InstrumentOf(itemID int) (Instrument, bool) {
	for _, in := range Instruments {
		if in.ItemID == itemID {
			return in, true
		}
	}
	return Instrument{}, false
}

// InstrumentsOf lists a family's instruments, crude first.
func InstrumentsOf(f Family) []Instrument {
	var out []Instrument
	for _, in := range Instruments {
		if in.Family == f {
			out = append(out, in)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Tier < out[j].Tier })
	return out
}

// BestInstrument picks the best instrument of a family among the item IDs
// the company carries (carried says whether it carries one).
func BestInstrument(f Family, carried func(itemID int) bool) (Instrument, bool) {
	list := InstrumentsOf(f)
	for i := len(list) - 1; i >= 0; i-- {
		if carried(list[i].ItemID) {
			return list[i], true
		}
	}
	return Instrument{}, false
}

// Player is a member who can play in a song: their skill, and for the
// three instrument families the instrument the company has for them.
type Player struct {
	Key    string
	Name   string
	Family Family
	Level  int
	// Tier and Quirk are the instrument's; both zero for the voice.
	Tier  int
	Quirk Quirk
}

// Strength is the player's level plus their instrument's tier (voice: the
// level plus one), 2 to 8.
func (p Player) Strength() int {
	if p.Family == FamilyVoice {
		return p.Level + 1
	}
	return p.Level + p.Tier
}

// SongPlayer is one member who played, kept on the rest for practice.
type SongPlayer struct {
	Key    string `yaml:"key"`
	Name   string `yaml:"name,omitempty"`
	Family Family `yaml:"family"`
}

// FamilyPlay is one family's part in a song: its best player and their
// strength after the weather.
type FamilyPlay struct {
	Family   Family `yaml:"family"`
	Strength int    `yaml:"strength"`
	Player   string `yaml:"player,omitempty"`
	// Instrument is the item name played (empty for the voice).
	Instrument string `yaml:"instrument,omitempty"`
	Quirk      Quirk  `yaml:"quirk,omitempty"`
}

// Song is what a camp rest's song fixed when the rest began: one play per
// family, and everyone who played (all of them practise).
type Song struct {
	Plays   []FamilyPlay `yaml:"plays,omitempty"`
	Players []SongPlayer `yaml:"players,omitempty"`
}

// Empty reports whether nobody played.
func (s Song) Empty() bool { return len(s.Plays) == 0 }

// Wet reports whether a weather condition's name is rain or snow, which
// dulls strings and winds.
func Wet(weatherName string) bool {
	name := strings.ToLower(weatherName)
	for _, w := range []string{"rain", "storm", "snow", "sleet", "blizzard", "hail", "drizzle"} {
		if strings.Contains(name, w) {
			return true
		}
	}
	return false
}

// PlanSong works out a song from the members who can play. wet halves the
// strings and winds strength (rounding up) unless a tent is pitched, which
// the caller leaves out of wet. A masterwork with the rainproof quirk shrugs
// the weather off.
func PlanSong(players []Player, wet bool) Song {
	var song Song
	best := map[Family]Player{}
	order := make([]Player, len(players))
	copy(order, players)
	sort.SliceStable(order, func(i, j int) bool { return order[i].Key < order[j].Key })
	for _, p := range order {
		if !ValidFamily(p.Family) || p.Level < 1 {
			continue
		}
		song.Players = append(song.Players, SongPlayer{Key: p.Key, Name: p.Name, Family: p.Family})
		if cur, ok := best[p.Family]; !ok || p.Strength() > cur.Strength() {
			best[p.Family] = p
		}
	}
	for _, info := range Families {
		p, ok := best[info.Family]
		if !ok {
			continue
		}
		strength := p.Strength()
		if wet && (info.Family == FamilyStrings || info.Family == FamilyWinds) && p.Quirk != QuirkRainproof {
			strength = (strength + 1) / 2
		}
		play := FamilyPlay{Family: info.Family, Strength: strength, Player: p.Name, Quirk: p.Quirk}
		if in, ok := tierInstrument(p); ok {
			play.Instrument = in
		}
		song.Plays = append(song.Plays, play)
	}
	return song
}

// tierInstrument names the instrument a player plays, from its family and
// tier.
func tierInstrument(p Player) (string, bool) {
	if p.Family == FamilyVoice {
		return "", false
	}
	for _, in := range InstrumentsOf(p.Family) {
		if in.Tier == p.Tier {
			return in.Name, true
		}
	}
	return "", false
}

// Strength is a family's strength in the song (0 when it is not played).
func (s Song) Strength(f Family) int {
	for _, p := range s.Plays {
		if p.Family == f {
			return p.Strength
		}
	}
	return 0
}

// StrongFamilies is how many families play at strength 4 or more.
func (s Song) StrongFamilies() int {
	n := 0
	for _, p := range s.Plays {
		if p.Strength >= 4 {
			n++
		}
	}
	return n
}

// Ensemble is three or more families each at strength 4+: the sleepers'
// Rested becomes Well Rested.
func (s Song) Ensemble() bool { return s.StrongFamilies() >= 3 }

// Full is all four families at strength 4+: every family's effect is a
// quarter stronger.
func (s Song) Full() bool { return s.StrongFamilies() >= len(Families) }

func (s Song) boost(v int) int {
	if s.Full() {
		return v * 5 / 4
	}
	return v
}

// RestedPct is the strings' effect: percent longer Rested and Well Rested
// (5% a point, at most 40%, a quarter more for a full ensemble).
func (s Song) RestedPct() int { return s.boost(min(40, 5*s.Strength(FamilyStrings))) }

// FatigueBonus is the winds' effect: Fatigue restored on top of the rest's
// (1 a point, at most 8).
func (s Song) FatigueBonus() int { return s.boost(min(8, s.Strength(FamilyWinds))) }

// DrumSpeed is the drums' effect: speed in the first battle after the rest
// (1 for every 2 points, at most 4).
func (s Song) DrumSpeed() int { return s.boost(min(4, s.Strength(FamilyDrums)/2)) }

// AilmentPct is the voice's effect: percent faster ailments fade (5% a
// point, at most 40%).
func (s Song) AilmentPct() int { return s.boost(min(40, 5*s.Strength(FamilyVoice))) }

// Raid and thief chance added per playing family, in percent: music
// carries. A hushed instrument adds none; a muffled one half.
const (
	raidPerFamily  = 10
	raidDrums      = 20
	maxSongRaidPct = 100 + 3*raidPerFamily + raidDrums
)

// RaidPct is the multiplier on the chance that raiders and thieves come,
// in percent (100 is no change).
func (s Song) RaidPct() int {
	pct := 100
	for _, p := range s.Plays {
		add := raidPerFamily
		if p.Family == FamilyDrums {
			add = raidDrums
		}
		switch p.Quirk {
		case QuirkHushed:
			add = 0
		case QuirkMuffled:
			add /= 2
		}
		pct += add
	}
	return min(pct, maxSongRaidPct)
}

// CostLine says what the song costs in raiders and thieves ("" when it
// adds nothing), so a player weighs the buffs against the risk.
func (s Song) CostLine() string {
	if s.Empty() || s.RaidPct() <= 100 {
		return ""
	}
	return fmt.Sprintf("The song carries: raiders and thieves are %d%% more likely to come.", s.RaidPct()-100)
}

// Effects is one line per family played, naming what its strength gives.
func (s Song) Effects() []string {
	var out []string
	for _, p := range s.Plays {
		var does string
		switch p.Family {
		case FamilyStrings:
			does = fmt.Sprintf("Rested and Well Rested last %d%% longer", s.RestedPct())
		case FamilyWinds:
			does = fmt.Sprintf("the rest restores %d more Fatigue", s.FatigueBonus())
		case FamilyDrums:
			does = fmt.Sprintf("+%d speed in the first battle after the rest", s.DrumSpeed())
		case FamilyVoice:
			does = fmt.Sprintf("ailments fade %d%% faster", s.AilmentPct())
		}
		out = append(out, fmt.Sprintf("%s (strength %d): %s", FamilyName(p.Family), p.Strength, does))
	}
	if s.Ensemble() {
		line := "Ensemble: those who sleep wake Well Rested"
		if s.Full() {
			line += "; all four families together make every effect a quarter stronger"
		}
		out = append(out, line)
	}
	return out
}

// Covered says how many of the four families play: "3 of 4".
func (s Song) Covered() string { return fmt.Sprintf("%d of %d", len(s.Plays), len(Families)) }

// DrumBuffIDs are the Drumbeat buffs, +1 to +5 speed (the fifth is a full
// ensemble's boosted best), by speed.
var DrumBuffIDs = []int{9401, 9402, 9403, 9404, 9405}

// DrumBuffFor is the buff giving speed (1 to 5), or 0 for none.
func DrumBuffFor(speed int) int {
	if speed < 1 {
		return 0
	}
	return DrumBuffIDs[min(speed, len(DrumBuffIDs))-1]
}

// IsDrumBuff reports whether buffID is a Drumbeat buff.
func IsDrumBuff(buffID int) bool {
	for _, id := range DrumBuffIDs {
		if id == buffID {
			return true
		}
	}
	return false
}

// FadedBattles is how many battles an ailment with left battles to run has
// after a song fades it by pct percent: at least one battle shorter for any
// pct above 0, never below 0.
func FadedBattles(left, pct int) int {
	if left <= 0 || pct <= 0 {
		return max(left, 0)
	}
	cut := (left*pct + 99) / 100
	return max(0, left-max(cut, 1))
}

// --- inn gigs ---

// Gig rules: an inn posts one gig a game evening, between GigStartHour
// and GigEndHour on the world clock, and a company may be paid for one
// every GigCooldown of real time.
const (
	GigStartHour = 19
	GigEndHour   = 21
	GigDuration  = 60 * time.Second
	GigCooldown  = 3 * time.Hour
	// GigMinFamilies is how many families a gig needs.
	GigMinFamilies = 2
	// DefaultGigBand is the zone band a settlement with none pays by.
	DefaultGigBand = 5
)

// InGigWindow reports whether an hour of the world clock (0-23) is inside
// the gig window.
func InGigWindow(hour int) bool { return hour >= GigStartHour && hour < GigEndHour }

// GigWindowText is the window as the notice board gives it.
func GigWindowText() string {
	return fmt.Sprintf("%02d:00 to %02d:00", GigStartHour, GigEndHour)
}

// GigPay is the gold a gig earns: the zone band's top level plus two, times
// the families' summed strength, halved; a three-strong ensemble earns a
// quarter more and a full one half again. A weak pair of players earns
// pennies; a full ensemble in a high zone earns about an hour's hunting.
func GigPay(bandHigh int, song Song) int {
	if bandHigh < 1 {
		bandHigh = DefaultGigBand
	}
	sum := 0
	for _, p := range song.Plays {
		sum += p.Strength
	}
	pay := (bandHigh + 2) * sum / 2
	switch {
	case song.Full():
		pay = pay * 3 / 2
	case song.Ensemble():
		pay = pay * 5 / 4
	}
	return max(pay, 1)
}

// CrowdLine is what the crowd makes of a gig of this song.
func CrowdLine(song Song) string {
	sum := 0
	for _, p := range song.Plays {
		sum += p.Strength
	}
	switch {
	case song.Full():
		return "The room goes wild: stamping, singing, coins on the boards."
	case song.Ensemble():
		return "The crowd is won over and calls for another."
	case sum >= 10:
		return "The crowd listens, tips a little, and goes back to its ale."
	}
	return "The crowd is polite, and the hat comes back light."
}

// Gig is a durable performance at an inn: a company that has started one
// stays until it ends. Pay is fixed when it starts and credited once.
type Gig struct {
	RoomID       int       `yaml:"room_id"`
	StartedAtUTC time.Time `yaml:"started_at_utc"`
	Pay          int       `yaml:"pay"`
	Song         Song      `yaml:"song"`
	Day          uint64    `yaml:"day,omitempty"`
	Done         bool      `yaml:"done,omitempty"`
	Settled      bool      `yaml:"settled,omitempty"`
}

// Valid reports whether the gig is well formed.
func (g Gig) Valid() bool { return g.RoomID > 0 && !g.StartedAtUTC.IsZero() && g.Pay >= 0 }

// Performing reports whether the gig is still running.
func (g Gig) Performing() bool { return !g.Done }

// Due reports whether the gig's time is up.
func (g Gig) Due(now time.Time) bool {
	return !g.Done && !now.UTC().Before(g.StartedAtUTC.Add(GigDuration))
}

// Remaining is the time left on a running gig.
func (g Gig) Remaining(now time.Time) time.Duration {
	if g.Done {
		return 0
	}
	left := g.StartedAtUTC.Add(GigDuration).Sub(now.UTC())
	return min(max(left, 0), GigDuration)
}

// GigLog is a leader's gig history: the limits that keep gigs from being
// farmed, and the gig in progress.
type GigLog struct {
	// LastPaidAtUTC is when a gig last paid: one every GigCooldown.
	LastPaidAtUTC time.Time `yaml:"last_paid_at_utc,omitempty"`
	// Evenings is, by inn room, the game day of the last paid gig there:
	// one paid gig per inn per game evening.
	Evenings map[int]uint64 `yaml:"evenings,omitempty"`
	Current  *Gig           `yaml:"current,omitempty"`
}

// GigRefusal says why a company cannot start a paid gig at an inn now, or
// "" when it can. hour is the world clock's hour, day its day number.
func (l GigLog) GigRefusal(roomID, hour int, day uint64, now time.Time, song Song) string {
	switch {
	case l.Current != nil && !l.Current.Settled:
		return "Your company is already playing."
	case !InGigWindow(hour):
		return fmt.Sprintf("The inn takes gigs between %s. It is not that hour now.", GigWindowText())
	case len(song.Plays) < GigMinFamilies:
		return fmt.Sprintf("A gig needs at least %d families playing; your company can field %d. See music.", GigMinFamilies, len(song.Plays))
	}
	if d, ok := l.Evenings[roomID]; ok && d == day {
		return "Your company has already played here this evening."
	}
	if !l.LastPaidAtUTC.IsZero() {
		if wait := l.LastPaidAtUTC.Add(GigCooldown).Sub(now.UTC()); wait > 0 {
			return fmt.Sprintf("Your company played for pay not long ago. Inns will have you again in about %s.", roundWait(wait))
		}
	}
	return ""
}

// roundWait is a wait as "2 hours" or "40 minutes", rounded up.
func roundWait(d time.Duration) string {
	minutes := int((d + time.Minute - 1) / time.Minute)
	if minutes >= 90 {
		hours := (minutes + 59) / 60
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d %s", max(minutes, 1), plural(max(minutes, 1), "minute"))
}

// Started is the log with a gig begun: the evening and the cooldown are
// spent when the gig is paid, not now, so a gig that never finishes costs
// nothing.
func (l GigLog) Started(g Gig) GigLog {
	l.Current = &g
	return l
}

// Paid is the log once the current gig has been paid: it records the
// evening and the time, and keeps the settled gig until the next starts.
func (l GigLog) Paid(now time.Time) GigLog {
	if l.Current == nil {
		return l
	}
	g := *l.Current
	g.Settled = true
	l.Current = &g
	l.LastPaidAtUTC = now.UTC()
	evenings := make(map[int]uint64, len(l.Evenings)+1)
	for room, day := range l.Evenings {
		if day == g.Day {
			evenings[room] = day
		}
	}
	evenings[g.RoomID] = g.Day
	l.Evenings = evenings
	return l
}
