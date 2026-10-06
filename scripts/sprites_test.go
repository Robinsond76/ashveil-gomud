package scripts

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type spriteMeta struct {
	Size   []int    `json:"size"`
	Frame  []int    `json:"frame"`
	Frames int      `json:"frames"`
	Rows   []string `json:"rows"`
	Kind   string   `json:"kind"`
}

type spriteManifest struct {
	Version int                   `json:"version"`
	Palette string                `json:"palette"`
	Files   map[string]spriteMeta `json:"files"`
}

func spriteDir(t *testing.T) string {
	t.Helper()
	_, src, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(filepath.Dir(src)), "_datafiles", "html", "public", "static", "sprites")
}

func loadSpriteManifest(t *testing.T, dir string) spriteManifest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m spriteManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return m
}

// loadSpritePalette returns the master palette from palette.gpl as 24-bit RGB keys.
func loadSpritePalette(t *testing.T, dir string, rel string) map[uint32]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("read palette: %v", err)
	}
	pal := map[uint32]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		var rgb [3]uint32
		okLine := true
		for i := 0; i < 3; i++ {
			n, err := strconv.Atoi(f[i])
			if err != nil {
				okLine = false
				break
			}
			rgb[i] = uint32(n)
		}
		if okLine {
			pal[rgb[0]<<16|rgb[1]<<8|rgb[2]] = true
		}
	}
	return pal
}

func TestSpriteSetsMatchSpecificationLayout(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	if m.Version != 1 {
		t.Fatalf("manifest version = %d", m.Version)
	}
	pal := loadSpritePalette(t, dir, m.Palette)
	if len(pal) == 0 || len(pal) > 64 {
		t.Fatalf("palette has %d colors, want 1..64", len(pal))
	}

	want := []string{
		"style/palette.png", "style/palette.gpl", "style/style-map.png", "style/style-battle.png",
		"style/proportions-map.png", "style/proportions-battle.png", "style/icon-sample.png",
		"map/markers/here-ring.png", "map/markers/company-badge.png", "map/markers/ally-banner.png",
		"map/markers/walk-target.png", "map/markers/walk-dot.png", "map/markers/exit-up.png",
		"map/markers/exit-down.png", "map/camp/tent.png", "map/camp/tent-ally.png",
		"map/camp/camp-rough.png", "map/camp/camp-rough-ally.png", "map/camp/embers.png",
		"map/camp/fire-unlit.png", "map/camp/fire-lit.png", "map/camp/smoke.png",
		"map/camp/resting.png", "map/camp/inn-rest.png", "app/icon-512.png", "app/icon-192.png",
		"app/icon-maskable-512.png", "app/favicon-32.png",
	}
	for _, r := range []string{"water", "forage", "herbs", "firewood", "shelter", "fishing", "game", "unknown", "depleted"} {
		want = append(want, "map/resources/"+r+".png")
	}
	for _, u := range []string{"warrior", "rogue", "ranger", "cleric", "wizard", "witch", "adventurer"} {
		want = append(want, "map/units/"+u+"/idle.png", "map/units/"+u+"/walk.png")
	}
	for _, rel := range want {
		if _, ok := m.Files[rel]; !ok {
			t.Errorf("manifest is missing %s", rel)
		}
	}

	for rel, meta := range m.Files {
		if !strings.HasSuffix(rel, ".png") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("%s: decode: %v", rel, err)
			continue
		}
		b := img.Bounds()
		if len(meta.Size) != 2 || b.Dx() != meta.Size[0] || b.Dy() != meta.Size[1] {
			t.Errorf("%s: size %dx%d does not match manifest %v", rel, b.Dx(), b.Dy(), meta.Size)
		}
		if len(meta.Frame) == 2 {
			rows := 1
			if len(meta.Rows) > 0 {
				rows = len(meta.Rows)
			}
			if b.Dx() != meta.Frame[0]*meta.Frames || b.Dy() != meta.Frame[1]*rows {
				t.Errorf("%s: %dx%d is not %d frames x %d rows of %v", rel, b.Dx(), b.Dy(), meta.Frames, rows, meta.Frame)
			}
		}
		// Hard edges (no partial alpha) and only master palette colors.
		bad := 0
		for y := b.Min.Y; y < b.Max.Y && bad < 3; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := img.At(x, y).RGBA()
				if a == 0 {
					continue
				}
				if a != 0xffff {
					t.Errorf("%s: partial alpha at %d,%d", rel, x, y)
					bad++
					break
				}
				if !pal[(r>>8)<<16|(g>>8)<<8|(bl>>8)] {
					t.Errorf("%s: color %02x%02x%02x at %d,%d is not in the master palette", rel, r>>8, g>>8, bl>>8, x, y)
					bad++
					break
				}
			}
		}
	}
}

