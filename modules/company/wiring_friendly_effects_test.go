package company

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func friendlyCast(t *testing.T, b *brawl) characters.SpellAggroInfo {
	t.Helper()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.SpellBook["healall"] = 1
	b.aria.Character.ManaMax.Value = 100
	b.aria.Character.Mana = 100
	b.cmd("cast", "healall")
	require.NotNil(t, b.aria.Character.Aggro)
	require.Equal(t, characters.SpellCast, b.aria.Character.Aggro.Type)
	return b.aria.Character.Aggro.SpellInfo
}

func TestManualGroupHealIncludesOwnCompany(t *testing.T) {
	b := newBrawl(t)
	info := friendlyCast(t, b)
	assert.Equal(t, []int{7}, info.TargetUserIds)
	require.Len(t, info.TargetMobInstanceIds, 4)
	require.NotNil(t, info.FriendlyTargets)
	mana := b.aria.Character.Mana
	for _, id := range info.TargetMobInstanceIds {
		m := mobs.GetInstance(id)
		m.Character.Health = 1
	}
	b.aria.Character.Health = 1
	_, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, info)
	require.NoError(t, err)
	assert.Greater(t, b.aria.Character.Health, 1)
	for _, id := range info.TargetMobInstanceIds {
		assert.Greater(t, mobs.GetInstance(id).Character.Health, 1)
	}
	assert.Equal(t, mana, b.aria.Character.Mana, "completion does not charge mana again")
	assert.Equal(t, 100-spells.GetSpell("healall").Cost, mana)
}

func TestFriendlyCastsPruneChangedTargets(t *testing.T) {
	for _, change := range []string{"move", "death", "transfer", "dismiss", "expire", "newcomer"} {
		t.Run(change, func(t *testing.T) {
			b := newBrawl(t)
			info := friendlyCast(t, b)
			patient := b.companion(1)
			patient.Character.Health = 1
			switch change {
			case "move":
				b.road.RemoveMob(patient.InstanceId)
				patient.Character.RoomId++
			case "death":
				patient.Character.Health = 0
			case "transfer":
				patient.Character.Charm(8, -1, "")
			case "dismiss":
				b.cmd("company", "dismiss Tamsin")
			case "expire":
				patient.Character.Charmed.Expire()
			case "newcomer":
				patient.Character.Charm(7, -1, "") // no longer a registered companion charm
			}
			result := effecttargets.Resolve(7, 0, info)
			assert.NotContains(t, result.TargetMobInstanceIds, patient.InstanceId)
			_, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, info)
			require.NoError(t, err)
			if change != "dismiss" {
				want := 1
				if change == "death" {
					want = 0
				}
				assert.Equal(t, want, patient.Character.Health)
			}
		})
	}
}

func TestHelpfulSourceOwnershipAndMovementRevalidate(t *testing.T) {
	b := newBrawl(t)
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value = 100
	oswin.Character.Mana = 100
	_, err := mobcommands.TryCommand("cast", "healall", oswin.InstanceId)
	require.NoError(t, err)
	require.NotNil(t, oswin.Character.Aggro)
	info := oswin.Character.Aggro.SpellInfo
	require.Len(t, info.TargetMobInstanceIds, 4)
	oswin.Character.Charm(8, -1, "")
	result := effecttargets.Resolve(0, oswin.InstanceId, info)
	assert.Empty(t, result.TargetUserIds)
	assert.Empty(t, result.TargetMobInstanceIds)
	b.aria.Character.Aggro = nil
	info = friendlyCast(t, b)
	b.aria.Character.RoomId++
	result = effecttargets.Resolve(7, 0, info)
	assert.Empty(t, result.TargetUserIds)
	assert.Empty(t, result.TargetMobInstanceIds)
	b.aria.Character.RoomId = b.road.RoomId
}

func TestFriendlyEligibilityConsentAndWoundCaps(t *testing.T) {
	b := newBrawl(t)
	other := users.NewUserRecord(8, 1)
	other.Character.RoomId = b.road.RoomId
	other.Character.Health = 10
	users.SetTestUser(other)
	b.road.AddPlayer(8)
	t.Cleanup(func() { b.road.RemovePlayer(8) })
	temporary := mobs.NewMobById(9101, b.road.RoomId)
	require.NotNil(t, temporary)
	temporary.Character.Charm(7, -1, "")
	b.road.AddMob(temporary.InstanceId)
	b.aria.Character.TrackCharmed(temporary.InstanceId, true)
	info := effecttargets.Resolve(7, 0, characters.SpellAggroInfo{SpellId: "healall"})
	assert.NotContains(t, info.TargetMobInstanceIds, temporary.InstanceId)
	assert.NotContains(t, info.TargetUserIds, 8)
	// Allied and area group scopes fail closed without 33d's consent provider.
	sp := spells.GetSpell("healall")
	original := sp.Scope
	t.Cleanup(func() { sp.Scope = original })
	previous := effecttargets.SetAlliedLeaders(nil)
	t.Cleanup(func() { effecttargets.SetAlliedLeaders(previous) })
	sp.Scope = spells.ScopeAllied
	info = effecttargets.Resolve(7, 0, characters.SpellAggroInfo{SpellId: "healall"})
	assert.NotContains(t, info.TargetUserIds, 8)
	b.aria.Character.Health = 0
	// Another living caster can heal a downed player, but never a fallen companion.
	oswin := b.companion(2)
	patient := b.companion(1)
	patient.Character.Health = 0
	info = effecttargets.Resolve(0, oswin.InstanceId, characters.SpellAggroInfo{SpellId: "healall"})
	assert.Contains(t, info.TargetUserIds, 7)
	assert.NotContains(t, info.TargetMobInstanceIds, patient.InstanceId)
	b.aria.Character.Health = -10
	info = effecttargets.Resolve(0, oswin.InstanceId, characters.SpellAggroInfo{SpellId: "healall"})
	assert.NotContains(t, info.TargetUserIds, 7)
	b.aria.Character.Health = 10
	patient.Character.Health = 1
	patient.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Points: 3, Light: true}}
	limit := patient.Character.HealthLimit()
	patient.Character.Health = limit - 1
	sp.Scope = original
	info = effecttargets.Resolve(0, oswin.InstanceId, characters.SpellAggroInfo{SpellId: "healall"})
	_, err := scripting.TrySpellScriptEvent("onMagic", 0, oswin.InstanceId, info)
	require.NoError(t, err)
	assert.LessOrEqual(t, patient.Character.Health, limit)
}

