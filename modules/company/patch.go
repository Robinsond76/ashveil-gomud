package company

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/camping"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// Phase 35b: after a battle, the company's healers patch everyone up with
// their mana: Minor Heal only (no tend, no supplies), each member healed to
// the company's healing threshold (company tactics healing), each healer
// stopping at its mana reserve (its strategy's reserve). `company patch`
// runs the same plan on demand. A player's company patches only its own
// members, so allies (33d) keep their own mana. Everything runs on the
// game loop and never advances the clock.

const (
	patchInBattle = "Not in the middle of a battle. Your healers patch the company up once the fighting is done."
	patchTravel   = "Not while your company is travelling. Patch up once you have arrived."
	patchResting  = "Not while your company is resting. The rest will see to everyone."
)

// onBattleEnded patches the company up after its battle, if the player and
// the company are still out of battle: a waiting group may already have
// begun the next one in the same pass, and then there is no time to patch.
// A leader downed in a won battle is patched back to their feet.
func (m *CompanyModule) onBattleEnded(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.BattleEnded)
	if !ok {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil || user.Character == nil {
		return events.Continue
	}
	if reason := patchBlocked(user); reason != "" || companionsFighting(m.woundMembers(user)) {
		return events.Continue
	}
	lines, _ := m.patch(user)
	for _, line := range lines {
		user.SendText(line)
	}
	return events.Continue
}

// companionsFighting reports whether a companion still has a foe, as when
// allies (33d) fight on after the player's own battle has ended.
func companionsFighting(members []woundMember) bool {
	for _, w := range members {
		if !w.leader() && w.char.Aggro != nil {
			return true
		}
	}
	return false
}

// patchBlocked is why the company can't patch up now, or "".
func patchBlocked(user *users.UserRecord) string {
	if _, inBattle := battle.Current(user.UserId); inBattle {
		return patchInBattle
	}
	if blocked, _ := expedition.MovementBlocked(user.UserId); blocked {
		return patchTravel
	}
	if blocked, _ := camping.MovementBlocked(user.UserId); blocked {
		return patchResting
	}
	return ""
}

// patchUserCommand is the `patch` alias of `company patch`.
func (m *CompanyModule) patchUserCommand(_ string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(m.patchCommand(user))
	return true, nil
}

// patchCommand is `company patch` (alias `patch`).
func (m *CompanyModule) patchCommand(user *users.UserRecord) string {
	if reason := patchBlocked(user); reason != "" {
		return reason
	}
	members := m.woundMembers(user)
	if companyFighting(user, members) {
		return patchInBattle
	}
	if user.Character.Health < 1 {
		return "You're in no state to tend anyone."
	}
	lines, healed := m.patch(user)
	if !healed {
		return strings.Join(append(lines, m.patchIdle(user, members)), "\n")
	}
	return strings.Join(lines, "\n")
}

// patchIdle says why `company patch` did nothing.
func (m *CompanyModule) patchIdle(user *users.UserRecord, members []woundMember) string {
	healBelow := strategy.TacticsFor(user.UserId).Patch
	hurt, knowers := false, false
	for _, p := range patientsOf(members) {
		if p.Health >= 1 && p.Health < wounds.HealTarget(p, healBelow) {
			hurt = true
		}
	}
	for _, w := range members {
		knowers = knowers || w.knows("heal")
	}
	switch {
	case !hurt:
		return fmt.Sprintf("No one is below the company's patch threshold (%d%% of their wound limit). Use heal wounds to tend everyone fully.", healBelow)
	case !knowers:
		return "There's no one in the company who can heal. Use heal wounds to open the packs."
	default:
		return "Your healers are down to their mana reserves. A camp rest, an inn or a mana draught refills mana."
	}
}

// patch heals the company to its patch threshold with Minor Heal, each
// healer stopping at its reserve, applies it, and narrates it. healed is
// whether anyone was healed.
func (m *CompanyModule) patch(user *users.UserRecord) (lines []string, healed bool) {
	members := m.woundMembers(user)
	who := byKey(members)
	rules := spellRules()
	var healers []wounds.Healer
	for _, w := range members {
		if !w.knows("heal") || w.char.Mana < rules.HealCost || w.char.Health < 1 {
			continue
		}
		key := domain.LeaderMemberKey
		if !w.leader() {
			key = domain.CompanionMemberKey(w.companionID)
		}
		reservePct := enemyparty.MemberStrategy(user.UserId, key).Reserve
		healers = append(healers, wounds.Healer{
			Key: w.key, Mana: w.char.Mana, Heal: true,
			HealBonus: healBonus(w.char), HealPct: w.char.HealingBonusPct(),
			Reserve: w.char.ManaMax.Value * reservePct / 100,
		})
	}
	if len(healers) == 0 {
		return nil, false
	}
	res := wounds.Patch(patientsOf(members), healers, rules, strategy.TacticsFor(user.UserId).Patch, util.Rand)
	if len(res.Steps) == 0 {
		return nil, false
	}
	m.applyPlan(who, res)
	lines = append(lines, "Your company patches itself up.")
	lines = append(lines, stepLines(who, res.Steps)...)
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	return lines, true
}
