package items

import "time"

// Phase 43b weapon poisons. The catalogue lives here, beside the coating
// fields on Item, so the combat, status and camping packages share one
// source. Effects are combat statuses (internal/status) on the victim.

// Coating limits (docs/designs/2026-10-01-weapon-poisons-design.md).
const (
	CoatMinutes     = 10 // real minutes a coating lasts from application
	CoatContacts    = 8  // damaging contacts a coating lasts
	DeliveryPct     = 40 // chance a contact delivers the poison
	ResistantPct    = 20 // chance against a resistant target
	PoisonItemFirst = 280
)

// Susceptibility of a creature to weapon poison.
const (
	PoisonNormal    = ``
	PoisonResistant = `resistant`
	PoisonImmune    = `immune`
)

// Poison is one weapon poison.
type Poison struct {
	ID     string // the word players type
	Name   string
	VialID int    // the shop item that is one dose
	BuffID int    // the combat status it leaves on the victim
	Effect string // one line, as help and previews say it
	Rounds int    // combat rounds the effect lasts
}

// Poisons is the launch catalogue.
var Poisons = []Poison{
	{ID: "bitterleaf", Name: "Bitterleaf", VialID: 280, BuffID: 1120, Rounds: 3, Effect: "1 damage a round"},
	{ID: "leechbane", Name: "Leechbane", VialID: 281, BuffID: 1121, Rounds: 3, Effect: "healing received cut by a quarter"},
	{ID: "leadroot", Name: "Leadroot", VialID: 282, BuffID: 1122, Rounds: 2, Effect: "physical damage dealt cut by 15%"},
	{ID: "mirethorn", Name: "Mirethorn", VialID: 283, BuffID: 1123, Rounds: 2, Effect: "dodge chance down 10 points"},
}

// PoisonByID finds a poison by its id.
func PoisonByID(id string) (Poison, bool) {
	for _, p := range Poisons {
		if p.ID == id {
			return p, true
		}
	}
	return Poison{}, false
}

// PoisonByVial finds the poison a vial item is one dose of.
func PoisonByVial(itemID int) (Poison, bool) {
	for _, p := range Poisons {
		if p.VialID == itemID {
			return p, true
		}
	}
	return Poison{}, false
}

// CoatExpiry is when a coating applied at now runs out.
func CoatExpiry(now time.Time) time.Time {
	return now.Add(CoatMinutes * time.Minute)
}
