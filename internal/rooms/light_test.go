package rooms

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/parties"
)

func TestAmbientLevel(t *testing.T) {
	cases := []struct {
		name string
		c    LightConditions
		want int
	}{
		{"clear day", LightConditions{}, 2},
		{"fog by day", LightConditions{FogMod: -1}, 1},
		{"thick fog by day never blinds", LightConditions{FogMod: -2}, 1},
		{"moonlit night", LightConditions{Night: true, Moonlight: 2}, 1},
		{"dim moon night", LightConditions{Night: true, Moonlight: 1}, 1},
		{"moonless night", LightConditions{Night: true, Moonlight: 0}, 0},
		{"foggy moonlit night", LightConditions{Night: true, Moonlight: 2, FogMod: -1}, 0},
		{"lit street at night", LightConditions{Night: true, Moonlight: 0, LitBiome: true}, 1},
		{"lit street moonlit night", LightConditions{Night: true, Moonlight: 2, LitBiome: true}, 2},
		{"lit indoors", LightConditions{Indoor: true, LitBiome: true, Night: true}, 2},
		{"unlit indoors by day", LightConditions{Indoor: true}, 0},
		{"indoors ignores fog", LightConditions{Indoor: true, LitBiome: true, FogMod: -2}, 2},
		{"cave by day", LightConditions{DarkBiome: true}, 0},
		{"cave with fixture", LightConditions{DarkBiome: true, Fixture: true}, 1},
		{"moonless night with campfire", LightConditions{Night: true, Fixture: true}, 1},
		{"negative mutator clamps then fixture", LightConditions{MutatorMod: -2, Night: true, Fixture: true}, 1},
		{"positive mutator clamps", LightConditions{MutatorMod: 2}, 2},
	}
	for _, c := range cases {
		if got := ambientLevel(c.c); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestLightConditionsDescribe(t *testing.T) {
	lines := strings.Join(LightConditions{Night: true, Moonlight: 0, FogMod: -1, Fixture: true}.Describe(), " ")
	for _, want := range []string{"night", "moon", "fog", "light source"} {
		if !strings.Contains(strings.ToLower(lines), want) {
			t.Errorf("description %q missing %q", lines, want)
		}
	}
	if got := strings.Join(LightConditions{Indoor: true}.Describe(), " "); !strings.Contains(got, "indoors") {
		t.Errorf("indoor description %q", got)
	}
}

func TestViewerLevel(t *testing.T) {
	cases := []struct {
		name                              string
		ambient                           int
		nightVision, ownLight, partyLight bool
		want                              int
	}{
		{"dark, no light", 0, false, false, false, 0},
		{"dark, own torch", 0, false, true, false, 1},
		{"dark, party light", 0, false, false, true, 1},
		{"dark, both do not stack", 0, false, true, true, 1},
		{"dim, own torch", 1, false, true, false, 2},
		{"bright stays bright", 2, false, true, true, 2},
		{"night vision", 0, true, false, false, 2},
	}
	for _, c := range cases {
		if got := viewerLevel(c.ambient, c.nightVision, c.ownLight, c.partyLight); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestLightAllied(t *testing.T) {
	party := &parties.Party{LeaderUserId: 1, UserIds: []int{1, 2}, InviteUserIds: []int{3}}
	partyOf := func(userId int) *parties.Party {
		if userId >= 1 && userId <= 3 {
			return party
		}
		return nil
	}
	cases := []struct {
		a, b int
		want bool
	}{
		{5, 5, true},  // a user and their own companions share a leader
		{1, 2, true},  // same party
		{2, 1, true},  // symmetric
		{1, 3, false}, // invited is not a member
		{1, 9, false}, // stranger
		{0, 0, false}, // uncharmed mobs are nobody's allies
	}
	for _, c := range cases {
		if got := lightAllied(c.a, c.b, partyOf); got != c.want {
			t.Errorf("lightAllied(%d,%d) = %v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestLightFixtureRegistry(t *testing.T) {
	t.Cleanup(ResetLightFixtures)
	if roomHasFixtureLight(7) {
		t.Fatal("no providers means no fixture")
	}
	RegisterLightFixture(func(roomId int) bool { return roomId == 7 })
	if !roomHasFixtureLight(7) || roomHasFixtureLight(8) {
		t.Fatal("registered provider should light only room 7")
	}
}

func TestHitPenaltyForVisibility(t *testing.T) {
	t.Cleanup(func() { SetDarknessPenalties(DefaultDarkHitPenalty, DefaultDimHitPenalty) })
	SetDarknessPenalties(40, 10)
	cases := []struct {
		vis       int
		targetLit bool
		want      int
	}{
		{0, false, 40},
		{0, true, 10},
		{1, false, 10},
		{1, true, 10},
		{2, false, 0},
	}
	for _, c := range cases {
		if got := HitPenaltyForVisibility(c.vis, c.targetLit); got != c.want {
			t.Errorf("vis %d lit %v: got %d want %d", c.vis, c.targetLit, got, c.want)
		}
	}
	SetDarknessPenalties(-5, -1)
	if HitPenaltyForVisibility(0, false) != 0 {
		t.Fatal("negative penalties clamp to 0")
	}
}
