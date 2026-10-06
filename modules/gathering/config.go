package gathering

import (
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/gathering"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Config helpers: the plugin hands back loosely typed YAML.

func listOf(raw any) []any {
	list, _ := raw.([]any)
	return list
}

func fieldsOf(raw any) map[string]any {
	out := map[string]any{}
	switch v := raw.(type) {
	case map[string]any:
		for k, item := range v {
			out[strings.ToLower(k)] = item
		}
	case map[any]any:
		for k, item := range v {
			if name, ok := k.(string); ok {
				out[strings.ToLower(name)] = item
			}
		}
	}
	return out
}

func configInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}

func configString(raw any) string {
	value, _ := raw.(string)
	return value
}

func configDuration(raw any) (time.Duration, bool) {
	d, err := time.ParseDuration(strings.TrimSpace(configString(raw)))
	return d, err == nil && d >= 0
}

// parseSettings reads the module's config over the shipped defaults;
// malformed entries are skipped with a warning, never fatal.
func parseSettings(get func(string) any) gathering.Settings {
	s := gathering.DefaultSettings()
	if get == nil {
		return s
	}
	count := func(key string, into *int, lo, hi int) {
		if n, ok := configInt(get(key)); ok && n >= lo && n <= hi {
			*into = n
		}
	}
	count("HuntEncounterBonus", &s.HuntEncounterBonus, 0, 100)
	count("HerbMin", &s.HerbMin, 0, 10)
	count("HerbMax", &s.HerbMax, 0, 10)
	if s.HerbMax < s.HerbMin {
		d := gathering.DefaultSettings()
		s.HerbMin, s.HerbMax = d.HerbMin, d.HerbMax
	}
	count("ForageLevelsPerOne", &s.ForageLevelsPerOne, 1, 10)
	count("KnifeBonus", &s.KnifeBonus, 0, 10)
	count("DarkPct", &s.DarkPct, 0, 100)
	count("BitterWeedPct", &s.BitterWeedPct, 0, 100)
	count("ScribePct", &s.ScribePct, 0, 100)
	count("FirewoodBundles", &s.FirewoodBundles, 1, 20)
	count("ToolMultiplier", &s.ToolMultiplier, 1, 10)
	count("FieldSmithBonus", &s.FieldSmithBonus, 0, 10)
	count("FishAttempts", &s.FishAttempts, 1, 10)
	count("FishPct", &s.FishPct, 0, 100)
	count("RangerFishPct", &s.RangerFishPct, 0, 100)
	count("LineBreakPct", &s.LineBreakPct, 0, 100)
	count("GamePct", &s.GamePct, 0, 100)
	count("GameMeat", &s.GameMeat, 1, 10)
	count("ForageMeat", &s.ForageMeat, 0, 10)
	count("RangerGamePct", &s.RangerGamePct, 0, 100)
	count("GameCapPct", &s.GameCapPct, 0, 100)
	count("SnarePct", &s.SnarePct, 0, 100)
	count("GoodsPct", &s.GoodsPct, 0, 100)
	items := map[string]*int{
		"FirewoodItemId":     &s.Items.Firewood,
		"DampFirewoodItemId": &s.Items.DampFirewood,
		"FishingLineItemId":  &s.Items.FishingLine,
		"RawFishItemId":      &s.Items.RawFish,
		"BitterWeedItemId":   &s.Items.BitterWeed,
		"RawMeatItemId":      &s.Items.RawMeat,
	}
	for key, into := range items {
		if n, ok := configInt(get(key)); ok && n > 0 {
			*into = n
		}
	}
	if list := listOf(get("WetWeather")); len(list) > 0 {
		var wet []string
		for _, v := range list {
			if name := strings.ToLower(strings.TrimSpace(configString(v))); name != "" {
				wet = append(wet, name)
			}
		}
		s.WetWeather = wet
	}
	if list := listOf(get("Resources")); len(list) > 0 {
		for _, entry := range list {
			f := fieldsOf(entry)
			kind := gathering.Kind(strings.ToLower(strings.TrimSpace(configString(f["kind"]))))
			if !kind.Valid() {
				mudlog.Warn("gathering: Resources entry skipped", "entry", entry)
				continue
			}
			rule := s.Rules[kind]
			if d, ok := configDuration(f["duration"]); ok && d > 0 {
				rule.Duration = d
			}
			if d, ok := configDuration(f["regrow"]); ok && d > 0 {
				rule.Regrow = d
			}
			if n, ok := configInt(f["pool"]); ok && n > 0 && n <= 50 {
				rule.PoolMax = n
			}
			if n, ok := configInt(f["effortpct"]); ok && n >= 0 && n <= 400 {
				rule.EffortPc = n
			}
			s.Rules[kind] = rule
		}
	}
	tables := map[string]map[string]gathering.Table{
		"Herbs":     s.Herb,
		"RareHerbs": s.RareHerb,
		"Fish":      s.Fish,
		"Goods":     s.Goods,
	}
	for key, into := range tables {
		for _, entry := range listOf(get(key)) {
			f := fieldsOf(entry)
			zone := strings.TrimSpace(configString(f["zone"]))
			var table gathering.Table
			for _, it := range listOf(f["items"]) {
				fi := fieldsOf(it)
				id, okI := configInt(fi["itemid"])
				w, okW := configInt(fi["weight"])
				if okI && id > 0 && okW && w > 0 {
					table = append(table, gathering.Weighted{ItemID: id, Weight: w})
				}
			}
			if zone == "" || len(table) == 0 {
				mudlog.Warn("gathering: table entry skipped", "table", key, "entry", entry)
				continue
			}
			into[zone] = table
		}
	}
	return s
}
