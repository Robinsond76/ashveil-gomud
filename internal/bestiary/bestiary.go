// Package bestiary is Ashveil's Phase 66 bestiary: what a leader has
// learned about each kind of creature by fighting it. Knowledge comes in
// three tiers (lore, defences, habits and weaknesses) and is earned by
// kills, read from the kill tally the character already keeps (so it
// survives restart and copyover with the character, and holds nothing new
// to save). Every line of an entry is built from the creature's template,
// never written by hand per creature.
//
// Switching the source to the company chronicle (Phase 63) later means
// replacing KillsOf; nothing else reads the tally.
package bestiary

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/windup"
)

// Tier is how much a leader knows of one kind.
type Tier int

const (
	Unknown  Tier = iota // never beaten: not in the bestiary
	Lore                 // what it is and where it lives
	Defences             // how well it stands up to blows
	Habits               // how it fights, and where it is weak
)

// Name is the tier's name for players.
func (t Tier) Name() string {
	switch t {
	case Lore:
		return "lore"
	case Defences:
		return "defences"
	case Habits:
		return "habits"
	}
	return "unknown"
}

// Thresholds are the kills of a kind that earn each tier. A boss is met
// rarely, so it teaches faster.
func Thresholds(boss bool) [3]int {
	if boss {
		return [3]int{1, 2, 3}
	}
	return [3]int{1, 3, 6}
}

// TierFor is the tier that kills of a kind earn.
func TierFor(kills int, boss bool) Tier {
	t := Unknown
	for i, need := range Thresholds(boss) {
		if kills >= need {
			t = Tier(i + 1)
		}
	}
	return t
}

// KillsOf is the leader's kills by creature template id. It is the one
// place the bestiary reads its source.
func KillsOf(c *characters.Character) map[int]int {
	if c == nil {
		return nil
	}
	return c.KD.Kills
}

// Entry is what a leader knows of one kind. Lines are present only for
// the tiers earned.
type Entry struct {
	MobID    int
	Name     string
	Zone     string
	Level    int
	Boss     bool
	Kills    int
	Tier     Tier
	NextAt   int // kills of the kind that earn the next tier; 0 when all are known
	Lore     []string
	Defences []string
	Habits   []string
	// Notes are the short habit phrases for a battle caption ("heals its
	// allies"): only from the habits tier.
	Notes []string
}

// Build is the entry for a creature template at the given kills; a
// template that can't be known (practice foes, no template) is not ok.
func Build(spec *mobs.Mob, kills int) (Entry, bool) {
	if spec == nil || spec.Practice || kills < 1 {
		return Entry{}, false
	}
	e := Entry{
		MobID: int(spec.MobId), Name: spec.Character.Name, Zone: spec.Zone,
		Level: spec.Character.Level, Boss: spec.Boss, Kills: kills,
	}
	e.Tier = TierFor(kills, spec.Boss)
	if e.Tier == Unknown {
		return Entry{}, false
	}
	for _, need := range Thresholds(spec.Boss) {
		if kills < need {
			e.NextAt = need
			break
		}
	}
	e.Lore = loreLines(spec)
	if e.Tier >= Defences {
		e.Defences = defenceLines(spec)
	}
	if e.Tier >= Habits {
		e.Habits, e.Notes = habitLines(spec)
	}
	return e, true
}

// Known is every kind the leader has fought, by zone then name.
func Known(kills map[int]int) []Entry {
	var out []Entry
	for id, n := range kills {
		if e, ok := Build(mobs.GetMobSpec(mobs.MobId(id)), n); ok {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Zone != b.Zone {
			return a.Zone < b.Zone
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.MobID < b.MobID
	})
	return out
}

// Find matches a known kind by name (exact, else prefix, else contained),
// the best-known first among equals. Only known kinds are searched.
func Find(entries []Entry, search string) (Entry, bool) {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return Entry{}, false
	}
	for _, match := range []func(name string) bool{
		func(n string) bool { return n == search },
		func(n string) bool { return strings.HasPrefix(n, search) },
		func(n string) bool { return strings.Contains(n, search) },
	} {
		var best *Entry
		for i := range entries {
			if !match(strings.ToLower(entries[i].Name)) {
				continue
			}
			if best == nil || entries[i].Kills > best.Kills {
				best = &entries[i]
			}
		}
		if best != nil {
			return *best, true
		}
	}
	return Entry{}, false
}

