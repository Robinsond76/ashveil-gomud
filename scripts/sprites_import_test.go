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

// writeMaster writes a synthetic 2272x832 map-unit master: a 64x224 block in
// every cell with its lowest row on feet (239 for a good sheet).
func writeMaster(t *testing.T, path string, feet int, gutterPixel bool) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8*256+7*32, 3*256+2*32))
	for r := 0; r < 3; r++ {
		for c := 0; c < 8; c++ {
			x0, y0 := c*288, r*288
			for y := feet - 223; y <= feet; y++ {
				for x := 96; x < 160; x++ {
					img.SetNRGBA(x0+x, y0+y, color.NRGBA{200, 120, 80, 255})
				}
			}
		}
	}
	if gutterPixel {
		img.SetNRGBA(260, 100, color.NRGBA{255, 0, 0, 255})
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

// TestImportSheetGrid runs import_sheet.py on synthetic masters: a good sheet
// gives 128 px density-4 frames with binary alpha; a wrong feet row or a
// pixel in a gutter is refused.
func TestImportSheetGrid(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import PIL, numpy").Run(); err != nil {
		t.Skip("Pillow or numpy not available")
	}
	script, err := filepath.Abs(filepath.Join("sprites", "import_sheet.py"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(master string) (string, string, error) {
		out := t.TempDir()
		b, err := exec.Command("python3", "-I", script, master, "probe", "--out", out).CombinedOutput()
		return out, string(b), err
	}

	src := t.TempDir()
	good := filepath.Join(src, "good.png")
	writeMaster(t, good, 239, false)
	out, msg, err := run(good)
	if err != nil {
		t.Fatalf("good master refused: %v\n%s", err, msg)
	}
	for name, size := range map[string][2]int{"idle": {256, 384}, "walk": {768, 384}} {
		f, err := os.Open(filepath.Join(out, "map", "units", "probe", name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != size[0] || b.Dy() != size[1] {
			t.Errorf("%s is %dx%d, want %dx%d", name, b.Dx(), b.Dy(), size[0], size[1])
		}
		lowest := -1
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a != 0 && a != 0xffff {
					t.Fatalf("%s has partial alpha at %d,%d", name, x, y)
				}
				if a != 0 {
					lowest = y
				}
			}
		}
		if lowest != 119 {
			t.Errorf("%s: lowest opaque row %d, want 119", name, lowest)
		}
	}
	var index map[string]spriteMeta
	b, err := os.ReadFile(filepath.Join(out, "imported.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &index); err != nil {
		t.Fatal(err)
	}
	walk := index["map/units/probe/walk.png"]
	if walk.Density != 4 || walk.FeetBaseline != 120 || walk.Frames != 6 || walk.FrameMs != 120 {
		t.Errorf("walk entry = %+v", walk)
	}

	for _, tc := range []struct {
		name    string
		feet    int
		gutter  bool
		problem string
	}{
		{"feet", 235, false, "not 239"},
		{"gutter", 239, true, "gutters"},
	} {
		bad := filepath.Join(src, tc.name+".png")
		writeMaster(t, bad, tc.feet, tc.gutter)
		if _, msg, err := run(bad); err == nil || !strings.Contains(msg, tc.problem) {
			t.Errorf("%s: want refusal naming %q, got err %v: %s", tc.name, tc.problem, err, msg)
		}
	}
}
