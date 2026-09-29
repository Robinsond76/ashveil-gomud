package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestScriptActorGetCombatPronoun(t *testing.T) {
	mob := ScriptActor{characterRecord: &characters.Character{Pronouns: "she"}}
	assert.Equal(t, "she", mob.GetCombatPronoun("subject"))
	assert.Equal(t, "her", mob.GetCombatPronoun("object"))
	assert.Equal(t, "her", mob.GetCombatPronoun("possessive"))
	assert.Empty(t, mob.GetCombatPronoun("unknown"))

	user := ScriptActor{userRecord: &users.UserRecord{}, characterRecord: &characters.Character{Pronouns: "he"}}
	assert.Equal(t, "they", user.GetCombatPronoun("subject"))
	assert.Equal(t, "them", user.GetCombatPronoun("object"))
	assert.Equal(t, "their", user.GetCombatPronoun("possessive"))
}

func TestScriptCombatIdentity(t *testing.T) {
	battle.Reset()
	t.Cleanup(battle.Reset)
	battle.Begin(71, 18, 1, "foes", []int{41, 42})
	battle.AssignEnemyNames(71, []battle.EnemyName{{InstanceId: 41, BaseName: "bandit cutthroat"}, {InstanceId: 42, BaseName: "bandit cutthroat"}})
	m := &mobs.Mob{InstanceId: 42, Character: characters.Character{Name: "bandit cutthroat", Pronouns: "she"}}
	actor := ScriptActor{mobRecord: m, mobInstanceId: 42, characterRecord: &m.Character}
	for _, tc := range []struct {
		name   string
		lang   ScriptLang
		source string
	}{
		{"js", LangJS, `function run(a) { return a.GetCharacterName(false) + "|" + a.GetCombatName(false) + "|" + a.GetCombatPronoun("subject") + "|" + a.GetCombatPronoun("object") + "|" + a.GetCombatPronoun("possessive"); }`},
		{"lua", LangLua, `function run(a) return a:GetCharacterName(false) .. "|" .. a:GetCombatName(false) .. "|" .. a:GetCombatPronoun("subject") .. "|" .. a:GetCombatPronoun("object") .. "|" .. a:GetCombatPronoun("possessive") end`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vm := loadEngine(t, tc.lang, tc.source)
			fn, ok := vm.GetFunction("run")
			require.True(t, ok)
			result, err := vm.Call(time.Second, fn, vm.ToValue(actor))
			require.NoError(t, err)
			assert.Equal(t, `bandit cutthroat|the <ansi fg="mobname">second cutthroat</ansi>|she|her|her`, result.Export())
		})
	}
	assert.Equal(t, "bandit cutthroat", m.Character.Name)
}
