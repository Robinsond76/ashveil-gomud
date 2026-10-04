package combat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/statmods"
)

// armorBaseHP is the representative HP pool used when converting defense into
// an eHP-equivalent score. It matches the baseline character used by the
// weapon ranker so the two ranking systems share a common reference point.
const armorBaseHP = 100

// ArmorRank holds the computed metrics for a single armor item spec.
type ArmorRank struct {
	ItemId int
	Name   string
	Slot   string
	Cursed bool

	// Defense is the raw DamageReduction value from the item spec.
	Defense int

	// AdjDefense is Defense as Character.GetDefense() counts it. Since
	// Phase 30g2 no slot has a multiplier (a shield blocks instead), so it
	// equals Defense; it stays for the rankings' API.
	AdjDefense float64

	// StatBonus is the sum of all raw stat mod values on the item (signed).
	StatBonus int

	// BuffCount is the number of WornBuffIds on the item (passive while equipped).
	BuffCount int

	// BuffValue is the aggregate GetValue() of all WornBuffIds.
	BuffValue int

	// Score is the unified eHP-equivalent ranking number.
	//
	// It is computed as:
	//
	//   defenseEHP + statValue + buffValue/10
	//
	// where:
	//   defenseEHP  = (AdjDefense / 200) * armorBaseHP
	//                 Expected HP-equivalent from damage reduction.
	//                 applyDefenseReduction draws uniform [0, DamageReduction),
	//                 so the expected fractional reduction per hit is
	//                 DamageReduction/200. Multiplied by the baseline HP pool
	//                 this gives how many effective extra HP the piece provides.
	//
	//   statValue   = per-stat weighted sum using weights derived from the
	//                 combat engine's own range constants:
	//                   strength   DamagePerStrength + DamageEdgeMax/span  – damage output (below cap)
	//                   speed      (ToHitMax-ToHitEven)/span           – hit chance
	//                   perception (CritMultMax-CritMultMin)/span * base – crit/dodge
	//                   smarts     (CritChanceMax-CritChanceEven)/span – crit chance
	//                 (span is StatEdgeSpan, the 30g6 stat edge)
	//                   vitality / healthmax / manamax / healthrecovery / manarecovery: 1.0
	//                   damage (flat per-hit bonus): 1.0
	//                   attacks (extra attack/round): avgWeaponDPS proxy = 3.0
	//                   everything else: 1.0
	//
	//   buffValue/10 = aggregate buff GetValue() scaled down to the same order
	//                 of magnitude as the other components.
	Score float64
}

// wornBuffCount returns the number of WornBuffIds on a spec that resolve to
// a valid buff. Used by both weapon and armor rankers.
func wornBuffCount(spec items.ItemSpec) int {
	count := 0
	for _, bId := range spec.WornBuffIds {
		if buffs.GetBuffSpec(bId) != nil {
			count++
		}
	}
	return count
}

// wornBuffValue returns the aggregate GetValue() of all WornBuffIds on a spec.
func wornBuffValue(spec items.ItemSpec) int {
	val := 0
	for _, bId := range spec.WornBuffIds {
		if bs := buffs.GetBuffSpec(bId); bs != nil {
			val += bs.GetValue()
		}
	}
	return val
}

// armorSlotSet is a set of every item type that occupies an equipment slot
// and provides passive protection or stat benefits. Built from items.ArmorSlots()
// so adding or removing slots only requires changing that one function.
var armorSlotSet = func() map[items.ItemType]bool {
	m := make(map[items.ItemType]bool)
	for _, s := range items.ArmorSlots() {
		m[s] = true
	}
	return m
}()

// statWeight returns the eHP-equivalent weight for one point of a given stat
// mod name, derived from the combat engine's configured range constants.
func statWeight(statName string) float64 {
	cfg := configs.GetCombatConfig()
	span := float64(cfg.StatEdgeSpan)
	if span <= 0 {
		span = 10
	}
	switch statName {
	case string(statmods.Strength):
		// Approximate marginal damage below the cap; rankings use no attacker stats.
		return float64(cfg.DamagePerStrength) + float64(cfg.DamageEdgeMax)/span
	case string(statmods.Speed):
		// +1 speed moves hitChance by about (ToHitMax-ToHitEven)/span
		return float64(int(cfg.ToHitMax)-int(cfg.ToHitEven)) / span
	case string(statmods.Smarts):
		// +1 smarts moves critChance by about (CritChanceMax-CritChanceEven)/span
		return float64(int(cfg.CritChanceMax)-int(cfg.CritChanceEven)) / span
	case string(statmods.Perception):
		// +1 perception moves critMultiplier by (CritMultMax-CritMultMin)/span
		// and dodgeChance by (DodgeChanceMax-DodgeChanceMin)/span; use the larger
		multRange := float64(cfg.CritMultMax-cfg.CritMultMin) / span
		dodgeRange := float64(int(cfg.DodgeChanceMax)-int(cfg.DodgeChanceMin)) / span
		if multRange > dodgeRange {
			return multRange
		}
		return dodgeRange
	case string(statmods.Damage):
		// flat damage per hit - weight 1:1 with HP
		return 1.0
	case string(statmods.Attacks):
		// extra attack per round; proxy average weapon DPS contribution
		return 3.0
	case string(statmods.Vitality),
		string(statmods.HealthMax),
		string(statmods.ManaMax),
		string(statmods.HealthRecovery),
		string(statmods.ManaRecovery),
		string(statmods.Mysticism):
		return 1.0
	default:
		return 1.0
	}
}

