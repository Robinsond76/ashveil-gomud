package company

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/engagement"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brawl is the Phase 29a wiring world: the 2026-09-26 5v5 simulation
// rebuilt through the real entry points with the shipped config.
type brawl struct {
	t        *testing.T
	road     *rooms.Room
	aria     *users.UserRecord
	messages *[]string
	bandits  map[string][]int // name -> live instance ids
	round    uint64
}

// brawlBandits is the bandits' spawn group: they are non-hostile, and
// since Phase 32d a shared tag groups only hostile mobs.
const brawlBandits = "brawl:bandits"

// banditMob is a non-hostile bandit sharing the party's groups tag.
func banditMob(id int, name string, level int) string {
	return fmt.Sprintf("mobid: %d\nzone: brawl\nhostile: false\nmaxwander: 0\nactivitylevel: 0\ngroups: [bandits]\ncharacter:\n  name: %s\n  raceid: 1\n  level: %d\n", id, name, level)
}

// hostileMob is a hostile mob that stays put, in a group when one is named.
func hostileMob(id int, name, group, extra string) string {
	groups := ""
	if group != "" {
		groups = "groups: [" + group + "]\n"
	}
	return fmt.Sprintf("mobid: %d\nzone: brawl\nhostile: true\nmaxwander: 0\nactivitylevel: 0\n%s%scharacter:\n  name: %s\n  raceid: 1\n  level: 1\n", id, groups, extra, name)
}

// spawnRoom is a room whose spawn list names these mobs, one entry each.
func spawnRoom(roomId int, mobIds ...int) string {
	out := fmt.Sprintf("roomid: %d\nzone: brawl\ntitle: Alley\ndescription: An alley.\nbiome: road\nspawninfo:\n", roomId)
	for _, id := range mobIds {
		out += fmt.Sprintf("  - mobid: %d\n", id)
	}
	return out
}

func copyShipped(t *testing.T, dataDir string, rels ...string) {
	t.Helper()
	shipped := shippedWorld(t)
	for _, rel := range rels {
		require.NoError(t, filepath.Walk(filepath.Join(shipped, rel), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			r, err := filepath.Rel(shipped, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			dst := filepath.Join(dataDir, r)
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			return os.WriteFile(dst, data, 0600)
		}))
	}
}

