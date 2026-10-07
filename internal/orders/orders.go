// Package orders holds Ashveil's battle orders (Phase 61): up to three
// "when this, do that" rules a company member carries into a battle, set
// beforehand like its strategy. Each round, before a member's role and
// target rule, its orders are read in order; the first whose condition holds
// and whose action can be carried out fires.
//
// This package is GoMud-free: conditions are read from a plain Snapshot, and
// internal/hooks carries the chosen action out. The durable lists live in
// modules/strategy behind the Provider seam.
package orders

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MaxOrders is the most orders a member carries.
const MaxOrders = 3

// When is what an order waits for.
type When string

const (
	AllyHurt   When = "ally"     // an ally is below a share of its health
	SelfHurt   When = "self"     // the member is below a share of its health
	Chanting   When = "chanting" // a foe is chanting a spell
	Boss       When = "boss"     // a boss stands among the foes
	FirstRound When = "first"    // the battle's first round
	FoeKind    When = "foe"      // a foe of a kind stands
)

// Whens is every condition, in the order they are listed.
var Whens = []When{AllyHurt, SelfHurt, Chanting, Boss, FirstRound, FoeKind}

// Do is what an order does.
type Do string

const (
	Heal      Do = "heal"      // heal the ally (or the member) the condition named
	Break     Do = "break"     // turn on the foe the condition named (a chanter: break its chant)
	Guard     Do = "guard"     // guard the ally the condition named this round
	Strongest Do = "strongest" // cast its heaviest attack, ignoring its mana reserve
	Hold      Do = "hold"      // cast no attack spell this round, keeping its mana
)

// Dos is every action, in the order they are listed.
var Dos = []Do{Heal, Break, Guard, Strongest, Hold}

// Kinds are the foe kinds a FoeKind condition reads.
var Kinds = []string{"caster", "healer"}

// Pct bounds: a health share is a multiple of 5 from 10 to 90.
const (
	MinPct = 10
	MaxPct = 90
)

// Order is one rule. Pct is the health share of AllyHurt and SelfHurt; Kind
// the foe kind of FoeKind; each is blank otherwise.
type Order struct {
	When When   `yaml:"when"`
	Pct  int    `yaml:"pct,omitempty"`
	Kind string `yaml:"kind,omitempty"`
	Do   Do     `yaml:"do"`
}

var whenAliases = map[string]When{
	"ally": AllyHurt, "allies": AllyHurt, "friend": AllyHurt, "hurt": AllyHurt,
	"self": SelfHurt, "me": SelfHurt, "i": SelfHurt,
	"chanting": Chanting, "chant": Chanting, "casting": Chanting,
	"boss": Boss, "first": FirstRound, "opening": FirstRound, "start": FirstRound,
	"foe": FoeKind, "enemy": FoeKind,
}

var doAliases = map[string]Do{
	"heal": Heal, "mend": Heal, "break": Break, "interrupt": Break, "strike": Break,
	"guard": Guard, "protect": Guard,
	"strongest": Strongest, "strong": Strongest, "best": Strongest,
	"hold": Hold, "save": Hold,
}

// Validate checks an order's fields and that its action makes sense with
// its condition.
func (o Order) Validate() error {
	switch o.When {
	case AllyHurt, SelfHurt:
		if o.Pct < MinPct || o.Pct > MaxPct || o.Pct%5 != 0 || o.Kind != "" {
			return fmt.Errorf("a health share is a multiple of 5 from %d to %d", MinPct, MaxPct)
		}
	case FoeKind:
		ok := false
		for _, k := range Kinds {
			ok = ok || k == o.Kind
		}
		if !ok || o.Pct != 0 {
			return fmt.Errorf("a foe kind is one of: %s", strings.Join(Kinds, ", "))
		}
	case Chanting, Boss, FirstRound:
		if o.Pct != 0 || o.Kind != "" {
			return errors.New("that condition takes no number or kind")
		}
	default:
		return fmt.Errorf("%q is not a condition", o.When)
	}
	switch o.Do {
	case Heal:
		if o.When != AllyHurt && o.When != SelfHurt {
			return errors.New("heal needs an ally or self condition: there must be someone to heal")
		}
	case Guard:
		if o.When != AllyHurt {
			return errors.New("guard needs an ally condition: there must be someone to guard")
		}
	case Break:
		if o.When != Chanting && o.When != Boss && o.When != FoeKind {
			return errors.New("break needs a foe condition (chanting, boss or foe): there must be a foe to turn on")
		}
	case Strongest, Hold:
	default:
		return fmt.Errorf("%q is not an action", o.Do)
	}
	return nil
}

