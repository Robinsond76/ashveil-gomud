package scripts

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommissionedBattleArtIsImported (E3): every battle unit and backdrop
// with an approved master is imported as exact pixel art: density 2 (8/3
// for small creatures), frames the 1x frame times the density, 4 idle
// frames, within 60 KB a frame (400 KB a backdrop); and each base class
// keeps a 1x look sheet the creation panel's skin and hair swap repaints.
func TestCommissionedBattleArtIsImported(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	oneX := map[string]int{"S": 48, "M": 64, "L": 72, "XL": 96}
	units := 0
	for rel, meta := range m.Files {
		if !strings.HasPrefix(rel, "battle/units/") || !strings.HasSuffix(rel, "/idle.png") {
			continue
		}
		units++
		if meta.Source != "imported" {
			t.Errorf("%s: not imported (every battle unit has an approved master)", rel)
			continue
		}
		want := 2.0
		if meta.SizeClass == "S" {
			want = 8.0 / 3
		}
		if math.Abs(meta.densityF()-want) > 1e-9 {
			t.Errorf("%s: density %v, want %v", rel, meta.densityF(), want)
		}
		if f := float64(oneX[meta.SizeClass]) * want; len(meta.Frame) != 2 || math.Abs(float64(meta.Frame[0])-f) > 0.01 || meta.Frames != 4 {
			t.Errorf("%s: %d frames of %v, want 4 of %v", rel, meta.Frames, meta.Frame, f)
		}
		if info, err := os.Stat(filepath.Join(dir, rel)); err != nil || info.Size() > 4*60*1024 {
			t.Errorf("%s: over the 240 KB budget (%v)", rel, err)
		}
	}
	if units != 159 {
		t.Errorf("found %d battle units, want 159 (158 roster units and the warhound)", units)
	}
	backdrops := 0
	for rel, meta := range m.Files {
		if !strings.HasPrefix(rel, "battle/backgrounds/") {
			continue
		}
		backdrops++
		if meta.Source != "imported" || meta.density() != 2 || len(meta.Size) != 2 || meta.Size[0] != 640 || meta.Size[1] != 360 {
			t.Errorf("%s: source %q density %v size %v, want imported 640x360 at density 2", rel, meta.Source, meta.densityF(), meta.Size)
		}
		if info, err := os.Stat(filepath.Join(dir, rel)); err != nil || info.Size() > 400*1024 {
			t.Errorf("%s: over the 400 KB budget (%v)", rel, err)
		}
	}
	if backdrops != 16 {
		t.Errorf("found %d backdrops, want 16", backdrops)
	}
	for _, c := range baseClasses {
		rel := "battle/units/" + c + "/look.png"
		meta, ok := m.Files[rel]
		if !ok {
			t.Errorf("manifest is missing %s (the creation panel's look preview)", rel)
			continue
		}
		if meta.Kind != "battle-look" || meta.density() != 1 || meta.Source == "imported" || len(meta.Frame) != 2 || meta.Frame[0] != 64 {
			t.Errorf("%s: kind %q density %v source %q frame %v, want a drawn 1x battle-look", rel, meta.Kind, meta.densityF(), meta.Source, meta.Frame)
		}
	}
}

// writeStrip writes a 1 x 4 battle master of cell x cell frames with equal
// gutters, each frame filled by fill(frame, x, y).
func writeStrip(t *testing.T, root, rel string, cell, gutter int, fill func(f, x, y int) color.NRGBA) {
	t.Helper()
	w := 4*cell + 3*gutter
	img := image.NewNRGBA(image.Rect(0, 0, w, cell))
	for f := 0; f < 4; f++ {
		for y := 0; y < cell; y++ {
			for x := 0; x < cell; x++ {
				img.SetNRGBA(f*(cell+gutter)+x, y, fill(f, x, y))
			}
		}
	}
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		t.Fatal(err)
	}
}

