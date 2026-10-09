package camping

import (
	"sort"
	"strings"
)

// Phase 51 rest duties. A member's duty for a camp rest is sleep (the
// default) or one of the work duties below. Assignments are kept on the
// camp (Camp.Duties) until changed, locked on the rest when it starts
// (RestSession.Duties) and settled when it ends, so a restart never
// re-rolls them. A member with no entry sleeps.

// Duty is what one member does during a camp rest.
type Duty string

const (
	DutySleep  Duty = "sleep"
	DutyWatch  Duty = "watch"
	DutyTend   Duty = "tend"
	DutyForage Duty = "forage"
	DutyCook   Duty = "cook"
	DutyBrew   Duty = "brew"
)

// DutyInfo describes a duty for the command and the web picker.
type DutyInfo struct {
	Duty Duty
	// Does is the one-line effect shown to players.
	Does string
}

// AllDuties is every duty in display order; the first is the default.
var AllDuties = []DutyInfo{
	{DutySleep, "sleeps: full rest, and the Rested buff"},
	{DutyWatch, "keeps watch: adds to the chance of spotting raiders, and ends the rest no better than Ready"},
	{DutyTend, "tends the company: one lasting wound with the surgeon's kit, or one member's blades with a whetstone"},
	{DutyForage, "forages for food (once per forage cooldown)"},
	{DutyCook, "cooks one dish from the pack and cargo"},
	{DutyBrew, "brews flasks for the company's Alchemists first"},
}

// ParseDuty reads a duty word; "rest" and "none" mean sleep.
func ParseDuty(word string) (Duty, bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	switch word {
	case "rest", "none", "off":
		return DutySleep, true
	}
	for _, info := range AllDuties {
		if string(info.Duty) == word {
			return info.Duty, true
		}
	}
	return DutySleep, false
}

// WatchFatigueCap is the highest fatigue a member who kept watch can end a
// rest on: the top of the Ready band (survival.BandFor), never Rested.
const WatchFatigueCap = 75

// DutyOf is a member's duty in an assignment map; a missing or unknown
// entry sleeps.
func DutyOf(duties map[string]string, member string) Duty {
	d, ok := ParseDuty(duties[member])
	if !ok {
		return DutySleep
	}
	return d
}

// WithDuty is a copy of duties with member given duty. Sleep removes the
// entry, so the map holds only work. The map is replaced whole, never
// edited in place.
func WithDuty(duties map[string]string, member string, duty Duty) map[string]string {
	out := make(map[string]string, len(duties)+1)
	for k, v := range duties {
		out[k] = v
	}
	if duty == DutySleep {
		delete(out, member)
	} else {
		out[member] = string(duty)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DutyMembers lists the members holding duty, sorted by key.
func DutyMembers(duties map[string]string, duty Duty) []string {
	var out []string
	for member, raw := range duties {
		if d, ok := ParseDuty(raw); ok && d == duty && d != DutySleep {
			out = append(out, member)
		}
	}
	sort.Strings(out)
	return out
}

// OnDuty reports whether member works instead of sleeping.
func OnDuty(duties map[string]string, member string) bool {
	return DutyOf(duties, member) != DutySleep
}

// LockDuties copies the assignments of the members present (a key set) for
// a rest: someone away from the camp when it begins sleeps.
func LockDuties(duties map[string]string, present map[string]bool) map[string]string {
	var out map[string]string
	for member := range duties {
		if present[member] && OnDuty(duties, member) {
			out = WithDuty(out, member, DutyOf(duties, member))
		}
	}
	return out
}

// DutyOf is a member's duty on this rest (sleep with none).
func (s RestSession) DutyOf(member string) Duty { return DutyOf(s.Duties, member) }
