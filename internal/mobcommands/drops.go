package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	relics := mob.Boss && len(loot.RelicsOf(int(mob.MobId))) > 0
	if (!ok && !relics) || len(contributors) == 0 {
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
		if relics {
			add(relicDrop(mob, uid, label, src))
		}
		if !ok {
			// A boss outside any drop profile drops only its relic.
			if len(d.Items) > 0 {
				out[uid] = d
				noteDrop(uid, d)
			}
			continue
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
			trailGold, _ := enemyparty.CompanyEffect(uid, classes.TrailGold) // Phase 38c2: Trailwise
			trailLoot, _ := enemyparty.CompanyEffect(uid, classes.TrailLoot)
			add(loot.EquipmentWith(loot.Cache, mob.Character.Level, profile, label+"'s company", src, trailLoot))
			if hasGoods {
				add(loot.Goods(loot.Cache, goods, src), nil)
			}
			d.Gold = loot.CacheGold(mob.Character.Level, mob.EncounterBoss, src)
			d.Gold += d.Gold * trailGold / 100
		}
		if len(d.Items) > 0 || d.Gold > 0 {
			out[uid] = d
			noteDrop(uid, d)
		}
	}
	return out
}

// noteDrop adds a company's drop to the battle summary's spoils line.
func noteDrop(uid int, d personalDrop) {
	var lines []string
	for _, it := range d.Items {
		lines = append(lines, it.DisplayName())
	}
	if d.Gold > 0 {
		lines = append(lines, fmt.Sprintf("%d gold", d.Gold))
	}
	loot.NoteSpoils(uid, lines...)
}

// relicLuckKey is the leader's saved count of kills of one boss that
// dropped no relic (bad-luck protection, Phase 36d).
func relicLuckKey(mobID int) string { return fmt.Sprintf("relicluck-%d", mobID) }

// relicDrop is one company's relic roll for a boss kill. The count of
// relic-less kills is kept on the leader's character, so it persists with
// the company across restarts.
func relicDrop(mob *mobs.Mob, uid int, label string, src loot.Source) ([]items.Item, error) {
	var char *characters.Character
	if u := users.GetByUserId(uid); u != nil {
		char = u.Character
	}
	missed := 0
	if char != nil {
		if n, ok := char.GetMiscData(relicLuckKey(int(mob.MobId))).(int); ok {
			missed = n
		}
	}
	itm, missed, err := loot.RelicRoll(int(mob.MobId), missed, label, src)
	if char != nil {
		char.SetMiscData(relicLuckKey(int(mob.MobId)), missed)
	}
	if err != nil || itm == nil {
		return nil, err
	}
	return []items.Item{*itm}, nil
}