// RankArmor computes ranking metrics for every loaded armor/wearable spec
// and returns slices sorted by three different criteria:
//
//   - byDefense     – raw DamageReduction value
//   - byAdjDefense  – DamageReduction adjusted for slot multiplier
//   - byScore       – unified eHP-equivalent score (defense + stats + buffs)
//
// All three slices contain the same entries; only the order differs.
func RankArmor() (byDefense, byAdjDefense, byScore []ArmorRank) {
	allSpecs := items.GetAllItemSpecs()

	ranks := make([]ArmorRank, 0, len(allSpecs))

	for _, spec := range allSpecs {
		if !armorSlotSet[spec.Type] {
			continue
		}

		defense := spec.DamageReduction

		adjDefense := float64(defense)

		// defenseEHP: expected HP-equivalent from damage reduction.
		// applyDefenseReduction draws uniform [0, DamageReduction), so the
		// expected fractional reduction per hit is AdjDefense/200.
		defenseEHP := (adjDefense / 200.0) * armorBaseHP

		statBonus := 0
		statValue := 0.0
		for k, v := range spec.StatMods {
			statBonus += v
			statValue += float64(v) * statWeight(k)
		}

		buffCount := wornBuffCount(spec)
		buffValue := wornBuffValue(spec)

		score := defenseEHP + statValue + float64(buffValue)/10.0

		ranks = append(ranks, ArmorRank{
			ItemId:     spec.ItemId,
			Name:       spec.Name,
			Slot:       string(spec.Type),
			Cursed:     spec.Cursed,
			Defense:    defense,
			AdjDefense: adjDefense,
			StatBonus:  statBonus,
			BuffCount:  buffCount,
			BuffValue:  buffValue,
			Score:      score,
		})
	}

	byDefense = make([]ArmorRank, len(ranks))
	copy(byDefense, ranks)
	sort.Slice(byDefense, func(i, j int) bool {
		if byDefense[i].Defense != byDefense[j].Defense {
			return byDefense[i].Defense > byDefense[j].Defense
		}
		return byDefense[i].Score > byDefense[j].Score
	})

	byAdjDefense = make([]ArmorRank, len(ranks))
	copy(byAdjDefense, ranks)
	sort.Slice(byAdjDefense, func(i, j int) bool {
		if byAdjDefense[i].AdjDefense != byAdjDefense[j].AdjDefense {
			return byAdjDefense[i].AdjDefense > byAdjDefense[j].AdjDefense
		}
		return byAdjDefense[i].Score > byAdjDefense[j].Score
	})

	byScore = make([]ArmorRank, len(ranks))
	copy(byScore, ranks)
	sort.Slice(byScore, func(i, j int) bool {
		return byScore[i].Score > byScore[j].Score
	})

	return byDefense, byAdjDefense, byScore
}

// FormatArmorRankings returns a human-readable table of all three ranking
// views, suitable for logging or printing to a terminal.
func FormatArmorRankings() string {
	byDefense, byAdjDefense, byScore := RankArmor()

	var sb strings.Builder

	writeTable := func(title string, rows []ArmorRank, scoreLabel string, score func(ArmorRank) string) {
		fmt.Fprintf(&sb, "\n=== %s ===\n", title)
		fmt.Fprintf(&sb, "%-4s %-32s %-10s %-8s %-6s %-6s %s\n",
			"Rank", "Name", "Slot", "Cursed", "Buffs", "Stats", scoreLabel)
		fmt.Fprintln(&sb, strings.Repeat("-", 90))
		for i, r := range rows {
			cursed := ""
			if r.Cursed {
				cursed = "yes"
			}
			fmt.Fprintf(&sb, "%-4d %-32s %-10s %-8s %-6d %-6d %s\n",
				i+1, r.Name, r.Slot, cursed, r.BuffCount, r.StatBonus, score(r))
		}
	}

	writeTable("Ranked by Defense (DamageReduction)", byDefense, "Defense", func(r ArmorRank) string {
		return fmt.Sprintf("%d", r.Defense)
	})
	writeTable("Ranked by Adjusted Defense", byAdjDefense, "AdjDefense", func(r ArmorRank) string {
		return fmt.Sprintf("%.1f", r.AdjDefense)
	})
	writeTable("Ranked by Score (eHP-equivalent)", byScore, "Score", func(r ArmorRank) string {
		return fmt.Sprintf("%.2f", r.Score)
	})

	return sb.String()
}
