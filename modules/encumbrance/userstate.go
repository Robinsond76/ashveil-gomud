package encumbrance

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a leader's cargo.
type stateContributor struct{ m *EncumbranceModule }

func (stateContributor) Name() string { return "encumbrance" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return userstate.Maps{c.m.cargo}.Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.cargo == nil {
		return nil
	}
	if err := (userstate.Maps{c.m.cargo}).Apply(userID, data); err != nil {
		return err
	}
	return c.m.saveLocked()
}
