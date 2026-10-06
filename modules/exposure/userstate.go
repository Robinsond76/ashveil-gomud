package exposure

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a user's exposure.
type stateContributor struct{ m *ExposureModule }

func (stateContributor) Name() string { return "exposure" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return userstate.Maps{c.m.registry.Exposure}.Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.registry.Exposure == nil {
		return nil
	}
	if err := (userstate.Maps{c.m.registry.Exposure}).Apply(userID, data); err != nil {
		return err
	}
	return c.m.saveLocked()
}
