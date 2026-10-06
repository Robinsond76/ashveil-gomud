package mount

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a leader's herd.
type stateContributor struct{ m *MountModule }

func (stateContributor) Name() string { return "mount" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return userstate.Maps{c.m.herds}.Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.herds == nil {
		return nil
	}
	if err := (userstate.Maps{c.m.herds}).Apply(userID, data); err != nil {
		return err
	}
	return c.m.saveLocked()
}
