package orders

import "sync"

// Provider is the durable store of orders (modules/strategy). Keys are
// formation member keys: "leader" for the player, "companion:<id>" for a
// companion.
type Provider interface {
	// StoredOrders is the member's orders, in the order they are read.
	StoredOrders(userID int, key string) []Order
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the store. The module calls it once, at init.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// For is a member's orders, or none when no store is registered.
func For(userID int, key string) []Order {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.StoredOrders(userID, key)
}

// Lines are a member's orders in words, one per order.
func Lines(list []Order) []string {
	out := make([]string, len(list))
	for i, o := range list {
		out[i] = o.Describe()
	}
	return out
}

// Preset is the starting orders for an archetype, from the menus above: a
// warrior guards a hurt ally and breaks chants, a healer tends the hurt
// first, a wizard breaks chants and spends its best on a boss, and anyone
// else breaks chants and goes for a healer.
func Preset(archetype string) []Order {
	switch archetype {
	case "warrior":
		return []Order{{When: AllyHurt, Pct: 40, Do: Guard}, {When: Chanting, Do: Break}}
	case "cleric", "alchemist":
		return []Order{{When: AllyHurt, Pct: 50, Do: Heal}, {When: SelfHurt, Pct: 40, Do: Heal}}
	case "wizard", "shaman":
		return []Order{{When: Chanting, Do: Break}, {When: Boss, Do: Strongest}}
	case "witch":
		return []Order{{When: Chanting, Do: Break}, {When: FirstRound, Do: Hold}}
	}
	return []Order{{When: Chanting, Do: Break}, {When: FoeKind, Kind: "healer", Do: Break}}
}
