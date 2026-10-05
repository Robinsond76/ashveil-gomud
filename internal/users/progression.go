package users

// migrateStatPointRhythm preserves spent points and pays the peak-level
// difference once. The marker and award reach disk in the same atomic save.
func migrateStatPointRhythm(u *UserRecord) error {
	c := u.Character
	before, marker := c.StatPoints, c.StatPointRhythm
	if !c.CatchUpStatPoints() {
		return nil
	}
	if err := SaveUserAtomic(*u); err != nil {
		c.StatPoints = before
		c.StatPointRhythm = marker
		return err
	}
	return nil
}
