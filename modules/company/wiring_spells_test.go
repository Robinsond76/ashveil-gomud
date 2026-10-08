package company

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ariaHears records what Aria herself is told, to her or to her room.
func (b *brawl) ariaHears() *[]string {
	b.t.Helper()
	events.ProcessEvents()
	var heard []string
	freshEvents(b.t)
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		m := e.(events.Message)
		if m.UserId == 7 || (m.UserId == 0 && m.RoomId == b.road.RoomId && !slices.Contains(m.ExcludeUserIds, 7)) {
			heard = append(heard, companyTagPattern.ReplaceAllString(m.Text, ""))
		}
		return events.Continue
	})
	b.t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &heard
}

var (
	missileLine = regexp.MustCompile(`releases a streak of cold light into the bandit captain\. \((\d+) damage\)`)
	spellLine   = regexp.MustCompile(`\((chanting: .*|\d+ healed)\)|light|prayer|praying`)
	healedLine  = regexp.MustCompile(`(?m)^    .*\byou \((\d+) healed\)`)
)

// TestSpellNarrationThroughTheRealCast (Phase 29c): through the real cast
// command and combat round, Magic Missile chants with its rounds and
// lands with its damage; Minor Heal All, cast by Brother Oswin, prints one
// cast line and one indented line naming everyone healed. No spell line
// breaks the narration voice.
func TestSpellNarrationThroughTheRealCast(t *testing.T) {
	b := newBrawl(t)
	heard := b.ariaHears()
	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	require.NotNil(t, captain)
	b.cmd("attack", "bandit cutthroats")
	b.toughen()
	b.fight()

	// Phase 32c: once a battle starts nothing typed changes it, so the
	// missile is Brother Oswin's, cast the way a companion casts.
	oswin := b.companion(2)
	oswin.Character.SpellBook["mm"] = 1
	var dealt []string
	for try := 0; try < 30 && dealt == nil; try++ {
		b.toughen()
		hardTo(&captain.Character, 1000)
		oswin.Character.ManaMax.Value = 100
		oswin.Character.Mana = 100
		_, err := mobcommands.TryCommand("cast", "mm #"+strconv.Itoa(captain.InstanceId), oswin.InstanceId)
		require.NoError(t, err)
		if oswin.Character.Aggro == nil || oswin.Character.Aggro.Type != characters.SpellCast {
			continue
		}
		for i := 0; i < 3; i++ {
			b.toughen()
			got := b.fight()
			if strings.Contains(got, "fizzles") {
				break
			}
			if m := missileLine.FindStringSubmatch(strings.Join(*heard, "")); m != nil {
				dealt = m
				break
			}
		}
	}
	require.NotNil(t, dealt, "a missile lands:\n%s", strings.Join(*heard, ""))
	n, _ := strconv.Atoi(dealt[1])
	// 35a2: 7 + 1d6 + level/10 + Mysticism/15, times the caster's spell
	// factor (its Attack against the captain's Evasion).
	o := &oswin.Character
	flat := 7 + o.Level/10 + o.Stats.Mysticism.ValueAdj/15
	factor := 1 + 0.5*characters.SkillEdge(o.AttackSkill(), captain.Character.Evasion())
	low, high := max(1, int(float64(flat+1)*factor)), max(1, int(float64(flat+6)*factor))
	assert.True(t, n >= low && n <= high, "(%d+1d6)×%.2f damage, got %d", flat, factor, n)
	all := strings.Join(*heard, "")
	assert.Contains(t, all, "(chanting: Magic Missile, 2 turns)", "the chant names the spell and its rounds")
	assert.Contains(t, all, "(chanting: Magic Missile, 1 turn)")

	// Minor Heal All from Brother Oswin, the company wounded first.
	oswin.Character.SpellBook["healall"] = 1
	var list []string
	for try := 0; try < 30 && list == nil; try++ {
		b.toughen()
		b.aria.Character.Health = 900
		oswin.Character.ManaMax.Value = 100
		oswin.Character.Mana = 100
		_, err := mobcommands.TryCommand("cast", "healall", oswin.InstanceId)
		require.NoError(t, err)
		if oswin.Character.Aggro == nil || oswin.Character.Aggro.Type != characters.SpellCast {
			continue
		}
		*heard = nil
		for i := 0; i < 4; i++ {
			b.aria.Character.Health = min(b.aria.Character.Health, 900)
			*heard = nil
			b.fight()
			round := strings.Join(*heard, "")
			if m := healedLine.FindStringSubmatch(round); m != nil {
				list = m
				assert.Contains(t, round, "Brother Oswin ", "one cast line names the healer")
				assert.Regexp(t, `\(\d+ healed\) · `, round, "the healed are listed on one line")
				break
			}
			if strings.Contains(round, "fizzles") {
				break
			}
		}
	}
	require.NotNil(t, list, "Oswin's heal lands")
	healed, _ := strconv.Atoi(list[1])
	// 35a2: 55% of 8 + 2d4 + level/6, times the healer's heal factor.
	base := 8 + oswin.Character.Level/6
	heal := 0.55 * (1 + float64(oswin.Character.HealingBonusPct())/100)
	assert.True(t, healed >= int(float64(base+2)*heal) && healed <= int(float64(base+8)*heal), "(%d+2d4)×%.3f healed, got %d", base, heal, healed)

	for _, line := range strings.Split(all+strings.Join(*heard, ""), "\n") {
		if spellLine.MatchString(line) {
			assert.Empty(t, util.NarrationVoiceProblem(line), line)
		}
	}
}