// TestImportBattle runs import_battle.py (E3) on synthetic masters: a
// person drawn on a 5 px grain imports as exact 128 px frames with its feet
// row; a frame offset from the grain is shifted onto it; a gutter pixel, art
// off any grain, a wrong cell size and a unit the roster doesn't know are
// refused.
func TestImportBattle(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import PIL, numpy").Run(); err != nil {
		t.Skip("Pillow or numpy not available")
	}
	script, err := filepath.Abs(filepath.Join("sprites", "import_battle.py"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(src string, only ...string) (string, string, error) {
		out := t.TempDir()
		args := []string{"-I", script, "--src", src, "--out", out}
		if len(only) > 0 {
			args = append(append(args, "--only"), only...)
		}
		b, err := exec.Command("python3", args...).CombinedOutput()
		return out, string(b), err
	}
	readPNG := func(path string) image.Image {
		t.Helper()
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	// A body of 5 px art pixels: columns 50-74 art px, rows 40-115, so its
	// feet are on art row 115 (master y 575-579); frame 3's foot reaches
	// one art row lower (116), so the sheet's feet are the lowest of all.
	// Frame f breathes by lifting its top f art pixels.
	body := func(shift int) func(f, x, y int) color.NRGBA {
		return func(f, x, y int) color.NRGBA {
			ax, ay := (x-shift)/5, (y-shift)/5
			low := 115
			if f == 2 && ax < 55 {
				low = 116
			}
			if x < shift || y < shift || ax < 50 || ax > 74 || ay < 40+f || ay > low {
				return color.NRGBA{}
			}
			if (ax+ay)%2 == 0 {
				return color.NRGBA{180, 60, 40, 255}
			}
			return color.NRGBA{60, 40, 30, 255}
		}
	}

	src := t.TempDir()
	writeStrip(t, src, "A6/battle/units/warrior/idle.png", 640, 40, body(0))
	out, msg, err := run(src)
	if err != nil {
		t.Fatalf("a good master was refused: %v\n%s", err, msg)
	}
	img := readPNG(filepath.Join(out, "battle/units/warrior/idle.png"))
	if b := img.Bounds(); b.Dx() != 512 || b.Dy() != 128 {
		t.Fatalf("sheet is %v, want 512x128", b)
	}
	// Exact: art pixel (50, 115) is red, (51, 115) dark, nothing partial.
	if r, _, _, a := img.At(50, 115).RGBA(); r>>8 != 60 && r>>8 != 180 || a>>8 != 255 {
		t.Errorf("art pixel (50,115) = r %d a %d, want an exact art colour", r>>8, a>>8)
	}
	for y := 0; y < 128; y++ {
		for x := 0; x < 512; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 && a != 0xffff {
				t.Fatalf("partial alpha at %d,%d: the shrink was not exact", x, y)
			}
		}
	}
	var index map[string]map[string]any
	raw, _ := os.ReadFile(filepath.Join(out, "imported.json"))
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	e := index["battle/units/warrior/idle.png"]
	if e["density"] != float64(2) || e["feet_baseline"] != float64(116) || e["size_class"] != "M" || e["family"] != "class" {
		t.Errorf("warrior entry = %v (feet are the lowest row of any frame)", e)
	}
	good, _ := os.ReadFile(filepath.Join(out, "battle/units/warrior/idle.png"))

	// A sheet drawn 2 px off the grain is shifted onto it, to the same result.
	shifted := t.TempDir()
	writeStrip(t, shifted, "A6/battle/units/warrior/idle.png", 640, 40, body(2))
	sout, msg, err := run(shifted)
	if err != nil {
		t.Errorf("an offset grain should be shifted, not refused: %v\n%s", err, msg)
	} else if got, _ := os.ReadFile(filepath.Join(sout, "battle/units/warrior/idle.png")); string(got) != string(good) {
		t.Error("the shifted import should equal the one drawn on the grain")
	}

	// 3 px off (over half the grain) moves the smaller way, 2 px down and
	// right, onto the next grid line: the same art one art pixel down-right.
	shifted3 := t.TempDir()
	writeStrip(t, shifted3, "A6/battle/units/warrior/idle.png", 640, 40, body(3))
	if s3, msg, err := run(shifted3); err != nil {
		t.Errorf("an offset of 3 should be shifted down and right, not refused: %v\n%s", err, msg)
	} else {
		moved := readPNG(filepath.Join(s3, "battle/units/warrior/idle.png"))
		for y := 0; y < 127; y++ {
			for x := 0; x < 127; x++ {
				if moved.At(x+1, y+1) != img.At(x, y) {
					t.Fatalf("the import shifted by 3 differs from the art one pixel down-right at %d,%d", x, y)
				}
			}
		}
	}

	// A small creature (768 px cells, 6 px grain) imports at density 8/3,
	// and --only imports just the paths it names.
	both := t.TempDir()
	writeStrip(t, both, "A6/battle/units/warrior/idle.png", 640, 40, body(0))
	writeStrip(t, both, "A7/battle/units/rat/idle.png", 768, 48, func(f, x, y int) color.NRGBA {
		if x/6 > 40 && x/6 < 90 && y/6 > 90+f%2 && y/6 <= 115 {
			return color.NRGBA{120, 110, 100, 255}
		}
		return color.NRGBA{}
	})
	rout, msg, err := run(both, "battle/units/rat/idle.png")
	if err != nil {
		t.Fatalf("the rat was refused: %v\n%s", err, msg)
	}
	if _, err := os.Stat(filepath.Join(rout, "battle/units/warrior/idle.png")); err == nil {
		t.Error("--only imported a path it did not name")
	}
	raw, _ = os.ReadFile(filepath.Join(rout, "imported.json"))
	index = nil
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	if d, ok := index["battle/units/rat/idle.png"]["density"].(float64); !ok || math.Abs(d-8.0/3) > 1e-9 {
		t.Errorf("rat density = %v, want 8/3", index["battle/units/rat/idle.png"]["density"])
	}

	// A backdrop on a 4 px grain imports as an exact 640x360 picture.
	bg := t.TempDir()
	{
		im := image.NewNRGBA(image.Rect(0, 0, 2560, 1440))
		for y := 0; y < 1440; y++ {
			for x := 0; x < 2560; x++ {
				ax, ay := x/4, y/4
				im.SetNRGBA(x, y, color.NRGBA{uint8(ax % 16 * 16), uint8(ay % 8 * 32), 90, 255})
			}
		}
		p := filepath.Join(bg, "A6/battle/backgrounds/plains.png")
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		fo, _ := os.Create(p)
		_ = png.Encode(fo, im)
		fo.Close()
	}
	bout, msg, err := run(bg)
	if err != nil {
		t.Fatalf("the backdrop was refused: %v\n%s", err, msg)
	}
	bimg := readPNG(filepath.Join(bout, "battle/backgrounds/plains.png"))
	if b := bimg.Bounds(); b.Dx() != 640 || b.Dy() != 360 {
		t.Fatalf("backdrop is %v, want 640x360", b)
	}
	for _, xy := range [][2]int{{0, 0}, {17, 5}, {333, 200}, {639, 359}} {
		r, g, b, _ := bimg.At(xy[0], xy[1]).RGBA()
		if r>>8 != uint32(xy[0]%16*16) || g>>8 != uint32(xy[1]%8*32) || b>>8 != 90 {
			t.Errorf("backdrop pixel %v = %d,%d,%d, not the exact art pixel", xy, r>>8, g>>8, b>>8)
		}
	}

	for _, tc := range []struct {
		name    string
		write   func(root string)
		problem string
	}{
		{"gutter pixel", func(root string) {
			writeStrip(t, root, "A6/battle/units/warrior/idle.png", 640, 40, body(0))
			// A white speck in the gutter after frame 1.
			p := filepath.Join(root, "A6/battle/units/warrior/idle.png")
			fh, _ := os.Open(p)
			im, _ := png.Decode(fh)
			fh.Close()
			nr := im.(*image.NRGBA)
			nr.SetNRGBA(650, 300, color.NRGBA{255, 255, 255, 255})
			fo, _ := os.Create(p)
			_ = png.Encode(fo, nr)
			fo.Close()
		}, "gutter"},
		{"off grain", func(root string) {
			writeStrip(t, root, "A6/battle/units/warrior/idle.png", 640, 40, func(f, x, y int) color.NRGBA {
				if x > 250 && x < 370 && y > 200 && y < 580 {
					return color.NRGBA{uint8(x * 7), uint8(y * 3), 40, 255}
				}
				return color.NRGBA{}
			})
		}, "not drawn on a 5 px grain"},
		{"wrong cell", func(root string) {
			writeStrip(t, root, "A6/battle/units/warrior/idle.png", 600, 40, body(0))
		}, "not 600"},
		{"unknown unit", func(root string) {
			writeStrip(t, root, "A7/battle/units/no-such-beast/idle.png", 640, 40, body(0))
		}, "not a battle unit"},
		{"off-grain backdrop", func(root string) {
			im := image.NewNRGBA(image.Rect(0, 0, 2560, 1440))
			for y := 0; y < 1440; y++ {
				for x := 0; x < 2560; x++ {
					im.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 0, 255})
				}
			}
			p := filepath.Join(root, "A6/battle/backgrounds/plains.png")
			_ = os.MkdirAll(filepath.Dir(p), 0o755)
			fo, _ := os.Create(p)
			_ = png.Encode(fo, im)
			fo.Close()
		}, "not drawn on a 4 px grain"},
	} {
		root := t.TempDir()
		tc.write(root)
		if _, msg, err := run(root); err == nil || !strings.Contains(msg, tc.problem) {
			t.Errorf("%s: want a refusal naming %q, got err %v: %s", tc.name, tc.problem, err, msg)
		}
	}
}
