package tutorial

// Phase 27a: the stages, their gates, and the progress kept on the
// character. See docs/superpowers/specs/2026-09-25-phase-27a-tutorial-framework-design.md.

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
)

// StageID names a stage. Later phases insert theirs before Departure.
type StageID string

const (
	StageCharacter StageID = "character"
	StageCompany   StageID = "company"
	StageFormation StageID = "formation"
	StageDeparture StageID = "departure"
)

// Stage is one lesson: the room it is taught in (an index into the
// configured tutorial rooms), its goal, and what to try.
type Stage struct {
	ID    StageID
	Room  int
	Title string
	Intro string
	Goal  string
	Hints []string
	Done  string
}

var stages []Stage

func init() {
	stages = []Stage{
		{
			ID: StageCharacter, Room: 0, Title: "Your character",
			Intro: "You chose your path when you made your character, and it gave you a starter kit. Before you set out, look yourself over.",
			Goal:  "Look yourself over: status, inventory, experience, and conditions.",
			Hints: []string{
				`<ansi fg="command">status</ansi> is your character sheet: your path, vitals, hunger, thirst, fatigue, and company.`,
				`<ansi fg="command">inventory</ansi> shows your gear and your company's load.`,
				`<ansi fg="command">experience</ansi> shows your level and progress.`,
				`<ansi fg="command">conditions</ansi> lists what is affecting you, and for how long.`,
				`Short forms work too. Your prompt adds warnings such as Hungry or Dark only when something needs your attention.`,
			},
			Done: "You know yourself well enough.",
		},
		{
			ID: StageCompany, Room: 1, Title: "Your company",
			Intro: "You lead a company: you and up to four companions who follow you, fight beside you, and need food and rest like you do. It isn't a party of other players; players can form parties as well.",
			Goal:  "Recruit two companions.",
			Hints: []string{
				`<ansi fg="command">company recruit</ansi> shows who is hiring here. Recruit two: <ansi fg="command">company recruit tamsin</ansi> and <ansi fg="command">company recruit oswin</ansi>. Each is free, once.`,
				`<ansi fg="command">company status</ansi> shows your companions. A company holds five at most, you included.`,
				`Companions keep their own gear (<ansi fg="command">company gear</ansi>). <ansi fg="command">company dismiss</ansi> lets one go, with their gear.`,
			},
			Done: "Your company is gathered.",
		},
		{
			ID: StageFormation, Room: 2, Title: "Formation",
			Intro: "Your company fights in a formation of three rows of three. Row 1 is the front: those in it shield the rows behind, and only long weapons reach from the back. The grid is your battle order, not a map.",
			Goal:  "Put one companion in the front row and another behind it.",
			Hints: []string{
				`<ansi fg="command">formation</ansi> shows the grid, row by row.`,
				`<ansi fg="command">formation move <name> <row> <col></ansi> places someone, e.g. <ansi fg="command">formation move tamsin 1 2</ansi>.`,
				`<ansi fg="command">formation swap <a> <b></ansi> trades two members' places.`,
			},
			Done: "A sound formation.",
		},
		{
			ID: StageDeparture, Room: 3, Title: "Departure",
			Intro: "Your training is done. Look over your company and your pack before you go.",
			Goal:  "When you're ready, go through the gate.",
			Hints: []string{
				`<ansi fg="command">company status</ansi>, <ansi fg="command">inventory</ansi>, and <ansi fg="command">status</ansi> one last time.`,
				`Go through the <ansi fg="exit">gate</ansi> to begin your journey.`,
			},
		},
	}
}

// stageIndex is a stage's position; -1 for an unknown ID.
func stageIndex(id StageID) int {
	for i, s := range stages {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// inspections are the Character stage's commands, by registered name.
var inspections = []string{"status", "inventory", "experience", "conditions"}

// Progress states.
const (
	stateNone      = ""
	stateActive    = "active"
	stateSkipped   = "skipped"
	stateGraduated = "graduated"
)

// MiscData keys; the values are strings, so they survive the user file.
const (
	keyState = "tutorial-state"
	keyStage = "tutorial-stage"
	keySeen  = "tutorial-seen"
)

// progress is a character's place in the course.
type progress struct {
	State string
	Stage StageID
	Seen  map[string]bool
}

func progressOf(c *characters.Character) progress {
	p := progress{Seen: map[string]bool{}}
	p.State, _ = c.GetMiscData(keyState).(string)
	stage, _ := c.GetMiscData(keyStage).(string)
	p.Stage = StageID(stage)
	seen, _ := c.GetMiscData(keySeen).(string)
	for _, s := range strings.Split(seen, ",") {
		if s != "" {
			p.Seen[s] = true
		}
	}
	return p
}

func (p progress) save(c *characters.Character) {
	c.SetMiscData(keyState, p.State)
	c.SetMiscData(keyStage, string(p.Stage))
	seen := make([]string, 0, len(p.Seen))
	for s := range p.Seen {
		seen = append(seen, s)
	}
	sort.Strings(seen)
	c.SetMiscData(keySeen, strings.Join(seen, ","))
}

// characterDone: every inspection was run.
func characterDone(p progress) bool {
	for _, cmd := range inspections {
		if !p.Seen[cmd] {
			return false
		}
	}
	return true
}

func living(members []company.MemberView) map[company.MemberKey]bool {
	out := map[company.MemberKey]bool{}
	for _, m := range members {
		if m.Status != company.MemberDead {
			out[company.CompanionMemberKey(m.ID)] = true
		}
	}
	return out
}

// companyDone: two living companions.
func companyDone(members []company.MemberView) bool {
	return len(living(members)) >= 2
}

// formationDone: a living companion in the front row and another behind.
func formationDone(f company.Formation, members []company.MemberView) bool {
	alive := living(members)
	front, rear := false, false
	for r := 0; r < company.FormationRows; r++ {
		for c := 0; c < company.FormationCols; c++ {
			if key := f.At(r, c); alive[key] {
				if r == 0 {
					front = true
				} else {
					rear = true
				}
			}
		}
	}
	return front && rear
}
