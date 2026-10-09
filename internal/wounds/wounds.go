// Package wounds is Ashveil's wounds (Phase 30b): a hit that leaves a
// mark healing can't erase at once. A wound holds back some of a fighter's
// health: while wounded, healing stops at the wound limit, though the true
// maximum is unchanged. Lasting wounds (from critical hits) stay until
// treated or rested away; light wounds (crushing blows, a bleed that runs
// its course) close when the fight ends. The package is pure: it holds the
// rules, the text, and the `heal wounds` planner; characters, combat, and
// modules wire it in.
package wounds

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Kind is what sort of wound it is, which decides what treats it.
type Kind string

const (
	Cut      Kind = "cut"
	Puncture Kind = "puncture"
	Fracture Kind = "fracture"
	Bruise   Kind = "bruise" // light only: a crushing blow
)

// Wound is one wound on a fighter.
type Wound struct {
	Kind   Kind   `yaml:"kind"`
	Place  string `yaml:"place"`           // text only: "arm", "scalp"
	Points int    `yaml:"points"`          // health held back
	Light  bool   `yaml:"light,omitempty"` // closes at fight end
	// Battles is how many battles a lasting puncture has stayed open
	// (Phase 55); three of them fester into a fever.
	Battles int `yaml:"battles,omitempty"`
}

// Festering counts one more battle on every lasting puncture wound in ws
// and reports whether any has now stayed open for limit battles; those
// counters start over. Treating or tending a wound (Close, Treat) also
// starts its count over, so only a wound left untreated festers.
func Festering(ws []Wound, limit int) bool {
	festered := false
	for i := range ws {
		if ws[i].Kind != Puncture || ws[i].Light || ws[i].Points <= 0 {
			continue
		}
		ws[i].Battles++
		if ws[i].Battles >= limit {
			ws[i].Battles = 0
			festered = true
		}
	}
	return festered
}

// Roll picks a whole number in [0, n).
type Roll func(n int) int

// CrushingPct is the share of max health, in percent, a single strike must
// deal to leave a light wound.
const CrushingPct = 25

// places a wound of each kind may be, for its text.
var places = map[Kind][]string{
	Cut:      {"arm", "leg", "side", "shoulder", "scalp", "hand"},
	Puncture: {"arm", "leg", "side", "shoulder", "thigh"},
	Fracture: {"arm", "leg", "ribs", "hand", "collarbone"},
	Bruise:   {"ribs", "shoulder", "side", "back"},
}

func place(k Kind, roll Roll) string {
	list := places[k]
	if len(list) == 0 {
		return "body"
	}
	if roll == nil {
		return list[0]
	}
	return list[roll(len(list))]
}

func half(n int) int { return (n + 1) / 2 }

// KindFor is the wound a weapon subtype leaves.
func KindFor(subtype items.ItemSubType) Kind {
	switch subtype {
	case items.Stabbing, items.Shooting:
		return Puncture
	case items.Bludgeoning:
		return Fracture
	}
	// slashing, cleaving, whipping, claws, and natural (generic) attacks
	return Cut
}

// FromCrit is the lasting wound a critical hit of damage (after armor)
// leaves: half the damage, rounded up, at least 1.
func FromCrit(subtype items.ItemSubType, damage int, roll Roll) Wound {
	k := KindFor(subtype)
	return Wound{Kind: k, Place: place(k, roll), Points: max(1, half(damage))}
}

// Crushing is the light wound a strike that isn't a crit leaves, and
// whether it leaves one: only a strike of at least CrushingPct of max
// health, holding back a quarter of its damage, rounded up.
func Crushing(damage, maxHealth int, roll Roll) (Wound, bool) {
	if damage <= 0 || maxHealth <= 0 || damage*100 < maxHealth*CrushingPct {
		return Wound{}, false
	}
	return Wound{Kind: Bruise, Place: place(Bruise, roll), Points: max(1, (damage+3)/4), Light: true}, true
}