// Parse reads "<condition> then <action>": for example "ally 50 then heal",
// "self 40 then heal", "chanting then break", "boss then strongest",
// "foe healer then break", "first then hold".
func Parse(words []string) (Order, error) {
	split := -1
	for i, w := range words {
		if w == "then" || w == "do" || w == "->" {
			split = i
		}
	}
	if split < 1 || split >= len(words)-1 {
		return Order{}, errors.New(`say it as "<condition> then <action>", for example "ally 50 then heal"`)
	}
	cond, act := words[:split], words[split+1:]
	when, ok := whenAliases[strings.ToLower(cond[0])]
	if !ok {
		return Order{}, fmt.Errorf("%q is not a condition (%s)", cond[0], listWhens())
	}
	o := Order{When: when}
	rest := cond[1:]
	switch when {
	case AllyHurt, SelfHurt:
		if len(rest) > 0 && strings.EqualFold(rest[0], "below") {
			rest = rest[1:]
		}
		if len(rest) != 1 {
			return Order{}, fmt.Errorf("%s needs a health share, like %q", when, string(when)+" 50")
		}
		n, err := strconv.Atoi(strings.TrimSuffix(rest[0], "%"))
		if err != nil {
			return Order{}, fmt.Errorf("%q is not a health share", rest[0])
		}
		o.Pct = n
	case FoeKind:
		if len(rest) != 1 {
			return Order{}, fmt.Errorf("foe needs a kind: %s", strings.Join(Kinds, ", "))
		}
		o.Kind = strings.ToLower(rest[0])
	default:
		if len(rest) != 0 {
			return Order{}, fmt.Errorf("%s takes nothing after it", when)
		}
	}
	if len(act) != 1 {
		return Order{}, errors.New("an order has one action")
	}
	do, ok := doAliases[strings.ToLower(act[0])]
	if !ok {
		return Order{}, fmt.Errorf("%q is not an action (%s)", act[0], listDos())
	}
	o.Do = do
	return o, o.Validate()
}

func listWhens() string {
	return "ally <n>, self <n>, chanting, boss, first, foe <kind>"
}

func listDos() string {
	names := make([]string, len(Dos))
	for i, d := range Dos {
		names[i] = string(d)
	}
	return strings.Join(names, ", ")
}

// Command is the order as it is typed: "ally 50 then heal".
func (o Order) Command() string {
	var cond string
	switch o.When {
	case AllyHurt, SelfHurt:
		cond = fmt.Sprintf("%s %d", o.When, o.Pct)
	case FoeKind:
		cond = "foe " + o.Kind
	default:
		cond = string(o.When)
	}
	return cond + " then " + string(o.Do)
}

// Condition is the condition in words.
func (o Order) Condition() string {
	switch o.When {
	case AllyHurt:
		return fmt.Sprintf("an ally is below %d%% health", o.Pct)
	case SelfHurt:
		return fmt.Sprintf("it is below %d%% health", o.Pct)
	case Chanting:
		return "a foe is chanting"
	case Boss:
		return "a boss stands among the foes"
	case FirstRound:
		return "the battle opens"
	case FoeKind:
		return "a " + o.Kind + " stands among the foes"
	}
	return string(o.When)
}