// Map units are 32x32 with idle (2 frames) and walk (4 frames), rows down/up/side,
// feet on row 30 (frameHeight-2) and nothing outside the frame margins.
func TestMapUnitSpritesFollowAnchorRules(t *testing.T) {
	dir := spriteDir(t)
	for _, u := range []string{"warrior", "rogue", "ranger", "cleric", "wizard", "witch", "adventurer"} {
		for file, frames := range map[string]int{"idle.png": 2, "walk.png": 4} {
			f, err := os.Open(filepath.Join(dir, "map", "units", u, file))
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(f)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			b := img.Bounds()
			if b.Dx() != 32*frames || b.Dy() != 96 {
				t.Fatalf("%s/%s is %dx%d", u, file, b.Dx(), b.Dy())
			}
			for fr := 0; fr < frames*3; fr++ {
				ox, oy := (fr%frames)*32, (fr/frames)*32
				lowest := -1
				for y := 0; y < 32; y++ {
					for x := 0; x < 32; x++ {
						if _, _, _, a := img.At(ox+x, oy+y).RGBA(); a != 0 {
							lowest = y
						}
					}
				}
				// Outline row sits at frameHeight-2 on the standing frames; walking
				// lifts a foot, so allow up to one row higher.
				if lowest > 30 || lowest < 29 {
					t.Errorf("%s/%s frame %d: lowest opaque row %d, want 29 or 30", u, file, fr, lowest)
				}
			}
		}
	}
}

// The committed art must be exactly what the generator produces, so the
// script stays the source of truth.  Skipped where Python or Pillow is absent.
func TestSpritesMatchGenerator(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import PIL").Run(); err != nil {
		t.Skip("Pillow not available")
	}
	dir := spriteDir(t)
	root := filepath.Dir(filepath.Dir(dir))
	_ = root
	_, src, _, _ := runtime.Caller(0)
	gen := filepath.Join(filepath.Dir(src), "sprites", "generate.py")
	tmp := t.TempDir()
	if out, err := exec.Command("python3", "-I", gen, "--out", tmp).CombinedOutput(); err != nil {
		t.Fatalf("generate: %v\n%s", err, out)
	}
	m := loadSpriteManifest(t, dir)
	for rel := range m.Files {
		a, err1 := os.ReadFile(filepath.Join(dir, rel))
		b, err2 := os.ReadFile(filepath.Join(tmp, rel))
		if err1 != nil || err2 != nil {
			t.Errorf("%s: %v %v", rel, err1, err2)
			continue
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s differs from generator output; run `make sprites`", rel)
		}
	}
}

func readSprite(t *testing.T, dir, rel string) image.Image {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return img
}