// EasyFightGap is how many levels below its target an attacker must be for
// the fight to count as easy for the target (Phase 35b).
const EasyFightGap = 3

// EasyFight reports whether a blow comes from a foe EasyFightGap or more
// levels below its target.
func EasyFight(attackerLevel, targetLevel int) bool {
	return targetLevel-attackerLevel >= EasyFightGap
}

// EasyCrit is what a crit from a much weaker foe leaves (Phase 35b): no
// wound when the target stays at or above half its max health after the
// blow, else a light wound of a crushing blow's size in place of a lasting
// one. ok is false for no wound.
func EasyCrit(w Wound, damage, healthAfter, maxHealth int) (Wound, bool) {
	if healthAfter*2 >= maxHealth {
		return Wound{}, false
	}
	w.Points = max(1, (damage+3)/4)
	w.Light = true
	return w, true
}

// Bled is the light wound a bleed of stacks leaves when it runs its course.
func Bled(stacks int, roll Roll) Wound {
	return Wound{Kind: Cut, Place: place(Cut, roll), Points: max(1, stacks), Light: true}
}

// Beaten is the lasting wound a defeat leaves (Phase 53): pct percent of
// max health, at least 1, of a kind picked by roll.
func Beaten(maxHealth, pct int, roll Roll) Wound {
	kinds := []Kind{Cut, Puncture, Fracture}
	k := kinds[0]
	if roll != nil {
		k = kinds[max(0, min(roll(len(kinds)), len(kinds)-1))]
	}
	return Wound{Kind: k, Place: place(k, roll), Points: max(1, maxHealth*max(pct, 0)/100)}
}

// Total is the health all ws hold back.
func Total(ws []Wound) int {
	t := 0
	for _, w := range ws {
		t += max(0, w.Points)
	}
	return t
}

// Floor is the lowest the limit can go: a quarter of max health, rounded
// up, and at least 1.
func Floor(maxHealth int) int {
	return max(1, (maxHealth+3)/4)
}

// Limit is how far healing can restore a fighter of maxHealth with ws.
func Limit(maxHealth int, ws []Wound) int {
	if maxHealth < 1 {
		return maxHealth
	}
	return min(maxHealth, max(maxHealth-Total(ws), Floor(maxHealth)))
}

// Lasting is the wounds among ws that outlast a fight.
func Lasting(ws []Wound) []Wound {
	var out []Wound
	for _, w := range ws {
		if !w.Light && w.Points > 0 {
			out = append(out, w)
		}
	}
	return out
}

// HasLight reports whether any of ws is light.
func HasLight(ws []Wound) bool {
	for _, w := range ws {
		if w.Light {
			return true
		}
	}
	return false
}

// CloseLight drops every light wound.
func CloseLight(ws []Wound) []Wound {
	var out []Wound
	for _, w := range ws {
		if !w.Light && w.Points > 0 {
			out = append(out, w)
		}
	}
	return out
}

// worst is the index of the lasting wound holding back the most (the first
// of equals), matching want when it isn't "", or -1.
func worst(ws []Wound, want func(Kind) bool) int {
	best := -1
	for i, w := range ws {
		if w.Light || w.Points <= 0 || (want != nil && !want(w.Kind)) {
			continue
		}
		if best < 0 || w.Points > ws[best].Points {
			best = i
		}
	}
	return best
}

// Close closes points from the worst lasting wound. It returns the wounds
// after, the wound as it was, the points closed, and whether there was one
// to close. A wound closed to 0 is dropped.
func Close(ws []Wound, points int) ([]Wound, Wound, int, bool) {
	return closeWhere(ws, points, nil)
}

func closeWhere(ws []Wound, points int, want func(Kind) bool) ([]Wound, Wound, int, bool) {
	i := worst(ws, want)
	if i < 0 || points <= 0 {
		return ws, Wound{}, 0, false
	}
	out := append([]Wound(nil), ws...)
	was := out[i]
	closed := min(points, was.Points)
	out[i].Points -= closed
	out[i].Battles = 0 // Phase 55: a treated puncture starts its fever count over
	if out[i].Points <= 0 {
		out = append(out[:i], out[i+1:]...)
	}
	return out, was, closed, true
}

