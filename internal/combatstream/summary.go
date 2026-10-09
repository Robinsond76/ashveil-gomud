package combatstream

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Enemy endings in a Summary.
const (
	EndingYielded  = "yielded"
	EndingSlain    = "slain"
	EndingBeaten   = "beaten" // a practice foe (the tutorial's straw soldiers)
	EndingFled     = "fled"
	EndingLeft     = "left"
	EndingStanding = "still standing"
)

// Amount is a named count: damage or kills by a member, uses of a guard.
type Amount struct {
	Who   Ref
	Value int
}

// Count is a count by name: effects applied.
type Count struct {
	Name  string
	Count int
}

// Hit is one blow or spell's damage.
type Hit struct {
	Source Ref
	Target Ref
	Damage int
	Crit   bool
}

// EnemyEnding is how one enemy of the fight ended it.
type EnemyEnding struct {
	Ref    Ref
	Ending string
}

// Summary is a finished fight, built only from its events (and the
// company's health read from the world at its end).
type Summary struct {
	FightID      uint64
	Outcome      string
	GroupName    string // the enemy group's name, "" if unnamed
	LeaderUserId int
	StartRound   uint64
	EndRound     uint64

	CompanyDamage int // weapon and spell damage the company dealt
	EnemyDamage   int // weapon and spell damage the enemies dealt
	Healing       int // healing the company received
	HeldBack      int // healing a wound limit kept back

	MostDamage []Amount // company members, most first
	HighestHit *Hit

	// CompanyDefenses and EnemyDefenses are the strikes each side's
	// active defenses stopped (Phase 30g2).
	CompanyDefenses DefenseCounts
	EnemyDefenses   DefenseCounts

	InterruptsDealt  []string // what the company interrupted
	InterruptsFailed int
	InterruptsTaken  int
	Guards           []Amount
	Effects          []Count
	Kills            []Amount // company members, most first

	// Phase 62: why the fight went as it did.
	Taken       []Amount // damage each company member took, most first
	NeverLanded []Swings // company members who swung and never landed a blow
	Moves       []Count  // class abilities the company used
	Sigil       string   // the sigil in force over the battle, set by the caller

	// Spoils is what the leader's company took (Phase 37), set by the
	// caller when the fight ends: item names and gold, as they fell.
	Spoils []string

	Enemies []EnemyEnding
	Company []MemberHealth
}

// Swings is what became of a member's strikes: how many were thrown, and
// why they did nothing.
type Swings struct {
	Who      Ref
	Thrown   int
	Missed   int // the roll fell short
	Turned   int // a block, parry or dodge stopped it
	Absorbed int // it landed and armor or a ward took it all
}

// DefenseCounts is how many strikes a side blocked, parried, and dodged.
type DefenseCounts struct {
	Blocked, Parried, Dodged int
}

// add counts one defended strike by its outcome (combat.Defense*).
func (d *DefenseCounts) add(defense string) {
	switch defense {
	case "blocked":
		d.Blocked++
	case "parried":
		d.Parried++
	case "dodged":
		d.Dodged++
	}
}

// words is the counts that aren't zero, "3 blocked, 1 dodged".
func (d DefenseCounts) words() string {
	var parts []string
	for _, c := range []struct {
		n    int
		word string
	}{{d.Blocked, "blocked"}, {d.Parried, "parried"}, {d.Dodged, "dodged"}} {
		if c.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c.n, c.word))
		}
	}
	return strings.Join(parts, ", ")
}

// tally is a fight's running totals, folded from its events as they
// arrive so that the events themselves need not be kept.
type tally struct {
	companyDamage, enemyDamage int
	healing, heldBack          int
	companyDefenses            DefenseCounts
	enemyDefenses              DefenseCounts
	damage                     map[string]int
	highest                    *Hit
	interruptsDealt            []string
	interruptsFailed           int
	interruptsTaken            int
	guards                     map[string]int
	effects                    map[string]int
	effectOrder                []string
	kills                      map[string]int
	refs                       map[string]Ref
	taken                      map[string]int
	swings                     map[string]*Swings
	swingOrder                 []string
	moves                      map[string]int
	moveOrder                  []string
}

