package company

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

type nativeRuntime struct{}

func (nativeRuntime) ResolveTemplate(name string) (int, bool) {
	id := mobs.MobIdByName(name)
	return int(id), id > 0
}

// Spawn creates the companion's live mob. With a state (Phase 22b), the mob
// is spawned at the saved level and the template's minted gear is replaced
// by copies of the saved gear.
func (nativeRuntime) Spawn(leaderUserID, roomID, mobTemplateID int, state *domain.MemberState, identity domain.Identity, growth domain.GrowthWeights) (int, error) {
	leader := users.GetByUserId(leaderUserID)
	if leader == nil {
		return 0, fmt.Errorf("company: leader %d is unavailable", leaderUserID)
	}
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, fmt.Errorf("company: room %d is unavailable", roomID)
	}
	// Companions never roll elite: their level comes from the record (or
	// the template, for a new recruit), not from chance.
	level := 0
	if state != nil {
		level = state.Level
	}
	mob := mobs.NewMobByIdNoElite(mobs.MobId(mobTemplateID), roomID, level)
	if mob == nil {
		return 0, fmt.Errorf("company: mob template %d is unavailable", mobTemplateID)
	}
	// Phase 33h1: the level's points dealt by the companion's growth, in
	// place of the template's even spread, before its vitals are set.
	retrainMob(mob, growth)
	if state != nil {
		applyState(mob, *state)
	} else {
		mob.Character.Health = mob.Character.HealthMax.Value
		mob.Character.Mana = mob.Character.ManaMax.Value
	}
	// Phase 32a2: a generated recruit's own name and description.
	if identity.Name != "" {
		mob.Character.Name = identity.Name
		// Generated identities have no authored pronouns of their own.
		mob.Character.Pronouns = "they"
	}
	if identity.Description != "" {
		mob.Character.Description = identity.Description
	}
	mob.Character.CharmAsCompanion(leaderUserID, -2, characters.CharmExpiredRevert)
	leader.Character.TrackCharmed(mob.InstanceId, true)
	room.AddMob(mob.InstanceId)
	return mob.InstanceId, nil
}

func (nativeRuntime) IsLive(instanceID int) bool { return mobs.MobInstanceExists(instanceID) }

func (nativeRuntime) WithLeader(leaderUserID, instanceID int) bool {
	leader := users.GetByUserId(leaderUserID)
	mob := mobs.GetInstance(instanceID)
	return leader != nil && mob != nil && mob.Character.RoomId == leader.Character.RoomId
}

func (nativeRuntime) IsAttached(leaderUserID, instanceID int) bool {
	leader := users.GetByUserId(leaderUserID)
	mob := mobs.GetInstance(instanceID)
	return leader != nil && mob != nil && mob.Character.IsCharmed(leaderUserID) &&
		mob.Character.Charmed.RoundsRemaining != 0 &&
		slices.Contains(leader.Character.GetCharmIds(), instanceID)
}

func (nativeRuntime) Detach(leaderUserID, instanceID int) {
	if leader := users.GetByUserId(leaderUserID); leader != nil {
		leader.Character.TrackCharmed(instanceID, false)
	}
	if mob := mobs.GetInstance(instanceID); mob != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			room.RemoveMob(instanceID)
		}
	}
	mobs.DestroyInstance(instanceID)
}

func (nativeRuntime) Relocate(instanceID, roomID int) bool {
	mob := mobs.GetInstance(instanceID)
	if mob == nil || mob.Character.Health < 1 {
		return false
	}
	to := rooms.LoadRoom(roomID)
	if to == nil {
		return false
	}
	mob.Character.Aggro = nil
	if mob.Character.RoomId == roomID {
		return true
	}
	if from := rooms.LoadRoom(mob.Character.RoomId); from != nil {
		from.RemoveMob(instanceID)
	}
	to.AddMob(instanceID)
	return true
}

func (nativeRuntime) Standing(instanceID int) (int, bool, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil || mob.Character.Health < 1 {
		return 0, false, false
	}
	return mob.Character.RoomId, mob.Character.Aggro != nil, true
}

// Retrain re-deals a live mob's stat points by its growth weights (Phase
// 33h1). Health and mana are only held to their new maxima, never raised.
func (nativeRuntime) Retrain(instanceID int, growth domain.GrowthWeights) bool {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return false
	}
	retrainMob(mob, growth)
	mob.Character.Health = min(mob.Character.Health, mob.Character.HealthLimit())
	mob.Character.Mana = min(mob.Character.Mana, mob.Character.ManaMax.Value)
	return true
}

// retrainMob sets each stat's training to the template's own plus the
// level's stat points dealt by the weights. Training is derived, never
// saved, so the same level and weights always give the same stats.
func retrainMob(mob *mobs.Mob, growth domain.GrowthWeights) {
	var base [6]int
	if spec := mobs.GetMobSpec(mob.MobId); spec != nil {
		base = trainingOf(&spec.Character)
	}
	dealt := domain.Deal(characters.StatPointsAtLevel(mob.Character.Level), growth)
	c := &mob.Character
	for i, training := range []*int{
		&c.Stats.Strength.Training,
		&c.Stats.Speed.Training,
		&c.Stats.Smarts.Training,
		&c.Stats.Vitality.Training,
		&c.Stats.Mysticism.Training,
		&c.Stats.Perception.Training,
	} {
		*training = base[i] + dealt[i]
	}
	c.StatPoints = 0
	c.Validate()
}

