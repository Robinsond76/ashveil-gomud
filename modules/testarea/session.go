package testarea

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"gopkg.in/yaml.v2"
)

// Session is one admin's trip into the test area: everything needed to put
// them back exactly as they were. It is saved the moment the trip starts, so
// a disconnect or a restart still lets them return.
type Session struct {
	Taken time.Time `yaml:"taken"`
	// Room is where they stood.
	Room int `yaml:"room"`
	// User is the whole user record (character, inventory, equipment, gold,
	// experience, buffs, alignment, ...), as the user file would hold it.
	User string `yaml:"user"`
	// States is each module's per-user state (company, class, herd, camp,
	// cargo, survival needs, ...), by contributor name.
	States map[string]string `yaml:"states,omitempty"`
}

// Registry is the saved set of sessions, by user id.
type Registry struct {
	Sessions map[int]Session `yaml:"sessions"`
}

// Store persists the registry.
type Store interface {
	Load() (map[int]Session, error)
	Save(map[int]Session) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load() (map[int]Session, error) {
	data, err := s.plug.ReadBytes("sessions")
	if errors.Is(err, os.ErrNotExist) {
		return map[int]Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	if reg.Sessions == nil {
		reg.Sessions = map[int]Session{}
	}
	return reg.Sessions, nil
}

func (s pluginStore) Save(sessions map[int]Session) error {
	return s.plug.WriteStruct("sessions", Registry{Sessions: sessions})
}

// takeSession captures a user: the record, then every module's state.
func takeSession(user *users.UserRecord) (Session, error) {
	record, err := yaml.Marshal(user)
	if err != nil {
		return Session{}, fmt.Errorf("capture user: %w", err)
	}
	states, err := userstate.CaptureAll(user.UserId)
	if err != nil {
		return Session{}, err
	}
	snap := Session{Taken: time.Now(), Room: user.Character.RoomId, User: string(record), States: map[string]string{}}
	for name, data := range states {
		snap.States[name] = string(data)
	}
	return snap, nil
}

// restoredRecord is the saved user record with the account's live
// credentials kept: a password or role changed while away is not undone.
func (s Session) restoredRecord(live *users.UserRecord) (*users.UserRecord, error) {
	restored := &users.UserRecord{}
	if err := yaml.Unmarshal([]byte(s.User), restored); err != nil {
		return nil, fmt.Errorf("read saved user: %w", err)
	}
	if restored.Character == nil {
		return nil, errors.New("the saved user has no character")
	}
	restored.UserId = live.UserId
	restored.Password, restored.Role, restored.Permissions = live.Password, live.Role, live.Permissions
	restored.Character.SetUserId(restored.UserId)
	// What a login does to a loaded character, so its derived values are
	// in place as they were before the trip.
	_ = restored.Character.Validate(true)
	return restored, nil
}

// stateBytes is the snapshot's module states as contributor input.
func (s Session) stateBytes() map[string][]byte {
	out := make(map[string][]byte, len(s.States))
	for name, data := range s.States {
		out[name] = []byte(data)
	}
	return out
}