func newTally() tally {
	return tally{damage: map[string]int{}, guards: map[string]int{}, effects: map[string]int{}, kills: map[string]int{}, refs: map[string]Ref{}, taken: map[string]int{}, swings: map[string]*Swings{}, moves: map[string]int{}}
}

func (t *tally) add(f *fight, e Event) {
	_, sourceCompany := f.company[e.Source.Key()]
	_, sourceEnemy := f.enemies[e.Source.Key()]
	_, targetCompany := f.company[e.Target.Key()]
	_, targetEnemy := f.enemies[e.Target.Key()]
	if !e.Source.Zero() {
		t.refs[e.Source.Key()] = e.Source
	}

	switch e.Kind {
	case Attack, SpellHit:
		for _, d := range e.Defenses {
			switch {
			case targetCompany:
				t.companyDefenses.add(d)
			case targetEnemy:
				t.enemyDefenses.add(d)
			}
		}
		if e.Kind == Attack && sourceCompany && targetEnemy {
			t.noteSwings(e)
		}
		if e.Damage <= 0 {
			return
		}
		if targetCompany {
			t.taken[e.Target.Key()] += e.Damage
			t.refs[e.Target.Key()] = e.Target
		}
		switch {
		case sourceCompany:
			t.companyDamage += e.Damage
			t.damage[e.Source.Key()] += e.Damage
		case sourceEnemy:
			t.enemyDamage += e.Damage
		}
		if t.highest == nil || e.Damage > t.highest.Damage {
			t.highest = &Hit{Source: e.Source, Target: e.Target, Damage: e.Damage, Crit: e.Crit}
		}
	case Heal:
		if targetCompany {
			t.healing += e.Amount
			t.heldBack += e.HeldBack
		}
	case StatusTick:
		// A status's damage has no attacker to credit; it counts to the
		// other side.
		if e.Damage <= 0 {
			return
		}
		switch {
		case targetEnemy:
			t.companyDamage += e.Damage
		case targetCompany:
			t.enemyDamage += e.Damage
			t.taken[e.Target.Key()] += e.Damage
			t.refs[e.Target.Key()] = e.Target
		}
	case Ability:
		if sourceCompany && e.Status != "" {
			if _, ok := t.moves[e.Status]; !ok {
				t.moveOrder = append(t.moveOrder, e.Status)
			}
			t.moves[e.Status]++
		}
	case StatusApplied:
		if e.Status == "" {
			return
		}
		if _, ok := t.effects[e.Status]; !ok {
			t.effectOrder = append(t.effectOrder, e.Status)
		}
		t.effects[e.Status]++
	case Interrupt:
		switch {
		case sourceCompany && e.Outcome == OutcomeSucceeded:
			t.interruptsDealt = append(t.interruptsDealt, e.Status)
		case sourceCompany && targetEnemy:
			t.interruptsFailed++
		case sourceEnemy && e.Outcome == OutcomeSucceeded:
			t.interruptsTaken++
		}
	case GuardUsed:
		if sourceCompany {
			t.guards[e.Source.Key()]++
		}
	case Death:
		if targetEnemy && e.Outcome == OutcomeSlain && sourceCompany {
			t.kills[e.Source.Key()]++
		}
	}
}

// noteSwings counts a company member's attack round's strikes by what
// became of them. A round with no recorded strikes counts as one.
func (t *tally) noteSwings(e Event) {
	k := e.Source.Key()
	sw := t.swings[k]
	if sw == nil {
		sw = &Swings{Who: e.Source}
		t.swings[k] = sw
		t.swingOrder = append(t.swingOrder, k)
	}
	if len(e.Strikes) == 0 {
		sw.Thrown++
		switch {
		case e.Outcome == OutcomeMiss:
			sw.Missed++
		case len(e.Defenses) > 0:
			sw.Turned++
		case e.Damage <= 0:
			sw.Absorbed++
		}
		return
	}
	for _, st := range e.Strikes {
		if st.Pet != "" {
			continue // a pet's bite is not the member's own swing
		}
		sw.Thrown++
		switch {
		case !st.Hit:
			sw.Missed++
		case st.Defense != "":
			sw.Turned++
		case st.Damage <= 0:
			sw.Absorbed++
		}
	}
}

