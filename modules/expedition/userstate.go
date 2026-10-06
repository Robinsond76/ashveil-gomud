package expedition

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a leader's journey.
// The test area refuses to start mid-journey, so what a snapshot holds is
// at most a finished session; restoring stops any timer the area started.
type stateContributor struct{ m *ExpeditionModule }

func (stateContributor) Name() string { return "expedition" }

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	return userstate.Maps{c.m.sessions}.Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	c.m.stopTimerLocked(userID)
	if c.m.sessions == nil {
		return nil
	}
	if err := (userstate.Maps{c.m.sessions}).Apply(userID, data); err != nil {
		return err
	}
	return c.m.saveLocked()
}
