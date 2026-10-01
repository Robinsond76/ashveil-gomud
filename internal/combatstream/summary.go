package combatstream

import (
	"fmt"
	"sort"
	"strings"
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

	Enemies []EnemyEnding
	Company []MemberHealth
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
}

func newTally() tally {
	return tally{damage: map[string]int{}, guards: map[string]int{}, effects: map[string]int{}, kills: map[string]int{}, refs: map[string]Ref{}}
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
		if e.Damage <= 0 {
			return
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
	line := func(label, body string) string { return fmt.Sprintf("%-15s%s", label, body) }
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
	out := []string{
		"── " + heading + " ──",
		line("Damage dealt", fmt.Sprintf("Company %d · Enemies %d", s.CompanyDamage, s.EnemyDamage)),
	}
	if s.Healing > 0 || s.HeldBack > 0 {
		body := fmt.Sprintf("Company %d", s.Healing)
		if s.HeldBack > 0 {
			body += fmt.Sprintf(" (%d held back by a wound)", s.HeldBack)
		}
		out = append(out, line("Healing", body))
	}
	if len(s.MostDamage) > 0 {
		out = append(out, line("Most damage", joinAmounts(s.MostDamage)))
	}
	if h := s.HighestHit; h != nil {
		body := fmt.Sprintf("%s %d on %s", name(h.Source), h.Damage, name(h.Target))
		if h.Crit {
			body += " (critical)"
		}
		out = append(out, line("Highest hit", body))
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
		out = append(out, line("Defenses", strings.Join(defenses, " · ")))
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
		out = append(out, line("Interrupts", strings.Join(parts, " · ")))
	}
	if len(s.Guards) > 0 {
		out = append(out, line("Guards", joinAmounts(s.Guards)))
	}
	if len(s.Effects) > 0 {
		parts := make([]string, 0, len(s.Effects))
		for _, c := range s.Effects {
			parts = append(parts, fmt.Sprintf("%s %d", c.Name, c.Count))
		}
		out = append(out, line("Effects", strings.Join(parts, " · ")))
	}
	if len(s.Kills) > 0 {
		out = append(out, line("Kills", joinAmounts(s.Kills)))
	}
	enemies := make([]string, 0, len(s.Enemies))
	for _, e := range s.Enemies {
		enemies = append(enemies, name(e.Ref)+" "+e.Ending)
	}
	if len(enemies) == 0 {
		enemies = append(enemies, "none")
	}
	out = append(out, line("Enemies", strings.Join(enemies, " · ")))
	members := make([]string, 0, len(s.Company))
	for _, m := range s.Company {
		if m.Fallen {
			members = append(members, name(m.Ref)+" fallen")
			continue
		}
		members = append(members, fmt.Sprintf("%s %d/%d", name(m.Ref), m.Health, m.Max))
	}
	out = append(out, line("Company", strings.Join(members, " · ")))
	return out
}
