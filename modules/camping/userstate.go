package camping

import "github.com/GoMudEngine/GoMud/internal/userstate"

// stateContributor lets the admin test area snapshot a leader's camp, inn
// stay, owed grants and poison plan. The test area refuses to start during
// a rest; restoring stops any timer the area started and forgets raiders it
// drew, then puts the saved state back.
type stateContributor struct{ m *CampingModule }

func (stateContributor) Name() string { return "camping" }

func (c stateContributor) maps() userstate.Maps {
	m := c.m
	return userstate.Maps{m.camps, m.recoveryApplied, m.stays, m.innRecoveryApplied, m.wellRestedPending,
		m.restedPending, m.owed, m.autoSharpen, m.poisonPlans, m.campRewards, m.lastRewards, m.restedDuties}
}

func (c stateContributor) ready() bool {
	m := c.m
	return m.camps != nil && m.recoveryApplied != nil && m.stays != nil && m.innRecoveryApplied != nil &&
		m.wellRestedPending != nil && m.restedPending != nil && m.owed != nil && m.autoSharpen != nil &&
		m.poisonPlans != nil && m.campRewards != nil && m.lastRewards != nil && m.restedDuties != nil
}

func (c stateContributor) Capture(userID int) ([]byte, error) {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	if !c.ready() {
		return nil, nil
	}
	return c.maps().Capture(userID)
}

func (c stateContributor) Restore(userID, _ int, data []byte) error {
	m := c.m
	m.mu.Lock()
	defer m.mu.Unlock()
	if !c.ready() {
		return nil
	}
	defer m.refreshLitRoomsLocked()
	m.stopTimerLocked(userID)
	m.stopInnTimerLocked(userID)
	delete(m.timerGeneration, userID)
	delete(m.innTimerGeneration, userID)
	delete(m.raiders, userID)
	if err := c.maps().Apply(userID, data); err != nil {
		return err
	}
	return m.saveLocked()
}