// Progress says what the next tier takes, or that the entry is full.
func (e Entry) Progress() string {
	if e.NextAt == 0 {
		return "everything is known"
	}
	left := e.NextAt - e.Kills
	next := Tier(int(e.Tier) + 1).Name()
	return fmt.Sprintf("%d more %s to learn its %s", left, plural(left, "kill", "kills"), next)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func loreLines(spec *mobs.Mob) []string {
	c := &spec.Character
	race := races.GetRace(c.RaceId)
	kind := "creature"
	if race != nil {
		kind = strings.ToLower(race.Name)
		if race.Size == races.Large {
			kind = "large " + kind
		} else if race.Size == races.Small {
			kind = "small " + kind
		}
	}
	where := ""
	if spec.Zone != "" {
		where = ", found in " + spec.Zone
	}
	// No level: room and encounter spawns set their own, so the template's
	// could disagree with the foe met (consider gives the real one).
	out := []string{fmt.Sprintf("%s %s%s.", article(kind), kind, where)}
	if spec.Boss {
		out = append(out, "A boss among its kind: tougher than the rest. A boss gives up its secrets in fewer kills.")
	}
	if d := strings.TrimSpace(c.Description); d != "" {
		out = append(out, d)
	} else if race != nil && strings.TrimSpace(race.Description) != "" {
		out = append(out, strings.TrimSpace(race.Description))
	}
	switch {
	case spec.Solitary:
		out = append(out, "Keeps to itself: you will not find it in a group.")
	default:
		noun := spec.GroupNoun
		if noun == "" && race != nil {
			noun = race.GroupNoun
		}
		if noun != "" {
			out = append(out, "Moves in a "+noun+".")
		}
	}
	if spec.Hostile {
		out = append(out, "Attacks on sight.")
	} else {
		out = append(out, "Leaves you be unless provoked.")
	}
	return out
}

func defenceLines(spec *mobs.Mob) []string {
	c := &spec.Character
	var out []string
	if armor := c.GetDefense(); armor > 0 {
		// The engine rolls 0 to armor-1 percent off each blow.
		out = append(out, fmt.Sprintf("Armor turns aside up to %d%% of a blow, about half that on average.", armor))
	} else {
		out = append(out, "Wears no armor worth the name: blows land in full.")
	}
	switch {
	case spec.EvasionSkill >= 3:
		out = append(out, "Quick to dodge: expect misses.")
	case spec.EvasionSkill > 0:
		out = append(out, "Somewhat quick to dodge.")
	case spec.EvasionSkill <= -3:
		out = append(out, "Slow on its feet: easy to hit.")
	case spec.EvasionSkill < 0:
		out = append(out, "A little slow to dodge.")
	}
	switch strings.ToLower(spec.PoisonSusceptibility) {
	case "immune":
		out = append(out, "Poison finds nothing to take hold of.")
	case "resistant":
		out = append(out, "Poison takes poorly: half as often.")
	}
	if !spec.TakesWounds() {
		out = append(out, "Takes no lasting wounds.")
	}
	if spec.Boss {
		out = append(out, "Resists hexes, and shakes them off in half the time.")
	}
	if spec.Reach {
		out = append(out, "Long limbs: it strikes from the second row as well.")
	}
	if spec.Sweep {
		out = append(out, "Sweeps its weapon across a whole row.")
	}
	return out
}

// targetWords is how a foe's re-aim rule reads.
var targetWords = map[string]string{
	"weakest":   "goes for whoever has the least health left",
	"strongest": "goes for whoever has the most health left",
	"wounded":   "goes for whoever is the most hurt",
	"nearest":   "goes for the nearest of you",
	"furthest":  "goes for the one standing furthest back",
	"leader":    "goes for your leader",
	"casters":   "goes for your spell-casters first",
	"healers":   "goes for your healers first",
	"armored":   "goes for the most heavily armored",
}

func habitLines(spec *mobs.Mob) (lines, notes []string) {
	c := &spec.Character
	switch spec.EnemyRole() {
	case "healer":
		lines = append(lines, "Tends its allies' wounds: bring it down first, or set an order (orders [who] add foe healer then break).")
		notes = append(notes, "heals its allies")
	case "caster":
		lines = append(lines, "Fights with spells: it chants before it casts, and a chant can be broken (orders [who] add chanting then break).")
		notes = append(notes, "casts spells")
	case "guardian":
		lines = append(lines, "Guards its allies, turning blows meant for them.")
		notes = append(notes, "guards its allies")
	}
	var names []string
	for id := range c.SpellBook {
		if sp := spells.GetSpell(id); sp != nil && sp.Name != "" {
			names = append(names, sp.Name)
		}
	}
	sort.Strings(names)
	if len(names) > 0 {
		lines = append(lines, "Knows: "+strings.Join(names, ", ")+".")
		if role := spec.EnemyRole(); role != "caster" && role != "healer" && castsInCombat(spec) {
			notes = append(notes, "casts spells")
		}
	}
	ids := make([]string, 0, len(spec.WindUps))
	for id := range spec.WindUps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ab, ok := windup.Get(id)
		if !ok {
			continue
		}
		how := "now and then"
		switch chance := spec.WindUps[id]; {
		case chance >= 50:
			how = "often"
		case chance < 25:
			how = "rarely"
		}
		when := "the turn after"
		if ab.Rounds > 1 {
			when = fmt.Sprintf("%d turns later", ab.Rounds)
		}
		line := fmt.Sprintf("Winds up %s %s, in plain view; it lands %s for %dx damage", ab.Name, how, when, ab.Multiplier)
		if ab.KnockDown {
			line += " and knocks its target down"
		}
		lines = append(lines, line+". Heavy force breaks it: a landed crit, a stagger, a knockdown or a stun.")
		notes = append(notes, "winds up "+strings.ToLower(ab.Name))
	}
	if rule, noise, ok := spec.Personality(); ok {
		if w, known := targetWords[rule]; known {
			line := "It " + w
			// No figure: a group's tier raises the noise in battle.
			if noise > 0 {
				line += ", though some of its choices are at random"
			}
			lines = append(lines, line+".")
		}
	} else {
		lines = append(lines, "It goes for whoever has the least health left.")
	}
	switch {
	case spec.AttackSkill >= 3:
		lines = append(lines, "Well trained: its blows land more often.")
	case spec.AttackSkill <= -3:
		lines = append(lines, "Poorly trained: it misses often.")
	}
	if spoils := spoilNames(spec); len(spoils) > 0 {
		lines = append(lines, "Carries: "+strings.Join(spoils, ", ")+".")
	}
	if line := trophyLine(spec); line != "" {
		lines = append(lines, line)
	}
	return lines, notes
}

