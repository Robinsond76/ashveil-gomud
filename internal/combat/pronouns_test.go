package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func pronounOptions(prefix string) items.MessageOptions {
	return items.MessageOptions{items.ItemMessage("{sourcehe}/{sourcehim}/{sourcehis} " + prefix + " {targethe}/{targethim}/{targethis}. You keep your footing.")}
}

func TestCombatPronounTokens(t *testing.T) {
	cases := []struct {
		name       string
		source     characters.Character
		target     characters.Character
		sourceType SourceTarget
		targetType SourceTarget
		wantSource string
		wantTarget string
	}{
		{"she and it", characters.Character{Name: "source", Pronouns: "she"}, characters.Character{Name: "target", Pronouns: "it"}, Mob, Mob, "She/her/her", "it/it/its"},
		{"he and she", characters.Character{Name: "source", Pronouns: "he"}, characters.Character{Name: "target", Pronouns: "she"}, Mob, Mob, "He/him/his", "she/her/her"},
		{"they and he", characters.Character{Name: "source", Pronouns: "they"}, characters.Character{Name: "target", Pronouns: "he"}, Mob, Mob, "They/them/their", "he/him/his"},
		{"it and they", characters.Character{Name: "source", Pronouns: "it"}, characters.Character{Name: "target", Pronouns: "they"}, Mob, Mob, "It/it/its", "they/them/their"},
		{"user source stays they in beast form", characters.Character{Name: "source", Pronouns: "she", FormRaceId: 21}, characters.Character{Name: "target", Pronouns: "it"}, User, Mob, "They/them/their", "it/it/its"},
		{"user target stays they in beast form", characters.Character{Name: "source", Pronouns: "she"}, characters.Character{Name: "target", Pronouns: "he", FormRaceId: 21}, Mob, User, "She/her/her", "they/them/their"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, layout := range []struct {
				name                   string
				sourceRoom, targetRoom int
			}{{"together", 1, 1}, {"separate", 1, 2}} {
				t.Run(layout.name, func(t *testing.T) {
					source, target := tc.source, tc.target
					source.RoomId, target.RoomId = layout.sourceRoom, layout.targetRoom
					options := pronounOptions(layout.name)
					toSource, toTarget, toSourceRoom, toTargetRoom := buildCombatMessages(&source, &target, tc.sourceType, tc.targetType, "weapon", "1", 1, options, options, options, options, options, options, options, options)
					want := tc.wantSource + " " + layout.name + " " + tc.wantTarget + ". You keep your footing."
					for _, line := range []string{string(toSource), string(toTarget), string(toSourceRoom), string(toTargetRoom)} {
						assert.Equal(t, want, line)
						assert.NotContains(t, line, "{source")
						assert.NotContains(t, line, "{target")
					}
				})
			}
		})
	}
}

func TestMobCombatCharacterUsesBattleNameWithoutMutatingMob(t *testing.T) {
	mob := &mobs.Mob{InstanceId: 9911, Character: characters.Character{Name: "bandit cutthroat", Health: 50}}
	battle.Reset()
	t.Cleanup(battle.Reset)
	battle.Begin(7, 1, 1, "bandits", []int{mob.InstanceId, 9912})
	battle.AssignEnemyNames(7, []battle.EnemyName{{InstanceId: mob.InstanceId, BaseName: mob.Character.Name}, {InstanceId: 9912, BaseName: mob.Character.Name}})

	copy := mobCombatCharacter(mob)
	assert.Equal(t, "bandit cutthroat", mob.Character.Name)
	assert.Equal(t, "first cutthroat", copy.Name)
	assert.Equal(t, "bandit cutthroat", mob.Character.Name)
}

func TestClawsTheyFormsAvoidSingularVerbs(t *testing.T) {
	loadTestData(t)
	source := characters.Character{Name: "claw fighter", Pronouns: "they", RoomId: 1}
	target := characters.Character{Name: "target", Pronouns: "it", RoomId: 1}
	for _, options := range []items.AttackOptions{
		items.GetPreAttackMessage(items.Claws, items.Wait),
		items.GetAttackMessage(items.Claws, 80, false),
	} {
		for seed := 1; seed <= max(len(options.Together.ToAttacker), len(options.Together.ToDefender), len(options.Together.ToRoom)); seed++ {
			toSource, toTarget, toSourceRoom, _ := buildCombatMessages(
				&source, &target, Mob, Mob, "claws", "1", seed,
				options.Together.ToAttacker, options.Together.ToDefender, options.Together.ToRoom, nil,
				options.Separate.ToAttacker, options.Separate.ToDefender, options.Separate.ToAttackerRoom, options.Separate.ToDefenderRoom,
			)
			for _, line := range []string{string(toSource), string(toTarget), string(toSourceRoom)} {
				lower := strings.ToLower(line)
				assert.NotContains(t, lower, "they watches")
				assert.NotContains(t, lower, "they shakes")
			}
		}
	}
}

func hasMessage(messages []string, want string) bool {
	return strings.Contains(strings.Join(messages, "\n"), want)
}
