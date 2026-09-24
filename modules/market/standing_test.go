package market

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/standing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// standingStub gives every leader the same standing in Dunmar.
type standingStub struct{ company int }

func (s standingStub) For(_ int, zone string) (standing.Standing, bool) {
	if zone != "Dunmar" {
		return standing.Standing{}, false
	}
	st := standing.Assess(s.company, 40, standing.DefaultRules())
	st.Zone = zone
	return st, true
}
func (standingStub) BlackMarketTag() string { return "blackmarket" }

func useStanding(t *testing.T, company int) {
	t.Helper()
	standing.SetProvider(standingStub{company: company})
	t.Cleanup(func() { standing.SetProvider(nil) })
}

func blackMarketRoom() *rooms.Room {
	return &rooms.Room{RoomId: 2006, Zone: "Dunmar", Title: "Tanner's Back Alley", Tags: []string{"blackmarket"}}
}

func (w *tradeWorld) runIn(t *testing.T, room *rooms.Room, rest string) string {
	t.Helper()
	*w.messages = nil
	handled, err := w.module.userCommand(rest, w.user, room, 0)
	require.NoError(t, err)
	require.True(t, handled)
	events.ProcessEvents()
	return stripTags(joinLines(*w.messages))
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

func TestListingAppliesDistrustedMarkup(t *testing.T) {
	useStanding(t, -60) // gap 100 from Dunmar: distrusted
	w := newTradeWorld(t, 100)
	out := w.run(t, "")
	// Base ask 27, bid 22: 27*1.2 = 32.4 -> 33; 22*0.8 = 17.6 -> 17.
	assert.Regexp(t, `wolf hide\s+33 gold\s+17 gold\s+scarce`, out)
	assert.Contains(t, out, "Your company is distrusted here: you pay 20% more and are paid 20% less.")
}

func TestBuySellApplyMarkup(t *testing.T) {
	useStanding(t, -60)
	w := newTradeWorld(t, 32)
	assert.Contains(t, w.run(t, "buy hide"), "costs 33 gold here, and you don't have enough", "affordability uses the marked-up price")
	w.assertUnchanged(t, 32, map[int]int{28: 0}, map[int]int{28: 4})
	w.user.Character.Gold = 100
	assert.Contains(t, w.run(t, "buy hide"), "for 33 gold")
	assert.Equal(t, 67, w.user.Character.Gold)
	// Stock is now 3: ask 28, bid 28 - 5 = 23, marked down to 18.
	assert.Contains(t, w.run(t, "sell hide"), "for 18 gold")
	assert.Equal(t, 85, w.user.Character.Gold, "the round trip still loses")
}

func TestShunnedRefusedAtMarket(t *testing.T) {
	useStanding(t, -100) // gap 140: shunned
	w := newTradeWorld(t, 100)
	w.user.Character.StoreItem(items.New(28))
	for _, rest := range []string{"", "buy hide", "sell hide"} {
		assert.Contains(t, w.run(t, rest), "The traders of Dunmar won't deal with your company.")
	}
	w.assertUnchanged(t, 100, map[int]int{28: 1}, map[int]int{28: 4})
}

func TestBlackMarketServesOnlyOutlaws(t *testing.T) {
	t.Run("shunned trade at normal prices", func(t *testing.T) {
		useStanding(t, -100)
		w := newTradeWorld(t, 100)
		out := w.runIn(t, blackMarketRoom(), "")
		assert.Contains(t, out, "Black market prices in Dunmar:")
		assert.Regexp(t, `wolf hide\s+27 gold\s+22 gold`, out)
		assert.Contains(t, w.runIn(t, blackMarketRoom(), "buy hide"), "for 27 gold")
		assert.Equal(t, 3, stockOf(t, w.store.saved, "Dunmar", 28), "the same zone ledger")
	})
	t.Run("distrusted are served", func(t *testing.T) {
		useStanding(t, -60)
		w := newTradeWorld(t, 100)
		assert.Contains(t, w.runIn(t, blackMarketRoom(), "buy hide"), "for 27 gold", "no standing markup at the black market")
	})
	t.Run("welcome are turned away", func(t *testing.T) {
		useStanding(t, 40)
		w := newTradeWorld(t, 100)
		assert.Contains(t, w.runIn(t, blackMarketRoom(), "buy hide"), "No one here will deal with your company.")
		w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 4})
	})
	t.Run("no standing provider, no black market", func(t *testing.T) {
		w := newTradeWorld(t, 100)
		assert.Contains(t, w.runIn(t, blackMarketRoom(), "buy hide"), "There's no market here.")
	})
}

func TestNoStandingProviderUnchanged(t *testing.T) {
	w := newTradeWorld(t, 100)
	out := w.run(t, "")
	assert.Regexp(t, `wolf hide\s+27 gold\s+22 gold\s+scarce`, out)
	assert.NotContains(t, out, "more")
}

func TestMarketHintSkipsBlackMarket(t *testing.T) {
	useStanding(t, 40)
	w := newTradeWorld(t, 100)
	var asked []string
	w.module.marketRooms = func(zone, tag string) []string {
		asked = append(asked, tag)
		return []string{"Dunmar Market Square"}
	}
	out := w.runIn(t, &rooms.Room{RoomId: 2001, Zone: "Dunmar"}, "")
	assert.Contains(t, out, "The market in Dunmar is at Dunmar Market Square.")
	assert.Equal(t, []string{"market"}, asked, "only ordinary market rooms are named")
}

func TestRoomWithBothTags(t *testing.T) {
	both := &rooms.Room{RoomId: 2004, Zone: "Dunmar", Title: "Dunmar Market Square", Tags: []string{"market", "blackmarket"}}
	t.Run("welcome trade at the ordinary market", func(t *testing.T) {
		useStanding(t, 40)
		w := newTradeWorld(t, 100)
		out := w.runIn(t, both, "")
		assert.Contains(t, out, "Market prices in Dunmar:")
		assert.Regexp(t, `wolf hide\s+27 gold`, out)
	})
	t.Run("outlaws get the black market", func(t *testing.T) {
		useStanding(t, -60)
		w := newTradeWorld(t, 100)
		out := w.runIn(t, both, "")
		assert.Contains(t, out, "Black market prices in Dunmar:")
		assert.Regexp(t, `wolf hide\s+27 gold`, out, "no markup")
	})
}

// noSettlements names a black-market tag but knows no settlements.
type noSettlements struct{}

func (noSettlements) For(int, string) (standing.Standing, bool) { return standing.Standing{}, false }
func (noSettlements) BlackMarketTag() string                    { return "blackmarket" }

func TestBlackMarketOutsideSettlementServesNoOne(t *testing.T) {
	standing.SetProvider(noSettlements{})
	t.Cleanup(func() { standing.SetProvider(nil) })
	w := newTradeWorld(t, 100)
	assert.Contains(t, w.runIn(t, blackMarketRoom(), "buy hide"), "No one here will deal with your company.")
	w.assertUnchanged(t, 100, map[int]int{28: 0}, map[int]int{28: 4})
}
