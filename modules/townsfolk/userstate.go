package townsfolk

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot what a player has been
// told, so a trip's tellings leave no trace and can be heard again.
type stateContributor struct{ m *Module }

func (stateContributor) Name() string { return "townsfolk" }

func (c stateContributor) maps() userstate.Maps { return userstate.Maps{c.m.state} }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	if c.m.state == nil {
		c.m.mu.Unlock()
		return nil
	}
	if err := c.maps().Apply(userID, data); err != nil {
		c.m.mu.Unlock()
		return err
	}
	err := c.m.saveLocked()
	c.m.mu.Unlock()
	c.m.push(userID)
	return err
}
