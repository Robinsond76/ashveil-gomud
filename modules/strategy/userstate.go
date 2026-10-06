package strategy

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a player's strategies
// and company tactics.
type stateContributor struct{ m *StrategyModule }

func (stateContributor) Name() string { return "strategy" }

func (c stateContributor) maps() userstate.Maps {
	return userstate.Maps{c.m.registry.Players, c.m.registry.Tactics}
}

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	if c.m.registry.Players == nil || c.m.registry.Tactics == nil {
		c.m.mu.Unlock()
		return nil
	}
	err := c.maps().Apply(userID, data)
	c.m.mu.Unlock()
	if err != nil {
		return err
	}
	return c.m.save()
}