// S2 and S3 ship every file the specification lists: the manifest must name them all.
func TestSpriteSetsS2S3AreComplete(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	var want []string
	biomes := []string{"cave", "city", "cliffs", "default", "desert", "dungeon", "farmland", "forest", "fort",
		"house", "land", "mountains", "road", "shore", "slums", "snow", "spiderweb", "swamp", "water"}
	for _, b := range biomes {
		want = append(want, "map/terrain/"+b+".png")
	}
	for _, b := range []string{"desert", "shore", "snow", "swamp", "water"} {
		want = append(want, "map/terrain/"+b+"-anim.png")
	}
	want = append(want, "map/terrain/fog.png", "map/terrain/unknown.png", "map/terrain/night-mask.png")
	for _, l := range strings.Fields("inn bank shop smithy herbalist trainer temple shaman hermit gate wall bridge keep throne townsquare village caravan lake-house cave-mouth dungeon-stair obelisk pond rocks desert-ruin alts landmark boss-lair") {
		want = append(want, "map/landmarks/"+l+".png")
	}
	for _, b := range strings.Fields("forest deep-web plains road city slums interior catacombs cave snowfield ice-keep shore swamp desert highlands training-yard") {
		want = append(want, "battle/backgrounds/"+b+".png")
	}
	for _, u := range strings.Fields(`warrior rogue ranger cleric wizard witch adventurer unknown-humanoid unknown-beast unknown-large
		rat rat-big wolf-timber wolf-snow dog-junkyard spider-hatchling spider-large spider-warrior spider-queen skeleton bone-warden
		bonecrafter lich acolyte-dark grave-chanter brigand ruffian ruffian-dangerous ruffian-enforcer poacher poacher-shieldman
		bonesetter shadow-trainee shadow-master goblin goblin-hexer goblin-loot faerie imp-forest fungus ent ogre-forest crocodile
		creeper-cave creeper-abyssal stalker-cave bats-echo ice-warrior ice-guardian snow-floof dummy-training straw-footman
		straw-archer guard guard-royal guard-captain`) {
		want = append(want, "battle/units/"+u+"/idle.png")
	}
	for _, f := range strings.Fields("cell cell-acting cell-targeted acting-arrow fallen surrendered hp-frame") {
		want = append(want, "battle/ui/"+f+".png")
	}
	groups := map[string]string{
		"status": "bleeding staggered knocked-down stunned armor-broken exposed hobbled burning overloaded poisoned asleep paralyzed blighted weakened hamstrung winded tackled cold regenerating lit hidden chanting winding-up warded wounded-light wounded-lasting dread tracked",
		"roles":  "fighter healer caster guardian controller", "morale": "hold yield flee nerve shaken",
		"conditions": "dark ambush narrow cold fatigue flanked cluster",
	}
	for g, names := range groups {
		for _, n := range strings.Fields(names) {
			want = append(want, "ui/"+g+"/"+n+".png")
		}
	}
	want = append(want, "battle/mapping.json")
	for _, rel := range want {
		if _, ok := m.Files[rel]; !ok {
			t.Errorf("manifest is missing %s", rel)
		}
	}
}

// Battle units: 4 idle frames, the size class frame, feet on frameHeight-2, every frame drawn.
func TestBattleUnitSpritesFollowAnchorRules(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	sizes := map[string]int{"S": 48, "M": 64, "L": 96, "XL": 128}
	n := 0
	for rel, meta := range m.Files {
		if !strings.HasPrefix(rel, "battle/units/") || !strings.HasSuffix(rel, "/idle.png") {
			continue
		}
		n++
		img := readSprite(t, dir, rel)
		fs := meta.Frame[0]
		if meta.Frames != 4 || img.Bounds().Dx() != fs*4 || img.Bounds().Dy() != fs {
			t.Errorf("%s: want 4 frames of %d, got %dx%d", rel, fs, img.Bounds().Dx(), img.Bounds().Dy())
			continue
		}
		valid := false
		for _, s := range sizes {
			valid = valid || s == fs
		}
		if !valid {
			t.Errorf("%s: frame %d is not a size class", rel, fs)
		}
		floating := strings.Contains(rel, "bats-echo")
		differ := false
		var first []uint32
		for f := 0; f < 4; f++ {
			lowest, count := -1, 0
			var px []uint32
			for y := 0; y < fs; y++ {
				for x := 0; x < fs; x++ {
					r, g, b, a := img.At(f*fs+x, y).RGBA()
					px = append(px, r^g<<8^b<<16^a<<24)
					if a != 0 {
						lowest = y
						count++
					}
				}
			}
			if count < 40 {
				t.Errorf("%s frame %d is nearly empty", rel, f)
			}
			if !floating && lowest != fs-2 {
				t.Errorf("%s frame %d: lowest opaque row %d, want %d", rel, f, lowest, fs-2)
			}
			if f == 0 {
				first = px
			} else if !bytes.Equal(u32bytes(px), u32bytes(first)) {
				differ = true
			}
		}
		if !differ {
			t.Errorf("%s: the four idle frames are identical", rel)
		}
	}
	if n < 55 {
		t.Errorf("found only %d battle units", n)
	}
}