// newBrawl loads the shipped config (F3: the harness that found 1-HP
// companions ran without it) and a disposable world holding the shipped
// companions and a five-bandit party (two cutthroats, a bruiser, a
// slinger, and a captain, who stands front-left as the toughest). Aria
// (level 3) summons Tamsin, Oswin, Garrick, and Ysolde.
func newBrawl(t *testing.T) *brawl {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	require.NoError(t, configs.ReloadConfig())
	// Phase 32e: this brawl forces every member's health to 1000 each round
	// (toughen); a level-up mid-round would recompute it and could let a
	// member fall. Kill XP is switched off so nobody levels here.

	dataDir := t.TempDir()
	useDataDir(t, dataDir)
	copyShipped(t, dataDir, "items", "races", "combat-messages", "biomes", "keywords.yaml", "spells", "skills",
		"mobs/dunmar/61-tamsin_reed.yaml", "mobs/dunmar/62-brother_oswin.yaml",
		"mobs/dunmar/63-garrick_vane.yaml", "mobs/old_kings_road/64-ysolde.yaml")
	fixtures := map[string]string{
		"rooms/brawl/zone-config.yaml":          "name: brawl\nroomid: 920101\n",
		"rooms/brawl/920101.yaml":               "roomid: 920101\nzone: brawl\ntitle: Road\ndescription: A road.\nbiome: road\nexits:\n  east:\n    roomid: 920102\n",
		"rooms/brawl/920102.yaml":               "roomid: 920102\nzone: brawl\ntitle: Verge\ndescription: The road's verge.\nbiome: road\nexits:\n  west:\n    roomid: 920101\n",
		"mobs/brawl/9101-bandit_cutthroat.yaml": banditMob(9101, "bandit cutthroat", 1),
		"mobs/brawl/9102-bandit_bruiser.yaml":   banditMob(9102, "bandit bruiser", 1),
		"mobs/brawl/9103-bandit_slinger.yaml":   banditMob(9103, "bandit slinger", 1),
		"mobs/brawl/9104-bandit_captain.yaml":   banditMob(9104, "bandit captain", 4),
		// A fence who trades with the bandits: same group, but a shop.
		"mobs/brawl/9105-bandit_fence.yaml": banditMob(9105, "bandit fence", 1) + "  shop:\n    - itemid: 10002\n      price: 50\n",
		// Phase 29b2 spawn fixtures: hostile mobs and rooms with spawn lists.
		"mobs/brawl/9106-ruffian.yaml":    hostileMob(9106, "ruffian", "slum-ruffians", ""),
		"mobs/brawl/9107-big_rat.yaml":    hostileMob(9107, "big rat", "rats", ""),
		"mobs/brawl/9108-cave_troll.yaml": hostileMob(9108, "cave troll", "", "solitary: true\n"),
		"rooms/brawl/920103.yaml":         spawnRoom(920103, 9106),
		"rooms/brawl/920104.yaml":         spawnRoom(920104, 9107, 9106),
		"rooms/brawl/920105.yaml":         spawnRoom(920105, 9108),
		"rooms/brawl/920106.yaml":         spawnRoom(920106, 9106, 9106, 9106, 9106, 9106, 9106),
	}
	for path, data := range fixtures {
		full := filepath.Join(dataDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
		require.NoError(t, os.WriteFile(full, []byte(data), 0600))
	}
	for _, dir := range []string{"users", "plugin-data", "buffs"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, dir), 0755))
	}
	races.LoadDataFiles()
	spells.LoadSpellFiles()
	skills.LoadDataFiles()
	items.LoadDataFiles()
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	mobs.LoadDataFiles()
	keywords.LoadAliases()
	road := rooms.LoadRoom(920101)
	require.NotNil(t, road)

	useFakeLifecycle(t, &fakeLifecycle{})
	require.NotNil(t, module)
	t.Cleanup(plugins.SnapshotLoadStateForTest())
	plugins.Load(dataDir)
	require.NoError(t, module.loadErr)
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["Modules.company.AllowedCompanionMobIDs"] = []any{61, 62, 63, 64}
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	// Apply after RestoreOverrides, which validates zero XPScale back to 100.
	gameplay := configs.GetGamePlayConfig()
	gameplay.XPScale = 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	aria := users.NewUserRecord(7, 1)
	aria.Username = "aria"
	aria.Password = "$2a$test"
	aria.Character.Name = "Aria"
	aria.Character.RaceId = 1
	aria.Character.Level = 3
	aria.Character.RoomId = road.RoomId
	aria.Character.Validate()
	aria.Character.Health = aria.Character.HealthMax.Value
	users.SetTestUser(aria)
	road.AddPlayer(aria.UserId)
	t.Cleanup(func() {
		for _, instance := range mobs.GetAllMobInstanceIds() {
			nativeRuntime{}.Detach(7, instance)
			if mob := mobs.GetInstance(instance); mob != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					room.RemoveMob(instance)
				}
			}
			mobs.DestroyInstance(instance)
		}
		road.RemovePlayer(7)
		module.registry = *domain.NewRegistry()
		module.instances = map[int]map[int]int{}
	})

	// The test stands in for the world loop: queued mob commands run, and
	// idle mobs look for trouble.
	mid := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.MobInstanceId > 0 {
			c, rest, _ := strings.Cut(in.InputText, " ")
			_, _ = mobcommands.TryCommand(strings.ToLower(c), rest, in.MobInstanceId)
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Input{}, mid) })
	idle := events.RegisterListener(events.MobIdle{}, hooks.HandleIdleMobs)
	t.Cleanup(func() { events.UnregisterListener(events.MobIdle{}, idle) })

	// Each brawl reports to a stream of its own (Phase 29b), so no fight
	// is carried from one test into the next.
	t.Cleanup(combatstream.UseForTest(combatstream.New()))
	battle.Reset() // and no battle (29b2)
	t.Cleanup(battle.Reset)
	// Nor a grudge: an earlier fight leaves the bandits' group hostile to
	// Aria (user 7) for many rounds, and they would open the next brawl.
	mobs.ResetHostility()
	t.Cleanup(mobs.ResetHostility)

	b := &brawl{t: t, road: road, aria: aria, messages: captureCompanyMessages(t), bandits: map[string][]int{}}
	for _, name := range []string{"tamsin reed", "brother oswin", "garrick vane", "ysolde"} {
		require.Contains(t, b.cmd("company", "summon "+name), "Companion summoned")
	}
	// A level-1 mob spawns holding level 2's experience (mobs.NewMobById
	// sets XPTL(0), which clamps to XPTL(1)), so without kill XP its first
	// kill still levels it and refills its health. Start the level-1
	// companions at no experience, so nobody levels here.
	for id := 1; id <= 4; id++ {
		if mob := b.companion(id); mob.Character.Level == 1 {
			mob.Character.Experience = 0
		}
	}
	for _, id := range []int{9101, 9101, 9102, 9103, 9104} {
		m := mobs.NewMobById(mobs.MobId(id), road.RoomId)
		require.NotNil(t, m)
		// Phase 32d: non-hostile mobs are a group only by a spawn group.
		m.SpawnGroup = brawlBandits
		road.AddMob(m.InstanceId)
		b.bandits[m.Character.Name] = append(b.bandits[m.Character.Name], m.InstanceId)
	}
	return b
}

