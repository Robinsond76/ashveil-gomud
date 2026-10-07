package awakening

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	bladeID   = 99821
	capID     = 99822
	leaderUID = 99820
	companion = 5
	instance  = 99825
	bossMob   = 99826
)

func relics(t *testing.T) {
	t.Helper()
	blade := &items.ItemSpec{ItemId: bladeID, Name: "Test Edge", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Tier: 5,
		Damage: items.Damage{Attacks: 1, DiceCount: 1, SideCount: 6},
		Relic: &items.RelicSpec{Signature: "Edge", Effects: map[string]int{classes.Wounded: 20}, ILvl: 20, Mob: bossMob, Chance: 5,
			Awakenings: []items.AwakeningSpec{
				{Name: "Bone-Breaker", Kind: items.AwakenSlay, Races: []string{"skeleton"}, Count: 2, Target: "skeletons", Effects: map[string]int{classes.Damage: 1}},
				{Name: "Lord's End", Kind: items.AwakenLair, Mob: bossMob, Target: "the bone lord", Effects: map[string]int{classes.Wounded: 5}},
				{Name: "Deep Hold", Kind: items.AwakenPlace, Zone: "Deep Keep", Target: "Deep Keep", Effects: map[string]int{classes.Attack: 3}},
			}}}
	cap := &items.ItemSpec{ItemId: capID, Name: "Test Cap", Type: items.Head, Subtype: items.Wearable, Tier: 5, DamageReduction: 2,
		Relic: &items.RelicSpec{Set: "awaketest", ILvl: 20, Mob: bossMob, Chance: 5,
			Awakenings: []items.AwakeningSpec{{Name: "Bone Guard", Kind: items.AwakenSlay, Races: []string{"skeleton"}, Count: 1, Target: "skeletons", Effects: map[string]int{classes.Armor: 2}}}}}
	items.SetTestItemSpec(blade)
	items.SetTestItemSpec(cap)
	t.Cleanup(func() { items.RemoveTestItemSpec(bladeID); items.RemoveTestItemSpec(capID) })
}

type fakeCompany struct{}

func (fakeCompany) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (fakeCompany) InstanceFor(leader, id int) (int, bool) {
	return instance, leader == leaderUID && id == companion
}
func (fakeCompany) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) { return 0, "", false }
func (fakeCompany) CompanyMembers(int) ([]company.MemberView, bool) {
	return []company.MemberView{{ID: companion, Name: "Tobin"}}, true
}

type world struct {
	leader *users.UserRecord
	mate   *mobs.Mob
	said   []string
}

// newWorld seats a leader and a companion in room 1 with the real default
// member lookup.
func newWorld(t *testing.T) *world {
	t.Helper()
	relics(t)
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	chronicle.SetProvider(chronicle.NewMemory())
	t.Cleanup(func() { chronicle.SetProvider(nil) })
	company.SetFormationProvider(fakeCompany{})
	t.Cleanup(func() { company.SetFormationProvider(nil) })

	w := &world{}
	w.leader = users.NewUserRecord(leaderUID, 0)
	w.leader.Character.Name = "Mara"
	w.leader.Character.RoomId = 1
	w.leader.Character.Health = 10
	users.SetTestUser(w.leader)
	w.mate = &mobs.Mob{MobId: 77, InstanceId: instance}
	w.mate.Character = *characters.New()
	w.mate.Character.Name = "Tobin"
	w.mate.Character.RoomId = 1
	w.mate.Character.Health = 10
	mobs.SetTestInstance(w.mate)
	t.Cleanup(func() { mobs.RemoveTestInstance(instance) })

	prev := notify
	notify = func(_ int, text string) { w.said = append(w.said, text) }
	t.Cleanup(func() { notify = prev })
	return w
}

func wield(c *characters.Character, id int) *items.Item {
	if id == bladeID {
		c.Equipment.Weapon = items.New(id)
		return &c.Equipment.Weapon
	}
	c.Equipment.Head = items.New(id)
	return &c.Equipment.Head
}

