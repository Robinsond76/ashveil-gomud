package company

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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
	m.plug.AddUserCommand("company", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(m.save)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.MobDeath{}, m.onMobDeath)
}

const companyUsage = "Usage: company summon <mob-id-or-name> | company status | company dismiss"

// allowedTemplateIDs normalizes values returned by YAML/config decoding.
func allowedTemplateIDs(raw any) map[int]struct{} {
	allowed := map[int]struct{}{}
	switch values := raw.(type) {
	case []int:
		for _, id := range values {
			allowed[id] = struct{}{}
		}
	case []interface{}:
		for _, value := range values {
			if id, ok := value.(int); ok {
				allowed[id] = struct{}{}
			}
		}
	}
	return allowed
}

func (m *CompanyModule) allowedTemplates() map[int]struct{} {
	if m.plug != nil {
		return allowedTemplateIDs(m.plug.Config.Get("AllowedCompanionMobIDs"))
	}
	return map[int]struct{}{58: {}}
}

func (m *CompanyModule) summon(leaderUserID, roomID int, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return companyUsage, fmt.Errorf("company: companion selector is required")
	}
	templateID, err := strconv.Atoi(selector)
	if err != nil {
		var ok bool
		templateID, ok = m.runtime.ResolveTemplate(strings.ToLower(selector))
		if !ok {
			return "", fmt.Errorf("company: unknown mob template %q", selector)
		}
	}
	if err := m.registry.Summon(leaderUserID, templateID, m.allowedTemplates()); err != nil {
		return "", err
	}
	instanceID, err := m.runtime.Spawn(leaderUserID, roomID, templateID)
	if err != nil {
		m.registry.Dismiss(leaderUserID)
		return "", err
	}
	m.liveByLeader[leaderUserID] = instanceID
	m.save()
	return fmt.Sprintf("Companion summoned: %s.", templateName(templateID, selector)), nil
}

func templateName(templateID int, fallback string) string {
	if spec := mobs.GetMobSpec(mobs.MobId(templateID)); spec != nil && spec.Character.Name != "" {
		return spec.Character.Name
	}
	return fallback
}

func (m *CompanyModule) status(leaderUserID int) string {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "No companion."
	}
	name := templateName(record.Companion.MobTemplateID, strconv.Itoa(record.Companion.MobTemplateID))
	if instanceID, tracked := m.liveByLeader[leaderUserID]; tracked {
		if m.runtime.IsLive(instanceID) {
			return fmt.Sprintf("Companion: %s (present).", name)
		}
		delete(m.liveByLeader, leaderUserID)
	}
	return fmt.Sprintf("Companion: %s (awaiting restoration).", name)
}

func (m *CompanyModule) dismiss(leaderUserID int) (string, error) {
	if instanceID, tracked := m.liveByLeader[leaderUserID]; tracked {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(leaderUserID, instanceID)
		}
		delete(m.liveByLeader, leaderUserID)
	}
	m.registry.Dismiss(leaderUserID)
	m.save()
	return "Companion dismissed.", nil
}

func (m *CompanyModule) userCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) == 0 {
		user.SendText(companyUsage)
		return true, nil
	}
	switch args[0] {
	case "summon":
		if len(args) < 2 {
			user.SendText(companyUsage)
			return true, nil
		}
		roomID := user.Character.RoomId
		if room != nil {
			roomID = room.RoomId
		}
		text, err := m.summon(user.UserId, roomID, strings.Join(args[1:], " "))
		if err != nil {
			user.SendText(err.Error())
			return true, nil
		}
		user.SendText(text)
	case "status":
		user.SendText(m.status(user.UserId))
	case "dismiss":
		text, err := m.dismiss(user.UserId)
		if err != nil {
			return true, err
		}
		user.SendText(text)
	default:
		user.SendText(companyUsage)
	}
	return true, nil
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
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}
	if err := m.restoreForLeader(evt.UserId, user.Character.RoomId); err != nil {
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
