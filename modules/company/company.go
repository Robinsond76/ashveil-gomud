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
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

type Runtime interface {
	ResolveTemplate(string) (int, bool)
	Spawn(leaderUserID, roomID, mobTemplateID int) (int, error)
	IsLive(instanceID int) bool
	IsAttached(leaderUserID, instanceID int) bool
	Detach(leaderUserID, instanceID int)
}

type Store interface {
	Load(*domain.Registry) error
	Save(domain.Registry) error
}

type wireRecord struct {
	// LeaderUserID is decoded for shape compatibility; the companies map key is authoritative.
	LeaderUserID int                `yaml:"leader_user_id"`
	Companions   []domain.Companion `yaml:"companions"`
	Companion    *domain.Companion  `yaml:"companion"`
	Formation    domain.Formation   `yaml:"formation"`
}

type wireRegistry struct {
	Companies map[int]wireRecord `yaml:"companies"`
}

// decodeCompanies parses stored bytes, converting a legacy single-companion
// record into a roster entry with ID 1.
func decodeCompanies(data []byte, registry *domain.Registry) error {
	var wire wireRegistry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := domain.NewRegistry()
	for leaderID, wr := range wire.Companies {
		record := domain.Record{LeaderUserID: leaderID, Companions: wr.Companions, Formation: wr.Formation}
		if len(record.Companions) == 0 && wr.Companion != nil {
			legacy := *wr.Companion
			if legacy.ID == 0 {
				legacy.ID = 1
			}
			record.Companions = []domain.Companion{legacy}
		}
		loaded.Put(record)
	}
	*registry = *loaded
	return nil
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *domain.Registry) error {
	// ReadIntoStruct currently discards YAML decoding errors. Decode here so
	// unreadable company data cannot become an empty, writable registry.
	data, err := s.plug.ReadBytes("companies")
	if errors.Is(err, os.ErrNotExist) {
		*registry = *domain.NewRegistry()
		return nil
	}
	if err != nil {
		return err
	}
	return decodeCompanies(data, registry)
}
func (s pluginStore) Save(registry domain.Registry) error {
	return s.plug.WriteStruct("companies", registry)
}

type CompanyModule struct {
	plug      *plugins.Plugin
	store     Store
	registry  domain.Registry
	instances map[int]map[int]int
	runtime   Runtime
	loadErr   error
}

func init() {
	m := &CompanyModule{plug: plugins.New("company", "1.0"), instances: map[int]map[int]int{}}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.runtime = nativeRuntime{}
	m.plug.AddUserCommand("company", m.userCommand, false, false)
	m.plug.AddUserCommand("formation", m.formationCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("company: save", "error", err)
		}
	})
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.MobDeath{}, m.onMobDeath)
}

const companyUsage = "Usage: company summon <mob-id-or-name> | company status | company dismiss <member|all>"

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

func maxCompanionsFromConfig(raw any) int {
	n, ok := configInt(raw)
	if !ok || n < 1 {
		return domain.MaxCompanions
	}
	if n > domain.MaxCompanions {
		return domain.MaxCompanions
	}
	return n
}

func (m *CompanyModule) maxCompanions() int {
	if m.plug != nil {
		return maxCompanionsFromConfig(m.plug.Config.Get("MaxCompanions"))
	}
	return domain.MaxCompanions
}

func configInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

func (m *CompanyModule) instance(leaderUserID, companionID int) (int, bool) {
	if m.instances == nil {
		return 0, false
	}
	instanceID, ok := m.instances[leaderUserID][companionID]
	return instanceID, ok
}

func (m *CompanyModule) setInstance(leaderUserID, companionID, instanceID int) {
	if m.instances == nil {
		m.instances = map[int]map[int]int{}
	}
	if m.instances[leaderUserID] == nil {
		m.instances[leaderUserID] = map[int]int{}
	}
	m.instances[leaderUserID][companionID] = instanceID
}

func (m *CompanyModule) clearInstance(leaderUserID, companionID int) {
	if m.instances == nil {
		return
	}
	delete(m.instances[leaderUserID], companionID)
	if len(m.instances[leaderUserID]) == 0 {
		delete(m.instances, leaderUserID)
	}
}

func (m *CompanyModule) companionForInstance(instanceID int) (int, int, bool) {
	for leaderUserID, byCompanion := range m.instances {
		for companionID, tracked := range byCompanion {
			if tracked == instanceID {
				return leaderUserID, companionID, true
			}
		}
	}
	return 0, 0, false
}

func (m *CompanyModule) summon(leaderUserID, roomID int, selector string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return companyUsage, fmt.Errorf("company: companion selector is required")
	}
	templateID, err := m.resolveTemplateID(selector)
	if err != nil {
		return "", err
	}
	companion, err := m.registry.Summon(leaderUserID, templateID, m.allowedTemplates(), m.maxCompanions())
	if err != nil {
		return "", err
	}
	instanceID, err := m.runtime.Spawn(leaderUserID, roomID, templateID)
	if err != nil {
		m.registry.Dismiss(leaderUserID, companion.ID)
		return "", err
	}
	if err := m.save(); err != nil {
		m.runtime.Detach(leaderUserID, instanceID)
		m.registry.Dismiss(leaderUserID, companion.ID)
		return "", err
	}
	m.setInstance(leaderUserID, companion.ID, instanceID)
	return fmt.Sprintf("Companion summoned: %s (#%d).", templateName(templateID, selector), companion.ID), nil
}

