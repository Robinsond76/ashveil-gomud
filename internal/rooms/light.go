package rooms

import (
	"sync"
	"sync/atomic"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/sky"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
)

// Ashveil light model (Phase 14). A room's ambient light counts only the sky,
// the biome, weather, mutators, and fixtures (a `lit` room tag, or a
// registered fixture such as a lit campfire). A carried light helps only its
// bearer; a `partylight` (e.g. a floating light) helps its bearer's allies.
const (
	FlagLightSource = `lightsource`
	FlagPartyLight  = `partylight`
	FlagNightVision = `nightvision`

	// TagLit marks a room with a permanent light fixture (hearth, lanterns).
	TagLit = `lit`

	DefaultDarkHitPenalty = 40
	DefaultDimHitPenalty  = 10
)

// LightConditions are the inputs to a room's ambient light level.
type LightConditions struct {
	Indoor     bool
	DarkBiome  bool
	LitBiome   bool
	Night      bool
	Moonlight  int // 0-2, from sky.Moonlight
	FogMod     int // 0 or negative, from the zone's weather
	MutatorMod int
	Fixture    bool
}

func clampLight(level int) int {
	if level < 0 {
		return 0
	}
	if level > 2 {
		return 2
	}
	return level
}

// ambientLevel is the room-wide light level (0 dark, 1 this room only, 2 this
// room and its exits) before any viewer's own or party light.
func ambientLevel(c LightConditions) int {
	level := 0
	switch {
	case c.DarkBiome:
		level = 0
	case c.Indoor:
		if c.LitBiome {
			level = 2
		}
	case !c.Night:
		// Daylight: fog limits sight but never fully blinds.
		level = 2 + c.FogMod
		if level < 1 {
			level = 1
		}
	default:
		if c.Moonlight >= 1 {
			level = 1
		}
		if c.LitBiome {
			level++
		}
		level += c.FogMod
	}
	level = clampLight(level + c.MutatorMod)
	if c.Fixture {
		level++
	}
	return clampLight(level)
}

// Describe explains the ambient light in player-facing sentences.
func (c LightConditions) Describe() []string {
	var lines []string
	switch {
	case c.DarkBiome:
		lines = append(lines, "No natural light reaches this place.")
	case c.Indoor:
		if c.LitBiome {
			lines = append(lines, "You are indoors, in a lit place.")
		} else {
			lines = append(lines, "You are indoors, away from any natural light.")
		}
	case !c.Night:
		lines = append(lines, "It is daytime.")
	default:
		switch {
		case c.Moonlight >= 2:
			lines = append(lines, "It is night, but the moon gives a fair light.")
		case c.Moonlight == 1:
			lines = append(lines, "It is night; the moon gives a faint light.")
		default:
			lines = append(lines, "It is night, and no moon lights the way.")
		}
		if c.LitBiome {
			lines = append(lines, "Lanterns light the way here.")
		}
	}
	if !c.Indoor && !c.DarkBiome {
		switch {
		case c.FogMod <= -2:
			lines = append(lines, "Thick fog swallows the light.")
		case c.FogMod < 0:
			lines = append(lines, "Fog dims everything.")
		}
	}
	if c.MutatorMod < 0 {
		lines = append(lines, "An unnatural gloom hangs here.")
	} else if c.MutatorMod > 0 {
		lines = append(lines, "An unnatural brightness fills the air.")
	}
	if c.Fixture {
		lines = append(lines, "A light source here lights the area for everyone.")
	}
	return lines
}

// LightConditions gathers this room's ambient-light inputs from the shared
// clock, the zone's weather, the biome, mutators, and fixtures. It reads the
// clock and never advances it.
func (r *Room) LightConditions() LightConditions {
	c := LightConditions{Indoor: r.IsIndoor()}
	if biome := r.GetBiome(); biome != nil {
		c.DarkBiome = biome.IsDark()
		c.LitBiome = biome.IsLit()
	}
	gd := gametime.GetDate()
	c.Night = gd.Night

	condition, tracked := weather.CurrentCondition(r.Zone)
	cloudCover := 0
	if tracked {
		cloudCover = condition.CloudCover
		if !c.Indoor {
			c.FogMod = condition.VisibilityMod
		}
	}
	if gd.MoonCount > 0 {
		phase := sky.PhaseForDay(sky.AbsoluteDay(gd.RoundNumber, gd.RoundsPerDay), sky.CycleDays())
		c.Moonlight = sky.Moonlight(phase, cloudCover)
	}
	for mut := range r.ActiveMutators {
		if spec := mut.GetSpec(); spec != nil {
			c.MutatorMod += spec.LightMod
		}
	}
	c.Fixture = r.HasTag(TagLit) || roomHasFixtureLight(r.RoomId)
	return c
}

