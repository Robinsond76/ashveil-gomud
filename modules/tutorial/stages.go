package tutorial

// Phase 27a: the stages, their gates, and the progress kept on the
// character. See docs/designs/2026-09-25-phase-27a-tutorial-framework-design.md.
// Phase 27b adds Survival and Camp:
// docs/designs/2026-09-25-phase-27b-tutorial-survival-camp-design.md.
// Phase 27c adds Combat:
// docs/designs/2026-09-25-phase-27c-tutorial-practice-fight-design.md.
// Phase 27d adds Alignment:
// docs/designs/2026-09-25-phase-27d-tutorial-alignment-panel-design.md.

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
	StageAlignment StageID = "alignment"
	StageDeparture StageID = "departure"
)

// Stage is one lesson: the room it is taught in (an index into the
// configured tutorial rooms), its goal, and what to try. Inspections are
// commands the stage asks the player to run: a registered name, optionally
// with a subcommand ("company alignment") that must be the first word of the
// rest of the line.
type Stage struct {
	ID          StageID
	Room        int
	Title       string
	Intro       string
	Goal        string
	Hints       []string
	Done        string
	Inspections []string
	// Labels shows an inspection's checklist line in words other than its
	// key (27d review: "company inspect" is done by inspecting Corvin).
	Labels map[string]string
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
				`<ansi fg="command">experience</ansi> shows your level and progress, and each companion's (<ansi fg="command">help company</ansi> explains how they earn it).`,
				`<ansi fg="command">conditions</ansi> lists what is affecting you, and for how long.`,
				`Playing in the web client? The dock on the right shows your character, your company, and how it fights at a glance (<ansi fg="command">help webclient</ansi>).`,
				`Short forms work too. Your prompt adds warnings such as Hungry or Dark only when something needs your attention.`,
				`<ansi fg="command">help status</ansi> and <ansi fg="command">help archetype</ansi> explain your sheet and your path. <ansi fg="command">help adventure</ansi> lists every help page.`,
				`You can run this course again later, as a practice character that keeps nothing: <ansi fg="command">tutorial replay</ansi> (<ansi fg="command">help tutorial</ansi>).`,
			},
			Done: "You know yourself well enough.",
		},
		{
			ID: StageCompany, Room: 1, Title: "Your company",
			Intro: "You lead a company: you and up to four companions who follow you, fight beside you, and need food and rest like you do. It isn't a party of other players; players can form parties as well.",
			Goal:  "Recruit two companions.",
			Hints: []string{
				`<ansi fg="command">look</ansi> shows the hiring post and who is waiting on it (<ansi fg="command">look post</ansi> reads it again). <ansi fg="command">look tamsin</ansi> reads about one candidate.`,
				`<ansi fg="command">company recruit</ansi> shows who is hiring here. Recruit two: <ansi fg="command">company recruit tamsin</ansi> and <ansi fg="command">company recruit oswin</ansi>. Each is free, once.`,
				`<ansi fg="command">company status</ansi> shows your companions. A company holds five at most, you included.`,
				`Safe <ansi fg="command">ask</ansi> orders manage a present member outside battle; members fight automatically. Conversation and observation remain available (<ansi fg="command">help ask</ansi>).`,
				`In Character > Gear, select a slot and a compatible cargo item to preview Current → After before equipping; Preview removal returns worn gear to cargo (<ansi fg="command">help equipment</ansi>, <ansi fg="command">help equip</ansi>). <ansi fg="command">help treasury</ansi> covers pooled gold.`,
				`Companions keep their own gear (<ansi fg="command">company gear</ansi>). <ansi fg="command">company dismiss</ansi> lets one go, with their gear.`,
				`Companions grow by their archetype: each level's stat points go where the archetype needs them, and <ansi fg="command">company growth</ansi> can favour one stat. Contract quests pay every companion with you when you turn them in (<ansi fg="command">help growth</ansi>, <ansi fg="command">help contracts</ansi>).`,
				`<ansi fg="command">help company</ansi> covers recruiting and your roster in full.`,
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
				`<ansi fg="command">help formation</ansi> explains reach and interception in full.`,
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
				`<ansi fg="command">strain</ansi> shows how worn your company is from walking; <ansi fg="command">cargo</ansi> shows your load. Too heavy a load tires everyone faster, and a full company can't pick up anything more. Each member, a pack, and a horse add room (<ansi fg="command">help cargo</ansi>, <ansi fg="command">help mount</ansi>).`,
				`<ansi fg="command">company meal</ansi> feeds and waters everyone at once, from the cargo, then their own packs, then yours; <ansi fg="command">company inventory</ansi> shows what everyone carries (<ansi fg="command">help company-meal</ansi>, <ansi fg="command">help company-inventory</ansi>).`,
				`Walking from room to room nearby is quick. A journey between places, with <ansi fg="command">travel</ansi>, takes real time and never hurries the world along.`,
				`Each has a help page: <ansi fg="command">help survival</ansi>, <ansi fg="command">help weather</ansi>, <ansi fg="command">help temperature</ansi>, <ansi fg="command">help strain</ansi>, <ansi fg="command">help cargo</ansi>, and <ansi fg="command">help travel</ansi>.`,
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
				`With a whetstone, <ansi fg="command">camp sharpen on</ansi> hones every blade in the company at the end of a rest, using the stone once. Whetstones are sold in markets (<ansi fg="command">help sharpen</ansi>).`,
				`An inn stay (<ansi fg="command">inn</ansi>) costs gold but leaves you Well Rested, which is better than Rested, and restores everyone's health and mana. A camp rest doesn't: out of a fight, your company recovers slowly as you go (<ansi fg="command">help readiness</ansi>).`,
				`A finished camp rest knits a broken bone for each splint and a cut for each bandage you carry, and an inn stay every wound. After a fight, <ansi fg="command">heal wounds</ansi> has your clerics, bandages, and splints tend the hurt (<ansi fg="command">help wounds</ansi>).`,
				`<ansi fg="command">help camp</ansi>, <ansi fg="command">help inn</ansi>, and <ansi fg="command">help cooking</ansi> explain resting and cooking in full.`,
				`Companions earn their keep at camp: a warrior keeps watch for raiders (<ansi fg="command">help campwatch</ansi>), a ranger forages, a cleric keeps a vigil, and <ansi fg="command">camp cook</ansi> cooks over your own fire.`,
			},
			Done: "Rested and ready.",
		},
		{
			ID: StageCombat, Room: 6, Title: "Combat",
			Intro: "Enemies fight as a band, in a formation like yours. Those in front shield those behind, and who can strike whom depends on where everyone stands. These straw soldiers can't hurt you; beat them to learn how a fight goes.",
			Goal:  "Beat all four straw soldiers, with your company's help.",
			Hints: []string{
				`Take your own place in the grid first, e.g. <ansi fg="command">formation move me 3 2</ansi>: where you stand decides what you can reach.`,
				`The squad is one group, "the straw squad": you fight a whole group, never one soldier. <ansi fg="command">scout squad</ansi> shows how it stands, with a * on the ones you can reach from your place (<ansi fg="command">help scout</ansi>), and ends with your company's assessment: how risky the fight looks, in words, and what it can't judge (<ansi fg="command">help assessment</ansi>).`,
				`<ansi fg="command">attack squad</ansi> starts the fight; your companions join in. From then on the battle plays out on its own, from how you set your company up: set your formation before you attack.`,
				`Group healing helps the present company even without a player party; it cannot revive a fallen companion. Targets are checked again when the chant finishes, and no one can heal either side of a battle they aren't part of (<ansi fg="command">help friendly-effects</ansi>).`,
				`The archer stands behind the footmen: once you stand in the grid, a blow aimed at it is caught by the footman in front of it (interception).`,
				`Everyone starts on the foe their strategy picks (the weakest they can reach, unless you set another), and when a foe falls, turns to the next the same way, on their own. <ansi fg="command">formation reach [name]</ansi> shows who can reach what; long weapons reach one rank deeper, bows any.`,
				`<ansi fg="command">strategy</ansi> shows how each of you fights: a role (fighter, healer, caster, guardian) and whom to go for. Healers heal and casters cast on their own, with real mana. Set them before you attack, e.g. <ansi fg="command">strategy me strongest</ansi> (<ansi fg="command">help strategy</ansi>).`,
				`Warriors, rogues, and rangers also use a class ability on their own when the moment comes: a warrior tackles a foe to the ground, a rogue strikes a foe that is down or exposed, its first blow that lands a critical hit, a ranger aims a shot. <ansi fg="command">strategy [who]</ansi> shows when each is used, and <ansi fg="command">strategy [who] abilities off</ansi> holds them back (<ansi fg="command">help abilities</ansi>).`,
				`<ansi fg="command">company tactics</ansi> sets orders for the whole company: a focus everyone goes for (their leader, their casters, the weakest...) and how hurt someone must be before your healers heal. The focus is the one thing you can change mid-battle, once a round: <ansi fg="command">company tactics focus leader</ansi> (<ansi fg="command">help tactics</ansi>).`,
				`A companion (or you) can be a guardian, who steps in front of a ward to take blows meant for them (2 guards, one back every 2 rounds): <ansi fg="command">strategy tamsin guard me</ansi> before you attack, with the two of you in the same or neighbouring columns (<ansi fg="command">help guardian</ansi>).`,
				`Foes may surrender when their group loses nerve. After victory you may spare or execute them; your companions react by their beliefs. A losing company may hesitate or temporarily flee (<ansi fg="command">help morale</ansi>, <ansi fg="command">help mercy</ansi>).`,
				`A spell is chanted for a round or two, and a blow that draws blood on the caster may break it (a critical or staggering blow always does): keep your healers and casters in the back row or guarded. A blow someone blocks with a shield may be bashed back, and the bash can stun. A big foe such as an ogre may wind up a heavy blow in plain view, and only a critical hit or a blow that staggers, knocks down, or stuns breaks it (<ansi fg="command">help interrupts</ansi>).`,
				`Where several bands stand together, you fight them one at a time: the others wait their turn, and the next fight begins as soon as one ends.`,
				`Assigned packs and eligible horses provide shared cargo capacity; worn weapons and armor only affect combat burden (<ansi fg="command">help pack</ansi>, <ansi fg="command">help cargo</ansi>).`,
				`A solo leader stays at row 2, column 2; recruits take a vacant cell automatically. Rearrange them before battle (<ansi fg="command">help formation</ansi>).`,
				`Once the battle starts, your formation and strategies are fixed until it ends, and <ansi fg="command">retreat</ansi> or <ansi fg="command">flee</ansi> takes you out: move anyone with <ansi fg="command">formation move</ansi> before you attack.`,
				`Watch your health in your prompt and <ansi fg="command">status</ansi>, and what ails you in <ansi fg="command">conditions</ansi>. A sharpened edge is spent one strike at a time, whether the blow does much or little.`,
				`Repeated enemies have fixed names: the first footman and the second footman. When the first falls, the second keeps its name. Pronouns follow the people and creatures in the fight (<ansi fg="command">help narration</ansi>).`,
				`A blow that would hit meets one defense: a shield blocks it, a weapon parries it, or its target dodges. A shield-bearer never dodges, and a stunned fighter has no defense at all. Armor takes its share of what gets through (<ansi fg="command">help defense</ansi>).`,
				`What you wear and carry yourself weighs on you in a fight: a heavy load against your Strength leaves you burdened, and a burdened fighter dodges less (a block or a parry is untouched). Your company's cargo and horses never lighten you. <ansi fg="command">status</ansi> shows your burden, and <ansi fg="command">look</ansi> at anyone shows theirs (<ansi fg="command">help burden</ansi>).`,
				`Each hit ends with what it did, e.g. (5 damage) or (critical hit, 9 damage); a line with no brackets is a miss, a block, a parry, or a dodge (<ansi fg="command">help narration</ansi>).`,
				`A damaging critical hit gets a pain reaction if its victim stays standing. A fatal critical goes straight to a death line (<ansi fg="command">help narration</ansi>).`,
				`A critical hit also leaves a mark that matches the weapon: bleeding, a stagger, a knockdown. Staggered and knocked-down fighters lose their next action, and it all ends with the fight (<ansi fg="command">help statuses</ansi>).`,
				`A round's lines come to you one by one, so you watch the fight unfold; your prompt's health catches up when they finish. <ansi fg="command">set combatpace fast</ansi>, normal, slow, or off sets how fast (<ansi fg="command">help combatpace</ansi>).`,
				`When the last foe falls, a battle summary shows the damage, the kills, and your company's health (<ansi fg="command">help battle-summary</ansi>).`,
				`Playing in the web client? The Combat tab shows the battle as it goes: both formations, who strikes whom, and how hurt each foe looks (<ansi fg="command">help webclient</ansi>).`,
				`To leave a battle, <ansi fg="command">retreat [exit]</ansi> (or <ansi fg="command">flee</ansi>): your company prepares for one round, then tries to withdraw together. A hobbled member pins everyone until the hurt passes (<ansi fg="command">help retreat</ansi>).`,
				`For every detail of how battles work, see <ansi fg="command">help combat</ansi>, and from there <ansi fg="command">help formation</ansi>, <ansi fg="command">help strategy</ansi>, <ansi fg="command">help targeting</ansi>, <ansi fg="command">help chemistry</ansi>, and <ansi fg="command">help light</ansi>.`,
			},
			Done: "The straw soldiers are beaten.",
		},
		{
			ID: StageAlignment, Room: 7, Title: "Alignment",
			Intro: "Everyone leans good or evil, your companions too. A company slowly drifts toward its members' common ground, and a companion who stands far from the rest loses loyalty and may one day walk away. You never drift: you set the course. Towns weigh a company's reputation as well.",
			Goal:  "Check your company's alignment, inspect the outlaw Corvin, and check your standing.",
			Hints: []string{
				`<ansi fg="command">company alignment</ansi> shows the company's average and each member's alignment and loyalty.`,
				`<ansi fg="command">company inspect corvin</ansi> weighs a would-be recruit against your company. Too far from its average, and a recruit refuses to join: Corvin is an outlaw your company won't take.`,
				`Companions drift a little toward the rest of the company over time. One kept too far from the others loses loyalty, and at none deserts.`,
				`<ansi fg="command">standing</ansi> shows how a settlement regards your company: good standing gets fair prices, poor standing higher ones, and outlaws are turned away but for the black market.`,
				`<ansi fg="command">help alignment</ansi> and <ansi fg="command">help standing</ansi> explain both in full; <ansi fg="command">help market</ansi> and <ansi fg="command">help rumors</ansi> cover trading in the towns.`,
			},
			Done:        "You know where your company stands.",
			Inspections: []string{"company alignment", "company inspect", "standing"},
			Labels:      map[string]string{"company inspect": "company inspect corvin"},
		},
		{
			ID: StageDeparture, Room: 3, Title: "Departure",
			Intro: "Your training is done. Look over your company and your pack before you go.",
			Goal:  "When you're ready, go through the gate.",
			Hints: []string{
				`After a battle, <ansi fg="command">loot</ansi> collects eligible spoils into cargo. <ansi fg="command">help loot</ansi> explains claims and optional autoloot.`,
				`<ansi fg="command">company status</ansi>, <ansi fg="command">inventory</ansi>, and <ansi fg="command">status</ansi> one last time.`,
				`Your companions keep their health and wounds when you log out: a hurt company is still hurt when you come back. Rest at an inn before a long road (<ansi fg="command">help readiness</ansi>).`,
				`Go through the <ansi fg="exit">gate</ansi> to begin your journey.`,
				`Settlements post recruits of your own, and the faces change over time: <ansi fg="command">look</ansi> at the Waymark Inn's hiring slate, and see <ansi fg="command">help company</ansi>.`,
				`Recruit for the road, too: a ranger reads the trail and eases rough ground, a rogue spots secret ways and haggles, a wizard forecasts the weather. <ansi fg="command">company specialists</ansi> shows who does what; see <ansi fg="command">help specialists</ansi>.`,
				`<ansi fg="command">help adventure</ansi> lists every part of the game. <ansi fg="command">help combat</ansi> covers battles, and <ansi fg="command">help death</ansi> and <ansi fg="command">help resurrect</ansi> what happens when someone falls.`,
				`<ansi fg="command">help party</ansi> explains player alliances. Following and allied group support require your consent; each owner commands their own company.`,
				`To start over with a new character on the same login, see <ansi fg="command">help delete</ansi>.`,
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
	for _, key := range s.Inspections {
		cmd, _, _ := strings.Cut(key, " ")
		if registered(cmd) {
			out = append(out, key)
		}
	}
	return out
}

// inspectionMatches reports whether a handled command (its registered
// name and the rest of the line) is an inspection: the command, and the
// key's subcommand, if it has one, as the first word of the rest.
func inspectionMatches(key, command, rest string) bool {
	keyCmd, keySub, _ := strings.Cut(key, " ")
	if command != keyCmd {
		return false
	}
	if keySub == "" {
		return true
	}
	words := strings.Fields(rest)
	return len(words) > 0 && strings.EqualFold(words[0], keySub)
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
	front, rear := formationRanks(f, members)
	return front && rear
}

// formationRanks: whether a living companion stands in the front row, and
// whether one stands behind it.
func formationRanks(f company.Formation, members []company.MemberView) (front, rear bool) {
	alive := living(members)
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
	return front, rear
}
