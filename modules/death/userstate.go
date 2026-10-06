package death

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot when a user was last
// returned to a church. It is held in memory only.
type stateContributor struct{ m *DeathModule }

func (stateContributor) Name() string { return "death" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return userstate.Maps{c.m.returned}.Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.returned == nil {
		return nil
	}
	return (userstate.Maps{c.m.returned}).Apply(userID, data)
}