// aimAt starts the fight with the bandits' group and then sets Aria's aim
// on one named bandit, as `attack <bandit>` did before Phase 32c (a fight is
// now started by naming a group, and the first aim is chosen for her). The
// 29a/29b scenarios that need a particular foe use it.
func (b *brawl) aimAt(name string) {
	b.t.Helper()
	_, id := b.road.FindByName(name)
	if id == 0 {
		_, id = rooms.LoadRoom(b.aria.Character.RoomId).FindByName(name)
	}
	require.NotZero(b.t, id, name)
	b.cmd("attack", fmt.Sprintf("#%d", id))
	b.aria.Character.SetAggro(0, id, characters.DefaultAttack)
}

func (b *brawl) cmd(c, rest string) string {
	b.t.Helper()
	*b.messages = nil
	_, err := usercommands.TryCommand(c, rest, b.aria.UserId, events.CmdSkipScripts)
	require.NoError(b.t, err)
	events.ProcessEvents()
	return companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
}

// fight runs one real combat round and the idle-mob pass after it.
func (b *brawl) fight() string {
	b.t.Helper()
	*b.messages = nil
	b.round++
	hooks.DoCombat(events.NewRound{RoundNumber: b.round})
	events.ProcessEvents()
	hooks.IdleMobs(events.NewRound{RoundNumber: b.round})
	events.ProcessEvents()
	return companyTagPattern.ReplaceAllString(strings.Join(*b.messages, "\n"), "")
}

func (b *brawl) companion(id int) *mobs.Mob {
	b.t.Helper()
	instance, ok := module.instance(7, id)
	require.True(b.t, ok)
	mob := mobs.GetInstance(instance)
	require.NotNil(b.t, mob)
	return mob
}

// companyInstances is the set of live companion instance ids.
func (b *brawl) companyInstances() map[int]bool {
	out := map[int]bool{}
	for _, instance := range module.instances[7] {
		out[instance] = true
	}
	return out
}

