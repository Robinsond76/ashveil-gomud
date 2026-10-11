package scripts

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLookMasksAreImported (E1b): every map figure and battle idle whose A11
// masks mark skin or hair ships a look mask, named by the sheet's look_mask,
// the sheet's size, red for skin and green for hair only, and marking only
// pixels the sheet draws.
func TestLookMasksAreImported(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	counts := map[string]int{}
	for rel, meta := range m.Files {
		if meta.LookMask == "" {
			continue
		}
		switch {
		case strings.HasPrefix(rel, "map/units/") && strings.HasSuffix(rel, "/idle.png"):
			counts["map idle"]++
		case strings.HasPrefix(rel, "map/units/") && strings.HasSuffix(rel, "/walk.png"):
			counts["map walk"]++
		case strings.HasPrefix(rel, "battle/units/") && strings.HasSuffix(rel, "/idle.png"):
			counts["battle idle"]++
		default:
			counts["other "+rel]++
		}
		mask, ok := m.Files[meta.LookMask]
		if !ok {
			t.Errorf("%s: look_mask %s is not in the manifest", rel, meta.LookMask)
			continue
		}
		if mask.Kind != "look-mask" || mask.Of != rel || mask.densityF() != meta.densityF() || len(mask.Size) != 2 || mask.Size[0] != meta.Size[0] || mask.Size[1] != meta.Size[1] {
			t.Errorf("%s: mask entry %+v does not match its sheet (size %v density %v)", meta.LookMask, mask, meta.Size, meta.densityF())
			continue
		}
		sheet, look := readSprite(t, dir, rel), readSprite(t, dir, meta.LookMask)
		if sheet.Bounds() != look.Bounds() {
			t.Errorf("%s: mask is %v, sheet %v", meta.LookMask, look.Bounds(), sheet.Bounds())
			continue
		}
		marked, bad, outside := 0, 0, 0
		b := look.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := look.At(x, y).RGBA()
				if a == 0 {
					continue
				}
				marked++
				if !(r == 0xffff && g == 0 && bl == 0) && !(r == 0 && g == 0xffff && bl == 0) {
					bad++
				}
				if _, _, _, sa := sheet.At(x, y).RGBA(); sa == 0 {
					outside++
				}
			}
		}
		if marked == 0 || bad > 0 || outside > 0 {
			t.Errorf("%s: %d marked, %d not pure red or green, %d where the sheet is transparent", meta.LookMask, marked, bad, outside)
		}
	}
	// The A11 delivery: 71 map classes and 72 battle idles show skin or hair.
	want := map[string]int{"map idle": 71, "map walk": 71, "battle idle": 72}
	for k, n := range counts {
		if want[k] != n {
			t.Errorf("%d %s sheets have look masks, want %d", n, k, want[k])
		}
	}
	for k, n := range want {
		if counts[k] == 0 && n > 0 {
			t.Errorf("no %s sheet has a look mask", k)
		}
	}
}

