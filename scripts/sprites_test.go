package scripts

import (
	"bytes"
	"encoding/json"
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
