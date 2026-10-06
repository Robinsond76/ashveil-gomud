package archetype

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a player's archetype,
// class, talents, autoskill toggles and owed kit.
type stateContributor struct{ m *ArchetypeModule }

func (stateContributor) Name() string { return "archetype" }

func (c stateContributor) maps() userstate.Maps {
	r := c.m.registry
	return userstate.Maps{r.Players, r.Autoskill, r.Kits, r.Classes}
}

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.registry == nil {
		return nil, nil
	}
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if c.m.registry == nil || c.m.registry.Players == nil || c.m.registry.Autoskill == nil || c.m.registry.Kits == nil || c.m.registry.Classes == nil {
		return nil
	}
	if err := c.maps().Apply(userID, data); err != nil {
		return err
	}
	return c.m.saveLocked()
}
