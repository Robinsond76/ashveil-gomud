package scripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
)

// Every promoted class whose mechanics are built has map and battle art
// (Phase 40s5); planned elite classes wait for the later art pass with their
// phases (38c, 39).
func TestEveryBuiltClassHasArt(t *testing.T) {
	m := loadSpriteManifest(t, spriteDir(t))
	built := 0
	for _, c := range classes.All() {
		if c.Planned {
			continue
		}
		built++
		for _, rel := range []string{"battle/units/" + c.ID + "/idle.png", "map/units/" + c.ID + "/idle.png", "map/units/" + c.ID + "/walk.png"} {
			if _, ok := m.Files[rel]; !ok {
				t.Errorf("class %s has no art: manifest is missing %s", c.ID, rel)
			}
		}
	}
	if built != len(promotedClasses) {
		t.Errorf("%d built classes but %d have art listed: update promotedClasses or the generator", built, len(promotedClasses))
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
