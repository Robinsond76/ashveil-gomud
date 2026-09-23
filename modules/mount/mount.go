// Package mount owns the durable leader-keyed mount assignment, the
// configured mount-type table, and the player-facing mount command. It
// registers itself as internal/mount's Provider so modules/encumbrance can
// consult a leader's mount cargo-capacity bonus without importing this
// package.
package mount

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const mountUsage = `Usage: mount | mount stable <type> | mount release`

// Registry is the durable, leader-keyed set of mount assignments.
type Registry struct {
	Mounts map[int]mount.Mount `yaml:"mounts"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Mounts: map[int]mount.Mount{}}
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Mounts: make(map[int]mount.Mount, len(r.Mounts))}
	for leaderUserID, m := range r.Mounts {
		out.Mounts[leaderUserID] = m
	}
	return out
}

// Store abstracts durable registry persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *Registry) error {
	data, err := s.plug.ReadBytes("mount")
	if errors.Is(err, os.ErrNotExist) {
		*registry = *NewRegistry()
		return nil
	}
	if err != nil {
		return err
	}
	return decodeRegistry(data, registry)
}

func (s pluginStore) Save(registry Registry) error {
	return s.plug.WriteStruct("mount", registry)
}

// decodeRegistry parses stored bytes, dropping only entries keyed by an
// invalid leader ID. Invalid mount content is retained for operator repair.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for leaderUserID, m := range wire.Mounts {
		if leaderUserID <= 0 {
			continue
		}
		m.LeaderUserID = leaderUserID
		loaded.Mounts[leaderUserID] = m
	}
	*registry = *loaded
	return nil
}

// MountModule owns durable leader-keyed mount assignments for one plugin.
type MountModule struct {
	plug  *plugins.Plugin
	store Store

	specs   map[string]mount.MountSpec
	mounts  map[int]mount.Mount
	loadErr error

	mu sync.Mutex
}

var (
	_ mount.Provider       = (*MountModule)(nil)
	_ mount.ReliefProvider = (*MountModule)(nil)
)

func init() {
	m := &MountModule{
		plug:   plugins.New("mount", "1.0"),
		specs:  map[string]mount.MountSpec{},
		mounts: map[int]mount.Mount{},
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("mount", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("mount: save", "error", err)
		}
	})
	mount.SetProvider(m)
}

func (m *MountModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("mount: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("mount: persistence unavailable")
	}
	return nil
}

func (m *MountModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *MountModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	registry := Registry{Mounts: m.mounts}
	if err := m.store.Save(registry); err != nil {
		return fmt.Errorf("mount: save failed; please retry: %w", err)
	}
	return nil
}

// load parses the mount-type configuration, loads the durable registry, and
// logs (without discarding) any invalid or now-unrecognized mount record for
// operator repair. Like modules/encumbrance, there is no real-time or
// round-driven state to reschedule here.
func (m *MountModule) load() {
	if m.store == nil {
		return
	}
	loaded := NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("mount: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mounts = loaded.Mounts
	if m.mounts == nil {
		m.mounts = map[int]mount.Mount{}
	}
	m.loadErr = nil
	if m.plug != nil {
		m.specs = parseMountSpecs(m.plug.Config.Get("Mounts"))
	}
	for leaderUserID, rec := range m.mounts {
		if err := rec.Validate(); err != nil {
			mudlog.Warn("mount: recovery invalid mount", "leader", leaderUserID, "error", err)
			continue
		}
		if _, known := m.specs[rec.Type]; !known {
			mudlog.Warn("mount: recovery unknown mount type", "leader", leaderUserID, "type", rec.Type)
		}
	}
}

// CapacityBonusGrams implements mount.Provider. A leader with no tracked
// mount, or one whose type no longer has a configured spec, contributes 0
// rather than a guessed default.
func (m *MountModule) CapacityBonusGrams(leaderUserID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.mounts[leaderUserID]
	if !ok {
		return 0
	}
	spec, known := m.specs[rec.Type]
	if !known {
		return 0
	}
	return spec.CargoCapacityBonusGrams
}

// currentSpecLocked is the configured spec of the leader's mount.
func (m *MountModule) currentSpecLocked(leaderUserID int) (mount.MountSpec, bool) {
	rec, ok := m.mounts[leaderUserID]
	if !ok {
		return mount.MountSpec{}, false
	}
	spec, known := m.specs[rec.Type]
	return spec, known
}

// Relief implements mount.ReliefProvider: (100, 0) without a recognised
// mount.
func (m *MountModule) Relief(leaderUserID int) (fatiguePct, riders int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	spec, ok := m.currentSpecLocked(leaderUserID)
	if !ok {
		return 100, 0
	}
	return spec.EffectiveFatiguePct(), spec.EffectiveRiders()
}

// TravelDurationPct implements mount.ReliefProvider: 100 without a
// recognised mount.
func (m *MountModule) TravelDurationPct(leaderUserID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	spec, ok := m.currentSpecLocked(leaderUserID)
	if !ok {
		return 100
	}
	return spec.EffectiveTravelDurationPct()
}

// stable assigns a configured mount type to the leader, replacing any
// existing assignment.
func (m *MountModule) stable(user *users.UserRecord, mountType string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	mountType = strings.TrimSpace(mountType)
	if mountType == "" {
		return mountUsage
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	spec, ok := m.specs[mountType]
	if !ok {
		return fmt.Sprintf(`There is no mount type called "%s".`, mountType)
	}
	rec, err := mount.Established(user.UserId, spec.Type)
	if err != nil {
		return "You can't stable that mount."
	}
	original, hadEntry := m.mounts[user.UserId]
	m.mounts[user.UserId] = rec
	if err := m.saveLocked(); err != nil {
		if hadEntry {
			m.mounts[user.UserId] = original
		} else {
			delete(m.mounts, user.UserId)
		}
		return err.Error()
	}
	return fmt.Sprintf("You stable a %s.", spec.Type)
}

// release removes the leader's current mount assignment.
func (m *MountModule) release(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	original, ok := m.mounts[leaderUserID]
	if !ok {
		return "You have no mount to release."
	}
	delete(m.mounts, leaderUserID)
	if err := m.saveLocked(); err != nil {
		m.mounts[leaderUserID] = original
		return err.Error()
	}
	return "You release your mount."
}

// status renders the leader's current mount assignment.
func (m *MountModule) status(leaderUserID int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.mounts[leaderUserID]
	if !ok {
		return `You have no mount. Use "mount stable <type>" to get one.`
	}
	spec, known := m.specs[rec.Type]
	if !known {
		return fmt.Sprintf("You have a %s, but it's no longer a recognized mount type.", rec.Type)
	}
	return fmt.Sprintf("You have a %s. %s (+%.1f kg cargo capacity)", rec.Type, spec.Description, float64(spec.CargoCapacityBonusGrams)/1000)
}

func (m *MountModule) userCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		user.SendText(m.status(user.UserId))
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "stable":
		user.SendText(m.stable(user, strings.Join(args[1:], " ")))
	case "release":
		user.SendText(m.release(user.UserId))
	case "status":
		user.SendText(m.status(user.UserId))
	default:
		user.SendText(mountUsage)
	}
	return true, nil
}

// parseMountSpecs normalizes the configured mount-type list, rejecting a
// malformed entry rather than applying a guess, matching
// modules/weather's parseBiomeTables and modules/encumbrance's parseConfig.
func parseMountSpecs(raw any) map[string]mount.MountSpec {
	specs := map[string]mount.MountSpec{}
	list, ok := raw.([]any)
	if !ok {
		return specs
	}
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			continue
		}
		spec := mount.MountSpec{
			Type:                    strings.TrimSpace(configString(fields["type"])),
			Description:             configString(fields["description"]),
			CargoCapacityBonusGrams: int(configFloat(fields["cargocapacitybonuskg"]) * 1000),
			TravelDurationPct:       configInt(fields["traveldurationpct"]),
			FatiguePct:              configInt(fields["fatiguepct"]),
			Riders:                  configInt(fields["riders"]),
		}
		if err := spec.Validate(); err != nil {
			mudlog.Warn("mount: invalid mount spec", "type", spec.Type, "error", err)
			continue
		}
		if _, exists := specs[spec.Type]; exists {
			mudlog.Warn("mount: duplicate mount type", "type", spec.Type)
			continue
		}
		specs[spec.Type] = spec
	}
	return specs
}

func stringMap(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[strings.ToLower(key)] = item
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if name, ok := key.(string); ok {
				out[strings.ToLower(name)] = item
			}
		}
		return out
	}
	return nil
}

func configInt(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return n
		}
	}
	return 0
}

func configFloat(raw any) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func configString(raw any) string {
	value, _ := raw.(string)
	return value
}