// landed is how many of a member's strikes got damage through.
func (sw Swings) landed() int { return sw.Thrown - sw.Missed - sw.Turned - sw.Absorbed }

// amounts turns a key->value map into Amounts, most first, ties by name.
func (t *tally) amounts(m map[string]int) []Amount {
	var out []Amount
	for k, v := range m {
		if v > 0 {
			out = append(out, Amount{Who: t.refs[k], Value: v})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Who.Name < out[j].Who.Name
	})
	return out
}

func (f *fight) summary(round uint64, outcome string, final Final) *Summary {
	t := &f.tally
	s := &Summary{
		FightID:          f.id,
		Outcome:          outcome,
		GroupName:        f.groupName,
		LeaderUserId:     f.leaderUserId,
		StartRound:       f.startRound,
		EndRound:         round,
		CompanyDamage:    t.companyDamage,
		EnemyDamage:      t.enemyDamage,
		Healing:          t.healing,
		HeldBack:         t.heldBack,
		MostDamage:       t.amounts(t.damage),
		HighestHit:       t.highest,
		CompanyDefenses:  t.companyDefenses,
		EnemyDefenses:    t.enemyDefenses,
		InterruptsDealt:  append([]string(nil), t.interruptsDealt...),
		InterruptsFailed: t.interruptsFailed,
		InterruptsTaken:  t.interruptsTaken,
		Guards:           t.amounts(t.guards),
		Kills:            t.amounts(t.kills),
		Taken:            t.amounts(t.taken),
	}
	for _, k := range t.swingOrder {
		if sw := t.swings[k]; sw.Thrown > 0 && sw.landed() == 0 {
			s.NeverLanded = append(s.NeverLanded, *sw)
		}
	}
	for _, name := range t.moveOrder {
		s.Moves = append(s.Moves, Count{Name: name, Count: t.moves[name]})
	}
	for _, m := range final.Company {
		if f.down[m.Ref.Key()] == OutcomeSlain {
			m.Fallen = true // slain in the fight, whatever their health now
		}
		s.Company = append(s.Company, m)
	}
	gone := map[string]bool{}
	for _, r := range final.Gone {
		gone[r.Key()] = true
	}
	for _, name := range t.effectOrder {
		s.Effects = append(s.Effects, Count{Name: name, Count: t.effects[name]})
	}
	for _, k := range f.enemyOrder {
		ending := EndingStanding
		switch {
		case f.down[k] == OutcomeSlain:
			ending = EndingSlain
		case f.down[k] == OutcomeBeaten:
			ending = EndingBeaten
		case f.yielded[k]:
			ending = EndingYielded
		case f.fled[k]:
			ending = EndingFled
		case gone[k]:
			ending = EndingLeft
		}
		s.Enemies = append(s.Enemies, EnemyEnding{Ref: f.enemies[k], Ending: ending})
	}
	return s
}

// Headings by fight outcome.
var headings = map[string]string{
	OutcomeVictory:   "The fighting is over",
	OutcomeDefeat:    "The company is beaten",
	OutcomeBrokenOff: "The fight breaks off",
}

// namedHeading heads the summary of a fight with a named group.
func namedHeading(outcome, group string) string {
	switch outcome {
	case OutcomeVictory:
		return "The fight with " + group + " is over"
	case OutcomeDefeat:
		return "The company is beaten by " + group
	}
	return "The fight with " + group + " breaks off"
}

