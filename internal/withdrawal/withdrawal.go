// Package withdrawal reads company escape eligibility on the game loop.
// Requests live on runtime-only Aggro; no battle or mob ID is recovery authority.
package withdrawal

import (
	"errors"
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"slices"
	"sort"
)

// LeaderPinned answers a leader held by a hobbling or binding hurt.
const LeaderPinned = "Your legs will not carry you out of this. Your company holds its ground."

func Eligible(c *characters.Character) bool {
	return c != nil && c.Health > 0 && !c.CombatWithdrawn && !c.HasBuffFlag("no-flee") && !c.HasBuffFlag("no-go")
}

// Route resolves the effective exit (including mutators), refusing private,
// locked, missing and script-blocked routes. Automatic choice is deterministic.
func Route(u *users.UserRecord, r *rooms.Room, requested string) (string, int, error) {
	names := []string{requested}
	if requested == "" {
		names = nil
		seen := map[string]bool{}
		for name := range r.Exits {
			seen[name] = true
		}
		for mut := range r.ActiveMutators {
			for name := range mut.GetSpec().Exits {
				seen[name] = true
			}
		}
		for name := range seen {
			names = append(names, name)
		}
		sort.Strings(names)
	} else {
		name, _ := r.FindExitByName(requested)
		names[0] = name
	}
	for _, name := range names {
		ex, ok := r.GetExitInfo(name)
		if !ok || ex.Secret || ex.Lock.IsLocked() || ex.RoomId == r.RoomId || rooms.LoadRoom(ex.RoomId) == nil {
			continue
		}
		// Mobs cannot enter through a locked reverse door either.
		dest := rooms.LoadRoom(ex.RoomId)
		if reverse := dest.FindExitTo(r.RoomId); reverse != "" {
			rev, _ := dest.GetExitInfo(reverse)
			if rev.Lock.IsLocked() {
				continue
			}
		}
		if blocked, err := scripting.TryRoomTryExitEvent(name, u.UserId, r.RoomId); r.HasScript() && err != nil && !errors.Is(err, scripting.ErrEventNotFound) {
			return "", 0, fmt.Errorf("The exit permission could not be checked: %w", err)
		} else if blocked {
			continue
		}
		if blocked, err := scripting.TryRoomTryEnterEvent(u.UserId, ex.RoomId); dest.HasScript() && err != nil && !errors.Is(err, scripting.ErrEventNotFound) {
			return "", 0, fmt.Errorf("The entry permission could not be checked: %w", err)
		} else if blocked {
			continue
		}
		return name, ex.RoomId, nil
	}
	return "", 0, fmt.Errorf("There is no legal escape route for your company.")
}

func Capture(u *users.UserRecord, r *rooms.Room, name string) *characters.RetreatInfo {
	req := &characters.RetreatInfo{RoomID: r.RoomId, ExitName: name}
	for _, id := range r.GetMobs() {
		m := mobs.GetInstance(id)
		owner, key, ok := company.LeaderAndKeyForInstance(id)
		if !ok || owner != u.UserId || m == nil || m.Character.Health <= 0 || !m.Character.IsCompanion() || !m.Character.IsCharmed(u.UserId) || !slices.Contains(u.Character.GetCharmIds(), id) {
			continue
		}
		req.Members = append(req.Members, characters.RetreatMember{InstanceID: id, Key: string(key), Charm: m.Character.Charmed})
	}
	sort.Slice(req.Members, func(i, j int) bool { return req.Members[i].InstanceID < req.Members[j].InstanceID })
	return req
}

// Present checks the original members. Fallen or removed flight instances do
// not move; a living member's changed ownership or location cancels the order.
func Present(u *users.UserRecord, req *characters.RetreatInfo) ([]*mobs.Mob, error) {
	if req == nil || u.Character.RoomId != req.RoomID || u.Character.Health <= 0 {
		return nil, fmt.Errorf("You cannot lead the company out of this.")
	}
	if !Eligible(u.Character) {
		return nil, errors.New(LeaderPinned)
	}
	out := []*mobs.Mob{}
	for _, member := range req.Members {
		m := mobs.GetInstance(member.InstanceID)
		if m == nil || m.Character.Health <= 0 {
			continue
		}
		owner, key, ok := company.LeaderAndKeyForInstance(member.InstanceID)
		if !ok || owner != u.UserId || string(key) != member.Key || m.Character.Charmed != member.Charm || !m.Character.IsCharmed(u.UserId) || member.Charm.RoundsRemaining == 0 || !member.Charm.Companion || m.Character.RoomId != req.RoomID || !slices.Contains(u.Character.GetCharmIds(), member.InstanceID) {
			return nil, fmt.Errorf("%s is no longer in place for the retreat.", m.Character.Name)
		}
		if !Eligible(&m.Character) {
			return nil, fmt.Errorf("%s is pinned and cannot withdraw. Your company holds its ground until %s can move.", m.Character.Name, m.Character.Name)
		}
		out = append(out, m)
	}
	return out, nil
}

// Mobility uses the shipped personal burden and lasting wound health share.
func Mobility(c *characters.Character) float64 {
	points := 0
	for _, wound := range c.Wounds {
		if !wound.Light {
			points += max(0, wound.Points)
		}
	}
	injury := min(0.30, float64(points)/float64(max(1, c.HealthMax.Value)))
	return max(1, float64(c.Stats.Speed.ValueAdj)*(1-0.6*c.Burden())*(1-injury))
}

func Chance(companyMobility, pursuerMobility float64, cover int) int {
	if pursuerMobility <= 0 {
		return 100
	}
	return min(95, max(30, int(30+70*companyMobility/(companyMobility+pursuerMobility))+cover))
}