// writeMask writes a white-on-transparent mask of w x h marking mark(x, y).
func writeMask(t *testing.T, path string, w, h int, mark func(x, y int) bool) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if mark(x, y) {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// TestImportLookMasks runs both importers (E1b) on synthetic masters and
// masks: the map importer cuts idle.look.png and walk.look.png like their
// sheets and names them; the battle importer cuts idle.look.png; masks that
// mark nothing make no file; a mask of another size, marking transparent
// art, marking a pixel as both skin and hair, or missing is refused.
func TestImportLookMasks(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import PIL, numpy").Run(); err != nil {
		t.Skip("Pillow or numpy not available")
	}
	sheetScript, _ := filepath.Abs(filepath.Join("sprites", "import_sheet.py"))
	battleScript, _ := filepath.Abs(filepath.Join("sprites", "import_battle.py"))
	index := func(out string) map[string]map[string]any {
		t.Helper()
		var ix map[string]map[string]any
		raw, _ := os.ReadFile(filepath.Join(out, "imported.json"))
		if err := json.Unmarshal(raw, &ix); err != nil {
			t.Fatal(err)
		}
		return ix
	}
	const W, H = 8*256 + 7*32, 3*256 + 2*32
	// The master's figure is x 96-159, y 16-239 of each cell: skin on its top
	// 16 rows, hair on the 16 rows below (both on the 4 px grain).
	inFigure := func(x, y int, top, bottom int) bool {
		cx, cy := x%288, y%288
		return x/288 < 8 && y/288 < 3 && cx >= 96 && cx < 160 && cy >= top && cy < bottom
	}
	skin := func(x, y int) bool { return inFigure(x, y, 16, 32) }
	hair := func(x, y int) bool { return inFigure(x, y, 32, 48) }
	none := func(x, y int) bool { return false }

	mapRun := func(masks func(dir string)) (string, string, error) {
		src, mdir, out := t.TempDir(), t.TempDir(), t.TempDir()
		writeMaster(t, filepath.Join(src, "probe.png"), 239, false)
		masks(mdir)
		b, err := exec.Command("python3", "-I", sheetScript, filepath.Join(src, "probe.png"), "probe", "--out", out, "--masks", mdir).CombinedOutput()
		return out, string(b), err
	}
	out, msg, err := mapRun(func(d string) {
		writeMask(t, filepath.Join(d, "probe.skin.png"), W, H, skin)
		writeMask(t, filepath.Join(d, "probe.hair.png"), W, H, hair)
	})
	if err != nil {
		t.Fatalf("good map masks were refused: %v\n%s", err, msg)
	}
	ix := index(out)
	for _, name := range []string{"idle", "walk"} {
		rel := "map/units/probe/" + name + ".png"
		if ix[rel]["look_mask"] != "map/units/probe/"+name+".look.png" {
			t.Errorf("%s: look_mask = %v", rel, ix[rel]["look_mask"])
		}
		f, err := os.Open(filepath.Join(out, "map/units/probe/"+name+".look.png"))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		// The figure is x 48-79 in a 128 px frame; skin rows 8-15, hair 16-23.
		r, g, _, a := img.At(60, 10).RGBA()
		r2, g2, _, a2 := img.At(60, 20).RGBA()
		_, _, _, a3 := img.At(60, 60).RGBA()
		if a != 0xffff || r != 0xffff || g != 0 || a2 != 0xffff || g2 != 0xffff || r2 != 0 || a3 != 0 {
			t.Errorf("%s.look.png: skin at (60,10) rgba %d,%d,_,%d; hair at (60,20) %d,%d,_,%d; body (60,60) a %d", name, r>>8, g>>8, a>>8, r2>>8, g2>>8, a2>>8, a3>>8)
		}
	}
	// Re-importing the sheet with no mask files for it (none delivered, or
	// another masks folder) keeps the masks it has (review: the plain
	// --dir command used to delete them).
	again := t.TempDir()
	writeMaster(t, filepath.Join(again, "probe.png"), 239, false)
	if b, err := exec.Command("python3", "-I", sheetScript, filepath.Join(again, "probe.png"), "probe", "--out", out, "--masks", t.TempDir()).CombinedOutput(); err != nil {
		t.Fatalf("a re-import without masks failed: %v\n%s", err, b)
	}
	if _, err := os.Stat(filepath.Join(out, "map/units/probe/idle.look.png")); err != nil || index(out)["map/units/probe/idle.png"]["look_mask"] != "map/units/probe/idle.look.png" || index(out)["map/units/probe/walk.look.png"]["kind"] != "look-mask" {
		t.Errorf("a re-import without mask files dropped the existing masks: %v %v", err, index(out)["map/units/probe/idle.png"])
	}
	// A bad mask anywhere in a run writes nothing at all.
	{
		src, mdir, fresh := t.TempDir(), t.TempDir(), t.TempDir()
		writeMaster(t, filepath.Join(src, "aaa.png"), 239, false)
		writeMaster(t, filepath.Join(src, "zzz.png"), 239, false)
		writeMask(t, filepath.Join(mdir, "zzz.skin.png"), W, H, skin)
		writeMask(t, filepath.Join(mdir, "zzz.hair.png"), W, H, skin)
		b, err := exec.Command("python3", "-I", sheetScript, "--dir", src, "--out", fresh, "--masks", mdir).CombinedOutput()
		if err == nil || !strings.Contains(string(b), "both skin and hair") {
			t.Errorf("a bad mask was not refused: %v %s", err, b)
		}
		if _, err := os.Stat(filepath.Join(fresh, "map/units/aaa/idle.png")); err == nil {
			t.Error("a run with a bad mask wrote the units before it")
		}
	}

	out, msg, err = mapRun(func(d string) {
		writeMask(t, filepath.Join(d, "probe.skin.png"), W, H, none)
		writeMask(t, filepath.Join(d, "probe.hair.png"), W, H, none)
	})
	if err != nil {
		t.Fatalf("empty map masks were refused: %v\n%s", err, msg)
	} else if _, err := os.Stat(filepath.Join(out, "map/units/probe/idle.look.png")); err == nil || index(out)["map/units/probe/idle.png"]["look_mask"] != nil {
		t.Error("masks that mark nothing still made a look mask")
	}

	for _, tc := range []struct {
		name, problem string
		masks         func(d string)
	}{
		{"another size", "but its master is", func(d string) {
			writeMask(t, filepath.Join(d, "probe.skin.png"), W-4, H, skin)
			writeMask(t, filepath.Join(d, "probe.hair.png"), W, H, none)
		}},
		{"outside the art", "leaves transparent", func(d string) {
			writeMask(t, filepath.Join(d, "probe.skin.png"), W, H, func(x, y int) bool { return skin(x, y) || (x < 4 && y < 4) })
			writeMask(t, filepath.Join(d, "probe.hair.png"), W, H, none)
		}},
		{"both skin and hair", "both skin and hair", func(d string) {
			writeMask(t, filepath.Join(d, "probe.skin.png"), W, H, skin)
			writeMask(t, filepath.Join(d, "probe.hair.png"), W, H, skin)
		}},
		{"a missing mask", "is missing", func(d string) {
			writeMask(t, filepath.Join(d, "probe.skin.png"), W, H, skin)
		}},
	} {
		_, msg, err := mapRun(tc.masks)
		if err == nil || !strings.Contains(msg, tc.problem) {
			t.Errorf("%s: err %v, output %q; want a refusal mentioning %q", tc.name, err, msg, tc.problem)
		}
	}

	// Battle: a warrior idle on the 5 px grain (figure x 250-374, y 200-579)
	// with skin on its top 25 rows.
	src, bout := t.TempDir(), t.TempDir()
	fill := func(f, x, y int) color.NRGBA {
		if x >= 250 && x < 375 && y >= 200 && y < 580 {
			return color.NRGBA{150, 90, 60, 255}
		}
		return color.NRGBA{}
	}
	writeStrip(t, src, "A6/battle/units/warrior/idle.png", 640, 40, fill)
	bw := 4*640 + 3*40
	writeMask(t, filepath.Join(src, "A11/battle/units/warrior/idle.skin.png"), bw, 640, func(x, y int) bool {
		cx := x % 680
		return x/680 < 4 && cx < 640 && cx >= 250 && cx < 375 && y >= 200 && y < 225
	})
	writeMask(t, filepath.Join(src, "A11/battle/units/warrior/idle.hair.png"), bw, 640, none)
	if b, err := exec.Command("python3", "-I", battleScript, "--src", src, "--out", bout).CombinedOutput(); err != nil {
		t.Fatalf("battle masks were refused: %v\n%s", err, b)
	}
	bix := index(bout)
	if bix["battle/units/warrior/idle.png"]["look_mask"] != "battle/units/warrior/idle.look.png" || bix["battle/units/warrior/idle.look.png"]["kind"] != "look-mask" {
		t.Errorf("battle idle entry %v, mask entry %v", bix["battle/units/warrior/idle.png"], bix["battle/units/warrior/idle.look.png"])
	}
	f, err := os.Open(filepath.Join(bout, "battle/units/warrior/idle.look.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	// Shrunk by the 5 px grain: skin is rows 40-44 of x 50-74 in each 128 px frame.
	if r, _, _, a := img.At(128+60, 42).RGBA(); a != 0xffff || r != 0xffff {
		t.Errorf("battle mask frame 2 (60,42) = r %d a %d, want skin", r>>8, a>>8)
	}
	if _, _, _, a := img.At(60, 60).RGBA(); a != 0 {
		t.Error("battle mask marks the body below the skin")
	}
}