// livingBandits lists the bandits still alive in the road.
func (b *brawl) livingBandits() []*mobs.Mob {
	var out []*mobs.Mob
	for _, ids := range b.bandits {
		for _, id := range ids {
			if mob := mobs.GetInstance(id); mob != nil && mob.Character.Health > 0 && mob.Character.RoomId == b.road.RoomId {
				out = append(out, mob)
			}
		}
	}
	return out
}

// toughen makes every company member hard to kill (health changes clamp
// to HealthMax, so both are raised), so a fight runs to the bandits' end
// and the invariants are checked over all of it. fightToTheEnd renews it
// every round.
func (b *brawl) toughen() {
	b.aria.Character.HealthMax.Value = 1000
	b.aria.Character.Health = 1000
	for id := 1; id <= 4; id++ {
		mob := b.companion(id)
		mob.Character.HealthMax.Value = 1000
		mob.Character.Health = 1000
	}
}

// fightToTheEnd runs rounds until no bandit is standing, checking every
// round that no living combatant on either side ends a round idle while
// the other side can still fight (F2 and the kill cases). A company member
// is idle at a round's end only when a bandit fell that round: a kill ends
// the killer's aggro, and a death clears the leader's, until the next
// round's upkeep. The company can't fall (toughen), so a bandit never is.
func (b *brawl) fightToTheEnd(maxRounds int) (companyDamage int) {
	b.t.Helper()
	for i := 0; i < maxRounds && len(b.livingBandits()) > 0; i++ {
		before := len(b.livingBandits())
		b.toughen()
		b.t.Logf("round %d:\n%s", b.round+1, b.fight())
		companyDamage += 1000 - b.aria.Character.Health
		for id := 1; id <= 4; id++ {
			companyDamage += 1000 - b.companion(id).Character.Health
		}
		living := b.livingBandits()
		if len(living) == 0 {
			break
		}
		someoneFell := len(living) < before
		if !someoneFell {
			assert.NotNil(b.t, b.aria.Character.Aggro, "round %d: Aria is fighting", b.round)
			for id := 1; id <= 4; id++ {
				mob := b.companion(id)
				assert.NotNil(b.t, mob.Character.Aggro, "round %d: %s is fighting", b.round, mob.Character.Name)
			}
		}
		for _, mob := range living {
			assert.NotNil(b.t, mob.Character.Aggro, "round %d: %s #%d is fighting", b.round, mob.Character.Name, mob.InstanceId)
		}
	}
	assert.Empty(b.t, b.livingBandits(), "the fight runs until every bandit is down; none is left standing beside the company")
	return companyDamage
}

// swungAt reports whether seen holds one of Aria's own attack lines naming
// foe: to her ("You hit...", "The foe twists aside from your blow", "The
// foe sways aside, and your blow falls on empty air"; she fights with her
// fists, the generic pool), or about her to the room ("Aria's fists...").
func swungAt(seen, foe string) bool {
	for _, line := range strings.Split(seen, "\n") {
		if strings.Contains(line, "turn toward") || !strings.Contains(line, foe) {
			continue
		}
		if strings.HasPrefix(line, "You") || strings.HasPrefix(line, "Aria") || strings.Contains(line, "your blow") {
			return true
		}
	}
	return false
}