// Action is the action in words.
func (o Order) Action() string {
	switch o.Do {
	case Heal:
		if o.When == SelfHurt {
			return "heal itself"
		}
		return "heal that ally first"
	case Break:
		switch o.When {
		case Chanting:
			return "turn on the chanter to break its chant"
		case Boss:
			return "turn on the boss"
		}
		return "turn on that foe"
	case Guard:
		return "guard that ally"
	case Strongest:
		return "cast its strongest attack, whatever its reserve"
	case Hold:
		return "hold its mana: no attack spell"
	}
	return string(o.Do)
}

// Describe is the whole order in words: "When an ally is below 50% health,
// heal that ally first."
func (o Order) Describe() string {
	return "When " + o.Condition() + ", " + o.Action() + "."
}

// Ally is a member of the side as an order reads it.
type Ally struct {
	HP, MaxHP int
	// Down is a player at or below 0 health but not dead: still healed.
	Down bool
	// Pending is an ally a heal already covers this round.
	Pending bool
	// Self is the member whose orders are read.
	Self bool
}

// FoeInfo is a foe as an order reads it.
type FoeInfo struct {
	ID                       int
	Chanting, Caster, Healer bool
	Boss                     bool
}

// Snapshot is what a member's orders are read against.
type Snapshot struct {
	// FirstRound is true in the battle's opening round.
	FirstRound bool
	Allies     []Ally
	Foes       []FoeInfo
}

// Fire is an order that fires: which one, the ally (an index into
// Snapshot.Allies, -1 for none) and the foe (an id, 0 for none) its
// condition named.
type Fire struct {
	Index int
	Order Order
	Ally  int
	Foe   int
}

// Evaluate reads the orders in order and returns the first whose condition
// holds and that usable accepts (usable says whether the action can be
// carried out now: a heal needs a heal spell, a break a foe in reach). ok is
// false when none fires.
func Evaluate(list []Order, s Snapshot, usable func(Fire) bool) (Fire, bool) {
	for i, o := range list {
		f, held := condition(o, i, s)
		if !held {
			continue
		}
		if usable == nil || usable(f) {
			return f, true
		}
	}
	return Fire{}, false
}

func share(a Ally) int {
	max := a.MaxHP
	if max < 1 {
		max = 1
	}
	return a.HP * 1000 / max
}

// condition is whether the order's condition holds, and what it names.
func condition(o Order, index int, s Snapshot) (Fire, bool) {
	f := Fire{Index: index, Order: o, Ally: -1}
	switch o.When {
	case AllyHurt:
		for i, a := range s.Allies {
			if a.Self || (a.HP < 1 && !a.Down) || (o.Do == Heal && a.Pending) || a.HP*100 >= o.Pct*max(a.MaxHP, 1) {
				continue
			}
			if f.Ally < 0 || share(a) < share(s.Allies[f.Ally]) {
				f.Ally = i
			}
		}
		return f, f.Ally >= 0
	case SelfHurt:
		for i, a := range s.Allies {
			if !a.Self {
				continue
			}
			if (a.HP < 1 && !a.Down) || (o.Do == Heal && a.Pending) || a.HP*100 >= o.Pct*max(a.MaxHP, 1) {
				return f, false
			}
			f.Ally = i
			return f, true
		}
		return f, false
	case Chanting:
		for _, foe := range s.Foes {
			if foe.Chanting {
				f.Foe = foe.ID
				return f, true
			}
		}
	case Boss:
		for _, foe := range s.Foes {
			if foe.Boss {
				f.Foe = foe.ID
				return f, true
			}
		}
	case FirstRound:
		return f, s.FirstRound
	case FoeKind:
		for _, foe := range s.Foes {
			if (o.Kind == "caster" && foe.Caster) || (o.Kind == "healer" && foe.Healer) {
				f.Foe = foe.ID
				return f, true
			}
		}
	}
	return f, false
}
