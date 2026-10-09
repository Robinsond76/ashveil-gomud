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

// writeArt writes a w x h master under root, filled by fill(x, y).
func writeArt(t *testing.T, root, rel string, w, h int, fill func(x, y int) color.NRGBA) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, fill(x, y))
		}
	}
	path := filepath.Join(root, rel)
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

func flat(c color.NRGBA) func(x, y int) color.NRGBA { return func(int, int) color.NRGBA { return c } }

// TestImportArt runs import_art.py (E2) on synthetic masters: three terrain
// variants become a 3-frame density-4 sheet; an animated row cuts into its
// frames with or without 32 px gutters; an app icon is written at its exact
// size; a wrong size, a transparent terrain tile and a framed tile are
// refused.
func TestImportArt(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if err := exec.Command("python3", "-c", "import PIL, numpy").Run(); err != nil {
		t.Skip("Pillow or numpy not available")
	}
	script, err := filepath.Abs(filepath.Join("sprites", "import_art.py"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(src string, only ...string) (string, string, error) {
		out := t.TempDir()
		args := append([]string{"-I", script, "--src", src, "--out", out, "--only"}, only...)
		b, err := exec.Command("python3", args...).CombinedOutput()
		return out, string(b), err
	}
	grass := color.NRGBA{90, 140, 60, 255}

	src := t.TempDir()
	for i := 1; i <= 3; i++ {
		writeArt(t, src, "A2/map/terrain/land-"+string(rune('0'+i))+".png", 512, 512, flat(grass))
	}
	// here-ring: 4 frames of 512 with 32 px gutters; the second frame is red.
	writeArt(t, src, "A4/map/markers/here-ring.png", 4*512+3*32, 512, func(x, y int) color.NRGBA {
		if x >= 544 && x < 1056 {
			return color.NRGBA{255, 0, 0, 255}
		}
		return color.NRGBA{}
	})
	// fire-lit: 4 frames of 256 with no gutters.
	writeArt(t, src, "A4/map/camp/fire-lit.png", 4*256, 256, flat(color.NRGBA{200, 100, 20, 255}))
	writeArt(t, src, "A4/app/favicon-32.png", 256, 256, flat(color.NRGBA{10, 10, 10, 255}))

	out, msg, err := run(src, "map/terrain/land.png", "map/markers/here-ring.png", "map/camp/fire-lit.png", "app/favicon-32.png")
	if err != nil {
		t.Fatalf("import refused: %v\n%s", err, msg)
	}
	sizes := map[string][2]int{
		"map/terrain/land.png":      {384, 128},
		"map/markers/here-ring.png": {512, 128},
		"map/camp/fire-lit.png":     {256, 64},
		"app/favicon-32.png":        {32, 32},
	}
	imgs := map[string]image.Image{}
	for rel, want := range sizes {
		f, err := os.Open(filepath.Join(out, rel))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		imgs[rel] = img
		if b := img.Bounds(); b.Dx() != want[0] || b.Dy() != want[1] {
			t.Errorf("%s is %dx%d, want %dx%d", rel, b.Dx(), b.Dy(), want[0], want[1])
		}
	}
	// The gutters were skipped: frame 2 (x 128-255) is all red, frame 1 empty.
	ring := imgs["map/markers/here-ring.png"]
	if r, _, _, a := ring.At(130, 60).RGBA(); r>>8 != 255 || a>>8 != 255 {
		t.Errorf("here-ring frame 2 should be red, got r %d a %d", r>>8, a>>8)
	}
	if _, _, _, a := ring.At(60, 60).RGBA(); a != 0 {
		t.Error("here-ring frame 1 should be empty")
	}
	if r, g, b, _ := imgs["map/terrain/land.png"].At(200, 64).RGBA(); r>>8 != 90 || g>>8 != 140 || b>>8 != 60 {
		t.Errorf("land variant 2 colour %d,%d,%d, want 90,140,60", r>>8, g>>8, b>>8)
	}
	var index map[string]map[string]any
	raw, err := os.ReadFile(filepath.Join(out, "imported.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	if e := index["map/markers/here-ring.png"]; e["density"] != float64(4) || e["frames"] != float64(4) || e["frame_ms"] != float64(180) {
		t.Errorf("here-ring entry = %v", e)
	}
	if e := index["app/favicon-32.png"]; e["density"] != nil {
		t.Errorf("an app icon has no density: %v", e)
	}

	for _, tc := range []struct {
		name    string
		write   func(root string)
		only    string
		problem string
	}{
		{"wrong size", func(root string) {
			writeArt(t, root, "A3/map/landmarks/inn.png", 500, 500, flat(color.NRGBA{}))
		}, "map/landmarks/inn.png", "expected 1 frame(s) of 512x512"},
		{"transparent terrain", func(root string) {
			for i := 1; i <= 3; i++ {
				writeArt(t, root, "A2/map/terrain/land-"+string(rune('0'+i))+".png", 512, 512, flat(color.NRGBA{90, 140, 60, 200}))
			}
		}, "map/terrain/land.png", "must be opaque"},
		{"framed tile", func(root string) {
			for i := 1; i <= 3; i++ {
				writeArt(t, root, "A2/map/terrain/land-"+string(rune('0'+i))+".png", 512, 512, func(x, y int) color.NRGBA {
					if x < 8 || y < 8 || x >= 504 || y >= 504 {
						return color.NRGBA{10, 10, 10, 255}
					}
					return grass
				})
			}
		}, "map/terrain/land.png", "darker than the tile"},
		{"missing", func(string) {}, "map/landmarks/inn.png", "missing"},
	} {
		root := t.TempDir()
		tc.write(root)
		if _, msg, err := run(root, tc.only); err == nil || !strings.Contains(msg, tc.problem) {
			t.Errorf("%s: want a refusal naming %q, got err %v: %s", tc.name, tc.problem, err, msg)
		}
	}
}