// Item is a treatment item.
type Item string

const (
	Bandage Item = "bandage"
	Splint  Item = "splint"
)

// Points each item closes, and what a bandage heals of plain damage.
const (
	BandagePoints = 3
	SplintPoints  = 4
	BandageHeal   = 3
)

// Treats reports whether item treats a wound of kind.
func Treats(item Item, k Kind) bool {
	switch item {
	case Bandage:
		return k == Cut || k == Puncture || k == Bruise
	case Splint:
		return k == Fracture
	}
	return false
}

// Treat uses one item on the worst wound it treats. It returns the wounds
// after, the wound as it was, the points closed, and whether it was used.
func Treat(ws []Wound, item Item) ([]Wound, Wound, int, bool) {
	pts := BandagePoints
	if item == Splint {
		pts = SplintPoints
	}
	return closeWhere(ws, pts, func(k Kind) bool { return Treats(item, k) })
}

// Describe names a wound: "a cut to the arm", "a broken arm".
func Describe(w Wound) string {
	switch w.Kind {
	case Fracture:
		if w.Place == "ribs" {
			return "cracked ribs"
		}
		return "a broken " + w.Place
	case Puncture:
		return "a puncture in the " + w.Place
	case Bruise:
		return "a deep bruise on the " + w.Place
	}
	return "a cut to the " + w.Place
}

// Possessive names a wound on someone: "Tamsin Reed's broken arm",
// "your cut arm".
func Possessive(owner string, w Wound) string {
	adj := map[Kind]string{Cut: "cut", Puncture: "pierced", Fracture: "broken", Bruise: "bruised"}[w.Kind]
	if w.Kind == Fracture && w.Place == "ribs" {
		adj = "cracked"
	}
	return fmt.Sprintf("%s %s %s", owner, adj, w.Place)
}

// --- the heal wounds planner ---

// Patient is a company member who may be treated.
type Patient struct {
	Key         string
	Health, Max int
	Wounds      []Wound
	// Bleeding marks a downed player (Health below 1 but alive), whom
	// after-battle patching heals back to their feet (Phase 35b). A fallen
	// companion is not bleeding: it needs a resurrection.
	Bleeding bool
}

// Limit is the patient's wound limit.
func (p Patient) Limit() int { return Limit(p.Max, p.Wounds) }

// Hurt reports whether anything could help the patient: a lasting wound,
// or health below the limit.
func (p Patient) Hurt() bool {
	return len(Lasting(p.Wounds)) > 0 || p.Health < p.Limit()
}

// Healer is a member who can cast tend or heal.
type Healer struct {
	Key        string
	Mana       int
	Tend, Heal bool
	// HealBonus is added to each heal's dice: the heal spell's flat part
	// at the healer's level (Phase 35b: spellpower.Power.Flat).
	HealBonus int
	// HealPct is the percent the healer's gear adds to each heal (a holy
	// symbol's 5), as heal.js's HealFactor.
	HealPct int
	// Reserve is the mana a healer keeps when patching (Phase 35b: its
	// strategy's mana reserve); Plan ignores it.
	Reserve int
	// CostPct moves the healer's spell costs by a percent, signed (Phase 38b:
	// a class that heals dearly or patches cheaply); 0 pays the rules' costs.
	CostPct int
}

// CostOf is a spell's mana for this healer, never below 1.
func (h Healer) CostOf(base int) int {
	if h.CostPct == 0 || base <= 0 {
		return base
	}
	return max(1, base+(base*h.CostPct+50)/100)
}

// Stock is the treatment items the company can reach.
type Stock struct {
	Bandages, Splints int
}

// Rules is the spells' costs and dice (tend.js, heal.js).
type Rules struct {
	TendCost, HealCost int
	TendDice, HealDice [2]int // quantity, sides
}

