package market

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	defaultRumorTag     = "inn"
	defaultRumorRefresh = 150 // rounds; 10 minutes at 4-second rounds
	defaultRumorsPerAsk = 3
	maxRumorRefresh     = 100000
	maxRumorsPerAsk     = 10
)

// rumorPhrases holds two phrasings per rumour kind; %[1]s is the market,
// %[2]s the good. Rumours never quote a price.
var rumorPhrases = map[market.RumorKind][2]string{
	market.RumorScarce: {
		"They say %[1]s is short of %[2]s and will pay for it.",
		"A carter mutters that you can't find %[2]s at %[1]s for love or money.",
	},
	market.RumorGlut: {
		"A drover grumbles that %[1]s is drowning in %[2]s.",
		"Word is %[1]s has more %[2]s than it knows what to do with.",
	},
	market.RumorCheapest: {
		"Word is %[2]s comes cheapest at %[1]s.",
		"A trapper swears nowhere sells %[2]s for less than %[1]s.",
	},
	market.RumorBestBuyer: {
		"A peddler swears nobody pays better for %[2]s than %[1]s.",
		"They say %[1]s pays the best price going for %[2]s.",
	},
}

// snapshotNewsLocked replaces the news with every configured market's
// current stock.
func (m *MarketModule) snapshotNewsLocked() {
	news := make(map[string]ZoneMarket, len(m.markets))
	for zone := range m.markets {
		if snap, ok := m.zoneSnapshotLocked(zone); ok {
			news[zone] = snap
		}
	}
	m.news = news
}

// addMissingNewsLocked snapshots each configured market that has no news
// yet, reporting whether any was added.
func (m *MarketModule) addMissingNewsLocked() bool {
	added := false
	for zone := range m.markets {
		if _, ok := m.news[zone]; ok {
			continue
		}
		if snap, ok := m.zoneSnapshotLocked(zone); ok {
			m.news[zone] = snap
			added = true
		}
	}
	return added
}

// zoneSnapshotLocked copies a configured market's current stock, clamped
// to each good's bounds.
func (m *MarketModule) zoneSnapshotLocked(zone string) (ZoneMarket, bool) {
	zm, ok := m.zones[zone]
	if !ok {
		return ZoneMarket{}, false
	}
	snap := ZoneMarket{Goods: []GoodStock{}}
	for _, g := range m.markets[zone] {
		if stock, ok := zm.Stock(g.ItemID); ok {
			snap.Goods = append(snap.Goods, GoodStock{ItemID: g.ItemID, Stock: g.ClampStock(stock)})
		}
	}
	return snap, true
}

// newsSightings returns the news as sightings placed by zone name, in zone
// order, with the spread, rumours per ask, and market room tag; or an error
// while persistence is unavailable.
func (m *MarketModule) newsSightings() ([]market.Sighting, int, int, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return nil, 0, 0, "", err
	}
	zones := make([]string, 0, len(m.markets))
	for zone := range m.markets {
		zones = append(zones, zone)
	}
	sort.Strings(zones)
	sightings := []market.Sighting{}
	for _, zone := range zones {
		heard, ok := m.news[zone]
		if !ok {
			continue
		}
		for _, g := range m.markets[zone] {
			if stock, ok := heard.Stock(g.ItemID); ok {
				sightings = append(sightings, market.Sighting{Place: zone, Good: g, Stock: stock})
			}
		}
	}
	return sightings, m.spreadPct, m.rumorsPerAsk, m.roomTag, nil
}

func (m *MarketModule) rumorRoomTag() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rumorTag
}

// rumorsCommand tells a player in a rumour room (an inn) a few fuzzy,
// possibly stale hints drawn from the market news.
func (m *MarketModule) rumorsCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	if room == nil {
		return true, nil
	}
	if !room.HasTag(m.rumorRoomTag()) {
		user.SendText("You hear no talk of the markets here. Try an inn.")
		return true, nil
	}
	sightings, spreadPct, perAsk, marketTag, err := m.newsSightings()
	if err != nil {
		user.SendText("Nobody here has any news of the markets right now.")
		return true, nil
	}
	picked := market.PickRumors(market.Rumors(sightings, spreadPct), perAsk, m.roll)
	if len(picked) == 0 {
		user.SendText("Nobody here has heard any market news worth repeating.")
		return true, nil
	}
	lines := []string{"You listen to the talk of the markets. It's old news, and prices move:"}
	for _, r := range picked {
		phrases, ok := rumorPhrases[r.Kind]
		if !ok {
			mudlog.Warn("market: rumour kind has no phrasing", "kind", r.Kind)
			continue
		}
		place := fmt.Sprintf(`<ansi fg="room-title">%s</ansi>`, m.rumorPlace(r.Place, marketTag))
		good := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, m.itemNames(r.ItemID)[0])
		lines = append(lines, "  "+fmt.Sprintf(phrases[m.roll()%2], place, good))
	}
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

// rumorPlace names a market zone by its market room, the place a player
// must walk to, falling back to the zone name.
func (m *MarketModule) rumorPlace(zone, marketTag string) string {
	if titles := m.marketRoomTitles(zone, marketTag); len(titles) > 0 {
		return titles[0]
	}
	return zone
}

// parseRumorTag reads the rumour room tag, defaulting when blank.
func parseRumorTag(raw any) string {
	tag, _ := raw.(string)
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return defaultRumorTag
	}
	return tag
}

// parseRumorRefresh reads the news refresh interval in rounds, falling
// back to the default when missing or outside 1..100000.
func parseRumorRefresh(raw any) int {
	return configBounded(raw, "RumorRefreshRounds", 1, maxRumorRefresh, defaultRumorRefresh)
}

// parseRumorsPerAsk reads how many rumours one ask shows, falling back to
// the default when missing or outside 1..10.
func parseRumorsPerAsk(raw any) int {
	return configBounded(raw, "RumorsPerAsk", 1, maxRumorsPerAsk, defaultRumorsPerAsk)
}

func configBounded(raw any, name string, lo, hi, def int) int {
	if raw == nil {
		return def
	}
	n := configInt(raw)
	if n < lo || n > hi {
		mudlog.Warn("market: config value out of range; using default", "setting", name, "value", raw, "min", lo, "max", hi, "default", def)
		return def
	}
	return n
}