// Render lays the summary out as text lines for the player viewerUserId,
// who reads their own name as "You". A line with nothing to report is
// left out, except damage, the enemies, and the company.
func Render(s Summary, viewerUserId int) []string {
	name := func(r Ref) string {
		if r.UserId != 0 && r.UserId == viewerUserId {
			return "You"
		}
		if r.Name == "" {
			return "someone"
		}
		return r.Name
	}
	line := func(label, body string) []string { return summaryLine(label, body) }
	joinAmounts := func(as []Amount) string {
		parts := make([]string, 0, len(as))
		for _, a := range as {
			parts = append(parts, fmt.Sprintf("%s %d", name(a.Who), a.Value))
		}
		return strings.Join(parts, " · ")
	}

	heading, ok := headings[s.Outcome]
	if !ok {
		heading = headings[OutcomeBrokenOff]
	}
	if s.GroupName != "" {
		heading = namedHeading(s.Outcome, s.GroupName)
	}
	// Phase 87: grouped into sections with a blank line between them.
	// Each section is built in its own list, in the order it reads.
	var blows, taken, result []string
	blows = append(blows, line("Damage dealt", fmt.Sprintf("Company %d · Enemies %d", s.CompanyDamage, s.EnemyDamage))...)
	if s.Healing > 0 || s.HeldBack > 0 {
		body := fmt.Sprintf("Company %d", s.Healing)
		if s.HeldBack > 0 {
			body += fmt.Sprintf(" (%d held back by a wound)", s.HeldBack)
		}
		taken = append(taken, line("Healing", body)...)
	}
	if len(s.MostDamage) > 0 {
		blows = append(blows, line("Most damage", joinAmounts(s.MostDamage))...)
	}
	if h := s.HighestHit; h != nil {
		body := fmt.Sprintf("%s %d on %s", name(h.Source), h.Damage, name(h.Target))
		if h.Crit {
			body += " (critical)"
		}
		blows = append(blows, line("Highest hit", body)...)
	}
	// Phase 30g2: each side's blocks, parries, and dodges, when any.
	var defenses []string
	if w := s.CompanyDefenses.words(); w != "" {
		defenses = append(defenses, "Company "+w)
	}
	if w := s.EnemyDefenses.words(); w != "" {
		defenses = append(defenses, "Enemies "+w)
	}
	if len(defenses) > 0 {
		taken = append(taken, line("Defenses", strings.Join(defenses, " · "))...)
	}
	if len(s.InterruptsDealt) > 0 || s.InterruptsFailed > 0 || s.InterruptsTaken > 0 {
		dealt := fmt.Sprintf("dealt %d", len(s.InterruptsDealt))
		if len(s.InterruptsDealt) > 0 {
			dealt += " (" + strings.Join(s.InterruptsDealt, ", ") + ")"
		}
		// "failed" is the company's blows a foe's chant withstood (Phase
		// 30d1b: a blow breaks a chant only by chance), shown only when
		// one did.
		parts := []string{dealt}
		if s.InterruptsFailed > 0 {
			parts = append(parts, fmt.Sprintf("failed %d", s.InterruptsFailed))
		}
		parts = append(parts, fmt.Sprintf("taken %d", s.InterruptsTaken))
		taken = append(taken, line("Interrupts", strings.Join(parts, " · "))...)
	}
	if len(s.Guards) > 0 {
		taken = append(taken, line("Guards", joinAmounts(s.Guards))...)
	}
	if len(s.Effects) > 0 {
		parts := make([]string, 0, len(s.Effects))
		for _, c := range s.Effects {
			parts = append(parts, fmt.Sprintf("%s %d", c.Name, c.Count))
		}
		blows = append(blows, line("Effects", strings.Join(parts, " · "))...)
	}
	if len(s.Kills) > 0 {
		blows = append(blows, line("Kills", joinAmounts(s.Kills))...)
	}
	// Phase 62: why the fight went as it did.
	if len(s.Taken) > 0 {
		taken = append(taken, line("Damage taken", joinAmounts(s.Taken))...)
	}
	if len(s.NeverLanded) > 0 {
		parts := make([]string, 0, len(s.NeverLanded))
		for _, sw := range s.NeverLanded {
			parts = append(parts, name(sw.Who)+" "+sw.why())
		}
		taken = append(taken, line("Never landed", strings.Join(parts, " · "))...)
	}
	if len(s.Moves) > 0 {
		parts := make([]string, 0, len(s.Moves))
		for _, c := range s.Moves {
			parts = append(parts, fmt.Sprintf("%s %d", c.Name, c.Count))
		}
		blows = append(blows, line("Moves", strings.Join(parts, " · "))...)
	}
	if s.Sigil != "" {
		blows = append(blows, line("Sigil", s.Sigil)...)
	}
	if len(s.Spoils) > 0 {
		result = append(result, line("Spoils", strings.Join(s.Spoils, " · "))...)
	}
	enemies := make([]string, 0, len(s.Enemies))
	for _, e := range s.Enemies {
		enemies = append(enemies, name(e.Ref)+" "+e.Ending)
	}
	if len(enemies) == 0 {
		enemies = append(enemies, "none")
	}
	result = append(result, line("Enemies", strings.Join(enemies, " · "))...)
	members := make([]string, 0, len(s.Company))
	for _, m := range s.Company {
		if m.Fallen {
			members = append(members, name(m.Ref)+" fallen")
			continue
		}
		members = append(members, fmt.Sprintf("%s %d/%d", name(m.Ref), m.Health, m.Max))
	}
	result = append(result, line("Company", strings.Join(members, " · "))...)
	// The heading, then the sections that have lines, a blank line apart,
	// each under its section's title.
	out := []string{"── " + heading + " ──"}
	for _, sec := range []struct {
		title string
		lines []string
	}{{"The fight", blows}, {"What it cost", taken}, {"How it ended", result}} {
		if len(sec.lines) == 0 {
			continue
		}
		out = append(out, "", sec.title)
		out = append(out, sec.lines...)
	}
	return out
}

