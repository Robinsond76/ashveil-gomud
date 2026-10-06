package walking

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a user's fatigue carry.
type stateContributor struct{ m *WalkingModule }

func (stateContributor) Name() string { return "walking" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return userstate.Maps{c.m.registry.Carry}.Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	if c.m.registry.Carry == nil {
		c.m.mu.Unlock()
		return nil
	}
	err := (userstate.Maps{c.m.registry.Carry}).Apply(userID, data)
	c.m.dirty = true
	c.m.mu.Unlock()
	if err != nil {
		return err
	}
	return c.m.save()
}
