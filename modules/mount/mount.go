// Package mount owns each company's durable herd of horses, the configured
// horse types, and the player-facing mount command. It registers itself as
// internal/mount's Provider so modules/encumbrance, modules/walking, and
// modules/expedition can read the horses' capacity, riders, and pace
// without importing this package.
//
// Phase 32f: a company keeps up to one riding horse and one pack horse per
// member, bought at a stable. A pack horse carries load; a riding horse
// carries one member; each needs its own kind of saddle to do its job.
package mount

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const mountUsage = `Usage: mount | mount stable [type] | mount saddle <horse> <saddle> | mount unsaddle <horse> | mount release <horse>`

// defaultStableTag is the room tag where horses are sold when config sets
// none.
const defaultStableTag = "stable"

// legacyMount is Phase 10's one mount per company, read from old saves
// and turned into a herd of one on load.
type legacyMount struct {
	LeaderUserID int    `yaml:"leaderuserid"`
	Type         string `yaml:"type"`
}

// Registry is the durable, leader-keyed set of herds. Mounts holds an old
// save's single mounts until load migrates them; it is never written again.
type Registry struct {
	Herds  map[int]mount.Herd  `yaml:"herds,omitempty"`
	Mounts map[int]legacyMount `yaml:"mounts,omitempty"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Herds: map[int]mount.Herd{}}
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Herds: make(map[int]mount.Herd, len(r.Herds))}
	for leaderUserID, herd := range r.Herds {
		out.Herds[leaderUserID] = herd.Clone()
	}
	if len(r.Mounts) > 0 {
		out.Mounts = make(map[int]legacyMount, len(r.Mounts))
		for leaderUserID, m := range r.Mounts {
			out.Mounts[leaderUserID] = m
		}
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
// invalid leader ID. Invalid content is retained for operator repair.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for leaderUserID, herd := range wire.Herds {
		if leaderUserID <= 0 {
			continue
		}
		herd.LeaderUserID = leaderUserID
		loaded.Herds[leaderUserID] = herd
	}
	for leaderUserID, m := range wire.Mounts {
		if leaderUserID <= 0 {
			continue
		}
		if loaded.Mounts == nil {
			loaded.Mounts = map[int]legacyMount{}
		}
		m.LeaderUserID = leaderUserID
		loaded.Mounts[leaderUserID] = m
	}
	*registry = *loaded
	return nil
}

// MountModule owns durable leader-keyed herds for one plugin.
type MountModule struct {
	plug  *plugins.Plugin
	store Store
	// members is how many members a leader's company counts (the leader and
	// each counted companion); it caps the herd. Game loop only.
	members func(leaderUserID int) int

	specs map[string]mount.MountSpec
	// stableTag is the room tag where horses are sold; legacySaddle the
	// pack saddle fitted to an old save's single mount on migration.
	stableTag    string
	legacySaddle int
	herds        map[int]mount.Herd
	loadErr      error
	// saveUser writes the leader right after a herd change moves gold or a
	// saddle (32f review), so a crash can't split the two; nil skips.
	saveUser func(*users.UserRecord) error

	mu sync.Mutex
}

var (
	_ mount.Provider       = (*MountModule)(nil)
	_ mount.ReliefProvider = (*MountModule)(nil)
	_ mount.HerdProvider   = (*MountModule)(nil)
)

func init() {
	m := &MountModule{
		plug:      plugins.New("mount", "1.0"),
		members:   company.CountedMembers,
		specs:     map[string]mount.MountSpec{},
		stableTag: defaultStableTag,
		herds:     map[int]mount.Herd{},
		saveUser:  func(user *users.UserRecord) error { return users.SaveUser(*user) },
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("mount", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("mount: save", "error", err)
		}
	})
	mount.SetProvider(m)
	registered = m
}

// registered is the module instance init registers, for wiring tests.
var registered *MountModule

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
	if err := m.store.Save(Registry{Herds: m.herds}); err != nil {
		return fmt.Errorf("mount: save failed; please retry: %w", err)
	}
	return nil
}

// load parses the configuration, loads the durable registry, turns an old
// save's single mount into a herd of one (with the legacy pack saddle, so
// no one loses capacity), and logs (without discarding) any invalid herd
// or unknown horse type for operator repair. Like modules/encumbrance,
// there is no real-time or round-driven state to reschedule here.
func (m *MountModule) load() {
	if m.store == nil {
		return
	}
	if m.plug != nil {
		m.mu.Lock()
		m.specs = parseMountSpecs(m.plug.Config.Get("Mounts"))
		m.stableTag = defaultStableTag
		if tag := strings.TrimSpace(configString(m.plug.Config.Get("StableRoomTag"))); tag != "" {
			m.stableTag = tag
		}
		m.legacySaddle = configInt(m.plug.Config.Get("LegacySaddleItemId"))
		m.mu.Unlock()
	}
	loaded := NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("mount: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.herds = loaded.Herds
	if m.herds == nil {
		m.herds = map[int]mount.Herd{}
	}
	m.loadErr = nil
	for leaderUserID, old := range loaded.Mounts {
		if _, migrated := m.herds[leaderUserID]; migrated || old.Type == "" {
			continue
		}
		m.herds[leaderUserID] = mount.Herd{
			LeaderUserID: leaderUserID,
			Horses:       []mount.Horse{{ID: 1, Type: old.Type, SaddleItemId: max(m.legacySaddle, 0)}},
			NextID:       2,
		}
		mudlog.Info("mount: migrated a single mount to a herd", "leader", leaderUserID, "type", old.Type)
	}
	for leaderUserID, herd := range m.herds {
		if err := herd.Validate(); err != nil {
			mudlog.Warn("mount: recovery invalid herd", "leader", leaderUserID, "error", err)
			continue
		}
		for _, horse := range herd.Horses {
			if _, known := m.specs[horse.Type]; !known {
				mudlog.Warn("mount: recovery unknown horse type", "leader", leaderUserID, "horse", horse.ID, "type", horse.Type)
			}
		}
	}
}

func (m *MountModule) specsLocked() map[string]mount.MountSpec {
	return m.specs
}

// CapacityBonusGrams implements mount.Provider: what the leader's horses
// carry, each saddled or bare. A horse whose type is no longer configured
// carries nothing rather than a guessed default.
func (m *MountModule) CapacityBonusGrams(leaderUserID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.herds[leaderUserID].CapacityGrams(m.specsLocked())
}

// Relief implements mount.ReliefProvider: one rider per saddled riding
// horse; (100, 0) with none.
func (m *MountModule) Relief(leaderUserID int) (fatiguePct, riders int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.herds[leaderUserID].Relief(m.specsLocked())
}

// TravelDurationPct implements mount.ReliefProvider: the riding pace only
// when every counted member has a saddled riding horse; 100 otherwise.
func (m *MountModule) TravelDurationPct(leaderUserID int) int {
	members := m.memberCount(leaderUserID)
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.herds[leaderUserID].TravelDurationPct(m.specsLocked(), members)
}

// Herd implements mount.HerdProvider.
func (m *MountModule) Herd(leaderUserID int) []mount.HorseView {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.viewsLocked(leaderUserID)
}

func (m *MountModule) viewsLocked(leaderUserID int) []mount.HorseView {
	herd, ok := m.herds[leaderUserID]
	if !ok {
		return nil
	}
	views := []mount.HorseView{}
	for _, horse := range herd.Sorted() {
		view := mount.HorseView{ID: horse.ID, Name: strings.ReplaceAll(horse.Type, "-", " ")}
		if spec, known := m.specs[horse.Type]; known {
			view.Name = spec.Name()
			view.Kind = spec.Kind
			view.CapacityGrams = spec.CapacityGrams(horse.Saddled())
		}
		if horse.Saddled() {
			view.Saddle = itemName(horse.SaddleItemId)
		}
		views = append(views, view)
	}
	return views
}

// memberCount is the company's counted members; at least one (the leader).
// Read outside the module lock: it reads the company on the game loop.
func (m *MountModule) memberCount(leaderUserID int) int {
	if m.members == nil {
		return 1
	}
	return max(m.members(leaderUserID), 1)
}

func itemName(itemID int) string {
	if spec := items.GetItemSpec(itemID); spec != nil {
		return spec.Name
	}
	return "saddle"
}

func (m *MountModule) isStable(room *rooms.Room) bool {
	if room == nil {
		return false
	}
	m.mu.Lock()
	tag := m.stableTag
	m.mu.Unlock()
	for _, t := range room.GetTags() {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// stableList names what a stable sells, cheapest first.
func (m *MountModule) stableList() string {
	m.mu.Lock()
	specs := make([]mount.MountSpec, 0, len(m.specs))
	for _, spec := range m.specs {
		specs = append(specs, spec)
	}
	m.mu.Unlock()
	if len(specs) == 0 {
		return "The stable has no horses for sale."
	}
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].Price != specs[j].Price {
			return specs[i].Price < specs[j].Price
		}
		return specs[i].Type < specs[j].Type
	})
	lines := []string{"Horses for sale:"}
	for _, spec := range specs {
		lines = append(lines, fmt.Sprintf(`  <ansi fg="command">%s</ansi> (%s horse), <ansi fg="gold">%d gold</ansi>. %s`, spec.Type, spec.Kind, spec.Price, spec.Description))
	}
	lines = append(lines, `Buy one with <ansi fg="command">mount stable [type]</ansi>.`)
	return strings.Join(lines, "\n")
}

// stable buys a configured horse type in a stable room: one riding and one
// pack horse per counted member. Gold is taken only once the herd is saved.
func (m *MountModule) stable(user *users.UserRecord, room *rooms.Room, horseType string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if !m.isStable(room) {
		return `There's no stable here. Horses are bought at a stable (<ansi fg="command">help mount</ansi>).`
	}
	horseType = strings.ToLower(strings.TrimSpace(horseType))
	if horseType == "" {
		return m.stableList()
	}
	members := m.memberCount(user.UserId)

	m.mu.Lock()
	defer m.mu.Unlock()
	spec, ok := m.specs[horseType]
	if !ok {
		return fmt.Sprintf(`There is no horse type called "%s".`, horseType)
	}
	original, hadHerd := m.herds[user.UserId]
	herd := original.Clone()
	herd.LeaderUserID = user.UserId
	if err := herd.CanStable(spec.Kind, m.specs, members); err != nil {
		return fmt.Sprintf("Your company can keep one %s horse per member: %d already for %d.", spec.Kind, herd.CountKind(spec.Kind, m.specs), members)
	}
	if user.Character.Gold < spec.Price {
		return fmt.Sprintf(`A %s costs <ansi fg="gold">%d gold</ansi>, and you don't have enough.`, spec.Name(), spec.Price)
	}
	herd, horse := herd.Add(spec.Type)
	m.herds[user.UserId] = herd
	if err := m.saveLocked(); err != nil {
		if hadHerd {
			m.herds[user.UserId] = original
		} else {
			delete(m.herds, user.UserId)
		}
		return err.Error()
	}
	if spec.Price > 0 {
		user.Character.Gold -= spec.Price
		events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: -spec.Price})
		m.saveLeader(user)
	}
	return fmt.Sprintf(`You buy a %s (#%d) for <ansi fg="gold">%d gold</ansi>. Fit it with a %s saddle to put it to full use (<ansi fg="command">mount saddle #%d [saddle]</ansi>).`,
		spec.Name(), horse.ID, spec.Price, spec.Kind, horse.ID)
}

