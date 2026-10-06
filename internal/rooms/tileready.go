package rooms

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/exit"
)

// Phase 40d: tile-ready zones. A zone opts in with `tileready: true` in its
// zone-config.yaml and then keeps the map conventions in
// docs/designs/tile-ready-conventions.md, so one room is one tile. This
// validator is pure over room data; the world-data test runs it on every
// shipped tile-ready zone, and a fixture proves it catches each rule.

// maxFillerSentences is the longest description a plain filler room (one
// without a maplegend) may have, so walking stays quick to read.
const maxFillerSentences = 2

// ValidateTileReady returns one message per broken convention, sorted. cfg
// supplies the zone's default biome. legendOK reports whether a lowercased
// maplegend draws an S2 landmark (or is a deliberate glyph).
func ValidateTileReady(cfg *ZoneConfig, list []*Room, legendOK func(legend string) bool) []string {
	var problems []string
	add := func(r *Room, format string, args ...any) {
		problems = append(problems, fmt.Sprintf("room %d (%s): %s", r.RoomId, r.Title, fmt.Sprintf(format, args...)))
	}

	byId := make(map[int]*Room, len(list))
	for _, r := range list {
		byId[r.RoomId] = r
	}

	// Rule 1: hand-placed coordinates, one room per coordinate per level.
	taken := map[[3]int]*Room{}
	for _, r := range list {
		if !r.HasCoordinates {
			add(r, "has no hand-placed coordinates (hascoordinates: true)")
			continue
		}
		key := [3]int{r.MapX, r.MapY, r.MapZ}
		if other, dup := taken[key]; dup {
			add(r, "shares coordinate %d,%d,%d with room %d", r.MapX, r.MapY, r.MapZ, other.RoomId)
			continue
		}
		taken[key] = r
	}

	for _, r := range list {
		// Rule 2: each exit leads to the adjacent coordinate in its
		// direction, or declares a mapdirection.
		names := make([]string, 0, len(r.Exits))
		for name := range r.Exits {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			info := r.Exits[name]
			dest, same := byId[info.RoomId]
			if !same || !r.HasCoordinates || !dest.HasCoordinates {
				continue // another zone, or already reported above
			}
			dir := strings.ToLower(name)
			if info.MapDirection != "" {
				dir = strings.ToLower(info.MapDirection)
			}
			delta, known := exit.DirectionDeltas[dir]
			if !known {
				add(r, "exit %q has no compass direction; set mapdirection", name)
				continue
			}
			want := [3]int{r.MapX + delta[0], r.MapY + delta[1], r.MapZ + delta[2]}
			if got := [3]int{dest.MapX, dest.MapY, dest.MapZ}; got != want {
				add(r, "exit %q leads to room %d at %d,%d,%d, expected %d,%d,%d", name, dest.RoomId, got[0], got[1], got[2], want[0], want[1], want[2])
			}
		}

		// Rule 3: a biome (own or the zone's) and a known resource list.
		if r.Biome == "" && (cfg == nil || cfg.DefaultBiome == "") {
			add(r, "has no biome and the zone sets no defaultbiome")
		}
		for _, res := range r.Resources {
			if !IsKnownResource(res) {
				add(r, "unknown resource %q", res)
			}
		}

		// Rule 4: filler rooms read quickly.
		if r.MapLegend == "" {
			if n := sentenceCount(r.Description); n > maxFillerSentences {
				add(r, "filler description has %d sentences (at most %d; a landmark room sets a maplegend)", n, maxFillerSentences)
			}
		}

		// Rule 5: a landmark room's legend draws an S2 landmark.
		if r.MapLegend != "" && legendOK != nil && !legendOK(strings.ToLower(strings.TrimSpace(r.MapLegend))) {
			add(r, "maplegend %q does not map to an S2 landmark", r.MapLegend)
		}
	}
	sort.Strings(problems)
	return problems
}

// sentenceCount counts sentence ends (. ! ?) in a description.
func sentenceCount(text string) int {
	n := 0
	runes := []rune(strings.TrimSpace(text))
	for i, c := range runes {
		if c == '.' || c == '!' || c == '?' {
			if i+1 == len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' || runes[i+1] == '"' {
				n++
			}
		}
	}
	return n
}
