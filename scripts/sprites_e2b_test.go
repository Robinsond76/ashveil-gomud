package scripts

import (
	"testing"
)

// TestShoreCornersAreImported (E2b): the four A2b inside-corner overlays
// ship at density 4 (128 px for a 32 px tile), each opaque water in its own
// corner and fully transparent in the opposite one, so drawn over a shore
// piece it adds only the lake's corner.
func TestShoreCornersAreImported(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	// The corner pixel each overlay paints, and the opposite one it leaves clear.
	for c, px := range map[string][2][2]int{
		"ne": {{127, 0}, {0, 127}},
		"es": {{127, 127}, {0, 0}},
		"sw": {{0, 127}, {127, 0}},
		"nw": {{0, 0}, {127, 127}},
	} {
		rel := "map/terrain/shore-corner-" + c + ".png"
		meta, ok := m.Files[rel]
		if !ok {
			t.Errorf("manifest is missing %s", rel)
			continue
		}
		if meta.Kind != "terrain-corner" || meta.Source != "imported" || meta.density() != 4 || len(meta.Frame) != 2 || meta.Frame[0] != 128 || meta.Frames != 1 {
			t.Errorf("%s: kind %q source %q density %d frame %v x%d, want an imported 128 px terrain-corner", rel, meta.Kind, meta.Source, meta.density(), meta.Frame, meta.Frames)
			continue
		}
		img := readSprite(t, dir, rel)
		if _, _, b, a := img.At(px[0][0], px[0][1]).RGBA(); a != 0xffff || b == 0 {
			t.Errorf("%s: corner pixel %v is not opaque water (b %d a %d)", rel, px[0], b>>8, a>>8)
		}
		if _, _, _, a := img.At(px[1][0], px[1][1]).RGBA(); a != 0 {
			t.Errorf("%s: the opposite corner %v is painted (a %d); the overlay must be clear there", rel, px[1], a>>8)
		}
		painted := 0
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
					painted++
				}
			}
		}
		if share := float64(painted) / float64(128*128); share < 0.05 || share > 0.25 {
			t.Errorf("%s: %.0f%% painted, want a corner of water (5-25%%)", rel, share*100)
		}
	}
}