func (m *CompanyModule) resolveTemplateID(selector string) (int, error) {
	if id, err := strconv.Atoi(selector); err == nil {
		return id, nil
	}
	if id, ok := m.runtime.ResolveTemplate(strings.ToLower(selector)); ok {
		return id, nil
	}
	return 0, fmt.Errorf("company: unknown mob template %q", selector)
}

func templateName(templateID int, fallback string) string {
	if spec := mobs.GetMobSpec(mobs.MobId(templateID)); spec != nil && spec.Character.Name != "" {
		return spec.Character.Name
	}
	return fallback
}

func (m *CompanyModule) status(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "No companions."
	}
	lines := []string{fmt.Sprintf("Company companions (%d/%d):", len(record.Companions), m.maxCompanions())}
	for _, c := range record.Companions {
		state := "awaiting restoration"
		if instanceID, tracked := m.instance(leaderUserID, c.ID); tracked {
			if m.runtime.IsAttached(leaderUserID, instanceID) {
				state = "present"
			} else if !m.runtime.IsLive(instanceID) {
				m.clearInstance(leaderUserID, c.ID)
			}
		}
		lines = append(lines, fmt.Sprintf("  #%d %s (%s)", c.ID, templateName(c.MobTemplateID, strconv.Itoa(c.MobTemplateID)), state))
	}
	return strings.Join(lines, "\n")
}

func (m *CompanyModule) dismiss(leaderUserID int, selector string) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	selector = strings.TrimSpace(strings.ToLower(selector))
	if selector == "all" {
		return m.dismissAll(leaderUserID)
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "No companions.", nil
	}
	companion, ok := resolveCompanion(record, selector)
	if !ok {
		return "", fmt.Errorf("company: no companion matches %q", selector)
	}
	if !m.registry.Dismiss(leaderUserID, companion.ID) {
		return "", fmt.Errorf("company: companion #%d is no longer in the company", companion.ID)
	}
	if err := m.save(); err != nil {
		m.registry.Put(record)
		return "", err
	}
	if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
		if m.runtime.IsLive(instanceID) {
			m.runtime.Detach(leaderUserID, instanceID)
		}
		m.clearInstance(leaderUserID, companion.ID)
	}
	return fmt.Sprintf("Companion dismissed: %s (#%d).", templateName(companion.MobTemplateID, strconv.Itoa(companion.MobTemplateID)), companion.ID), nil
}

func (m *CompanyModule) dismissAll(leaderUserID int) (string, error) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "No companions.", nil
	}
	count := m.registry.DismissAll(leaderUserID)
	if err := m.save(); err != nil {
		m.registry.Put(record)
		return "", err
	}
	for _, companion := range record.Companions {
		if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
			if m.runtime.IsLive(instanceID) {
				m.runtime.Detach(leaderUserID, instanceID)
			}
			m.clearInstance(leaderUserID, companion.ID)
		}
	}
	return fmt.Sprintf("Dismissed %d companion(s).", count), nil
}

// resolveCompanion matches a companion by #id, bare numeric ID, exact name, or name substring.
func resolveCompanion(record domain.Record, selector string) (domain.Companion, bool) {
	selector = strings.ToLower(strings.TrimSpace(selector))
	if id, err := strconv.Atoi(strings.TrimPrefix(selector, "#")); err == nil {
		for _, c := range record.Companions {
			if c.ID == id {
				return c, true
			}
		}
		return domain.Companion{}, false
	}
	var exact, partial []domain.Companion
	for _, c := range record.Companions {
		name := strings.ToLower(templateName(c.MobTemplateID, ""))
		if name == selector {
			exact = append(exact, c)
		} else if strings.Contains(name, selector) {
			partial = append(partial, c)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(partial) == 1 {
		return partial[0], true
	}
	return domain.Companion{}, false
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
			return true, err
		}
		user.SendText(text)
	case "status":
		user.SendText(m.status(user.UserId))
	case "dismiss":
		if len(args) < 2 {
			user.SendText(companyUsage)
			return true, nil
		}
		text, err := m.dismiss(user.UserId, strings.Join(args[1:], " "))
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
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return nil
	}
	var firstErr error
	for _, companion := range record.Companions {
		if instanceID, tracked := m.instance(leaderUserID, companion.ID); tracked {
			if m.runtime.IsAttached(leaderUserID, instanceID) {
				continue
			}
			if m.runtime.IsLive(instanceID) {
				m.runtime.Detach(leaderUserID, instanceID)
			}
			m.clearInstance(leaderUserID, companion.ID)
		}
		instanceID, err := m.runtime.Spawn(leaderUserID, roomID, companion.MobTemplateID)
		if err != nil {
			m.clearInstance(leaderUserID, companion.ID)
			if firstErr == nil {
				firstErr = fmt.Errorf("company: restore leader %d companion %d: %w", leaderUserID, companion.ID, err)
			}
			continue
		}
		m.setInstance(leaderUserID, companion.ID, instanceID)
	}
	return firstErr
}

func (m *CompanyModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("company: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("company: persistence unavailable")
	}
	return nil
}

func (m *CompanyModule) save() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if err := m.store.Save(m.registry); err != nil {
		return fmt.Errorf("company: save failed; please retry: %w", err)
	}
	return nil
}

func (m *CompanyModule) load() {
	if m.store == nil {
		return
	}
	loaded := domain.NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("company: load", "error", err)
		return
	}
	if loaded.Companies == nil {
		loaded = domain.NewRegistry()
	}
	m.registry = *loaded
	m.loadErr = nil
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
	if leaderUserID, companionID, found := m.companionForInstance(evt.InstanceId); found {
		m.clearInstance(leaderUserID, companionID)
	}
	return events.Continue
}