// DefaultRules match tend.yaml/tend.js and heal.yaml's power dice; callers
// read the live costs and dice from the spell files (Phase 35b).
var DefaultRules = Rules{TendCost: 4, HealCost: 3, TendDice: [2]int{2, 3}, HealDice: [2]int{2, 4}}

// StepKind is one kind of treatment.
type StepKind string

const (
	StepTend        StepKind = "tend"
	StepHeal        StepKind = "heal"
	StepSplint      StepKind = "splint"
	StepBandage     StepKind = "bandage"      // on a wound
	StepBandageHeal StepKind = "bandage-heal" // on plain damage
)

// Step is one treatment, with the patient's state after it.
type Step struct {
	Kind    StepKind
	Healer  string // "" for an item
	Patient string
	Wound   Wound // the wound treated, as it was
	Closed  int   // wound points closed
	Healed  int   // health restored
	// After the step.
	Health, Limit, Max int
	HealerMana         int
}

// Result is a whole plan: its steps and everyone's state after.
type Result struct {
	Steps    []Step
	Patients []Patient
	Healers  []Healer
	Stock    Stock
	// Used is the items the steps use.
	Used Stock
}

func dice(q [2]int, roll Roll) int {
	t := 0
	for i := 0; i < q[0]; i++ {
		t += roll(q[1]) + 1
	}
	return t
}

func frac(p Patient) float64 {
	l := p.Limit()
	if l <= 0 {
		return 0
	}
	return float64(p.Health) / float64(l)
}

// Order sorts patients most hurt first: most lasting wound points, then
// lowest health against the limit, then by key.
func Order(ps []Patient) {
	sort.SliceStable(ps, func(i, j int) bool {
		wi, wj := Total(Lasting(ps[i].Wounds)), Total(Lasting(ps[j].Wounds))
		if wi != wj {
			return wi > wj
		}
		if fi, fj := frac(ps[i]), frac(ps[j]); fi != fj {
			return fi < fj
		}
		return ps[i].Key < ps[j].Key
	})
}

// Plan treats patients with healers and then items, in the order of the
// design: each patient in turn is tended (each lasting wound) and then
// healed to the limit by the healers, most mana first, until their mana
// runs out; then splints and bandages go on the remaining wounds, and
// bandages on anyone under half their limit. Patients who aren't hurt are
// left alone. Nothing is changed but the returned copies.
func Plan(patients []Patient, healers []Healer, stock Stock, rules Rules, roll Roll) Result {
	ps := make([]Patient, len(patients))
	for i, p := range patients {
		p.Wounds = append([]Wound(nil), p.Wounds...)
		ps[i] = p
	}
	Order(ps)
	hs := append([]Healer(nil), healers...)
	sort.SliceStable(hs, func(i, j int) bool { return hs[i].Mana > hs[j].Mana })

	res := Result{Stock: stock}
	after := func(s Step, p Patient, h *Healer) Step {
		s.Patient = p.Key
		s.Health, s.Limit, s.Max = p.Health, p.Limit(), p.Max
		if h != nil {
			s.Healer = h.Key
			s.HealerMana = h.Mana
		}
		return s
	}
	caster := func(tend bool) *Healer {
		for i := range hs {
			h := &hs[i]
			if tend && h.Tend && h.Mana >= h.CostOf(rules.TendCost) {
				return h
			}
			if !tend && h.Heal && h.Mana >= h.CostOf(rules.HealCost) {
				return h
			}
		}
		return nil
	}

	// Healers, patient by patient.
	for i := range ps {
		p := &ps[i]
		if p.Health < 1 {
			continue
		}
		for len(Lasting(p.Wounds)) > 0 {
			h := caster(true)
			if h == nil {
				break
			}
			h.Mana -= h.CostOf(rules.TendCost)
			ws, was, closed, ok := Close(p.Wounds, dice(rules.TendDice, roll))
			if !ok {
				break
			}
			p.Wounds = ws
			res.Steps = append(res.Steps, after(Step{Kind: StepTend, Wound: was, Closed: closed}, *p, h))
		}
		for p.Health < p.Limit() {
			h := caster(false)
			if h == nil {
				break
			}
			h.Mana -= h.CostOf(rules.HealCost)
			amt := min((dice(rules.HealDice, roll)+h.HealBonus)*(100+max(h.HealPct, 0))/100, p.Limit()-p.Health)
			p.Health += amt
			res.Steps = append(res.Steps, after(Step{Kind: StepHeal, Healed: amt}, *p, h))
		}
	}

	// Items, on what the healers left.
	for i := range ps {
		p := &ps[i]
		if p.Health < 1 {
			continue
		}
		for _, item := range []Item{Splint, Bandage} {
			for {
				left := &res.Stock.Bandages
				used := &res.Used.Bandages
				kind := StepBandage
				if item == Splint {
					left, used, kind = &res.Stock.Splints, &res.Used.Splints, StepSplint
				}
				if *left < 1 {
					break
				}
				ws, was, closed, ok := Treat(p.Wounds, item)
				if !ok {
					break
				}
				*left--
				*used++
				p.Wounds = ws
				res.Steps = append(res.Steps, after(Step{Kind: kind, Wound: was, Closed: closed}, *p, nil))
			}
		}
		for res.Stock.Bandages > 0 && p.Health*2 < p.Limit() {
			amt := min(BandageHeal, p.Limit()-p.Health)
			res.Stock.Bandages--
			res.Used.Bandages++
			p.Health += amt
			res.Steps = append(res.Steps, after(Step{Kind: StepBandageHeal, Healed: amt}, *p, nil))
		}
	}

	res.Patients = ps
	res.Healers = hs
	return res
}