// TestCombatFixesThroughTheRealRound drives Phase 29a through the real
// combat round with the shipped config: F1 (the leader reassigned from an
// unreachable target), the kill cases (the leader and the killer take the
// next target), F2 (the whole party joins and fights to the end), the
// enemy side, F3, and F4. The clock never moves.
func TestCombatFixesThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	turn, round := util.GetTurnCount(), util.GetRoundCount()

	// F3: with the shipped HPBase, level-1 companions have real HP.
	for id, name := range map[int]string{1: "Tamsin Reed", 2: "Brother Oswin"} {
		mob := b.companion(id)
		require.Equal(t, name, mob.Character.Name)
		require.Equal(t, 1, mob.Character.Level)
		assert.Greater(t, mob.Character.HealthMax.Value, 1, "%s has more than 1 HP", name)
	}

	// The simulation's formation: Aria front right, in column 3.
	for _, mv := range []string{"move #1 1 1", "move #3 1 2", "move me 1 3", "move #2 2 2", "move #4 3 2"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	captain := b.bandits["bandit captain"][0]

	// F4, out of a fight: a labelled demonstration.
	assert.Contains(t, b.cmd("formation", "reach me"), "Out of a fight, this shows plain-melee reach within your own company: Aria could reach Garrick Vane(#3).")

	// The enemy side: the captain (enemy column 1) is set on Aria, who
	// stands in column 3, out of its reach.
	mobs.GetInstance(captain).Character.SetAggro(7, 0, characters.DefaultAttack)

	// F1: Aria attacks the captain, out of her reach.
	b.aimAt("bandit captain")
	require.NotNil(t, b.aria.Character.Aggro)
	require.Equal(t, captain, b.aria.Character.Aggro.MobInstanceId)

	// F4, in a fight: the enemies she can reach with her own reach (her
	// fists), never the captain; Tamsin, in column 1, can reach it.
	got := b.cmd("formation", "reach me")
	assert.Contains(t, got, "In this fight, Aria can reach: bandit ")
	assert.NotContains(t, got, "bandit captain")
	assert.Contains(t, b.cmd("formation", "reach #1"), "bandit captain")

	b.toughen()
	got = b.fight()
	assert.NotContains(t, got, "You can't reach that target from here.")
	turned := regexp.MustCompile(`You can't reach the bandit captain from here\. You turn toward the ((?:first|second) cutthroat)\.`).FindStringSubmatch(got)
	require.Len(t, turned, 2, "Aria is told she turns from the captain:\n%s", got)
	assert.NotEqual(t, "bandit captain", turned[1])
	assert.True(t, swungAt(got, turned[1]), "Aria swung at her new target in the same round:\n%s", got)

	// F2: the whole party joined in round 1, not one member at a time.
	company := b.companyInstances()
	for _, mob := range b.livingBandits() {
		require.NotNil(t, mob.Character.Aggro, "%s #%d joined the fight", mob.Character.Name, mob.InstanceId)
		onCompany := mob.Character.Aggro.UserId == 7 || company[mob.Character.Aggro.MobInstanceId]
		assert.True(t, onCompany, "%s #%d fights the company", mob.Character.Name, mob.InstanceId)
	}
	// The enemy side: the captain turned from Aria, out of its reach.
	if c := mobs.GetInstance(captain); c != nil && c.Character.Health > 0 {
		require.NotNil(t, c.Character.Aggro)
		assert.Zero(t, c.Character.Aggro.UserId, "the captain turned on a companion it can reach")
	}

	// The kill case: a kill ends the killer's aggro (DoCombat's own kill
	// branch), and a mob's death clears every player's aggro on it
	// (suicide). Both rejoin at the start of the next round.
	b.aria.Character.EndAggro()
	garrick := b.companion(3)
	garrick.Character.EndAggro()
	got = b.fight()
	assert.Regexp(t, `You turn toward the (first|second) cutthroat\.`, got, "the leader rejoins")
	assert.Regexp(t, `Garrick Vane turns toward the (first|second) cutthroat\.`, got, "the killer rejoins")

	// Phase 32d: `break` is refused in a battle (only flee takes her out),
	// and she fights on.
	assert.Contains(t, b.cmd("break", ""), "Only flee takes you out of it.")
	assert.False(t, engagement.StoodDown(7), "nothing stood her down")
	_, inBattle := battle.Current(7)
	assert.True(t, inBattle, "her battle goes on")

	// And the fight runs to its end with nobody left idle.
	b.fightToTheEnd(200)

	assert.Equal(t, turn, util.GetTurnCount(), "never advances the clock")
	assert.Equal(t, round, util.GetRoundCount())
}

