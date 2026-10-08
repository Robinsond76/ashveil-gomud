package gametime

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"
)

// xtermBase is the standard xterm palette for colors 0-15.
var xtermBase = [16][3]int{
	{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0}, {0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
	{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// xtermLuminance returns the WCAG relative luminance of an xterm 256-color
// index: the base 16, the color cube (16-231) or grayscale (232-255).
func xtermLuminance(n int) float64 {
	var r, g, b int
	switch {
	case n < 16:
		r, g, b = xtermBase[n][0], xtermBase[n][1], xtermBase[n][2]
	case n >= 232:
		r = 8 + (n-232)*10
		g, b = r, r
	default:
		levels := []int{0, 95, 135, 175, 215, 255}
		n -= 16
		r, g, b = levels[n/36], levels[(n/6)%6], levels[n%6]
	}
	f := func(c int) float64 {
		v := float64(c) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*f(r) + 0.7152*f(g) + 0.0722*f(b)
}

// TestTimeOfDayColorsReadableOnBlack keeps the day, dusk and night clock
// colors, and text aliases that were once as dark as the old night blue,
// legible (4.5:1) against a black terminal.
func TestTimeOfDayColorsReadableOnBlack(t *testing.T) {
	for _, world := range []string{"default", "empty"} {
		raw, err := os.ReadFile("../../_datafiles/world/" + world + "/ansi-aliases.yaml")
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"night", "day", "day-dusk", "role-mod", "holy", "item-nothing", "item-cursed", "md-hr1", "md-hr2"} {
			m := regexp.MustCompile(`(?m)^\s+` + name + `:\s*(\d+)\s*(?:#.*)?$`).FindSubmatch(raw)
			if m == nil {
				t.Fatalf("%s: alias %q not found", world, name)
			}
			n, _ := strconv.Atoi(string(m[1]))
			if ratio := (xtermLuminance(n) + 0.05) / 0.05; ratio < 4.5 {
				t.Errorf("%s: alias %q (color %d) contrast %.2f is below 4.5", world, name, n, ratio)
			}
		}
	}
}
