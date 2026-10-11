package scripts

import (
	"encoding/json"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// e3cPoses are the A12 pilot's pose sheets: every base class attacks and is
// hurt; the ranged classes shoot; the casters cast.
var e3cPoses = map[string]int{"attack": 6, "hurt": 3, "shoot": 6, "cast": 4}

func e3cWants(class string) []string {
	poses := []string{"attack", "hurt"}
	switch class {
	case "ranger", "arbalist", "alchemist":
		poses = append(poses, "shoot")
	case "cleric", "wizard", "witch", "shaman", "dollmaster":
		poses = append(poses, "cast")
	}
	return poses
}

// a12bCreatures are the A12b creatures (docs/art/A12b-creature-actions.md):
// each attacks and is hurt; the hexer and the bonesetter cast; the poacher
// shoots.
var a12bCreatures = []string{"spider-hatchling", "fungus", "skeleton", "crocodile", "snow-floof", "imp-forest",
	"ruffian", "rat", "rat-big", "wolf-snow", "wolf-timber", "goblin-hexer", "bats-echo", "spider-warrior",
	"acolyte-dark", "ruffian-dangerous", "poacher", "dog-junkyard", "bonesetter", "ice-warrior", "stalker-cave",
	"spider-large", "unknown-humanoid", "unknown-beast", "unknown-large"}

func a12bWants(creature string) []string {
	poses := []string{"attack", "hurt"}
	switch creature {
	case "goblin-hexer", "bonesetter":
		poses = append(poses, "cast")
	case "poacher":
		poses = append(poses, "shoot")
	}
	return poses
}

// TestCommissionedBattlePosesAreImported (E3c, A12b): the 38 approved A12
// pose sheets ship for the 15 base classes and the 53 A12b sheets for 25
// creatures, each cut like its idle (the same frame
// size and density, so the screen stands them on the idle's feet row),
// with the pose's frame count, a looping cast, and within 60 KB a frame.
func TestCommissionedBattlePosesAreImported(t *testing.T) {
	dir := spriteDir(t)
	m := loadSpriteManifest(t, dir)
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var loose struct {
		Files map[string]map[string]any `json:"files"`
	}
	if err := json.Unmarshal(raw, &loose); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, c := range baseClasses {
		for _, p := range e3cWants(c) {
			want["battle/units/"+c+"/"+p+".png"] = true
		}
	}
	if len(want) != 38 {
		t.Fatalf("the pilot lists %d sheets, want 38", len(want))
	}
	for _, c := range a12bCreatures {
		for _, p := range a12bWants(c) {
			want["battle/units/"+c+"/"+p+".png"] = true
		}
	}
	if len(want) != 38+53 {
		t.Fatalf("A12 and A12b list %d sheets, want 91", len(want))
	}
	for rel, meta := range m.Files {
		if meta.Kind != "battle-pose" {
			continue
		}
		if !want[rel] {
			t.Errorf("%s: a pose sheet A12 and A12b did not order", rel)
		}
	}
	for rel := range want {
		meta, ok := m.Files[rel]
		if !ok {
			t.Errorf("manifest is missing %s", rel)
			continue
		}
		pose := strings.TrimSuffix(filepath.Base(rel), ".png")
		idle := m.Files[filepath.Dir(rel)+"/idle.png"]
		if meta.Kind != "battle-pose" || meta.Source != "imported" || meta.Frames != e3cPoses[pose] {
			t.Errorf("%s: kind %q source %q frames %d, want an imported battle-pose of %d", rel, meta.Kind, meta.Source, meta.Frames, e3cPoses[pose])
		}
		if len(meta.Frame) != 2 || len(idle.Frame) != 2 || meta.Frame[0] != idle.Frame[0] || meta.Frame[1] != idle.Frame[1] || meta.densityF() != idle.densityF() {
			t.Errorf("%s: frame %v density %v, want the idle's %v at %v", rel, meta.Frame, meta.densityF(), idle.Frame, idle.densityF())
		}
		if len(meta.Size) != 2 || meta.Size[0] != meta.Frames*meta.Frame[0] {
			t.Errorf("%s: size %v is not %d frames", rel, meta.Size, meta.Frames)
		}
		if loop, _ := loose.Files[rel]["loop"].(bool); loop != (pose == "cast") {
			t.Errorf("%s: loop %v, want only a cast to loop", rel, loop)
		}
		if info, err := os.Stat(filepath.Join(dir, rel)); err != nil || info.Size() > int64(meta.Frames)*60*1024 {
			t.Errorf("%s: over the 60 KB a frame budget (%v)", rel, err)
		}
	}
}

// TestImportBattlePoses runs import_battle.py (E3c) on a synthetic pose
// sheet: an A12 attack beside its A6 idle imports as 6 frames on the idle's
// cell, a cast is marked to loop, and an idle already in imported.json is
// enough; a pose with no idle, with the wrong frame count or on another cell
// size is refused.
func TestImportBattlePoses(t *testing.T) {
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
	run := func(src, out string) (string, error) {
		b, err := exec.Command("python3", "-I", script, "--src", src, "--out", out).CombinedOutput()
		return string(b), err
	}
	// A figure on the 5 px grain, its feet on art row 115; frame f leans f art pixels.
	fig := func(f, x, y int) color.NRGBA {
		ax, ay := x/5, y/5
		if ax < 50+f || ax > 74+f || ay < 40 || ay > 115 {
			return color.NRGBA{}
		}
		return color.NRGBA{uint8(100 + ax), 60, uint8(30 + ay), 255}
	}
	index := func(out string) map[string]map[string]any {
		t.Helper()
		var ix map[string]map[string]any
		raw, _ := os.ReadFile(filepath.Join(out, "imported.json"))
		if err := json.Unmarshal(raw, &ix); err != nil {
			t.Fatal(err)
		}
		return ix
	}

	src, out := t.TempDir(), t.TempDir()
	writeStrip(t, src, "A6/battle/units/warrior/idle.png", 640, 40, fig)
	writeStripN(t, src, "A12/battle/units/warrior/attack.png", 6, 640, 40, fig)
	writeStripN(t, src, "A12/battle/units/wizard/cast.png", 4, 640, 40, fig)
	writeStrip(t, src, "A6/battle/units/wizard/idle.png", 640, 40, fig)
	if msg, err := run(src, out); err != nil {
		t.Fatalf("good pose sheets were refused: %v\n%s", err, msg)
	}
	ix := index(out)
	a := ix["battle/units/warrior/attack.png"]
	if a["kind"] != "battle-pose" || a["pose"] != "attack" || a["frames"] != float64(6) || a["density"] != float64(2) || a["loop"] != nil {
		t.Errorf("attack entry = %v", a)
	}
	if sz, _ := a["size"].([]any); len(sz) != 2 || sz[0] != float64(768) || sz[1] != float64(128) {
		t.Errorf("attack sheet size = %v, want 6 frames of 128", a["size"])
	}
	if c := ix["battle/units/wizard/cast.png"]; c["loop"] != true || c["frames"] != float64(4) {
		t.Errorf("cast entry = %v, want 4 looping frames", c)
	}

	// A later run that brings only the pose stands it on the idle already imported.
	later := t.TempDir()
	writeStripN(t, later, "A12/battle/units/warrior/hurt.png", 3, 640, 40, fig)
	if msg, err := run(later, out); err != nil {
		t.Errorf("a pose whose idle is already imported was refused: %v\n%s", err, msg)
	} else if h := index(out)["battle/units/warrior/hurt.png"]; h["frames"] != float64(3) {
		t.Errorf("hurt entry = %v", h)
	}

	// A pose whose idle in the index was cut at another size is refused,
	// so it can't stand on the wrong feet row.
	stale := t.TempDir()
	if err := os.WriteFile(filepath.Join(stale, "imported.json"), []byte(`{"battle/units/rogue/idle.png": {"frame": [96, 96], "density": 1.5}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	writeStripN(t, other, "A12/battle/units/rogue/hurt.png", 3, 640, 40, fig)
	if msg, err := run(other, stale); err == nil || !strings.Contains(msg, "but the idle's are [96, 96]") {
		t.Errorf("a pose on another frame size than its idle: err %v, output %q; want a refusal", err, msg)
	}

	for _, tc := range []struct {
		name, problem string
		write         func(root string)
	}{
		{"no idle", "no imported idle", func(root string) {
			writeStripN(t, root, "A12/battle/units/rogue/attack.png", 6, 640, 40, fig)
		}},
		{"wrong frame count", "not 6 cells", func(root string) {
			writeStrip(t, root, "A6/battle/units/rogue/idle.png", 640, 40, fig)
			writeStripN(t, root, "A12/battle/units/rogue/attack.png", 4, 640, 40, fig)
		}},
		{"other cell size", "not 600", func(root string) {
			writeStrip(t, root, "A6/battle/units/rogue/idle.png", 640, 40, fig)
			writeStripN(t, root, "A12/battle/units/rogue/hurt.png", 3, 600, 40, fig)
		}},
	} {
		root := t.TempDir()
		tc.write(root)
		msg, err := run(root, t.TempDir())
		if err == nil || !strings.Contains(msg, tc.problem) {
			t.Errorf("%s: err %v, output %q; want a refusal mentioning %q", tc.name, err, msg, tc.problem)
		}
	}
}