// TestUnplacedCompanyFightsAndCanBeStruck: a company that never set its
// formation is fought like any other (29a found it could never be struck),
// and its members are reassigned when their targets fall.
func TestUnplacedCompanyFightsAndCanBeStruck(t *testing.T) {
	b := newBrawl(t)
	require.Contains(t, b.cmd("formation", ""), "Unplaced: leader")

	// Review fix: an enemy aimed at an unplaced companion keeps it (the
	// upkeep once counted unplaced companions as dead and pulled every
	// enemy onto the leader).
	tamsin := b.companion(1)
	slinger := mobs.GetInstance(b.bandits["bandit slinger"][0])
	slinger.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)

	b.aimAt("bandit cutthroat")
	// In a fight, an unplaced member can reach anyone standing.
	got := b.cmd("formation", "reach me")
	assert.Contains(t, got, "In this fight, Aria can reach: ")
	assert.Contains(t, got, "bandit captain")

	b.toughen()
	got = b.fight()
	assert.NotContains(t, got, "bandit slinger turns toward")
	if slinger.Character.Health > 0 {
		require.NotNil(t, slinger.Character.Aggro)
		assert.Equal(t, tamsin.InstanceId, slinger.Character.Aggro.MobInstanceId, "the slinger keeps Tamsin")
	}

	assert.Positive(t, b.fightToTheEnd(200), "an unplaced company can be struck")
}

// TestSoloPlayerWithARecordTurnsOnHerOwn: the 29a upkeep only keeps a
// company with a companion present. A player whose companions are all
// dismissed (the record stays) is never told she can't reach anyone, and
// since Phase 32c (a battle plays out on its own) she turns on the next foe
// of her battle by herself when hers falls; since 29b2 the group she
// struck comes at her ("turns toward Aria").
func TestSoloPlayerWithARecordTurnsOnHerOwn(t *testing.T) {
	b := newBrawl(t)
	b.cmd("company", "dismiss all")
	_, hasRecord := domain.FormationFor(7)
	require.True(t, hasRecord)
	require.Empty(t, b.companyInstances())

	b.aimAt("bandit cutthroat")
	for i := 0; i < 8; i++ {
		b.aria.Character.HealthMax.Value = 1000
		b.aria.Character.Health = 1000
		before := len(b.livingBandits())
		got := b.fight()
		assert.NotContains(t, got, "can't reach", "round %d", b.round)
		if len(b.livingBandits()) > 0 && len(b.livingBandits()) == before {
			assert.NotNil(t, b.aria.Character.Aggro, "round %d: she is fighting", b.round)
		}
	}
}

// TestShopkeeperInTheGroupStaysOut: a fence sharing the bandits' group is
// in their party, but the upkeep never draws a shopkeeper into the fight.
func TestShopkeeperInTheGroupStaysOut(t *testing.T) {
	b := newBrawl(t)
	for _, name := range []string{"bandit slinger", "bandit bruiser"} {
		id := b.bandits[name][0]
		b.road.RemoveMob(id)
		mobs.DestroyInstance(id)
		delete(b.bandits, name)
	}
	fence := mobs.NewMobById(9105, b.road.RoomId)
	require.NotNil(t, fence)
	fence.SpawnGroup = brawlBandits
	require.True(t, fence.HasShop())
	b.road.AddMob(fence.InstanceId)

	b.aimAt("bandit cutthroat")
	b.toughen()
	got := b.fight()
	assert.Contains(t, got, "bandit captain turns toward", "the rest of the party joins")
	assert.NotContains(t, got, "bandit fence turns toward", "the shopkeeper doesn't")
	for i := 0; i < 3; i++ {
		b.toughen()
		assert.NotContains(t, b.fight(), "bandit fence turns toward")
	}
}