// viewerLevel applies a viewer's own light, an ally's party light, and night
// vision to the room's ambient level. Own and party light don't stack.
func viewerLevel(ambient int, nightVision, ownLight, partyLight bool) int {
	if nightVision {
		return 2
	}
	if ownLight || partyLight {
		ambient++
	}
	return clampLight(ambient)
}

// lightAllied reports whether two light-group keys (a user id, or the user id
// a companion is charmed to) belong to the same company or GoMud party.
func lightAllied(a, b int, partyOf func(int) *parties.Party) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	if a == b {
		return true
	}
	p := partyOf(a)
	return p != nil && p.IsMember(a) && p.IsMember(b)
}

// HasPartyLightFor reports whether any ally of leaderKey in this room carries a
// party light.
func (r *Room) HasPartyLightFor(leaderKey int) bool {
	if leaderKey <= 0 {
		return false
	}
	for _, uid := range r.GetPlayers() {
		u := users.GetByUserId(uid)
		if u == nil || !u.Character.HasBuffFlag(FlagPartyLight) {
			continue
		}
		if lightAllied(leaderKey, uid, parties.Get) {
			return true
		}
	}
	for _, instanceId := range r.GetMobs() {
		m := mobs.GetInstance(instanceId)
		if m == nil || !m.Character.HasBuffFlag(FlagPartyLight) {
			continue
		}
		if lightAllied(leaderKey, m.Character.GetCharmedUserId(), parties.Get) {
			return true
		}
	}
	return false
}

// VisibilityForUser is what this user can see here (0-2).
func (r *Room) VisibilityForUser(u *users.UserRecord) int {
	c := u.Character
	return viewerLevel(r.GetVisibility(), c.HasBuffFlag(FlagNightVision), c.HasBuffFlag(FlagLightSource), r.HasPartyLightFor(u.UserId))
}

// VisibilityForMob is what this mob can see here (0-2). A companion shares
// its leader's party light.
func (r *Room) VisibilityForMob(m *mobs.Mob) int {
	c := &m.Character
	return viewerLevel(r.GetVisibility(), c.HasBuffFlag(FlagNightVision), c.HasBuffFlag(FlagLightSource), r.HasPartyLightFor(c.GetCharmedUserId()))
}

var (
	fixtureMu        sync.RWMutex
	fixtureProviders []func(roomId int) bool
)

// RegisterLightFixture adds a provider that reports rooms lit by a fixture
// it owns (e.g. modules/camping's lit campfires).
func RegisterLightFixture(provider func(roomId int) bool) {
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	fixtureProviders = append(fixtureProviders, provider)
}

// ResetLightFixtures clears every registered fixture provider (tests).
func ResetLightFixtures() {
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	fixtureProviders = nil
}

func roomHasFixtureLight(roomId int) bool {
	fixtureMu.RLock()
	providers := append([]func(int) bool(nil), fixtureProviders...)
	fixtureMu.RUnlock()
	for _, p := range providers {
		if p(roomId) {
			return true
		}
	}
	return false
}

var (
	darkHitPenalty atomic.Int64
	dimHitPenalty  atomic.Int64
)

func init() {
	SetDarknessPenalties(DefaultDarkHitPenalty, DefaultDimHitPenalty)
}

// SetDarknessPenalties sets the to-hit penalties for attacking in darkness
// (visibility 0) and dim light (visibility 1). Negative values clamp to 0.
func SetDarknessPenalties(dark, dim int) {
	if dark < 0 {
		dark = 0
	}
	if dim < 0 {
		dim = 0
	}
	darkHitPenalty.Store(int64(dark))
	dimHitPenalty.Store(int64(dim))
}

// HitPenaltyForVisibility is the to-hit penalty for an attacker who sees at
// level vis. A target carrying its own light is always at least dimly
// visible.
func HitPenaltyForVisibility(vis int, targetLit bool) int {
	if targetLit && vis < 1 {
		vis = 1
	}
	switch {
	case vis <= 0:
		return int(darkHitPenalty.Load())
	case vis == 1:
		return int(dimHitPenalty.Load())
	}
	return 0
}
