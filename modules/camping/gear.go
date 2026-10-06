package camping

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 40a3 camp gear: durable items a company hauls to camp better. Each
// is an ordinary item the company already carries (the leader's pack, a
// present member's pack, or the cargo), counted when a rest starts and
// locked on the rest (camping.RestSession), so losing one mid-rest changes
// nothing until the next rest. Separated and dead members are not present,
// so what they carry does not count.
const (
	bedrollItemID    = 45 // +25% fatigue recovery for one member each
	tentItemID       = 46 // shelter, and no rest-time cold
	fireSteelItemID  = 47 // damp firewood lights first time, at full warmth
	cookpotItemID    = 48 // one extra portion of multi-ingredient dishes
	campBellsItemID  = 49 // a flat chance to spot raiders; wears 10 rests
	surgeonKitItemID = 50 // treats a lasting wound at a rest; wears 5 uses
)

// Bells and trip lines raise a watch's chance to spot raiders: a flat
// chance with no watch posted, and a bonus on top of a watch's, capped at
// the incense cap (Phase 40a3).
const (
	bellsOnlyPct  = 20
	bellsBonusPct = 10
	bellsCapPct   = 90
)

// campGear is what the company has at camp.
type campGear struct {
	// Bedrolls are the member keys that sleep on one: the leader first,
	// then companions by number.
	Bedrolls []string
	// Tent is whether one is pitched; TentKind which (empty is canvas) and
	// Tents every kind carried (Phase 52).
	Tent      bool
	TentKind  camping.TentKind
	Tents     []camping.TentKind
	FireSteel bool
	Cookpot   bool
	Bells     bool
	Kit       bool
}

// gearCount is how many of an item the company carries at camp.
func (m *CampingModule) gearCount(leaderUserID, itemID int) int {
	count := m.itemCount
	if count == nil {
		count = company.CompanyItemCount
	}
	return count(leaderUserID, itemID)
}

// gearOf counts the company's gear. It calls into the company module, so
// it is read before taking m.mu.
func (m *CampingModule) gearOf(leaderUserID int) campGear {
	m.mu.Lock()
	choice := m.camps[leaderUserID].TentChoice
	m.mu.Unlock()
	tents := m.tentsCarried(leaderUserID)
	kind, pitched := camping.PickTent(tents, choice)
	g := campGear{
		Tent:      pitched,
		TentKind:  kind,
		Tents:     tents,
		FireSteel: m.gearCount(leaderUserID, fireSteelItemID) > 0,
		Cookpot:   m.gearCount(leaderUserID, cookpotItemID) > 0,
		Bells:     m.gearCount(leaderUserID, campBellsItemID) > 0,
		Kit:       m.gearCount(leaderUserID, surgeonKitItemID) > 0,
	}
	if n := m.gearCount(leaderUserID, bedrollItemID); n > 0 {
		keys := []string{string(survival.LeaderMemberKey)}
		_, roster := m.companions(leaderUserID)
		ids := append([]int(nil), roster...)
		sort.Ints(ids)
		for _, id := range ids {
			keys = append(keys, string(survival.CompanionMemberKey(id)))
		}
		g.Bedrolls = keys[:min(n, len(keys))]
	}
	return g
}

// tentsCarried is every kind of tent the company carries, in Tents order
// (Phase 52). It calls into the company module, so it is read before m.mu.
func (m *CampingModule) tentsCarried(leaderUserID int) []camping.TentKind {
	var out []camping.TentKind
	for _, t := range camping.Tents {
		if m.gearCount(leaderUserID, t.ItemID) > 0 {
			out = append(out, t.Kind)
		}
	}
	return out
}

// bedrollBonuses is the survival bonus map for a rest's locked bedrolls.
func bedrollBonuses(rest *camping.RestSession) map[survival.MemberKey]int {
	if rest == nil || len(rest.Bedrolls) == 0 {
		return nil
	}
	out := make(map[survival.MemberKey]int, len(rest.Bedrolls))
	for _, key := range rest.Bedrolls {
		out[survival.MemberKey(key)] = camping.BedrollBonusPct
	}
	return out
}

// spotChance is the percent chance a raid is spotted: a watch's own
// chance (watchPct, 0 with none posted), raised by bells and trip lines.
func spotChance(watchPosted bool, watchPct int, bells bool) int {
	switch {
	case watchPosted && bells:
		return min(bellsCapPct, watchPct+bellsBonusPct)
	case watchPosted:
		return watchPct
	case bells:
		return bellsOnlyPct
	}
	return 0
}

// gearLines is one line per piece of gear in use, for the camp view.
func (g campGear) lines(members int) []string {
	var out []string
	if n := len(g.Bedrolls); n > 0 {
		out = append(out, fmt.Sprintf("  Bedrolls: %d of %d members sleep on one (+%d%% fatigue recovered).", n, max(members, n), camping.BedrollBonusPct))
	}
	if g.Tent {
		tent := camping.TentOf(g.TentKind)
		out = append(out, "  "+util.CapitalizeFirst(tent.Name)+": "+tent.Effect+".")
		if len(g.Tents) > 1 {
			out = append(out, "  Tents carried: "+tentNames(g.Tents)+" (camp tent [name] chooses).")
		}
	}
	if g.FireSteel {
		out = append(out, "  Fire steel and tinder: damp firewood lights first time.")
	}
	if g.Cookpot {
		out = append(out, "  Iron cookpot: a multi-ingredient dish makes one extra portion.")
	}
	if g.Bells {
		out = append(out, fmt.Sprintf("  Camp bells and trip lines: raiders are spotted (%d%% with no watch, +%d with one) and thieves keep out.", bellsOnlyPct, bellsBonusPct))
	}
	if g.Kit {
		out = append(out, "  Field surgeon's kit: a healer with mana treats a lasting wound at the end of a rest.")
	}
	return out
}

// labels is one short label per piece of gear carried, for the web Camp
// tab (Phase 40a4).
func (g campGear) labels(members int) []string {
	var out []string
	if n := len(g.Bedrolls); n > 0 {
		out = append(out, fmt.Sprintf("Bedrolls %d/%d", n, max(members, n)))
	}
	if g.Tent {
		out = append(out, util.CapitalizeFirst(camping.TentOf(g.TentKind).Short)+" tent")
	}
	if g.FireSteel {
		out = append(out, "Fire steel")
	}
	if g.Cookpot {
		out = append(out, "Cookpot")
	}
	if g.Bells {
		out = append(out, "Bells and trip lines")
	}
	if g.Kit {
		out = append(out, "Surgeon's kit")
	}
	return out
}

// restGearText is the rest start's report of the gear locked for it.
func restGearText(rest *camping.RestSession, tent bool, members int) string {
	var parts []string
	if n := len(rest.Bedrolls); n > 0 {
		parts = append(parts, fmt.Sprintf("%d of %d on bedrolls (+%d%% fatigue)", n, max(members, n), camping.BedrollBonusPct))
	}
	if tent {
		parts = append(parts, "the "+camping.TentOf(rest.Tent).Name+" keeps out the weather and the cold")
	}
	if rest.Bells {
		parts = append(parts, "bells and trip lines strung")
	}
	if rest.Kit {
		parts = append(parts, "the surgeon's kit packed")
	}
	if len(parts) == 0 {
		return ""
	}
	return "Gear: " + strings.Join(parts, "; ") + "."
}

// tentNames lists tent kinds as "canvas, fur".
func tentNames(kinds []camping.TentKind) string {
	names := make([]string, 0, len(kinds))
	for _, k := range kinds {
		names = append(names, camping.TentOf(k).Short)
	}
	return strings.Join(names, ", ")
}