// trainingOf reads a character's training in domain.GrowthStats order.
func trainingOf(c *characters.Character) [6]int {
	return [6]int{
		c.Stats.Strength.Training,
		c.Stats.Speed.Training,
		c.Stats.Smarts.Training,
		c.Stats.Vitality.Training,
		c.Stats.Mysticism.Training,
		c.Stats.Perception.Training,
	}
}

// applyState puts a saved state on a freshly spawned mob: its level,
// experience, gold, copies of its gear in place of the template's, its
// lasting wounds, and its health and mana.
func applyState(mob *mobs.Mob, state domain.MemberState) {
	saved := state.Clone()
	if saved.Level > 0 {
		mob.Character.Level = saved.Level
	}
	if saved.Experience > 0 {
		mob.Character.Experience = saved.Experience
	}
	mob.Character.Gold = saved.Gold
	mob.Character.Equipment = saved.Equipment
	mob.Character.Items = saved.Items
	if mob.Character.Items == nil {
		mob.Character.Items = []items.Item{}
	}
	// Phase 30b: its lasting wounds come back with it (no fight survives a
	// respawn, so light ones don't), and its health stops at the limit.
	mob.Character.Wounds = wounds.CloseLight(saved.Wounds)
	mob.Character.Validate(true)
	// Phase 33h2: its saved health and mana, held to today's limits.
	mob.Character.Health, mob.Character.Mana = saved.Vitals.Resolve(mob.Character.HealthLimit(), mob.Character.ManaMax.Value)
}

// Snapshot reads a live mob's level, gear, wounds, and vitals.
func (nativeRuntime) Snapshot(instanceID int) (domain.MemberState, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return domain.MemberState{}, false
	}
	state := domain.MemberState{
		Level:      mob.Character.Level,
		Experience: mob.Character.Experience,
		Equipment:  mob.Character.Equipment,
		Items:      mob.Character.Items,
		Gold:       mob.Character.Gold,
		Wounds:     mob.Character.Wounds,
		Vitals:     &domain.Vitals{Health: mob.Character.Health, Mana: mob.Character.Mana},
	}
	return state.Clone(), true
}

// UseItem takes one use of a carried item from a live mob, removing it
// when used up.
func (nativeRuntime) UseItem(instanceID int, itm items.Item) bool {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return false
	}
	for i := range mob.Character.Items {
		if mob.Character.Items[i].Equals(itm) {
			mob.Character.UseItem(itm)
			return true
		}
	}
	return false
}

// CharmedByOther reports whether a live mob is now charmed by someone other
// than the leader (befriended away). An uncharmed companion, such as one
// whose charm expired when its leader left, is still the company's.
func (nativeRuntime) CharmedByOther(leaderUserID, instanceID int) bool {
	mob := mobs.GetInstance(instanceID)
	return mob != nil && mob.Character.IsCharmed() && !mob.Character.IsCharmed(leaderUserID)
}

// TemplateState is the state a companion of this template starts with,
// derived from the spec without spawning (the legacy upgrade). Every item
// gets its own fresh UUID.
func (nativeRuntime) GearGrams(instanceID int) (int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, false
	}
	total := 0
	for _, itm := range mob.Character.Equipment.GetAllItems() {
		total += itm.Weight()
	}
	for i := range mob.Character.Items {
		total += mob.Character.Items[i].Weight()
	}
	return total, true
}

// Carry reads a live mob's Strength and its largest pack (Phase 32f).
func (nativeRuntime) Carry(instanceID int) (int, int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, 0, false
	}
	return mob.Character.Stats.Strength.ValueAdj, domain.BestPackGrams(mob.Character.Items), true
}

func (nativeRuntime) TemplateState(mobTemplateID int) (domain.MemberState, bool) {
	spec := mobs.GetMobSpec(mobs.MobId(mobTemplateID))
	if spec == nil {
		return domain.MemberState{}, false
	}
	state := domain.MemberState{
		Level:     spec.Character.Level,
		Equipment: spec.Character.Equipment,
		Items:     spec.Character.Items,
		Gold:      spec.Character.Gold,
	}.Clone()
	if state.Level < 1 {
		state.Level = 1
	}
	for _, slot := range characters.AllSlots() {
		if itm := state.Equipment.Get(slot); itm != nil && itm.ItemId > 0 {
			itm.UUID = uuid.UUID{}
			itm.Validate()
		}
	}
	for i := range state.Items {
		state.Items[i].UUID = uuid.UUID{}
		state.Items[i].Validate()
	}
	return state, true
}

// Progress reads a live mob's level and experience progress.
func (nativeRuntime) Progress(instanceID int) (int, int, int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, 0, 0, false
	}
	into, tnl := mob.Character.XPTNLActual()
	return mob.Character.Level, into, tnl, true
}

// Mana reads a live mob's mana.
func (nativeRuntime) Mana(instanceID int) (int, int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, 0, false
	}
	return mob.Character.Mana, mob.Character.ManaMax.Value, true
}

// HealthLimit reads a live mob's wound limit (Phase 30b).
func (nativeRuntime) HealthLimit(instanceID int) (int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, false
	}
	return mob.Character.HealthLimit(), true
}

// Vitals reads a live mob's health.
func (nativeRuntime) Vitals(instanceID int) (int, int, bool) {
	mob := mobs.GetInstance(instanceID)
	if mob == nil {
		return 0, 0, false
	}
	return mob.Character.Health, mob.Character.HealthMax.Value, true
}
