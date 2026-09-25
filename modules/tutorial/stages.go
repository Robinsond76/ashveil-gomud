package tutorial

// Phase 27a: the stages, their gates, and the progress kept on the
// character. See docs/superpowers/specs/2026-09-25-phase-27a-tutorial-framework-design.md.
// Phase 27b adds Survival and Camp:
// docs/superpowers/specs/2026-09-25-phase-27b-tutorial-survival-camp-design.md.
// Phase 27c adds Combat:
// docs/superpowers/specs/2026-09-25-phase-27c-tutorial-practice-fight-design.md.

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
	StageSurvival  StageID = "survival"
	StageCamp      StageID = "camp"
	StageCombat    StageID = "combat"
	StageDeparture StageID = "departure"
)

// Stage is one lesson: the room it is taught in (an index into the
// configured tutorial rooms), its goal, and what to try. Inspections are
// commands, by registered name, the stage asks the player to run.
type Stage struct {
	ID          StageID
	Room        int
	Title       string
	Intro       string
	Goal        string
	Hints       []string
	Done        string
	Inspections []string
}

var stages []Stage

func init() {
	stages = []Stage{
		{
			ID: StageCharacter, Room: 0, Title: "Your character",
			Inspections: []string{"status", "inventory", "experience", "conditions"},
			Intro:       "You chose your path when you made your character, and it gave you a starter kit. Before you set out, look yourself over.",
			Goal:        "Look yourself over: status, inventory, experience, and conditions.",
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
			// Rooms 904 and 905 follow the Gate (903) in TutorialRooms.
			ID: StageSurvival, Room: 4, Title: "Survival",
			Intro: "Out here the weather is your enemy as much as any beast. You and every companion grow hungry, thirsty, and tired, the cold bites the unsheltered, and a heavy pack wears a company down.",
			Goal:  "Eat something and drink something, then check the weather, the temperature, your strain, and your cargo.",
			Hints: []string{
				`<ansi fg="command">eat <food></ansi> and <ansi fg="command">drink <drink></ansi>, e.g. <ansi fg="command">eat sandwich</ansi> and <ansi fg="command">drink waterskin</ansi>. Add a companion's name to feed them instead: <ansi fg="command">eat sandwich tamsin</ansi>.`,
				`<ansi fg="command">weather</ansi> shows the sky. Storms slow travel and spoil rest.`,
				`<ansi fg="command">temperature</ansi> shows how warm you are. Try it here in the open, then back in the Waking Hall: shelter, a fire, and warm clothes all help.`,
				`<ansi fg="command">strain</ansi> shows how worn your company is from walking; <ansi fg="command">cargo</ansi> shows your load. Too heavy a load tires everyone faster.`,
				`Walking from room to room nearby is quick. A journey between places, with <ansi fg="command">travel</ansi>, takes real time and never hurries the world along.`,
			},
			Done:        "You know what the road will ask of you.",
			Inspections: []string{"weather", "temperature", "strain", "cargo"},
		},
		{
			ID: StageCamp, Room: 5, Title: "Camp",
			Intro: "A company that never rests falls apart. In the wild you make camp, light a fire, and rest; a good rest leaves everyone Rested for a while. An inn's bed is better still: Well Rested.",
			Goal:  "Make camp, light a fire, and rest until your company is Rested.",
			Hints: []string{
				`<ansi fg="command">camp</ansi> makes camp here, <ansi fg="command">camp fire</ansi> lights the fire, and <ansi fg="command">camp rest</ansi> rests for about a minute. You stay put while you rest.`,
				`<ansi fg="command">camp status</ansi> shows the rest's progress and everyone's needs; <ansi fg="command">conditions</ansi> shows Rested afterwards.`,
				`With a whetstone, <ansi fg="command">camp sharpen on</ansi> hones every blade in the company at the end of a rest, using the stone once. Whetstones are sold in markets.`,
				`An inn stay (<ansi fg="command">inn</ansi>) costs gold but leaves you Well Rested, which is better than Rested.`,
			},
			Done: "Rested and ready.",
		},
		{
			ID: StageCombat, Room: 6, Title: "Combat",
			Intro: "Enemies fight as a band, in a formation like yours. Those in front shield those behind, and who can strike whom depends on where everyone stands. These straw soldiers can't hurt you; beat them to learn how a fight goes.",
			Goal:  "Beat all four straw soldiers, with your company's help.",
			Hints: []string{
				`Take your own place in the grid first, e.g. <ansi fg="command">formation move me 3 2</ansi>: where you stand decides what you can reach.`,
				`<ansi fg="command">attack footman</ansi> starts the fight; your companions join in. You keep fighting, round by round, until your target falls.`,
				`The archer stands behind the footmen: once you stand in the grid, an attack aimed at it is caught by the footman in front of it (interception).`,
				`"You can't reach that target from here" means it's too far across the grid, or too deep for your weapon: attack another footman, or move. <ansi fg="command">formation reach <name></ansi> shows who can reach what; long weapons reach one rank deeper, bows any.`,
				`When a foe falls, companions who were aiming at it turn to the next one they can reach, on their own. The one who struck it down picks a new target: <ansi fg="command">attack</ansi> again.`,
				`You can move anyone with <ansi fg="command">formation move</ansi> at any time, even mid-fight, to bring a foe within reach.`,
				`Watch your health in your prompt and <ansi fg="command">status</ansi>, and what ails you in <ansi fg="command">conditions</ansi>. A sharpened edge is spent one strike at a time, whether the blow does much or little.`,
			},
			Done: "The straw soldiers are beaten.",
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

// required is a stage's inspections whose commands are registered; a
// command whose module isn't loaded is never asked for.
func required(s Stage, registered func(string) bool) []string {
	var out []string
	for _, cmd := range s.Inspections {
		if registered(cmd) {
			out = append(out, cmd)
		}
	}
	return out
}

// inspected: every required inspection of a stage was run.
func inspected(s Stage, p progress, registered func(string) bool) bool {
	for _, cmd := range required(s, registered) {
		if !p.Seen[cmd] {
			return false
		}
	}
	return true
}

// Results the Survival stage records in Seen alongside its inspections.
const (
	seenFed     = "fed"
	seenWatered = "watered"
)

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
	// keySupplied marks the Survival stage's supplies as given (27b).
	keySupplied = "tutorial-supplied"
	// keyStrike marks a course camp strike that failed and is retried.
	keyStrike = "tutorial-strike"
)

// progress is a character's place in the course.
type progress struct {
	State    string
	Stage    StageID
	Seen     map[string]bool
	Supplied bool
}

func progressOf(c *characters.Character) progress {
	p := progress{Seen: map[string]bool{}}
	p.State, _ = c.GetMiscData(keyState).(string)
	stage, _ := c.GetMiscData(keyStage).(string)
	p.Stage = StageID(stage)
	supplied, _ := c.GetMiscData(keySupplied).(string)
	p.Supplied = supplied == "yes"
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
	if p.Supplied {
		c.SetMiscData(keySupplied, "yes")
	} else {
		c.SetMiscData(keySupplied, nil)
	}
}

// characterDone: every inspection was run.
func characterDone(p progress, registered func(string) bool) bool {
	return inspected(stages[stageIndex(StageCharacter)], p, registered)
}

// survivalDone: fed, watered, and every inspection run.
func survivalDone(p progress, registered func(string) bool) bool {
	return p.Seen[seenFed] && p.Seen[seenWatered] && inspected(stages[stageIndex(StageSurvival)], p, registered)
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