// release lets a horse go. Its saddle comes back to the leader's pack.
func (m *MountModule) release(user *users.UserRecord, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	original, ok := m.herds[user.UserId]
	if !ok || len(original.Horses) == 0 {
		return "You have no horses to release."
	}
	horse, found := original.Find(selector, m.specs)
	if !found {
		return fmt.Sprintf(`You have no horse "%s". Type <ansi fg="command">mount</ansi> to see your horses.`, strings.TrimSpace(selector))
	}
	herd, _, _ := original.Remove(horse.ID)
	m.herds[user.UserId] = herd
	if err := m.saveLocked(); err != nil {
		m.herds[user.UserId] = original
		return err.Error()
	}
	name := m.horseNameLocked(horse)
	if horse.Saddled() {
		m.giveSaddle(user, horse.SaddleItemId)
		m.saveLeader(user)
		return fmt.Sprintf("You release your %s (#%d), keeping its %s.", name, horse.ID, itemName(horse.SaddleItemId))
	}
	return fmt.Sprintf("You release your %s (#%d).", name, horse.ID)
}

// saddle fits a saddle from the leader's pack onto a horse of its kind; any
// saddle already on the horse comes back to the pack. Moving a saddle
// within the company never changes its load, so no capacity check applies.
func (m *MountModule) saddle(user *users.UserRecord, selector, saddleName string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if strings.TrimSpace(selector) == "" || strings.TrimSpace(saddleName) == "" {
		return mountUsage
	}
	saddleItem, found := user.Character.FindInBackpack(saddleName)
	if !found {
		return fmt.Sprintf(`You don't have a "%s".`, strings.TrimSpace(saddleName))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	original, ok := m.herds[user.UserId]
	if !ok || len(original.Horses) == 0 {
		return "You have no horses."
	}
	horse, found := original.Find(selector, m.specs)
	if !found {
		return fmt.Sprintf(`You have no horse "%s". Type <ansi fg="command">mount</ansi> to see your horses.`, strings.TrimSpace(selector))
	}
	spec, known := m.specs[horse.Type]
	if !known {
		return "That horse isn't a type anyone knows how to saddle."
	}
	kind := saddleItem.SaddleKind()
	if kind == "" {
		return fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> isn't a saddle.`, saddleItem.DisplayName())
	}
	if string(kind) != string(spec.Kind) {
		return fmt.Sprintf(`The <ansi fg="itemname">%s</ansi> is a %s saddle; your %s needs a %s saddle.`, saddleItem.DisplayName(), kind, spec.Name(), spec.Kind)
	}
	herd, old, _ := original.SetSaddle(horse.ID, saddleItem.ItemId)
	m.herds[user.UserId] = herd
	if err := m.saveLocked(); err != nil {
		m.herds[user.UserId] = original
		return err.Error()
	}
	user.Character.RemoveItem(saddleItem)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: saddleItem, Gained: false})
	text := fmt.Sprintf(`You fit the <ansi fg="itemname">%s</ansi> to your %s (#%d). It carries %s.`, saddleItem.DisplayName(), spec.Name(), horse.ID, carries(spec, true))
	if old > 0 {
		m.giveSaddle(user, old)
		text += fmt.Sprintf(" You keep the %s it wore.", itemName(old))
	}
	m.saveLeader(user)
	return text
}

