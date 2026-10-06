package users

import (
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/sigils"
)

// RoomSigil is a sigil lit in a room, with who laid it (Phase 54).
type RoomSigil struct {
	UserId  int
	Owner   string // the laying character's name
	Kind    sigils.Kind
	Minutes int   // whole minutes left, rounded up
	Expires int64 // the Unix second it fades
}

// SigilsIn lists the sigils lit in a room at now, from the companies online,
// by who laid them. A sigil whose leader is offline still burns down, but only
// shows (and serves a battle) while they are online.
func SigilsIn(roomId int, now time.Time) []RoomSigil {
	var out []RoomSigil
	for _, u := range GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		if l := u.Character.Sigil; l.In(roomId, now) {
			out = append(out, RoomSigil{UserId: u.UserId, Owner: u.Character.Name, Kind: l.Kind, Minutes: l.MinutesLeft(now), Expires: l.Expires})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserId < out[j].UserId })
	return out
}
