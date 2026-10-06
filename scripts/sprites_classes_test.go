package scripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
)

// Every built promoted class has map and battle art. A class that ships
// without art still plays (the battle screen draws its base class), but the
// final art pass closed the gap and this check now fails the build, so a new
// class phase must add its drawer to scripts/sprites/promoted.py and run
// `make sprites`. Classes marked Planned (routes whose mechanics are not
// delivered yet) are exempt; their art may be drawn ahead of the class.
func TestEveryBuiltClassHasArt(t *testing.T) {
	m := loadSpriteManifest(t, spriteDir(t))
	known := map[string]bool{}
	var pending []string
	for _, c := range classes.All() {
		known[c.ID] = true
		if c.Planned {
			continue
		}
		var missing []string
		for _, rel := range []string{"battle/units/" + c.ID + "/idle.png", "map/units/" + c.ID + "/idle.png", "map/units/" + c.ID + "/walk.png"} {
			if _, ok := m.Files[rel]; !ok {
				missing = append(missing, rel)
			}
		}
		if len(missing) == 0 {
			continue
		}
		pending = append(pending, c.ID)
		fallback := "a silhouette"
		if _, ok := m.Files["battle/units/"+c.Lineage+"/idle.png"]; ok {
			fallback = c.Lineage + " art"
		}
		t.Errorf("class %s has no art yet (battle screen shows %s): manifest is missing %s",
			c.ID, fallback, strings.Join(missing, ", "))
	}
	if len(pending) > 0 {
		t.Errorf("art pending for %d built classes: %s", len(pending), strings.Join(pending, " "))
	}
	// The art list the layout tests check must name real classes.
	for _, id := range promotedClasses {
		if !known[id] {
			t.Errorf("promotedClasses lists %q, which is not a class", id)
		}
	}
}

// Every lineage a character can belong to, the neutral ones included, has its
// own base figure: a member who has not promoted is drawn as its lineage, and
// an unpromoted Halberdier, Samurai or Shaman must not be a silhouette
// (Phase 40h). This one is strict: a new lineage ships its base art.
func TestEveryLineageHasBaseArt(t *testing.T) {
	m := loadSpriteManifest(t, spriteDir(t))
	lineages := classes.Lineages()
	if len(lineages) < 9 {
		t.Fatalf("expected the six base and three neutral lineages, got %v", lineages)
	}
	for _, l := range lineages {
		for _, rel := range []string{"battle/units/" + l + "/idle.png", "map/units/" + l + "/idle.png", "map/units/" + l + "/walk.png"} {
			if _, ok := m.Files[rel]; !ok {
				t.Errorf("lineage %s has no base art: manifest is missing %s", l, rel)
			}
		}
	}
}

// A mob's `sprite:` key must name a battle unit the manifest lists, or the
// battle screen silently falls back to a silhouette.
func TestMobSpriteKeysHaveArt(t *testing.T) {
	m := loadSpriteManifest(t, spriteDir(t))
	root := filepath.Clean(filepath.Join(spriteDir(t), "..", "..", "..", "..", "world", "default", "mobs"))
	re := regexp.MustCompile(`(?m)^sprite:\s*(\S+)\s*$`)
	n := 0
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if sm := re.FindSubmatch(raw); sm != nil {
			n++
			if _, ok := m.Files["battle/units/"+string(sm[1])+"/idle.png"]; !ok {
				t.Errorf("%s: sprite %q has no battle art", p, sm[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 50 {
		t.Errorf("found only %d mobs with a sprite key; is the mobs path right?", n)
	}
}

// The creature recruits and the Beast Tamer's bonded beasts have their own art:
// a hound and a stone golem on the map (a companion's key is its class id), and
// a war bear and a drake hatchling in battle, which the battle window and the
// summon mobs name.
func TestCreatureArtExists(t *testing.T) {
	m := loadSpriteManifest(t, spriteDir(t))
	for _, id := range []string{"hound", "stone-golem"} {
		for _, rel := range []string{"map/units/" + id + "/idle.png", "map/units/" + id + "/walk.png", "battle/units/" + id + "/idle.png"} {
			if _, ok := m.Files[rel]; !ok {
				t.Errorf("creature %s: manifest is missing %s", id, rel)
			}
		}
	}
	for _, id := range []string{"war-bear", "drake-hatchling"} {
		if _, ok := m.Files["battle/units/"+id+"/idle.png"]; !ok {
			t.Errorf("bonded beast %s has no battle art", id)
		}
	}
	// The battle window maps each beast kind to a unit; none may be a silhouette.
	js, err := os.ReadFile(filepath.Join(spriteDir(t), "..", "js", "windows", "window-battle.js"))
	if err != nil {
		t.Fatal(err)
	}
	line := regexp.MustCompile(`(?m)BEAST_SPRITES = \{([^}]*)\}`).FindSubmatch(js)
	if line == nil {
		t.Fatal("window-battle.js has no BEAST_SPRITES table")
	}
	kinds := regexp.MustCompile(`(\w+):\s*'([\w-]+)'`).FindAllSubmatch(line[1], -1)
	if len(kinds) != 4 {
		t.Fatalf("BEAST_SPRITES lists %d kinds, want wolf, warhound, bear and drake", len(kinds))
	}
	for _, k := range kinds {
		if strings.HasPrefix(string(k[2]), "unknown-") {
			t.Errorf("beast kind %s is drawn as the %s silhouette", k[1], k[2])
		}
		if _, ok := m.Files["battle/units/"+string(k[2])+"/idle.png"]; !ok {
			t.Errorf("beast kind %s names unit %s, which has no battle art", k[1], k[2])
		}
	}
}