// trophyLine names the trophies a kind of creature may drop (Phase 71): by
// race, with the chance an ordinary kill gives one. A boss always gives one.
func trophyLine(spec *mobs.Mob) string {
	if spec.Character.Zone == "Training" {
		return ""
	}
	kind := loot.Ordinary
	if spec.Boss {
		kind = loot.BossRoll
	}
	var names []string
	options := loot.Trophies(spec.Character.Race())
	for _, t := range options {
		if spec.Boss {
			names = append(names, t.Name)
			continue
		}
		names = append(names, fmt.Sprintf("%s (about %d in 100 kills)", t.Name, loot.TrophyChance(t, kind)/len(options))) // a kill picks one of the kind's trophies, then rolls it
	}
	if len(names) == 0 {
		return ""
	}
	if spec.Boss {
		return "Hunted for a trophy it always yields, one of: " + strings.Join(names, ", ") + " (help enchanting)."
	}
	return "Hunted for trophies: " + strings.Join(names, ", ") + " (help enchanting)."
}

// castsInCombat is whether the template casts in a fight: a spell book
// alone (a healer's cure kept for idle use) is not enough.
func castsInCombat(spec *mobs.Mob) bool {
	for _, cmd := range spec.CombatCommands {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(cmd)), "cast") {
			return true
		}
	}
	return false
}

