package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Phase 37 drop tables: what a death adds to each contributing company's
// own spoils. Every company that fought makes its own roll (personal loot,
// owner decision 2026-10-05); the corpse claim rules (33g) then keep it
// its own. It runs once per death, because Suicide does.

// personalDrop is one company's roll for a death.
type personalDrop struct {
	Items []items.Item
	Gold  int
}

// dropProfile is the zone's drop profile, and whether the zone has one. A
// zone opts in with a `loot:` block or an encounter band; a world with
// neither drops exactly as it did before this phase. A band alone sets item
// levels from the band, two over its top for the boss.
func dropProfile(room *rooms.Room) (loot.ZoneProfile, bool) {
	cfg := rooms.GetZoneConfig(room.Zone)
	if cfg == nil {
		return loot.ZoneProfile{}, false
	}
	p := cfg.Loot
	if p.ILvl.Low < 1 && cfg.Encounters.Band.Valid() {
		b := cfg.Encounters.Band
		p.ILvl = loot.Range{Low: b.Low, High: b.High + 2}
	}
	if p.ILvl.Low < 1 && p.Tier < 1 {
		return loot.ZoneProfile{}, false
	}
	return p, true
}

// lastOfGroup reports whether mob is the last of its random encounter's
// foes to have its death processed, so the group's cache rolls exactly once
// however many fall in one round.
func lastOfGroup(mob *mobs.Mob, room *rooms.Room) bool {
	if mob.EncounterID == "" {
		return false
	}
	for _, id := range room.GetMobs() {
		if id == mob.InstanceId {
			continue
		}
		if other := mobs.GetInstance(id); other != nil && other.EncounterID == mob.EncounterID && !other.DeathProcessed {
			return false
		}
	}
	return true
}

// zoneDrops makes each contributing company's roll for mob's death: the
// foe's own (ordinary, elite or boss) and, when it is the last of a random
// encounter's foes, the group's cache.
func zoneDrops(mob *mobs.Mob, room *rooms.Room, contributors []int) map[int]personalDrop {
	profile, ok := dropProfile(room)
	if !ok || len(contributors) == 0 {
		return nil
	}
	kind := loot.Ordinary
	switch {
	case mob.Boss:
		kind = loot.BossRoll
	case mob.IsElite:
		kind = loot.Elite
	}
	cache := lastOfGroup(mob, room)
	label := mob.Character.Name
	src := loot.GameSource()
	out := make(map[int]personalDrop, len(contributors))
	for _, uid := range contributors {
		var d personalDrop
		add := func(got []items.Item, err error) {
			if err != nil {
				mudlog.Warn("zone drop roll", "mob", mob.Character.Name, "error", err)
			}
			d.Items = append(d.Items, got...)
		}
		add(loot.Equipment(kind, mob.Character.Level, profile, label, src))
		var goods loot.Table
		hasGoods := false
		if mob.LootCategory != "" {
			goods, hasGoods = loot.GetTable(mob.LootCategory)
		}
		if kind == loot.BossRoll && hasGoods {
			add(loot.Goods(kind, goods, src), nil)
		}
		if cache {
			add(loot.Equipment(loot.Cache, mob.Character.Level, profile, label+"'s company", src))
			if hasGoods {
				add(loot.Goods(loot.Cache, goods, src), nil)
			}
			d.Gold = loot.CacheGold(mob.Character.Level, mob.EncounterBoss, src)
		}
		if len(d.Items) > 0 || d.Gold > 0 {
			out[uid] = d
			var lines []string
			for _, it := range d.Items {
				lines = append(lines, it.DisplayName())
			}
			if d.Gold > 0 {
				lines = append(lines, fmt.Sprintf("%d gold", d.Gold))
			}
			loot.NoteSpoils(uid, lines...)
		}
	}
	return out
}
