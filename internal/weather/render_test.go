package weather

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/sky"
)

func fog() Condition {
	c := clear()
	c.Name, c.Description, c.CloudCover, c.VisibilityMod = "fog", "Fog drifts between the trees.", 2, -1
	return c
}

func overcast() Condition {
	c := clear()
	c.Name, c.Description, c.CloudCover = "overcast", "Grey clouds hang low.", 3
	return c
}

func joined(lines []string) string { return strings.Join(lines, "\n") }

func TestRenderSkyOutdoorDayShowsWeather(t *testing.T) {
	lines := RenderSky(SkyView{HasMoon: true, Moon: sky.Full}, clear(), true, false)
	if len(lines) != 1 || lines[0] != clear().Description {
		t.Fatalf("expected only the weather line by day, got %q", lines)
	}
}

func TestRenderSkyNightShowsMoon(t *testing.T) {
	out := joined(RenderSky(SkyView{Night: true, HasMoon: true, Moon: sky.Full}, clear(), true, false))
	if !strings.Contains(out, clear().Description) || !strings.Contains(out, "full moon") {
		t.Fatalf("expected weather and full moon at night, got %q", out)
	}
}

func TestRenderSkyOvercastHidesMoon(t *testing.T) {
	out := joined(RenderSky(SkyView{Night: true, HasMoon: true, Moon: sky.Full}, overcast(), true, false))
	if strings.Contains(out, "full moon") {
		t.Fatalf("overcast should hide the moon, got %q", out)
	}
	full := joined(RenderSky(SkyView{Night: true, HasMoon: true, Moon: sky.Full}, overcast(), true, true))
	if !strings.Contains(full, "Clouds hide the moon") {
		t.Fatalf("full report should say clouds hide the moon, got %q", full)
	}
}

func TestRenderSkyUntrackedNightStillShowsMoon(t *testing.T) {
	out := joined(RenderSky(SkyView{Night: true, HasMoon: true, Moon: sky.WaxingGibbous}, Condition{}, false, false))
	if !strings.Contains(out, "waxing gibbous") {
		t.Fatalf("untracked outdoor night should still show the moon, got %q", out)
	}
}

func TestRenderSkyNoMoonWhenMoonless(t *testing.T) {
	out := joined(RenderSky(SkyView{Night: true, HasMoon: false, Moon: sky.Full}, clear(), true, true))
	if strings.Contains(out, "moon") {
		t.Fatalf("a world with no moon should never mention one, got %q", out)
	}
}

func TestRenderSkyIndoorHidesWeather(t *testing.T) {
	if lines := RenderSky(SkyView{Indoor: true, Night: true, HasMoon: true}, clear(), true, false); len(lines) != 0 {
		t.Fatalf("indoor room with no outdoor exit should show nothing on look, got %q", lines)
	}
	full := joined(RenderSky(SkyView{Indoor: true}, clear(), true, true))
	if strings.Contains(full, clear().Description) || !strings.Contains(full, "can't see the sky") {
		t.Fatalf("indoor weather report should refuse, got %q", full)
	}
}

func TestRenderSkyIndoorGlimpse(t *testing.T) {
	out := joined(RenderSky(SkyView{Indoor: true, GlimpseExit: "north"}, clear(), true, false))
	if !strings.Contains(out, "north") || !strings.Contains(out, clear().Description) {
		t.Fatalf("expected a glimpse through the north exit, got %q", out)
	}
}

func TestRenderSkyFullReportsFogAndCloud(t *testing.T) {
	out := joined(RenderSky(SkyView{}, fog(), true, true))
	if !strings.Contains(out, "broken clouds") || !strings.Contains(strings.ToLower(out), "fog") {
		t.Fatalf("expected cloud cover and fog in the full report, got %q", out)
	}
	brief := joined(RenderSky(SkyView{}, fog(), true, false))
	if strings.Contains(brief, "broken clouds") {
		t.Fatalf("look should not list cloud cover, got %q", brief)
	}
}

func TestRenderSkyFullUntrackedDay(t *testing.T) {
	out := joined(RenderSky(SkyView{}, Condition{}, false, true))
	if !strings.Contains(out, "can't tell what the weather is like") {
		t.Fatalf("expected untracked message, got %q", out)
	}
}
