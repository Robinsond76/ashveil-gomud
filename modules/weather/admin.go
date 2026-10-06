package weather

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/weather"
)

var _ weather.AdminProvider = (*WeatherModule)(nil)

// adminHoldRounds is how long an admin-set condition is held: effectively
// until it is set again. The round is a count, not game time; nothing here
// advances the clock.
const adminHoldRounds = 1 << 32

// AdminConditions implements weather.AdminProvider.
func (m *WeatherModule) AdminConditions(zone string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	table, ok := m.tableForZoneLocked(zone)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(table.Conditions))
	for _, c := range table.Conditions {
		names = append(names, c.Name)
	}
	return names
}

// AdminSetCondition implements weather.AdminProvider.
func (m *WeatherModule) AdminSetCondition(zone, name string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	table, ok := m.tableForZoneLocked(zone)
	if !ok {
		return "", fmt.Errorf("%s has no weather", zone)
	}
	round := m.roundFn()
	prev, had := m.zones[zone]
	var zw weather.ZoneWeather
	var err error
	if name = strings.ToLower(strings.TrimSpace(name)); name == "" || name == "auto" {
		zw, err = m.establishLocked(zone, table, round)
	} else {
		condition, known := conditionNamed(table, name)
		if !known {
			return "", fmt.Errorf("%s has no %q weather", zone, name)
		}
		zw, err = weather.Established(zone, condition, round+adminHoldRounds, round)
		zw.Next = condition.Name
	}
	if err != nil {
		return "", err
	}
	m.zones[zone] = zw
	if err := m.saveLocked(); err != nil {
		if had {
			m.zones[zone] = prev
		} else {
			delete(m.zones, zone)
		}
		return "", err
	}
	return zw.Current, nil
}
