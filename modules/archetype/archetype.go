// Package archetype owns Ashveil's exclusive character archetypes: the
// configured archetype table, each player's durable one-time choice, the
// grants applied on choosing, and the gating decisions the engine consults
// through internal/archetypes. Phase 17b adds utility skills (autoskill,
// traps, auto-light) in utility.go.
//
// Companion archetypes live on the company roster record
// (modules/company), so they are persisted and pruned with the roster.
package archetype

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const archetypeUsage = `Usage: archetype | archetype choose <name> [confirm]`

// Registry is the durable archetype state.
type Registry struct {
	// Players maps a user id to the archetype id they chose.
	Players map[int]string `yaml:"players"`
	// Autoskill maps a user id to per-utility toggles; a missing entry
	// means on (Phase 17b).
	Autoskill map[int]map[string]bool `yaml:"autoskill,omitempty"`
	// Disarmed maps a lock id to the round its trap re-arms at (Phase 17b).
	Disarmed map[string]uint64 `yaml:"disarmed,omitempty"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Players: map[int]string{}, Autoskill: map[int]map[string]bool{}, Disarmed: map[string]uint64{}}
}

// Clone returns a deep copy.
func (r Registry) Clone() Registry {
	out := *NewRegistry()
	for id, a := range r.Players {
		out.Players[id] = a
	}
	for id, toggles := range r.Autoskill {
		cp := make(map[string]bool, len(toggles))
		for k, v := range toggles {
			cp[k] = v
		}
		out.Autoskill[id] = cp
	}
	for lock, round := range r.Disarmed {
		out.Disarmed[lock] = round
	}
	return out
}

// Store abstracts durable persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *Registry) error {
	data, err := s.plug.ReadBytes("archetype")
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
	return s.plug.WriteStruct("archetype", registry)
}

// decodeRegistry parses stored bytes, dropping only entries keyed by an
// invalid user id or lock id. An archetype id that is no longer configured
// is kept (and logged at load) for operator repair.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for id, a := range wire.Players {
		if id <= 0 || strings.TrimSpace(a) == "" {
			continue
		}
		loaded.Players[id] = strings.ToLower(strings.TrimSpace(a))
	}
	for id, toggles := range wire.Autoskill {
		if id <= 0 || len(toggles) == 0 {
			continue
		}
		loaded.Autoskill[id] = toggles
	}
	for lock, round := range wire.Disarmed {
		if strings.TrimSpace(lock) == "" {
			continue
		}
		loaded.Disarmed[lock] = round
	}
	*registry = *loaded
	return nil
}

// ArchetypeModule owns archetype state for one plugin.
type ArchetypeModule struct {
	plug  *plugins.Plugin
	store Store

	// Dependency seams (native defaults; tests replace them).
	schoolOf    func(spellID string) (string, bool)
	skillExists func(skillID string) bool

	table    archetypes.Table
	registry *Registry
	loadErr  error
	config   utilityConfig

	mu sync.Mutex
}

var _ archetypes.Provider = (*ArchetypeModule)(nil)

func nativeSchoolOf(spellID string) (string, bool) {
	s := spells.GetSpell(spellID)
	if s == nil {
		return "", false
	}
	return string(s.School), true
}

func newModule() *ArchetypeModule {
	return &ArchetypeModule{
		schoolOf:    nativeSchoolOf,
		skillExists: skills.SkillExists,
		registry:    NewRegistry(),
		config:      defaultUtilityConfig(),
	}
}

func init() {
	m := newModule()
	m.plug = plugins.New("archetype", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("archetype", m.userCommand, true, false)
	m.plug.AddUserCommand("archetypereset", m.adminResetCommand, true, true)
	m.registerUtility()
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("archetype: save", "error", err)
		}
	})
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	archetypes.SetProvider(m)
}

func (m *ArchetypeModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("archetype: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("archetype: persistence unavailable")
	}
	return nil
}

func (m *ArchetypeModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *ArchetypeModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if err := m.store.Save(m.registry.Clone()); err != nil {
		return fmt.Errorf("archetype: save failed; please retry: %w", err)
	}
	return nil
}

// load parses the archetype table (after the engine loaded skills and
// spells, so references can be cross-checked) and the durable registry.
func (m *ArchetypeModule) load() {
	var rawTable, rawUtilities any
	var cfg utilityConfig
	if m.plug != nil {
		rawTable = m.plug.Config.Get("Archetypes")
		rawUtilities = m.plug.Config.Get("Utilities")
		cfg = parseUtilityConfig(m.plug.Config.Get)
	} else {
		cfg = defaultUtilityConfig()
	}
	table := m.buildTable(parseArchetypes(rawTable))
	cfg.UtilitySkills = parseUtilitySkills(rawUtilities, cfg.UtilitySkills)

	loaded := NewRegistry()
	var loadErr error
	if m.store != nil {
		if err := m.store.Load(loaded); err != nil {
			loadErr = err
			mudlog.Error("archetype: load", "error", err)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.table = table
	m.config = cfg
	m.loadErr = loadErr
	if loadErr != nil {
		return
	}
	m.registry = loaded
	for userID, id := range m.registry.Players {
		if _, ok := m.table.Get(id); !ok {
			mudlog.Warn("archetype: recovery unknown archetype", "user", userID, "archetype", id)
		}
	}
}

// buildTable validates the parsed list, then drops archetypes whose skill
// or spell references don't resolve against the loaded engine data.
func (m *ArchetypeModule) buildTable(list []archetypes.Archetype) archetypes.Table {
	valid := make([]archetypes.Archetype, 0, len(list))
	for _, a := range list {
		if err := a.Validate(); err != nil {
			mudlog.Warn("archetype: invalid archetype", "error", err)
			continue
		}
		unknown := false
		for _, skill := range a.Skills {
			if m.skillExists != nil && !m.skillExists(skill) {
				mudlog.Warn("archetype: unknown skill", "archetype", a.ID, "skill", skill)
				unknown = true
			}
		}
		if unknown {
			continue
		}
		if m.schoolOf != nil {
			if err := a.ValidateGrantedSpells(m.schoolOf); err != nil {
				mudlog.Warn("archetype: invalid spell grant", "error", err)
				continue
			}
		}
		valid = append(valid, a)
	}
	table, errs := archetypes.NewTable(valid)
	for _, err := range errs {
		mudlog.Warn("archetype: invalid archetype", "error", err)
	}
	return table
}

// --- archetypes.Provider --------------------------------------------------

func (m *ArchetypeModule) CanTrain(userID int, skillID string) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.table.CanTrain(m.registry.Players[userID], skillID)
}

func (m *ArchetypeModule) CanLearnSpell(userID int, spellID string) (bool, string) {
	school := ""
	if m.schoolOf != nil {
		s, ok := m.schoolOf(spellID)
		if !ok {
			// Unknown spells are left to the engine's own handling.
			return true, ""
		}
		school = s
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.table.CanLearnSchool(m.registry.Players[userID], school)
}

// SkillClaimantNames implements archetypes.ClaimantNamer.
func (m *ArchetypeModule) SkillClaimantNames(skillID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.table.SkillClaimantNames(skillID)
}

func (m *ArchetypeModule) Exists(archetypeID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.table.Get(archetypeID)
	return ok
}

func (m *ArchetypeModule) ArchetypeName(archetypeID string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.table.Get(archetypeID)
	return a.Name, ok
}

func (m *ArchetypeModule) PlayerArchetype(userID int) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.registry.Players[userID]
	return id, ok
}

// --- choice and grants ------------------------------------------------------

// applyGrants gives the user the archetype's starting skills and spells. It
// never lowers a higher existing skill level and is idempotent, so it is
// safe to re-apply on every login (crash recovery). It runs on the game
// loop and outside the module lock.
func applyGrants(user *users.UserRecord, a archetypes.Archetype) {
	if user == nil {
		return
	}
	skillIDs := make([]string, 0, len(a.GrantSkills))
	for id := range a.GrantSkills {
		skillIDs = append(skillIDs, id)
	}
	sort.Strings(skillIDs)
	for _, id := range skillIDs {
		if user.Character.GetSkillLevel(id) < a.GrantSkills[id] {
			user.Character.SetSkill(id, a.GrantSkills[id])
		}
	}
	for _, spell := range a.GrantSpells {
		user.Character.LearnSpell(spell)
	}
}

// choose records a first archetype choice. Without confirm it only previews.
func (m *ArchetypeModule) choose(user *users.UserRecord, name string, confirm bool) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return archetypeUsage
	}
	m.mu.Lock()
	a, ok := m.table.Get(name)
	if !ok {
		m.mu.Unlock()
		return fmt.Sprintf(`There is no archetype called "%s". Type "archetype" to see them.`, name)
	}
	if current, chosen := m.registry.Players[user.UserId]; chosen {
		m.mu.Unlock()
		currentName := current
		if c, known := m.table.Get(current); known {
			currentName = c.Name
		}
		return fmt.Sprintf("You are already a %s. The choice is permanent.", currentName)
	}
	if !confirm {
		m.mu.Unlock()
		return fmt.Sprintf("%s: %s\nThis choice is permanent. Type \"archetype choose %s confirm\" to become a %s.", a.Name, a.Description, a.ID, a.Name)
	}
	m.registry.Players[user.UserId] = a.ID
	if err := m.saveLocked(); err != nil {
		delete(m.registry.Players, user.UserId)
		m.mu.Unlock()
		return err.Error()
	}
	m.mu.Unlock()
	applyGrants(user, a)
	return fmt.Sprintf("You are now a %s.", a.Name)
}

// reset clears a player's choice (admin only). Granted skills and spells
// are kept: nothing already known is ever removed.
func (m *ArchetypeModule) reset(userID int) (string, error) {
	if err := m.persistenceAvailable(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	original, ok := m.registry.Players[userID]
	if !ok {
		return "That character has no archetype.", nil
	}
	delete(m.registry.Players, userID)
	if err := m.saveLocked(); err != nil {
		m.registry.Players[userID] = original
		return "", err
	}
	return "Archetype cleared.", nil
}

func (m *ArchetypeModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	id, chosen := m.registry.Players[evt.UserId]
	a, known := m.table.Get(id)
	m.mu.Unlock()
	if chosen && known {
		applyGrants(users.GetByUserId(evt.UserId), a)
	}
	return events.Continue
}

// list renders the archetype table and the user's current choice.
func (m *ArchetypeModule) list(userID int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	lines := []string{}
	if id, ok := m.registry.Players[userID]; ok {
		name := id
		if a, known := m.table.Get(id); known {
			name = a.Name
		}
		lines = append(lines, fmt.Sprintf("You are a %s.", name))
	} else {
		lines = append(lines, `You have not chosen an archetype. Use "archetype choose <name>".`)
	}
	if m.table.Len() == 0 {
		lines = append(lines, "No archetypes are configured.")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Archetypes:")
	for _, a := range m.table.List() {
		line := fmt.Sprintf("  %-8s %s Skills: %s.", a.Name, a.Description, strings.Join(a.Skills, ", "))
		if len(a.Schools) > 0 {
			line += " Spell schools: " + strings.Join(a.Schools, ", ") + "."
		}
		lines = append(lines, line)
	}
	lines = append(lines, "Skills no archetype claims are open to everyone.")
	return strings.Join(lines, "\n")
}

func (m *ArchetypeModule) userCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(rest))
	if len(args) == 0 || args[0] == "list" || args[0] == "status" {
		user.SendText(m.list(user.UserId))
		return true, nil
	}
	switch args[0] {
	case "choose":
		confirm := len(args) > 2 && args[len(args)-1] == "confirm"
		nameArgs := args[1:]
		if confirm {
			nameArgs = args[1 : len(args)-1]
		}
		user.SendText(m.choose(user, strings.Join(nameArgs, " "), confirm))
	default:
		user.SendText(archetypeUsage)
	}
	return true, nil
}

// adminResetCommand: archetypereset <character name> (online characters).
func (m *ArchetypeModule) adminResetCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	name := strings.TrimSpace(rest)
	if name == "" {
		user.SendText("Usage: archetypereset <character>")
		return true, nil
	}
	target := users.GetByCharacterName(name)
	if target == nil {
		user.SendText(fmt.Sprintf(`No online character called "%s".`, name))
		return true, nil
	}
	text, err := m.reset(target.UserId)
	if err != nil {
		return true, err
	}
	user.SendText(text)
	return true, nil
}

// --- config parsing ---------------------------------------------------------

// parseArchetypes reads the configured archetype list. Validation happens
// in buildTable; this only maps fields.
func parseArchetypes(raw any) []archetypes.Archetype {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]archetypes.Archetype, 0, len(list))
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			continue
		}
		a := archetypes.Archetype{
			ID:              configString(fields["archetypeid"]),
			Name:            configString(fields["name"]),
			Description:     configString(fields["description"]),
			Skills:          stringList(fields["skills"]),
			Schools:         stringList(fields["schools"]),
			GrantSpells:     stringList(fields["grantspells"]),
			Utility:         stringList(fields["utility"]),
			CompanionLevels: intList(fields["companionlevels"]),
			GrantSkills:     map[string]int{},
		}
		if grants, ok := fields["grantskills"].([]any); ok {
			for _, g := range grants {
				gf := stringMap(g)
				if gf == nil {
					continue
				}
				a.GrantSkills[configString(gf["skill"])] = configInt(gf["level"])
			}
		}
		out = append(out, a)
	}
	return out
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

func stringList(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func intList(raw any) []int {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(list))
	for _, v := range list {
		out = append(out, configInt(v))
	}
	return out
}

func configInt(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case uint64:
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

func configString(raw any) string {
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}
