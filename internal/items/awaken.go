package items

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/classes"
)

// Phase 67: relic awakenings. A relic carries up to MaxAwakenings deeds in
// its `relic.awakenings` list. Each is a condition (slay foes of a kind,
// defeat a lair's master, reach a zone) met while the relic is worn, and a
// power that joins the relic's effects the moment it is met. The spec is
// data; the progress is a plain slice on the item instance (one count per
// awakening, replaced whole so item copies never alias it).

// MaxAwakenings is the most awakenings one relic may carry.
const MaxAwakenings = 3

// Awakening condition kinds.
const (
	AwakenSlay  = "slay"  // slay Count foes whose race is in Races
	AwakenLair  = "lair"  // defeat the lair master Mob, Count times
	AwakenPlace = "place" // reach the zone Zone, Count times
)

// AwakeningSpec is one awakening of a relic.
type AwakeningSpec struct {
	Name    string         `yaml:"name"`
	Kind    string         `yaml:"kind"`
	Races   []string       `yaml:"races,omitempty"` // slay: race names (lowercase)
	Mob     int            `yaml:"mob,omitempty"`   // lair: the boss's template id
	Zone    string         `yaml:"zone,omitempty"`  // place: the zone entered
	Count   int            `yaml:"count,omitempty"` // how many (1 when omitted for lair and place)
	Target  string         `yaml:"target"`          // the condition in words: "ogres", "the forest ogre", "Hollowweb Deep"
	Effects map[string]int `yaml:"effects"`         // gear effects the awakened relic gains
}

// Need is the count the awakening asks for.
func (a AwakeningSpec) Need() int {
	if a.Count < 1 {
		return 1
	}
	return a.Count
}

// Condition says what the awakening asks, in words.
func (a AwakeningSpec) Condition() string {
	switch a.Kind {
	case AwakenSlay:
		return fmt.Sprintf("slay %d %s while it is worn", a.Need(), a.Target)
	case AwakenLair:
		if a.Need() > 1 {
			return fmt.Sprintf("defeat %s %d times while it is worn", a.Target, a.Need())
		}
		return fmt.Sprintf("defeat %s while it is worn", a.Target)
	default:
		if a.Need() > 1 {
			return fmt.Sprintf("reach %s %d times while it is worn", a.Target, a.Need())
		}
		return fmt.Sprintf("reach %s while it is worn", a.Target)
	}
}

// Matches reports whether a deed of the kind and subject advances the
// awakening: a slain foe's race, a lair master's template id as text, or a
// zone name.
func (a AwakeningSpec) Matches(kind, subject string) bool {
	if a.Kind != kind {
		return false
	}
	switch kind {
	case AwakenSlay:
		for _, r := range a.Races {
			if strings.EqualFold(r, subject) {
				return true
			}
		}
	case AwakenLair:
		return fmt.Sprint(a.Mob) == subject
	case AwakenPlace:
		return strings.EqualFold(a.Zone, subject)
	}
	return false
}

// maxAwakenedEffect is the most one awakening may add to a gear effect: a
// third of the effect's cap, at least 1. One-shot effects (cap 1) cannot be
// awakened.
func maxAwakenedEffect(e classes.GearEffect) int {
	if e.Max <= 1 {
		return 0
	}
	return max(1, e.Max/3)
}

// validateAwakenings holds a relic's awakenings to the gear caps: each
// awakening adds at most a third of an effect's cap, and the relic's base
// effects plus all its awakenings stay within the cap, so awakening is
// counted in the relic balance of Phase 36d.
func validateAwakenings(r RelicSpec) error {
	if len(r.Awakenings) == 0 {
		return nil
	}
	if len(r.Awakenings) > MaxAwakenings {
		return fmt.Errorf("a relic has at most %d awakenings", MaxAwakenings)
	}
	total := map[string]int{}
	for k, v := range r.Effects {
		total[k] += v
	}
	names := map[string]bool{}
	for _, a := range r.Awakenings {
		if strings.TrimSpace(a.Name) == "" || strings.TrimSpace(a.Target) == "" {
			return fmt.Errorf("an awakening needs a name and a target in words")
		}
		if names[a.Name] {
			return fmt.Errorf("awakening name %q repeats", a.Name)
		}
		names[a.Name] = true
		switch a.Kind {
		case AwakenSlay:
			if len(a.Races) == 0 || a.Count < 1 {
				return fmt.Errorf("awakening %q: slay needs races and a count", a.Name)
			}
			for _, race := range a.Races {
				if race != strings.ToLower(race) {
					return fmt.Errorf("awakening %q: race %q must be lowercase", a.Name, race)
				}
			}
		case AwakenLair:
			if a.Mob < 1 {
				return fmt.Errorf("awakening %q: lair needs a boss (mob)", a.Name)
			}
		case AwakenPlace:
			if strings.TrimSpace(a.Zone) == "" {
				return fmt.Errorf("awakening %q: place needs a zone", a.Name)
			}
		default:
			return fmt.Errorf("awakening %q: kind %q is not slay, lair or place", a.Name, a.Kind)
		}
		if a.Count < 0 || a.Count > 100 {
			return fmt.Errorf("awakening %q: count %d is outside 0 to 100", a.Name, a.Count)
		}
		if len(a.Effects) == 0 {
			return fmt.Errorf("awakening %q grants nothing", a.Name)
		}
		if err := classes.ValidateGearEffects(a.Effects); err != nil {
			return fmt.Errorf("awakening %q: %w", a.Name, err)
		}
		for k, v := range a.Effects {
			e, _ := classes.GearEffectFor(k)
			if limit := maxAwakenedEffect(e); v > limit {
				return fmt.Errorf("awakening %q: %q may add at most %d, not %d", a.Name, k, limit, v)
			}
			total[k] += v
			if total[k] > e.Max {
				return fmt.Errorf("awakening %q: the relic's %q would reach %d, over the cap of %d", a.Name, k, total[k], e.Max)
			}
		}
	}
	return nil
}

