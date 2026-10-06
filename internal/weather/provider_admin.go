package weather

import "errors"

// ErrAdminUnavailable is returned when no weather module is loaded.
var ErrAdminUnavailable = errors.New("weather controls are unavailable right now")

// AdminProvider is optionally implemented by the registered Provider, for
// the admin test area (modules/testarea).
type AdminProvider interface {
	// AdminConditions lists the condition names a zone's biome offers.
	AdminConditions(zone string) []string
	// AdminSetCondition holds a zone at a condition until it is set again
	// (an empty name rolls a fresh condition and lets the weather change
	// on its normal schedule). It returns the condition now in force.
	AdminSetCondition(zone, name string) (string, error)
}

func adminProvider() (AdminProvider, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	ap, ok := p.(AdminProvider)
	return ap, ok
}

// AdminConditions lists the condition names a zone's biome offers.
func AdminConditions(zone string) []string {
	ap, ok := adminProvider()
	if !ok {
		return nil
	}
	return ap.AdminConditions(zone)
}

// AdminSetCondition is AdminProvider.AdminSetCondition.
func AdminSetCondition(zone, name string) (string, error) {
	ap, ok := adminProvider()
	if !ok {
		return "", ErrAdminUnavailable
	}
	return ap.AdminSetCondition(zone, name)
}
