package company

import (
	"embed"
	"errors"
	"fmt"
	"os"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

//go:embed files/*
var files embed.FS

type Runtime interface {
	ResolveTemplate(string) (int, bool)
	Spawn(leaderUserID, roomID, mobTemplateID int) (int, error)
	IsLive(instanceID int) bool
	Detach(leaderUserID, instanceID int)
}

type Store interface {
	Load(*domain.Registry) error
	Save(domain.Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *domain.Registry) error {
	err := s.plug.ReadIntoStruct("companies", registry)
	if errors.Is(err, os.ErrNotExist) {
		*registry = *domain.NewRegistry()
		return nil
	}
	return err
}
func (s pluginStore) Save(registry domain.Registry) error {
	return s.plug.WriteStruct("companies", registry)
}

type CompanyModule struct {
	plug         *plugins.Plugin
	store        Store
	registry     domain.Registry
	liveByLeader map[int]int
	runtime      Runtime
}

func init() {
	m := &CompanyModule{plug: plugins.New("company", "1.0"), liveByLeader: map[int]int{}}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.runtime = nativeRuntime{}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(m.save)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.MobDeath{}, m.onMobDeath)
}

func (m *CompanyModule) restoreForLeader(leaderUserID, roomID int) error {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	if instanceID, tracked := m.liveByLeader[leaderUserID]; tracked {
		if m.runtime.IsLive(instanceID) {
			return nil
		}
		delete(m.liveByLeader, leaderUserID)
	}
	instanceID, err := m.runtime.Spawn(leaderUserID, roomID, record.Companion.MobTemplateID)
	if err != nil {
		delete(m.liveByLeader, leaderUserID)
		return fmt.Errorf("company: restore leader %d: %w", leaderUserID, err)
	}
	m.liveByLeader[leaderUserID] = instanceID
	return nil
}

func (m *CompanyModule) save() {
	if m.store != nil {
		if err := m.store.Save(m.registry); err != nil {
			mudlog.Error("company: save", "error", err)
		}
	}
}

func (m *CompanyModule) load() {
	if m.store == nil {
		return
	}
	if err := m.store.Load(&m.registry); err != nil {
		mudlog.Error("company: load", "error", err)
	}
	if m.registry.Companies == nil {
		m.registry = *domain.NewRegistry()
	}
}

func (m *CompanyModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Cancel
	}
	if err := m.restoreForLeader(evt.UserId, evt.RoomId); err != nil {
		mudlog.Warn("company: restore", "error", err)
	}
	return events.Continue
}

func (m *CompanyModule) onMobDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Cancel
	}
	for leader, instanceID := range m.liveByLeader {
		if instanceID == evt.InstanceId {
			delete(m.liveByLeader, leader)
			break
		}
	}
	return events.Continue
}