// unsaddle takes a horse's saddle off, back into the leader's pack.
func (m *MountModule) unsaddle(user *users.UserRecord, selector string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	original, ok := m.herds[user.UserId]
	if !ok || len(original.Horses) == 0 {
		return "You have no horses."
	}
	horse, found := original.Find(selector, m.specs)
	if !found {
		return fmt.Sprintf(`You have no horse "%s". Type <ansi fg="command">mount</ansi> to see your horses.`, strings.TrimSpace(selector))
	}
	if !horse.Saddled() {
		return fmt.Sprintf("Your %s (#%d) has no saddle.", m.horseNameLocked(horse), horse.ID)
	}
	herd, old, _ := original.SetSaddle(horse.ID, 0)
	m.herds[user.UserId] = herd
	if err := m.saveLocked(); err != nil {
		m.herds[user.UserId] = original
		return err.Error()
	}
	m.giveSaddle(user, old)
	m.saveLeader(user)
	return fmt.Sprintf("You take the %s off your %s (#%d).", itemName(old), m.horseNameLocked(horse), horse.ID)
}

// saveLeader writes the leader after the herd save, as a paid recruit does
// (modules/company). A failed save leaves the change in memory for the next
// autosave, logout, or copyover save.
func (m *MountModule) saveLeader(user *users.UserRecord) {
	if m.saveUser == nil {
		return
	}
	if err := m.saveUser(user); err != nil {
		mudlog.Error("mount: save leader after herd change", "user", user.UserId, "error", err)
	}
}

