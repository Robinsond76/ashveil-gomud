// Package sigils is Phase 54's rules: the four sigils a caster can lay in a
// room before a fight, what each costs and does, and how long one lasts. It
// is GoMud-free and pure. A sigil is laid with `cast sigil of [kind]` out of
// battle, costs mana and a sigil chalk, and lasts real minutes (it is stored
// as a wall-clock expiry, so it survives a restart or copyover and never
// moves global game time). Any battle the company fights in that room while
// it lasts uses it; enemies never lay sigils.
package sigils

import (
	"strings"
	"time"
)

// Kind is one sigil.
type Kind string

const (
	None      Kind = ""
	Fire      Kind = "fire"      // fire spells hit harder and leave the foe burning
	Ward      Kind = "ward"      // the company starts the battle under a small ward
	Stillness Kind = "stillness" // the foes' chants and sling shots start a round slower
	Mending   Kind = "mending"   // heals land stronger
)

// Kinds are the sigils a caster can lay, in the order they are listed.
var Kinds = []Kind{Fire, Ward, Stillness, Mending}

// Numbers the sigils move.
const (
	// Minutes is how long a sigil lasts, in real minutes.
	Minutes = 15
	// ChalkItemID is the reagent each sigil uses up (market-only, never bought back).
	ChalkItemID = 30060
	// FirePct is the extra damage a fire spell deals under a fire sigil.
	FirePct = 15
	// MendingPct is the extra a heal restores under a mending sigil.
	MendingPct = 25
	// StillRounds is how many rounds the foes stay chilled by a stillness sigil.
	StillRounds = 3
	// WardBlows is how many blows each member's ward absorbs.
	WardBlows = 1
)

// ManaCost is the mana a kind costs to lay.
func (k Kind) ManaCost() int {
	switch k {
	case Fire, Mending:
		return 12
	case Ward, Stillness:
		return 15
	}
	return 0
}

// WardCap is the most a sigil's ward takes from one blow: about half of
// a Priest's Ward, from a character's level.
func WardCap(level int) int { return 4 + max(level, 1)/3 }

// Name is the sigil as lines name it.
func (k Kind) Name() string {
	switch k {
	case Fire, Ward, Stillness, Mending:
		return string(k) + " sigil"
	}
	return ""
}

// Effect is what the sigil does, in a few words, for the battle screen.
func (k Kind) Effect() string {
	switch k {
	case Fire:
		return "fire spells 15% stronger and leave the foe burning"
	case Ward:
		return "company starts under a small ward"
	case Stillness:
		return "foe chants and sling shots start a round slower"
	case Mending:
		return "heals 25% stronger"
	}
	return ""
}

// Glow is the line `look` adds for a laid sigil.
func (k Kind) Glow() string {
	switch k {
	case Fire:
		return "a ring of ember-red chalk glows on the ground"
	case Ward:
		return "a ring of pale blue chalk shimmers on the ground"
	case Stillness:
		return "a ring of grey chalk lies unnaturally still on the ground"
	case Mending:
		return "a ring of soft green chalk glows on the ground"
	}
	return ""
}

// Parse reads `fire`, `fire sigil` or `of fire` as a kind.
func Parse(words string) (Kind, bool) {
	w := strings.ToLower(strings.TrimSpace(words))
	w = strings.TrimPrefix(w, "of ")
	w = strings.TrimSuffix(w, " sigil")
	for _, k := range Kinds {
		if w == string(k) {
			return k, true
		}
	}
	return None, false
}

// Laid is a sigil a company has laid: where, and when it fades (Unix
// seconds). It is saved with the leader's character.
type Laid struct {
	Kind    Kind  `yaml:"kind,omitempty"`
	RoomId  int   `yaml:"roomid,omitempty"`
	Expires int64 `yaml:"expires,omitempty"`
}

// Live reports whether the sigil is still lit at now.
func (l Laid) Live(now time.Time) bool {
	return l.Kind != None && l.Expires > now.Unix()
}

// In reports whether the sigil is lit at now in the room.
func (l Laid) In(roomId int, now time.Time) bool {
	return l.Live(now) && l.RoomId == roomId
}

// MinutesLeft is the whole minutes left, rounded up (0 when faded).
func (l Laid) MinutesLeft(now time.Time) int {
	if !l.Live(now) {
		return 0
	}
	return int((l.Expires - now.Unix() + 59) / 60)
}

// Lay is a sigil of kind laid in the room at now.
func Lay(kind Kind, roomId int, now time.Time) Laid {
	return Laid{Kind: kind, RoomId: roomId, Expires: now.Add(Minutes * time.Minute).Unix()}
}