// TierOf is the tier the given kills of a template earn, by the
// template's own boss flag (an encounter's boss is the same kind rolled
// stronger, like an elite, and shares the template's entry).
func TierOf(mobID, kills int) Tier {
	spec := mobs.GetMobSpec(mobs.MobId(mobID))
	if spec == nil || spec.Practice {
		return Unknown
	}
	return TierFor(kills, spec.Boss)
}

// spoilNames are the items the template wears and carries (what it can
// drop), by name.
func spoilNames(spec *mobs.Mob) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, slot := range characters.AllSlots() {
		it := spec.Character.Equipment.Get(slot)
		if it.ItemId > 0 {
			add(it.NameSimple())
		}
	}
	for i := range spec.Character.Items {
		add(spec.Character.Items[i].NameSimple())
	}
	sort.Strings(out)
	return out
}

// NotesFor are a creature's habit notes for the leader, from the habits
// tier only (a battle caption, a consider line). Nothing is said of a kind
// the leader has not learned that far.
func NotesFor(kills map[int]int, mobID int) []string {
	e, ok := Build(mobs.GetMobSpec(mobs.MobId(mobID)), kills[mobID])
	if !ok || e.Tier < Habits {
		return nil
	}
	return e.Notes
}

// FoeLine says, for the foes a leader can see, what the bestiary holds on
// each distinct kind, and which kinds are new to them. Empty when there is
// nothing to say. Only the foes passed are named: it is the caller's job to
// pass the visible ones.
func FoeLine(kills map[int]int, foes []*mobs.Mob) string {
	seen := map[int]bool{}
	var known, fresh []string
	for _, m := range foes {
		if m == nil || m.Practice || seen[int(m.MobId)] {
			continue
		}
		seen[int(m.MobId)] = true
		e, ok := Build(mobs.GetMobSpec(m.MobId), kills[int(m.MobId)])
		if !ok {
			fresh = append(fresh, m.Character.Name)
			continue
		}
		part := fmt.Sprintf("%s (%s", e.Name, e.Tier.Name())
		if len(e.Notes) > 0 {
			part += ": " + strings.Join(e.Notes, ", ")
		}
		known = append(known, part+")")
	}
	var parts []string
	if len(known) > 0 {
		parts = append(parts, "Known: "+strings.Join(known, "; ")+".")
	}
	if len(fresh) > 0 {
		parts = append(parts, "New to you: "+strings.Join(fresh, ", ")+".")
	}
	return strings.Join(parts, " ")
}

// LearnedLine is the line said when a kill takes a kind to a new tier.
func LearnedLine(name string, tier Tier) string {
	what := map[Tier]string{
		Lore:     "its lore is now in your bestiary",
		Defences: "you have learned its defences",
		Habits:   "you now know its habits and weaknesses",
	}[tier]
	return fmt.Sprintf(`<ansi fg="yellow-bold">Bestiary:</ansi> %s, %s (<ansi fg="command">bestiary %s</ansi>).`, name, what, strings.ToLower(name))
}

// MasteredDeed is the chronicle line for the kill that taught a company a
// kind's habits (Phase 85), built from the creature's template. It is not ok
// for a template that can't be known (none, or a practice foe).
func MasteredDeed(mobID int, place string) (chronicle.Entry, bool) {
	spec := mobs.GetMobSpec(mobs.MobId(mobID))
	if spec == nil || spec.Practice {
		return chronicle.Entry{}, false
	}
	return chronicle.Entry{
		Kind:    chronicle.Mastered,
		Subject: spec.Character.Name,
		Ref:     fmt.Sprintf("mob:%d", mobID),
		Zone:    spec.Zone,
		Place:   place,
	}, true
}

// article is "a" or "an" for the word that follows.
func article(word string) string {
	if word != "" && strings.ContainsRune("aeiou", rune(word[0])) {
		return "An"
	}
	return "A"
}
