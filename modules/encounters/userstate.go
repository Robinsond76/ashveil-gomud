package encounters

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a leader's grace and
// boss-lair cooldowns, so a trip's fights leave no breathing room behind.
type stateContributor struct{ m *EncountersModule }

func (stateContributor) Name() string { return "encounters" }

func (c stateContributor) maps() userstate.Maps { return userstate.Maps{c.m.graces, c.m.bosses} }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.graces == nil || c.m.bosses == nil {
		return nil
	}
	if err := c.maps().Apply(userID, data); err != nil {
		return err
	}
	return c.m.saveLocked()
}
