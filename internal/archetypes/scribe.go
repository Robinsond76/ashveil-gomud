package archetypes

// ScribeProvider is optionally implemented by the provider (Phase 36a): the
// company's Scribe reads its unidentified gear when a camp rest completes.
type ScribeProvider interface {
	// CampIdentify identifies, for the leader's company present at the
	// camp, every unidentified item a rest or the best Scribe covers, and
	// returns a line for each thing it did.
	CampIdentify(leaderUserID int) []string
}

// CampIdentify is nil without a provider that identifies gear.
func CampIdentify(leaderUserID int) []string {
	if sp, ok := current().(ScribeProvider); ok {
		return sp.CampIdentify(leaderUserID)
	}
	return nil
}
