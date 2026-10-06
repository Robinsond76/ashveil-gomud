package rooms

import (
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Phase 40a: room resources. A room lists what it provides in its
// `resources` field. The vocabulary is validated when a room loads; look,
// survival, camping and the map all read it through the helpers here.

// Resource ids. Water, forage and shelter are wired in 40a; herbs, firewood,
// fishing and game in 40a2 (modules/gathering).
const (
	ResourceWater    = `water`
	ResourceForage   = `forage`
	ResourceShelter  = `shelter`
	ResourceHerbs    = `herbs`
	ResourceFirewood = `firewood`
	ResourceFishing  = `fishing`
	ResourceGame     = `game`
)

type resourceInfo struct {
	ID    string
	Label string // how look and the map tooltip name it
	Shown bool   // false: reserved until its mechanic ships
	Camp  bool   // acts only on a camp rest: shown only where a camp can be made
}

// resourceTable fixes the vocabulary and the display order.
var resourceTable = []resourceInfo{
	{ResourceWater, `fresh water`, true, false},
	{ResourceForage, `forage`, true, true},
	{ResourceShelter, `shelter`, true, true},
	{ResourceHerbs, `herbs`, true, false},
	{ResourceFirewood, `firewood`, true, false},
	{ResourceFishing, `fishing`, true, false},
	{ResourceGame, `game`, true, false},
}

func lookupResource(id string) (resourceInfo, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, info := range resourceTable {
		if info.ID == id {
			return info, true
		}
	}
	return resourceInfo{}, false
}

// IsKnownResource reports whether id is in the resource vocabulary.
func IsKnownResource(id string) bool {
	_, ok := lookupResource(id)
	return ok
}

// ResourceLabel is the player-facing name of a resource id ("" if unknown).
func ResourceLabel(id string) string {
	info, _ := lookupResource(id)
	return info.Label
}

// NormalizeResources lowercases, de-duplicates and orders a resource list,
// dropping unknown entries. dropped holds what was removed.
func NormalizeResources(in []string) (kept, dropped []string) {
	have := map[string]bool{}
	for _, id := range in {
		if info, ok := lookupResource(id); ok {
			have[info.ID] = true
		} else {
			dropped = append(dropped, id)
		}
	}
	for _, info := range resourceTable {
		if have[info.ID] {
			kept = append(kept, info.ID)
		}
	}
	return kept, dropped
}

// normalizeResources cleans the room's own list, warning about entries it
// drops. Called as a room loads.
func (r *Room) normalizeResources() {
	if len(r.Resources) == 0 {
		return
	}
	kept, dropped := NormalizeResources(r.Resources)
	for _, id := range dropped {
		mudlog.Warn("rooms: unknown resource dropped", "room", r.RoomId, "resource", id)
	}
	r.Resources = kept
}

// HasResource reports whether the room provides a resource.
func (r *Room) HasResource(id string) bool {
	if r == nil {
		return false
	}
	id = strings.ToLower(strings.TrimSpace(id))
	for _, have := range r.Resources {
		if strings.ToLower(have) == id {
			return true
		}
	}
	return false
}

var (
	campableMu sync.RWMutex
	campable   func(*Room) bool
)

// SetCampableCheck registers the rule for whether a camp can be made in a
// room (modules/camping). Forage and shelter only act on a camp rest, so
// they are shown only where it says yes; with no rule set they are shown
// wherever the data lists them. nil clears it.
func SetCampableCheck(check func(*Room) bool) {
	campableMu.Lock()
	defer campableMu.Unlock()
	campable = check
}

func canCampIn(r *Room) bool {
	campableMu.RLock()
	check := campable
	campableMu.RUnlock()
	return check == nil || check(r)
}

// ShownResources is the room's resources that players can see and use
// today, in display order. Forage and shelter need a room a camp can be
// made in, so a marker never promises a rule that can't act. Never nil.
func (r *Room) ShownResources() []string {
	out := []string{}
	if r == nil {
		return out
	}
	for _, info := range resourceTable {
		if !info.Shown || !r.HasResource(info.ID) || (info.Camp && !canCampIn(r)) {
			continue
		}
		out = append(out, info.ID)
	}
	return out
}

var (
	depletedMu sync.RWMutex
	depleted   func(roomID int, resource string) bool
)

// SetDepletedCheck registers the rule for whether a gathering resource in a
// room is picked clean for now (modules/gathering, Phase 40a2). nil clears
// it; with none set nothing is ever depleted.
func SetDepletedCheck(check func(roomID int, resource string) bool) {
	depletedMu.Lock()
	defer depletedMu.Unlock()
	depleted = check
}

// IsDepleted reports whether a shown gathering resource here is picked
// clean right now.
func (r *Room) IsDepleted(id string) bool {
	if r == nil {
		return false
	}
	depletedMu.RLock()
	check := depleted
	depletedMu.RUnlock()
	return check != nil && r.HasResource(id) && check(r.RoomId, strings.ToLower(strings.TrimSpace(id)))
}

// DepletedResources is the room's shown resources that are picked clean for
// now, in display order. Never nil.
func (r *Room) DepletedResources() []string {
	out := []string{}
	for _, id := range r.ShownResources() {
		if r.IsDepleted(id) {
			out = append(out, id)
		}
	}
	return out
}

// ResourceLine is look's "Here: ..." line, or "" without shown resources.
// A picked-clean gathering resource says so.
func (r *Room) ResourceLine() string {
	ids := r.ShownResources()
	if len(ids) == 0 {
		return ``
	}
	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = ResourceLabel(id)
		if r.IsDepleted(id) {
			labels[i] += ` (picked clean)`
		}
	}
	return `Here: ` + strings.Join(labels, `, `) + `.`
}
