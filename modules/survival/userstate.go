package survival

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a leader's company
// needs (hunger, thirst, fatigue) and the operations already applied.
type stateContributor struct{ m *SurvivalModule }

func (stateContributor) Name() string { return "survival" }

func (c stateContributor) maps() userstate.Maps {
	r := &c.m.registry
	return userstate.Maps{r.Leaders, r.ReservedNextCompanionIDs, r.AppliedExertion, r.AppliedRestOperation}
}

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	r := &c.m.registry
	if r.Leaders == nil || r.ReservedNextCompanionIDs == nil || r.AppliedExertion == nil || r.AppliedRestOperation == nil {
		return nil
	}
	if err := c.maps().Apply(userID, data); err != nil {
		return err
	}
	return c.m.save()
}
