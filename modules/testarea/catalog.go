package testarea

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// catalogEntry is one item of the armory catalog, as the web client shows it.
type catalogEntry struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
	Family  string `json:"family,omitempty"`
	Tier    int    `json:"tier,omitempty"`
}

// catalogPayload is the `Armory` GMCP message: the whole catalog, which the
// web client's armory screen filters and searches itself. Filter is the word
// the admin typed, shown in the search box on open.
type catalogPayload struct {
	Filter string         `json:"filter,omitempty"`
	Items  []catalogEntry `json:"items"`
}

// catalogEntries is every item of the world, in a stable order: by type,
// then tier, then name.
func catalogEntries() []catalogEntry {
	specs := items.GetAllItemSpecs()
	out := make([]catalogEntry, 0, len(specs))
	for _, s := range specs {
		out = append(out, catalogEntry{ID: s.ItemId, Name: s.Name, Type: string(s.Type), Subtype: string(s.Subtype), Family: s.Family, Tier: s.Tier})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.Tier != b.Tier {
			return a.Tier < b.Tier
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	return out
}

func (e catalogEntry) matches(word string) bool {
	hay := strings.ToLower(e.Name + " " + e.Type + " " + e.Subtype + " " + e.Family)
	return strings.Contains(hay, word)
}

const catalogTextMax = 40

// toolCatalog is `testarea catalog [word]`. The web client gets the whole
// catalog as a screen with search, type filters and a Take button; every
// client gets text: the types with their counts, or the matches for a word.
// Nothing is given until the admin picks it (testarea give).
func toolCatalog(m *Module, user *users.UserRecord, args []string) string {
	all := catalogEntries()
	word := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
	sendCatalog(user, catalogPayload{Filter: word, Items: all})

	if word == "" {
		counts := map[string]int{}
		for _, e := range all {
			counts[e.Type]++
		}
		var b strings.Builder
		fmt.Fprintf(&b, "The catalog holds %d items. %s lists matches; %s takes one.\n", len(all), cmd("testarea catalog <word>"), cmd("testarea give <item or id> [count]"))
		for _, t := range sortedKeys(counts) {
			fmt.Fprintf(&b, "  %s (%d)\n", cmd(t), counts[t])
		}
		return strings.TrimRight(b.String(), "\n")
	}
	var lines []string
	for _, e := range all {
		if e.matches(word) {
			lines = append(lines, fmt.Sprintf("  #%d %s (%s%s%s)", e.ID, e.Name, e.Type, familySuffix(items.ItemSpec{Family: e.Family}), tierSuffix(e.Tier)))
		}
	}
	if len(lines) == 0 {
		return fmt.Sprintf("No item matches %q.", word)
	}
	more := ""
	if len(lines) > catalogTextMax {
		more = fmt.Sprintf("\n  ...and %d more; narrow the word.", len(lines)-catalogTextMax)
		lines = lines[:catalogTextMax]
	}
	return strings.Join(lines, "\n") + more + "\n" + cmd("testarea give <id> [count]") + " takes one."
}

func tierSuffix(tier int) string {
	if tier > 0 {
		return fmt.Sprintf(", tier %d", tier)
	}
	return ""
}

// sendCatalog hands the web client its screen; a client without GMCP just
// gets the text.
func sendCatalog(user *users.UserRecord, p catalogPayload) {
	if f, ok := usercommands.GetExportedFunction("SendGMCPEvent"); ok {
		if send, ok := f.(func(int, string, any)); ok {
			send(user.UserId, "Armory", p)
		}
	}
}

// ArmoryRoom is the test room whose catalog opens on arrival.
const ArmoryRoom = 90004

// onEnterArmory opens the catalog screen when an admin on a trip walks into
// (or is sent to) the armory, so the web client shows it without a typed
// command. Text clients already read the command in the room description.
func (m *Module) onEnterArmory(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId == 0 || evt.ToRoomId != ArmoryRoom || evt.FromRoomId == ArmoryRoom {
		return events.Continue
	}
	if _, in := m.session(evt.UserId); !in {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil || user.Character == nil || user.Character.RoomId != ArmoryRoom {
		return events.Continue
	}
	sendCatalog(user, catalogPayload{Items: catalogEntries()})
	return events.Continue
}
