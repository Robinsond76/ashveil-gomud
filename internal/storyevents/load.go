package storyevents

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v2"
)

// Parse reads one events file: a YAML list of events. Unknown fields are
// errors, so a misspelt key never silently disables a rule.
func Parse(data []byte) ([]Event, error) {
	var list []Event
	if err := yaml.UnmarshalStrict(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// Catalog is the usable events by id.
type Catalog struct {
	byID  map[string]Event
	order []string
}

// NewCatalog validates events and keeps the sound ones, in id order; later
// entries with a duplicate id are dropped. The returned problems name each
// rejected event.
func NewCatalog(list []Event, l Lookups) (Catalog, []string) {
	c := Catalog{byID: map[string]Event{}}
	var problems []string
	for _, e := range list {
		if _, dup := c.byID[e.ID]; dup {
			problems = append(problems, fmt.Sprintf("event %q: defined twice; the first is used", e.ID))
			continue
		}
		if errs := e.Validate(l); len(errs) > 0 {
			for _, msg := range errs {
				problems = append(problems, fmt.Sprintf("event %q: %s", e.ID, msg))
			}
			continue
		}
		c.byID[e.ID] = e
		c.order = append(c.order, e.ID)
	}
	sortStrings(c.order)
	return c, problems
}

func sortStrings(s []string) { sort.Strings(s) }

// Get returns an event by id.
func (c Catalog) Get(id string) (Event, bool) {
	e, ok := c.byID[id]
	return e, ok
}

// IDs lists the events in id order.
func (c Catalog) IDs() []string { return append([]string(nil), c.order...) }

// Len is how many events the catalog holds.
func (c Catalog) Len() int { return len(c.order) }

// Triggered lists the events with a trigger of this kind that matches the
// room (in id order). The caller applies each event's chance, level range,
// cooldown and requirement.
func (c Catalog) Triggered(kind string, roomID int, zone string, tags []string) []Triggered {
	var out []Triggered
	for _, id := range c.order {
		e := c.byID[id]
		for _, t := range e.Triggers {
			if t.Matches(kind, roomID, zone, tags) {
				out = append(out, Triggered{Event: e, Trigger: t})
				break
			}
		}
	}
	return out
}

// Triggered is an event and the trigger that matched.
type Triggered struct {
	Event   Event
	Trigger Trigger
}