func TestSlainFoesAdvanceOnlyWornRelicsOfAMatchingKind(t *testing.T) {
	w := newWorld(t)
	blade := wield(w.leader.Character, bladeID)
	carried := items.New(bladeID)
	w.leader.Character.Items = append(w.leader.Character.Items, carried)

	Slain(leaderUID, "Ogre")
	assert.Zero(t, blade.AwakeningProgress(0), "the wrong kind of foe")
	Slain(leaderUID, "Skeleton")
	assert.Equal(t, 1, blade.AwakeningProgress(0))
	assert.Zero(t, w.leader.Character.Items[0].AwakeningProgress(0), "a relic in the pack does nothing")
	assert.Empty(t, w.said)

	Slain(leaderUID, "skeleton")
	assert.True(t, blade.Awakened(0))
	require.Len(t, w.said, 1)
	assert.Contains(t, w.said[0], "Mara's Test Edge awakens: Bone-Breaker")
	assert.Contains(t, w.said[0], "+1 damage on every landed blow")

	Slain(leaderUID, "skeleton")
	assert.Equal(t, 2, blade.AwakeningProgress(0), "a woken awakening stops counting")
	assert.Len(t, w.said, 1, "and never wakes twice")

	// The wearer's real effects follow, through the gear cache.
	assert.Equal(t, 1, w.leader.Character.ClassEffects().Int(classes.Damage))
	log := chronicle.Query(leaderUID, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Awakened}})
	require.Len(t, log, 1)
	assert.Equal(t, []string{"leader"}, log[0].Keys)
	assert.Equal(t, "Bone-Breaker", log[0].Detail)
	assert.Equal(t, "item:"+"99821", log[0].Ref)
	assert.Contains(t, chronicle.Prose(log[0]), "Test Edge awoke to Bone-Breaker")
}

func TestACompanionsRelicProgressesWhileItStandsWithTheLeader(t *testing.T) {
	w := newWorld(t)
	cap := wield(&w.mate.Character, capID)

	w.mate.Character.RoomId = 2
	Slain(leaderUID, "skeleton")
	assert.Zero(t, cap.AwakeningProgress(0), "a companion away from the leader earns nothing")
	w.mate.Character.RoomId = 1
	w.mate.Character.Health = 0
	Slain(leaderUID, "skeleton")
	assert.Zero(t, cap.AwakeningProgress(0), "nor does a fallen one")
	w.mate.Character.Health = 10
	w.mate.Character.CombatWithdrawn = true
	Slain(leaderUID, "skeleton")
	assert.Zero(t, cap.AwakeningProgress(0), "nor one withdrawn from the fight")
	w.mate.Character.CombatWithdrawn = false

	Slain(leaderUID, "skeleton")
	assert.True(t, cap.Awakened(0))
	require.Len(t, w.said, 1)
	assert.Contains(t, w.said[0], "Tobin's Test Cap awakens: Bone Guard")
	log := chronicle.Query(leaderUID, chronicle.Filter{Kinds: []chronicle.Kind{chronicle.Awakened}})
	require.Len(t, log, 1)
	assert.Equal(t, []string{"companion:5"}, log[0].Keys, "the deed names the companion by its key")
	assert.Equal(t, []string{"Tobin"}, log[0].Members)
	assert.Equal(t, 2, w.mate.Character.ClassEffects().Int(classes.Armor))
}

// A lair's master falls: the chronicle's own boss deed advances the relic.
func TestTheChroniclesBossDeedWakesALairAwakening(t *testing.T) {
	w := newWorld(t)
	blade := wield(w.leader.Character, bladeID)

	chronicle.Record(leaderUID, chronicle.Entry{Kind: chronicle.Boss, Subject: "Other", Ref: "mob:12"})
	assert.Zero(t, blade.AwakeningProgress(1), "another boss")
	chronicle.Record(leaderUID, chronicle.Entry{Kind: chronicle.Relic, Ref: "mob:" + "99826"})
	assert.Zero(t, blade.AwakeningProgress(1), "only a boss deed counts")
	chronicle.Record(leaderUID, chronicle.Entry{Kind: chronicle.Boss, Subject: "The Bone Lord", Ref: "mob:99826"})
	assert.True(t, blade.Awakened(1))
	assert.Equal(t, 25, w.leader.Character.ClassEffects().Int(classes.Wounded), "20 from the signature, 5 awakened")
	assert.Contains(t, strings.Join(w.said, "\n"), "Lord's End")
}

func TestReachingAZoneWakesAPlaceAwakening(t *testing.T) {
	w := newWorld(t)
	blade := wield(w.leader.Character, bladeID)
	Reached(leaderUID, "Shallow Keep")
	assert.Zero(t, blade.AwakeningProgress(2))
	Reached(leaderUID, "deep keep")
	assert.True(t, blade.Awakened(2))
	assert.Equal(t, 3, w.leader.Character.ClassEffects().Int(classes.Attack))
	assert.Equal(t, 4, blade.AwakenedMask())
}

func TestAnAwakeningNeedsALeaderInTheWorld(t *testing.T) {
	relics(t)
	users.ResetActiveUsers()
	assert.NotPanics(t, func() { Slain(404, "skeleton"); Reached(404, "x") })
}
