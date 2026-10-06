// Package userstate is the seam for per-user state that lives outside the
// user record: each module that keeps state by user id (the company, the
// herd, the camp, the class, ...) registers a Contributor, so a snapshot of
// one user's whole state can be taken and put back (the admin test area,
// modules/testarea). Nothing here runs unless that command does.
package userstate

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"gopkg.in/yaml.v2"
)

// Contributor captures and restores one module's state for one user.
type Contributor interface {
	// Name is the module's stable key in a snapshot.
	Name() string
	// Capture is the user's state as bytes, durable across a restart. Nil
	// bytes mean the module holds nothing for the user.
	Capture(userID int) ([]byte, error)
	// Restore puts back exactly what Capture returned (nil: remove all of
	// the user's state), saves it, and settles any live objects. roomID is
	// where the user now stands, for contributors that respawn things.
	Restore(userID, roomID int, data []byte) error
}

var (
	mu           sync.Mutex
	contributors = map[string]Contributor{}
)

// Register adds a contributor, replacing one of the same name.
func Register(c Contributor) {
	mu.Lock()
	defer mu.Unlock()
	contributors[c.Name()] = c
}

// Names lists the registered contributors, sorted.
func Names() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, 0, len(contributors))
	for n := range contributors {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func sorted() []Contributor {
	mu.Lock()
	defer mu.Unlock()
	names := make([]string, 0, len(contributors))
	for n := range contributors {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Contributor, len(names))
	for i, n := range names {
		out[i] = contributors[n]
	}
	return out
}

// CaptureAll asks every contributor for the user's state. Contributors that
// hold nothing are left out of the result.
func CaptureAll(userID int) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, c := range sorted() {
		data, err := c.Capture(userID)
		if err != nil {
			return nil, fmt.Errorf("userstate: capture %s: %w", c.Name(), err)
		}
		if data != nil {
			out[c.Name()] = data
		}
	}
	return out, nil
}

// RestoreAll puts every contributor's state back. A contributor with no
// entry in the snapshot is told to clear the user's state, so what the user
// gained in the meantime is gone. Every contributor runs even after a
// failure; the errors are joined in the returned slice.
func RestoreAll(userID, roomID int, snap map[string][]byte) []error {
	var errs []error
	for _, c := range sorted() {
		if err := c.Restore(userID, roomID, snap[c.Name()]); err != nil {
			errs = append(errs, fmt.Errorf("userstate: restore %s: %w", c.Name(), err))
		}
	}
	return errs
}

// Maps captures and restores the entries for one user across several
// registries keyed by user id: each element is a map[int]T. T must survive
// a YAML round trip, which is how the owning module saves it anyway.
type Maps []any

type entry struct {
	Present bool   `yaml:"present"`
	Value   string `yaml:"value,omitempty"`
}

func (ms Maps) check() error {
	for i, m := range ms {
		v := reflect.ValueOf(m)
		if v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.Int {
			return fmt.Errorf("userstate: element %d is %T, not a map keyed by int", i, m)
		}
	}
	return nil
}

// Capture serializes the user's entry from every map. It returns nil when
// the user has an entry in none.
func (ms Maps) Capture(userID int) ([]byte, error) {
	if err := ms.check(); err != nil {
		return nil, err
	}
	entries := make([]entry, len(ms))
	any := false
	for i, m := range ms {
		v := reflect.ValueOf(m)
		got := v.MapIndex(reflect.ValueOf(userID).Convert(v.Type().Key()))
		if !got.IsValid() {
			continue
		}
		raw, err := yaml.Marshal(got.Interface())
		if err != nil {
			return nil, err
		}
		entries[i] = entry{Present: true, Value: string(raw)}
		any = true
	}
	if !any {
		return nil, nil
	}
	return yaml.Marshal(entries)
}

// Apply writes a Capture back into the maps: an entry that was present is
// set to its captured value, one that was not is deleted. nil data deletes
// every entry. The maps must be non-nil; the caller holds their lock.
func (ms Maps) Apply(userID int, data []byte) error {
	if err := ms.check(); err != nil {
		return err
	}
	var entries []entry
	if data != nil {
		if err := yaml.Unmarshal(data, &entries); err != nil {
			return err
		}
		if len(entries) != len(ms) {
			return fmt.Errorf("userstate: snapshot has %d entries, want %d", len(entries), len(ms))
		}
	}
	for i, m := range ms {
		v := reflect.ValueOf(m)
		key := reflect.ValueOf(userID).Convert(v.Type().Key())
		if data == nil || !entries[i].Present {
			v.SetMapIndex(key, reflect.Value{})
			continue
		}
		val := reflect.New(v.Type().Elem())
		if err := yaml.Unmarshal([]byte(entries[i].Value), val.Interface()); err != nil {
			return err
		}
		v.SetMapIndex(key, val.Elem())
	}
	return nil
}
