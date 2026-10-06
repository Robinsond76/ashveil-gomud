package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
)

// Phase 39c: a Shaman's weather. A call belongs to the caster's battle
// (internal/battle) and puts its status on the foes standing in it; the
// world's weather is never touched.

// battleLeader is the player whose battle the actor fights in: itself for a
// player, its company's leader for a companion.
func (a ScriptActor) battleLeader() int {
	if a.userId > 0 {
		return a.userId
	}
	if a.mobInstanceId > 0 {
		if leader, _, ok := company.LeaderAndKeyForInstance(a.mobInstanceId); ok {
			return leader
		}
	}
	return 0
}

// Weather is the weather of the actor's battle: "fog", "chill", "rain" or ""
// for none (or no battle).
func (a ScriptActor) Weather() string {
	leader := a.battleLeader()
	if leader == 0 {
		return ``
	}
	return string(battle.WeatherOf(leader).Kind)
}

// CallWeather calls a weather ("fog", "chill" or "rain") into the actor's
// battle for 3 combat rounds, plus what its class adds (WeatherLong), and
// replaces any other: the old weather's statuses come off the foes at once.
// Fog and Chill Wind put their status on every foe standing in the battle.
// It returns {landed, reason, rounds, replaced, endless}: reason is "landed", "same"
// (that weather is already up) or "invalid" (no battle, or no such weather).
func (a ScriptActor) CallWeather(kind string) map[string]any {
	out := map[string]any{`landed`: false, `reason`: `invalid`, `rounds`: 0, `replaced`: ``}
	k := stormcraft.Kind(kind)
	leader := a.battleLeader()
	if leader == 0 || a.characterRecord == nil || (k != stormcraft.Fog && k != stormcraft.Chill && k != stormcraft.Rain) {
		return out
	}
	b, ok := battle.Current(leader)
	if !ok {
		return out
	}
	if b.Weather.Kind == k && b.Weather.Left > 1 {
		out[`reason`] = `same`
		return out
	}
	rounds := stormcraft.Rounds + a.characterRecord.ClassEffects().Int(classes.WeatherLong)
	// Phase 39i: a Tempest Lord's Rain lasts the whole battle.
	endless := k == stormcraft.Rain && a.characterRecord.ClassEffects().Has(classes.RainEndless)
	if endless {
		rounds = stormcraft.EndlessRounds
	}
	out[`endless`] = endless
	replaced, _ := battle.CallWeather(leader, k, stormcraft.Triggers(rounds))
	out[`landed`], out[`reason`], out[`rounds`], out[`replaced`] = true, `landed`, rounds, string(replaced)
	for id := range b.Enemies {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Health < 1 {
			continue
		}
		// One weather at a time: the last one's mark comes off.
		m.Character.RemoveBuff(status.Fogbound)
		m.Character.RemoveBuff(status.Windchilled)
		switch k {
		case stormcraft.Fog:
			events.AddToQueue(events.Buff{MobInstanceId: id, BuffId: status.Fogbound, Source: `spell`, Triggers: stormcraft.Triggers(rounds)})
		case stormcraft.Chill:
			events.AddToQueue(events.Buff{MobInstanceId: id, BuffId: status.Windchilled, Source: `spell`, Triggers: stormcraft.Triggers(rounds)})
		}
	}
	return out
}
