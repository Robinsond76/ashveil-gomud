package scripts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// e2Sides are the 16 road and coast pieces: the joined edges in n, e, s, w
// order, or none.
var e2Sides = []string{"none", "n", "e", "s", "w", "ns", "ew", "ne", "es", "sw", "nw", "nes", "esw", "nsw", "new", "nesw"}

// e2Expected lists every runtime file the A2-A5 art fills (E2) with its 1x
// frame [w, h] and frame count. Runtime frames are these times 4.
func e2Expected() map[string][3]int {
	want := map[string][3]int{}
	for _, b := range []string{"cave", "city", "cliffs", "default", "desert", "dungeon", "farmland", "forest", "fort", "house", "land", "mountains", "slums", "snow", "spiderweb", "swamp", "water"} {
		want["map/terrain/"+b+".png"] = [3]int{32, 32, 3}
	}
	for _, b := range []string{"water", "swamp", "snow", "desert"} {
		want["map/terrain/"+b+"-anim.png"] = [3]int{32, 32, 4}
	}
	for _, s := range e2Sides {
		want["map/terrain/road-"+s+".png"] = [3]int{32, 32, 1}
		want["map/terrain/shore-"+s+".png"] = [3]int{32, 32, 1}
	}
	want["map/terrain/fog.png"] = [3]int{32, 32, 1}
	want["map/terrain/unknown.png"] = [3]int{32, 32, 1}
	for _, l := range []string{"inn", "bank", "shop", "smithy", "herbalist", "trainer", "temple", "shaman", "hermit", "gate", "wall", "bridge", "keep", "throne", "townsquare", "village", "caravan", "lake-house", "cave-mouth", "dungeon-stair", "obelisk", "pond", "rocks", "desert-ruin", "alts", "landmark", "boss-lair"} {
		want["map/landmarks/"+l+".png"] = [3]int{32, 32, 1}
	}
	for _, r := range []string{"water", "forage", "herbs", "firewood", "shelter", "fishing", "game", "unknown", "depleted"} {
		want["map/resources/"+r+".png"] = [3]int{16, 16, 1}
	}
	for name, f := range map[string][3]int{"here-ring": {32, 32, 4}, "company-badge": {12, 12, 1}, "ally-banner": {16, 16, 2}, "walk-target": {16, 16, 2}, "walk-dot": {8, 8, 1}, "exit-up": {8, 8, 1}, "exit-down": {8, 8, 1}} {
		want["map/markers/"+name+".png"] = f
	}
	for name, f := range map[string][3]int{"tent": {32, 32, 1}, "tent-ally": {32, 32, 1}, "camp-rough": {32, 32, 1}, "camp-rough-ally": {32, 32, 1}, "fire-unlit": {16, 16, 1}, "fire-lit": {16, 16, 4}, "embers": {16, 16, 3}, "smoke": {16, 16, 4}, "resting": {16, 16, 3}, "inn-rest": {16, 16, 1}} {
		want["map/camp/"+name+".png"] = f
	}
	groups := map[string][]string{
		"status":     {"bleeding", "staggered", "knocked-down", "stunned", "armor-broken", "exposed", "hobbled", "burning", "overloaded", "poisoned", "asleep", "paralyzed", "blighted", "weakened", "hamstrung", "winded", "tackled", "cold", "regenerating", "lit", "hidden", "chanting", "winding-up", "warded", "wounded-light", "wounded-lasting", "dread", "tracked"},
		"roles":      {"fighter", "healer", "caster", "guardian", "controller"},
		"morale":     {"hold", "yield", "flee", "nerve", "shaken"},
		"conditions": {"dark", "ambush", "narrow", "cold", "fatigue", "flanked", "cluster"},
	}
	for g, ids := range groups {
		for _, id := range ids {
			want["ui/"+g+"/"+id+".png"] = [3]int{16, 16, 1}
		}
	}
	for name, f := range map[string][3]int{"cell": {32, 16, 1}, "cell-acting": {32, 16, 4}, "cell-targeted": {32, 16, 2}, "acting-arrow": {8, 8, 2}, "fallen": {16, 16, 1}, "surrendered": {16, 16, 1}, "hp-frame": {32, 6, 1}} {
		want["battle/ui/"+name+".png"] = f
	}
	return want
}

// TestCommissionedTerrainAndIconsAreImported (E2): every A2-A5 file is
// imported at density 4 with its 1x frames times 4, within the standards'
// budgets (60 KB per terrain-sized frame, 12 KB per icon frame); terrain is
// opaque except the fog; animated biomes are replace sheets their base
// sheet points at; app icons are at their exact sizes.
func TestCommissionedTerrainAndIconsAreImported(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	want := e2Expected()
	if len(want) != 160 {
		t.Fatalf("expected 160 density-4 files, have %d", len(want))
	}
	for rel, f := range want {
		meta, ok := m.Files[rel]
		if !ok {
			t.Errorf("manifest is missing %s", rel)
			continue
		}
		if meta.Source != "imported" || meta.density() != 4 {
			t.Errorf("%s: source %q density %d, want imported density 4", rel, meta.Source, meta.density())
			continue
		}
		if len(meta.Frame) != 2 || meta.Frame[0] != f[0]*4 || meta.Frame[1] != f[1]*4 || meta.Frames != f[2] {
			t.Errorf("%s: %d frames of %v, want %d of %dx%d", rel, meta.Frames, meta.Frame, f[2], f[0]*4, f[1]*4)
			continue
		}
		budget := int64(12 * 1024)
		if f[0] >= 32 {
			budget = 60 * 1024
		}
		info, err := os.Stat(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if info.Size() > budget*int64(f[2]) {
			t.Errorf("%s is %d bytes, over %d (%d frames)", rel, info.Size(), budget*int64(f[2]), f[2])
		}
		if strings.HasPrefix(rel, "map/terrain/") && rel != "map/terrain/fog.png" {
			img := readSprite(t, dir, rel)
			b := img.Bounds()
		opaque:
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
						t.Errorf("%s: not opaque at %d,%d", rel, x, y)
						break opaque
					}
				}
			}
		}
	}
	for _, b := range []string{"water", "swamp", "snow", "desert"} {
		raw := loadRawEntry(t, dir, "map/terrain/"+b+".png")
		if raw["animated_overlay"] != b+"-anim.png" {
			t.Errorf("%s.png: animated_overlay %v, want %s-anim.png", b, raw["animated_overlay"], b)
		}
		anim := loadRawEntry(t, dir, "map/terrain/"+b+"-anim.png")
		if anim["replace"] != true || anim["frame_ms"] != float64(250) {
			t.Errorf("%s-anim.png: replace %v frame_ms %v, want a replace sheet at 250 ms", b, anim["replace"], anim["frame_ms"])
		}
	}
	for name, px := range map[string]int{"icon-512": 512, "icon-192": 192, "icon-maskable-512": 512, "favicon-32": 32} {
		rel := "app/" + name + ".png"
		meta := m.Files[rel]
		if meta.Source != "imported" || meta.density() != 1 || len(meta.Size) != 2 || meta.Size[0] != px || meta.Size[1] != px {
			t.Errorf("%s: source %q density %d size %v, want imported %dx%d", rel, meta.Source, meta.density(), meta.Size, px, px)
		}
	}
}

// loadRawEntry is one manifest entry as plain JSON, for fields spriteMeta
// leaves out.
func loadRawEntry(t *testing.T, dir, rel string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Files map[string]map[string]any `json:"files"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Files[rel]
}