// giveSaddle puts a saddle item into the leader's pack.
func (m *MountModule) giveSaddle(user *users.UserRecord, itemID int) {
	itm := items.New(itemID)
	if itm.ItemId == 0 {
		mudlog.Warn("mount: saddle has no item spec", "leader", user.UserId, "itemid", itemID)
		return
	}
	user.Character.StoreItem(itm)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: itm, Gained: true})
}

func (m *MountModule) horseNameLocked(horse mount.Horse) string {
	if spec, ok := m.specs[horse.Type]; ok {
		return spec.Name()
	}
	return strings.ReplaceAll(horse.Type, "-", " ")
}

// carries describes what a horse of this spec carries.
func carries(spec mount.MountSpec, saddled bool) string {
	kg := fmt.Sprintf("%.1f kg", float64(spec.CapacityGrams(saddled))/1000)
	if spec.Kind == mount.KindRiding && saddled {
		return "a rider and " + kg
	}
	return kg
}

// status lists the leader's horses and the room left in the herd.
func (m *MountModule) status(leaderUserID int) string {
	members := m.memberCount(leaderUserID)
	m.mu.Lock()
	defer m.mu.Unlock()
	herd, ok := m.herds[leaderUserID]
	if !ok || len(herd.Horses) == 0 {
		return `You have no horses. Buy one at a stable with <ansi fg="command">mount stable [type]</ansi> (<ansi fg="command">help mount</ansi>).`
	}
	lines := []string{fmt.Sprintf("Your horses (riding %d of %d, pack %d of %d):",
		herd.CountKind(mount.KindRiding, m.specs), members, herd.CountKind(mount.KindPack, m.specs), members)}
	for _, horse := range herd.Sorted() {
		spec, known := m.specs[horse.Type]
		if !known {
			lines = append(lines, fmt.Sprintf("  #%d %s: no longer a recognized horse type.", horse.ID, m.horseNameLocked(horse)))
			continue
		}
		saddle := "no saddle"
		if horse.Saddled() {
			saddle = itemName(horse.SaddleItemId)
		}
		lines = append(lines, fmt.Sprintf("  #%d %s, %s: carries %s.", horse.ID, spec.Name(), saddle, carries(spec, horse.Saddled())))
	}
	return strings.Join(lines, "\n")
}

