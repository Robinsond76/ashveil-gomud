package rooms

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Phase 40a: room resources. A room lists what it provides in its
// `resources` field. The vocabulary is validated when a room loads; look,
// survival, camping and the map all read it through the helpers here.

// Resource ids. The first three are shown and wired in 40a; the rest are
// accepted in data now and appear when 40a2 gives them their rules.
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
}

// resourceTable fixes the vocabulary and the display order.
var resourceTable = []resourceInfo{
	{ResourceWater, `fresh water`, true},
	{ResourceForage, `forage`, true},
	{ResourceShelter, `shelter`, true},
	{ResourceHerbs, `herbs`, false},
	{ResourceFirewood, `firewood`, false},
	{ResourceFishing, `fishing`, false},
	{ResourceGame, `game`, false},
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

// ShownResources is the room's resources that players can see and use
// today, in display order. Never nil.
func (r *Room) ShownResources() []string {
	out := []string{}
	if r == nil {
		return out
	}
	for _, info := range resourceTable {
		if info.Shown && r.HasResource(info.ID) {
			out = append(out, info.ID)
		}
	}
	return out
}

// ResourceLine is look's "Here: ..." line, or "" without shown resources.
func (r *Room) ResourceLine() string {
	ids := r.ShownResources()
	if len(ids) == 0 {
		return ``
	}
	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = ResourceLabel(id)
	}
	return `Here: ` + strings.Join(labels, `, `) + `.`
}
