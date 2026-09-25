// Package companyview is Phase 26a's read model: a read-only summary of a
// player and their company, assembled on the game loop from the providers
// that own each value, plus the labels every surface (text commands, the
// prompt, and in 26b the browser) renders them with. It stores nothing but
// the prompt cache. See
// docs/superpowers/specs/2026-09-24-phase-26a-company-summary-text-design.md.
package companyview

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/survival"
)

// maxWarnWords caps the prompt's warning cluster.
const maxWarnWords = 4

// Need is one survival need. Known is false when no provider reported it.
type Need struct {
	Known bool
	Value int
	Label string
}

// Warns reports whether the need is Low (Hungry, Thirsty, Tired) or worse.
func (n Need) Warns() bool {
	return n.Known && survival.BandFor(n.Value) <= survival.BandLow
}

var (
	coldLabels = map[climate.Band]string{climate.BandMild: "Chilled", climate.BandModerate: "Frostbitten", climate.BandSevere: "Hypothermic", climate.BandCritical: "Freezing to Death"}
	heatLabels = map[climate.Band]string{climate.BandMild: "Overheated", climate.BandModerate: "Heatstricken", climate.BandSevere: "Heat Exhausted", climate.BandCritical: "Heatstroke"}
)

// WarmthLabel names a signed exposure's band (negative is cold), matching
// the exposure module's band buffs; "" when comfortable.
func WarmthLabel(exposure int) string {
	band := climate.BandFor(exposure)
	if exposure < 0 {
		return coldLabels[band]
	}
	return heatLabels[band]
}

// LightLabel names a viewer's visibility (0 dark, 1 dim, 2 lit); "" when
// they can see.
func LightLabel(visibility int) string {
	switch {
	case visibility <= 0:
		return "Dark"
	case visibility == 1:
		return "Dim"
	}
	return ""
}

// LoadLabel names a company load. The configured band table has no names,
// so the label follows its effect: a band that slows the company is
// Burdened, and a load at or over capacity is Overloaded.
func LoadLabel(load encumbrance.Load, band encumbrance.LoadBand) string {
	total := load.TotalGrams()
	switch {
	case total > 0 && total >= load.CapacityGrams:
		return "Overloaded"
	case band.TravelDurationPct > 100 || band.FatiguePct > 100:
		return "Burdened"
	}
	return "Unburdened"
}

// ActivityKind is what a company is doing.
type ActivityKind int

const (
	Idle ActivityKind = iota
	Travelling
	// Stopped is a journey interrupted on the road.
	Stopped
	// Camped is a pitched camp, not resting.
	Camped
	CampRest
	InnStay
)

// Activity is what the company is doing, with its progress or time left.
type Activity struct {
	Kind      ActivityKind
	Percent   int
	Remaining time.Duration
	// Route is the journey's route name, when travelling.
	Route string
}

// Label is the activity in a few words; "" when idle.
func (a Activity) Label() string {
	switch a.Kind {
	case Travelling:
		return fmt.Sprintf("Travelling %d%%", a.Percent)
	case Stopped:
		return "Stopped"
	case Camped:
		return "Camped"
	case CampRest:
		return "Resting " + FormatRemaining(a.Remaining)
	case InnStay:
		return "At inn " + FormatRemaining(a.Remaining)
	}
	return ""
}

// FormatRemaining renders a duration as "1h 5m", "12m", or "45s".
func FormatRemaining(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", max(int(d/time.Second), 0))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%dh %dm", int(d/time.Hour), int(d%time.Hour/time.Minute))
}

// CompanyCount is "3" alive, or "3, 1 dead".
func CompanyCount(alive, dead int) string {
	if dead == 0 {
		return fmt.Sprint(alive)
	}
	return fmt.Sprintf("%d, %d dead", alive, dead)
}

// WarnCluster joins warning words for the prompt, skipping empty ones, at
// most maxWarnWords, each after a space; "" when there are none.
func WarnCluster(words []string) string {
	var b strings.Builder
	n := 0
	for _, w := range words {
		if w == "" || n == maxWarnWords {
			continue
		}
		b.WriteString(" " + w)
		n++
	}
	return b.String()
}