func (m *MountModule) userCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		user.SendText(m.status(user.UserId))
		return true, nil
	}
	tail := func(from int) string {
		if len(args) <= from {
			return ""
		}
		return strings.Join(args[from:], " ")
	}
	switch strings.ToLower(args[0]) {
	case "status", "list":
		user.SendText(m.status(user.UserId))
	case "stable", "buy":
		user.SendText(m.stable(user, room, tail(1)))
	case "release":
		user.SendText(m.release(user, tail(1)))
	case "saddle":
		if len(args) < 3 {
			user.SendText(mountUsage)
			return true, nil
		}
		user.SendText(m.saddle(user, args[1], tail(2)))
	case "unsaddle":
		user.SendText(m.unsaddle(user, tail(1)))
	default:
		user.SendText(mountUsage)
	}
	return true, nil
}

// parseMountSpecs normalizes the configured horse-type list, rejecting a
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
			Type:                 strings.ToLower(strings.TrimSpace(configString(fields["type"]))),
			Description:          configString(fields["description"]),
			Kind:                 mount.Kind(strings.ToLower(strings.TrimSpace(configString(fields["kind"])))),
			Price:                configInt(fields["price"]),
			BareCapacityGrams:    int(configFloat(fields["barecapacitykg"]) * 1000),
			SaddledCapacityGrams: int(configFloat(fields["saddledcapacitykg"]) * 1000),
			TravelDurationPct:    configInt(fields["traveldurationpct"]),
			FatiguePct:           configInt(fields["fatiguepct"]),
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
