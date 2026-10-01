package combat

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

type AttackResult struct {
	Hit  bool // defaults false
	Crit bool // defaults false
	// CritLanded is true when a critical strike got through the armor and
	// did damage (Crit is set before armor; Phase 30d1b's heavy force).
	CritLanded              bool
	BuffSource              []int // defaults 0
	BuffTarget              []int // defaults 0
	DamageToTarget          int   // defaults 0
	DamageToTargetReduction int   // defaults 0
	DamageToSource          int   // defaults 0
	DamageToSourceReduction int   // defaults 0
	MessagesToSource        []string
	MessagesToTarget        []string
	MessagesToSourceRoom    []string
	MessagesToTargetRoom    []string
	MessagesToRoomOld       []string
	// EdgeSpent is how many strikes of each attacking weapon's Phase 23b
	// edge the round spent, by equipment slot.
	EdgeSpent map[items.ItemType]int
	// WoundsToTarget is the wounds the round's strikes left on a woundable
	// target (Phase 30b), applied with the damage.
	WoundsToTarget []wounds.Wound
	// Defense (Phase 30g2): outcome of the active defense attempt, one of
	// DefenseDodged, DefenseParried, DefenseBlocked, or DefenseNone.
	Defense string
}

const (
	DefenseNone    = ""
	DefenseDodged  = "dodged"
	DefenseParried = "parried"
	DefenseBlocked = "blocked"
)

func (a *AttackResult) SendToSource(msg string) {
	a.MessagesToSource = append(a.MessagesToSource, msg)
}

func (a *AttackResult) SendToTarget(msg string) {
	a.MessagesToTarget = append(a.MessagesToTarget, msg)
}

func (a *AttackResult) SendToSourceRoom(msg string) {
	a.MessagesToSourceRoom = append(a.MessagesToSourceRoom, msg)
}

func (a *AttackResult) SendToTargetRoom(msg string) {
	a.MessagesToTargetRoom = append(a.MessagesToTargetRoom, msg)
}

func (a *AttackResult) SendToRoomOld(msg string) {
	a.MessagesToRoomOld = append(a.MessagesToRoomOld, msg)
}