// HasAwakenings reports whether an item is a relic with awakenings.
func (i *Item) HasAwakenings() bool {
	spec := i.GetSpec()
	return spec.Relic != nil && len(spec.Relic.Awakenings) > 0
}

// AwakeningProgress is the count toward awakening idx.
func (i *Item) AwakeningProgress(idx int) int {
	if idx < 0 || idx >= len(i.Awaken) {
		return 0
	}
	return i.Awaken[idx]
}

// Awakened reports whether awakening idx has been met.
func (i *Item) Awakened(idx int) bool {
	spec := i.GetSpec()
	if spec.Relic == nil || idx < 0 || idx >= len(spec.Relic.Awakenings) {
		return false
	}
	return i.AwakeningProgress(idx) >= spec.Relic.Awakenings[idx].Need()
}

// AwakenedMask is a bit per awakening met: it names the relic's state in
// the wearer's cached gear effects.
func (i *Item) AwakenedMask() int {
	mask := 0
	for idx := 0; idx < MaxAwakenings; idx++ {
		if i.Awakened(idx) {
			mask |= 1 << idx
		}
	}
	return mask
}

// AwakenedPowers lists the awakenings that have woken, in spec order.
func (i *Item) AwakenedPowers() []AwakeningSpec {
	spec := i.GetSpec()
	if spec.Relic == nil {
		return nil
	}
	var out []AwakeningSpec
	for idx, a := range spec.Relic.Awakenings {
		if i.Awakened(idx) {
			out = append(out, a)
		}
	}
	return out
}

// AwakenedEffects is what the relic's woken awakenings add, by effect key.
func (i *Item) AwakenedEffects() map[string]int {
	var fx map[string]int
	for _, a := range i.AwakenedPowers() {
		for k, v := range a.Effects {
			if fx == nil {
				fx = map[string]int{}
			}
			fx[k] += v
		}
	}
	return fx
}

// AdvanceAwakening adds n to awakening idx, capped at what it needs, and
// reports whether this advance woke it. The progress slice is replaced, not
// edited, so copies of the item keep theirs.
func (i *Item) AdvanceAwakening(idx, n int) bool {
	spec := i.GetSpec()
	if spec.Relic == nil || idx < 0 || idx >= len(spec.Relic.Awakenings) || n < 1 || i.Awakened(idx) {
		return false
	}
	progress := make([]int, len(spec.Relic.Awakenings))
	copy(progress, i.Awaken)
	need := spec.Relic.Awakenings[idx].Need()
	progress[idx] = min(need, progress[idx]+n)
	i.Awaken = progress
	return progress[idx] >= need
}

// AwakeningLines shows each awakening: its progress while it sleeps, and
// its power once it has woken. Empty for an item with none.
func (i *Item) AwakeningLines() []string {
	spec := i.GetSpec()
	if spec.Relic == nil || len(spec.Relic.Awakenings) == 0 {
		return nil
	}
	lines := make([]string, 0, len(spec.Relic.Awakenings))
	for idx, a := range spec.Relic.Awakenings {
		power := strings.Join(classes.DescribeGearEffects(a.Effects), "; ")
		if i.Awakened(idx) {
			lines = append(lines, fmt.Sprintf("Awakened, %s: %s.", a.Name, power))
			continue
		}
		lines = append(lines, fmt.Sprintf("Sleeping, %s (%d of %d): %s, and it wakes with %s.", a.Name, i.AwakeningProgress(idx), a.Need(), a.Condition(), power))
	}
	return lines
}

// AwakeningSummary is a short tag for a relic in a list: "1 of 3 awakened",
// or "" for an item with none.
func (i *Item) AwakeningSummary() string {
	spec := i.GetSpec()
	if spec.Relic == nil || len(spec.Relic.Awakenings) == 0 {
		return ""
	}
	return fmt.Sprintf("%d of %d awakened", len(i.AwakenedPowers()), len(spec.Relic.Awakenings))
}

// AwakeningStatus is one line of a relic's progress for a list: how many
// awakenings have woken and the first one still asleep with its count. Empty
// for an item with none.
func (i *Item) AwakeningStatus() string {
	spec := i.GetSpec()
	if spec.Relic == nil || len(spec.Relic.Awakenings) == 0 {
		return ""
	}
	line := i.AwakeningSummary()
	for idx, a := range spec.Relic.Awakenings {
		if !i.Awakened(idx) {
			return fmt.Sprintf("%s; next: %s (%d of %d)", line, a.Name, i.AwakeningProgress(idx), a.Need())
		}
	}
	return line
}
