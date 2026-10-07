// Package rites is Phase 74's rules: when a companion's loss is worth a
// funeral, who it touches, and what holding or skipping the rites does. It
// is GoMud-free and pure; modules/company keeps the saved occasions
// (`Record.Rites`) and applies the outcome, and modules/camping offers them
// when a camp or an inn stay begins. The rules name no god and no creed:
// phase 73's faiths will add words to a rite, not change what it does.
package rites

// Cause is why the company is mourning.
type Cause string

const (
	// Lost is a companion who fell and was not raised in time: gone for good.
	Lost Cause = "lost"
	// Left is a companion who lost faith in the company and walked away
	// after long service.
	Left Cause = "left"
)

// Causes is every cause, in the order help lists them.
var Causes = []Cause{Lost, Left}

// LongServiceRounds is how much service a companion who leaves needs before
// the company mourns them: the Trusted chemistry tier (3 game days of
// rounds served with the band). A companion who falls and is lost is
// mourned whatever their service.
const LongServiceRounds = 2700

// Mourns reports whether a loss is worth rites: any companion lost for
// good, and one who left after long service.
func Mourns(c Cause, serviceRounds int) bool {
	switch c {
	case Lost:
		return true
	case Left:
		return serviceRounds >= LongServiceRounds
	}
	return false
}

// Who is touched. Close are the companions who trusted the one gone (a
// bond at "trusts" or better); the Band is everyone else walking with the
// company. A rite names both when it is queued, since the bond ends with the
// companion.
const (
	CloseHold = 3  // loyalty a held rite gives each close companion
	BandHold  = 1  // ... and each of the rest
	CloseSkip = 5  // loyalty skipping takes from each close companion
	BandSkip  = 2  // ... and from each of the rest
	Floor     = 30 // skipping never takes loyalty below this
	Ceiling   = 80 // holding never lifts loyalty above this
	// BondGain is what a held rite does for each pair of close companions.
	BondGain = 2
)

// Steady is a companion's loyalty after a held rite touches them.
func Steady(loyalty int, close bool) int {
	gain := BandHold
	if close {
		gain = CloseHold
	}
	if loyalty >= Ceiling {
		return loyalty
	}
	return min(Ceiling, loyalty+gain)
}

// Grieve is a companion's loyalty after a skipped rite touches them.
func Grieve(loyalty int, close bool) int {
	loss := BandSkip
	if close {
		loss = CloseSkip
	}
	if loyalty <= Floor {
		return loyalty
	}
	return max(Floor, loyalty-loss)
}

// Phrase is the cause in words: "lost for good", "left the company".
func Phrase(c Cause) string {
	switch c {
	case Lost:
		return "lost for good"
	case Left:
		return "left the company"
	}
	return "gone"
}

// Verb is the departed in a sentence's middle: "fell and was not raised",
// "walked away".
func Verb(c Cause) string {
	switch c {
	case Lost:
		return "fell and was not raised in time"
	case Left:
		return "walked away after long service"
	}
	return "is gone"
}
