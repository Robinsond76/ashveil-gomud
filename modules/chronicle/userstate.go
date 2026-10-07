package chronicle

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a company's chronicle,
// so a trip's deeds leave no trace in the real record.
type stateContributor struct{ m *Module }

func (stateContributor) Name() string { return "chronicle" }

func (c stateContributor) maps() userstate.Maps { return userstate.Maps{c.m.logs} }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	if c.m.logs == nil {
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