func TestSingleSupportAndHarmfulTargetsKeepTheirContracts(t *testing.T) {
	b := newBrawl(t)
	patient := b.companion(1)
	patient.Character.Health = 1
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.SpellBook["heal"] = 1
	b.aria.Character.ManaMax.Value = 100
	b.aria.Character.Mana = 100
	b.cmd("cast", "heal #"+strconv.Itoa(patient.InstanceId))
	require.NotNil(t, b.aria.Character.Aggro)
	info := b.aria.Character.Aggro.SpellInfo
	assert.Equal(t, []int{patient.InstanceId}, info.TargetMobInstanceIds)
	patient.Character.Charm(8, -1, "")
	_, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, info)
	require.NoError(t, err)
	assert.Equal(t, 1, patient.Character.Health)
	harm := characters.SpellAggroInfo{SpellId: "mm", TargetMobInstanceIds: []int{patient.InstanceId, 1234}}
	assert.Equal(t, harm, effecttargets.Resolve(7, 0, harm), "friendly resolver cannot broaden harmful effects")
}

func TestFriendlyCastNeverAddsLateTargets(t *testing.T) {
	b := newBrawl(t)
	patient := b.companion(1)
	patient.Character.Health = 0
	info := friendlyCast(t, b)
	assert.NotContains(t, info.TargetMobInstanceIds, patient.InstanceId)
	patient.Character.Health = 1
	result := effecttargets.Resolve(7, 0, info)
	assert.NotContains(t, result.TargetMobInstanceIds, patient.InstanceId, "revived during chant is not a new target")
	_, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, info)
	require.NoError(t, err)
	assert.Equal(t, 1, patient.Character.Health)
}

func TestVoidHelpfulSpellHandlersBothEngines(t *testing.T) {
	for _, tc := range []struct{ ext, body string }{
		{"js", `function onCast(source, actors) {} function onMagic(source, actors) { actors[0].AddHealth(2); }`},
		{"lua", `function onCast(source, actors) end function onMagic(source, actors) actors[1]:AddHealth(2) end`},
	} {
		t.Run(tc.ext, func(t *testing.T) {
			b := newBrawl(t)
			sp := spells.GetSpell("healall")
			originalType, originalScope := sp.Type, sp.Scope
			sp.Type, sp.Scope = spells.HelpArea, spells.ScopeArea
			t.Cleanup(func() { sp.Type, sp.Scope = originalType, originalScope })
			path := sp.GetScriptPath()
			require.NoError(t, os.Remove(path))
			path = strings.TrimSuffix(path, ".js") + "." + tc.ext
			require.NoError(t, os.WriteFile(path, []byte(tc.body), 0600))
			scripting.InvalidateSpellVM("healall")
			t.Cleanup(scripting.ClearSpellVMs)
			b.aria.Character.Health = 1
			info := friendlyCast(t, b)
			require.NotNil(t, info.FriendlyTargets, "void onCast must proceed through actual cast initiation")
			handled, err := scripting.TrySpellScriptEvent("onMagic", 7, 0, info)
			require.NoError(t, err, "a present void handler is not a missing handler")
			assert.False(t, handled)
			assert.Equal(t, 3, b.aria.Character.Health)
		})
	}
}

func TestHelpfulCastPrunesBeforeRoundCompletion(t *testing.T) {
	b := newBrawl(t)
	patient := b.companion(1)
	patient.Character.Health = 1
	b.aria.Character.Health = 1
	stream := b.listen()
	info := friendlyCast(t, b)
	patient.Character.Charm(8, -1, "")
	b.aria.Character.SpellBook["healall"] = 1000
	// The engine retains a 1% fizzle chance even at maximum proficiency.
	// Retry the same captured targets rather than making randomness a failure.
	for attempt := 0; attempt < 5 && b.aria.Character.Health == 1; attempt++ {
		b.aria.Character.SetCast(2, info)
		for i := 0; i < 3; i++ {
			out := b.fight()
			assert.NotContains(t, out, patient.Character.Name+" (", "pruned patient is absent from healing narration")
		}
	}
	assert.Equal(t, 1, patient.Character.Health, "transferred patient cannot receive the completed heal")
	assert.Greater(t, b.aria.Character.Health, 1, "eligible caster receives the completed heal")
	for _, event := range *stream {
		if event.Kind == combatstream.Heal {
			assert.NotEqual(t, patient.InstanceId, event.Target.MobInstanceId)
		}
	}
}
