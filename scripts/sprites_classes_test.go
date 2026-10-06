package scripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
)

// Every built promoted class should have map and battle art (Phase 40s5).
// A class without art still plays: the battle screen draws its base class
// (or a silhouette for a lineage with no art yet), so missing art is logged
// rather than failed and class phases (38c, 39) are never blocked on art. The
// art pass runs this with ASHVEIL_ART_STRICT=1 to make the pending list fail.
func TestEveryBuiltClassHasArt(t *testing.T) {
	m := loadSpriteManifest(t, spriteDir(t))
	strict := os.Getenv("ASHVEIL_ART_STRICT") == "1"
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
		msg := "class %s has no art yet (battle screen shows %s): manifest is missing %s"
		if strict {
			t.Errorf(msg, c.ID, fallback, strings.Join(missing, ", "))
		} else {
			t.Logf(msg, c.ID, fallback, strings.Join(missing, ", "))
		}
	}
	if len(pending) > 0 && !strict {
		t.Logf("art pending for %d built classes: %s", len(pending), strings.Join(pending, " "))
	}
	// The art list the layout tests check must name real classes.
	for _, id := range promotedClasses {
		if !known[id] {
			t.Errorf("promotedClasses lists %q, which is not a class", id)
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