// HealTarget is the health a patch heals a patient to (Phase 35b): the
// healing threshold's percent of the wound limit, rounded up.
func HealTarget(p Patient, healBelow int) int {
	healBelow = min(max(healBelow, 0), 100)
	return (p.Limit()*healBelow + 99) / 100
}

// Patch is after-battle patching up (Phase 35b): heal spells only, no tend
// and no items. Each patient below the healing threshold (healBelow, a
// percent of their wound limit) is healed, most hurt first, until they
// reach it; each healer, most mana first, stops when a heal would take it
// below its Reserve. Nothing is changed but the returned copies.
func Patch(patients []Patient, healers []Healer, rules Rules, healBelow int, roll Roll) Result {
	ps := make([]Patient, len(patients))
	for i, p := range patients {
		p.Wounds = append([]Wound(nil), p.Wounds...)
		ps[i] = p
	}
	Order(ps)
	hs := append([]Healer(nil), healers...)
	sort.SliceStable(hs, func(i, j int) bool { return hs[i].Mana-hs[i].Reserve > hs[j].Mana-hs[j].Reserve })

	res := Result{}
	caster := func() *Healer {
		for i := range hs {
			if h := &hs[i]; h.Heal && h.Mana-h.CostOf(rules.HealCost) >= max(h.Reserve, 0) {
				return h
			}
		}
		return nil
	}
	for i := range ps {
		p := &ps[i]
		if p.Health < 1 && !p.Bleeding {
			continue
		}
		target := HealTarget(*p, healBelow)
		for p.Health < target {
			h := caster()
			if h == nil {
				break
			}
			h.Mana -= h.CostOf(rules.HealCost)
			amt := min((dice(rules.HealDice, roll)+h.HealBonus)*(100+max(h.HealPct, 0))/100, p.Limit()-p.Health)
			p.Health += amt
			res.Steps = append(res.Steps, Step{Kind: StepHeal, Healer: h.Key, Patient: p.Key, Healed: amt, Health: p.Health, Limit: p.Limit(), Max: p.Max, HealerMana: h.Mana})
		}
	}
	res.Patients = ps
	res.Healers = hs
	return res
}