// why says what became of the strikes that did nothing, "3 missed, 1
// dodged or parried".
func (sw Swings) why() string {
	var parts []string
	if sw.Missed > 0 {
		parts = append(parts, fmt.Sprintf("%d missed", sw.Missed))
	}
	if sw.Turned > 0 {
		parts = append(parts, fmt.Sprintf("%d turned aside", sw.Turned))
	}
	if sw.Absorbed > 0 {
		parts = append(parts, fmt.Sprintf("%d stopped by armor or a ward", sw.Absorbed))
	}
	return strings.Join(parts, ", ")
}

// summaryWidth is the widest a summary line runs (Phase 87 review): the web
// client on a 360px phone shows 49 columns with "Smaller text" off (57
// with it on), and a value the terminal wraps starts again under the
// labels and breaks a name mid-word.
const summaryWidth = 48

// summaryIndent lines a continued value up under its column.
var summaryIndent = strings.Repeat(" ", summaryLabelWidth)

const summaryLabelWidth = 15

// summaryLine is one labelled summary row, its value wrapped to
// summaryWidth between its " · " parts (or, for one long part, between
// words), each continuation indented under the value column.
func summaryLine(label, body string) []string {
	room := summaryWidth - summaryLabelWidth
	var rows []string
	cur := ""
	push := func() {
		if cur != "" {
			rows = append(rows, cur)
			cur = ""
		}
	}
	parts := strings.Split(body, " · ")
	for i, part := range parts {
		sep, tail := "", 0
		if i > 0 {
			sep = " · "
		}
		if i < len(parts)-1 {
			tail = 2 // room for the " ·" a wrap after this part leaves
		}
		if cur != "" && utf8.RuneCountInString(cur+sep+part)+tail <= room {
			cur += sep + part
			continue
		}
		if cur != "" {
			cur += " ·"
			push()
		}
		for _, word := range strings.Fields(part) {
			switch {
			case cur == "":
				cur = word
			case utf8.RuneCountInString(cur+" "+word) <= room:
				cur += " " + word
			default:
				push()
				cur = word
			}
		}
	}
	push()
	if len(rows) == 0 {
		rows = []string{""}
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		if i == 0 {
			out[i] = fmt.Sprintf("%-*s%s", summaryLabelWidth, label, r)
		} else {
			out[i] = summaryIndent + r
		}
	}
	return out
}
