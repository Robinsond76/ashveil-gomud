package storyevents

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a company's events, so
// a trip's scenes leave nothing behind and can be walked into again.
type stateContributor struct{ m *Module }

func (stateContributor) Name() string { return "storyevents" }

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
	// A modal left open on the web client closes with the old state.
	c.m.w.Push(userID, "Event", payload{Active: false})
	return err
}