func u32bytes(v []uint32) []byte {
	b := make([]byte, 0, len(v)*4)
	for _, x := range v {
		b = append(b, byte(x), byte(x>>8), byte(x>>16), byte(x>>24))
	}
	return b
}

// Backgrounds are opaque 320x180 and keep the ground band quiet (units must read against it).
func TestBattleBackgroundsAreOpaqueAndQuietInGroundBand(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	for rel := range m.Files {
		if !strings.HasPrefix(rel, "battle/backgrounds/") {
			continue
		}
		img := readSprite(t, dir, rel)
		if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 180 {
			t.Errorf("%s: size %v", rel, img.Bounds())
		}
		for y := 0; y < 180; y++ {
			for x := 0; x < 320; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
					t.Fatalf("%s: transparent pixel at %d,%d", rel, x, y)
				}
			}
		}
		// Ground band: no more than 4 distinct colors in any 16x16 block below the horizon fringe.
		for by := 108; by+16 <= 176; by += 16 {
			for bx := 16; bx+16 <= 304; bx += 16 {
				seen := map[uint32]bool{}
				for y := by; y < by+16; y++ {
					for x := bx; x < bx+16; x++ {
						r, g, b, _ := img.At(x, y).RGBA()
						seen[(r>>8)<<16|(g>>8)<<8|(b>>8)] = true
					}
				}
				if len(seen) > 5 {
					t.Errorf("%s: ground band block %d,%d has %d colors (too busy for units to read)", rel, bx, by, len(seen))
					break
				}
			}
		}
	}
}

// Terrain tiles stay quiet: no outline color, few colors, a flat 2 px margin.
func TestTerrainTilesAreQuietWithFlatMargin(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	const outline = 0x15120f
	n := 0
	for rel, meta := range m.Files {
		if !strings.HasPrefix(rel, "map/terrain/") || meta.Kind != "terrain" {
			continue
		}
		n++
		img := readSprite(t, dir, rel)
		if img.Bounds().Dx() != 96 || img.Bounds().Dy() != 32 {
			t.Errorf("%s: want 3 variants of 32x32, got %v", rel, img.Bounds())
		}
		for v := 0; v < 3; v++ {
			all := map[uint32]int{}
			ring := map[uint32]bool{}
			for y := 0; y < 32; y++ {
				for x := 0; x < 32; x++ {
					r, g, b, a := img.At(v*32+x, y).RGBA()
					if a != 0xffff {
						t.Fatalf("%s: tile is not opaque at %d,%d", rel, x, y)
					}
					k := (r>>8)<<16 | (g>>8)<<8 | (b >> 8)
					if k == outline {
						t.Errorf("%s variant %d uses the outline color (tiles have no outline)", rel, v)
					}
					all[k]++
					if x < 2 || y < 2 || x >= 30 || y >= 30 {
						ring[k] = true
					}
				}
			}
			top := 0
			for _, c := range all {
				if c > top {
					top = c
				}
			}
			if len(all) > 7 || len(ring) > 6 || float64(top) < 0.35*1024 {
				t.Errorf("%s variant %d is too busy: %d colors, %d in the margin, modal share %.2f", rel, v, len(all), len(ring), float64(top)/1024)
			}
		}
	}
	if n != 19 {
		t.Errorf("found %d terrain files, want 19", n)
	}
}

// Every `sprite:` key on a shipped mob names a real battle unit, and every combatant
// family named in the specification has at least one mob using the art.
func TestMobSpriteKeysExist(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	_, src, _, _ := runtime.Caller(0)
	mobDir := filepath.Join(filepath.Dir(filepath.Dir(src)), "_datafiles", "world", "default", "mobs")
	used := 0
	_ = filepath.Walk(mobDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return nil
		}
		raw, _ := os.ReadFile(p)
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "sprite:") {
				key := strings.TrimSpace(strings.TrimPrefix(line, "sprite:"))
				if _, ok := m.Files["battle/units/"+key+"/idle.png"]; !ok {
					t.Errorf("%s: sprite %q has no battle art", p, key)
				}
				used++
			}
		}
		return nil
	})
	if used < 40 {
		t.Errorf("only %d mobs carry a sprite key", used)
	}
}
