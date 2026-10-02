package company

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 33h1 companion growth. A companion's training is its level's stat
// points dealt by its archetype's weights plus its focus; nothing about it
// is saved except the focus, so every spawn and level-up derives the same
// stats from the same level.

var _ domain.GrowthProvider = (*CompanyModule)(nil)

const growthUsage = "Usage: company growth | company growth <member> <strength|speed|smarts|vitality|mysticism|perception|balanced>"

// growthWeightsOf is a companion's archetype weights with its focus.
func growthWeightsOf(c domain.Companion) domain.GrowthWeights {
	var w domain.GrowthWeights
	if c.Archetype != "" {
		if byStat, ok := archetypes.CompanionGrowth(c.Archetype); ok {
			w, _ = domain.GrowthWeightsFrom(byStat)
		}
	}
	return w.WithFocus(c.GrowthFocus)
}

// growthWeights reads the companion's current record.
func (m *CompanyModule) growthWeights(leaderUserID, companionID int) domain.GrowthWeights {
	if record, ok := m.registry.Get(leaderUserID); ok {
		for _, c := range record.Companions {
			if c.ID == companionID {
				return growthWeightsOf(c)
			}
		}
	}
	return domain.EvenGrowth
}

// retrain re-deals a companion's live mob, when it is out.
func (m *CompanyModule) retrain(leaderUserID, companionID int) {
	if instanceID, ok := m.instance(leaderUserID, companionID); ok && m.runtime.IsLive(instanceID) {
		m.runtime.Retrain(instanceID, m.growthWeights(leaderUserID, companionID))
	}
}

// RetrainCompanion implements domain.GrowthProvider for a live level-up.
func (m *CompanyModule) RetrainCompanion(instanceID int) bool {
	leaderUserID, companionID, ok := m.companionForInstance(instanceID)
	if !ok {
		return false
	}
	return m.runtime.Retrain(instanceID, m.growthWeights(leaderUserID, companionID))
}

// growth runs company growth.
func (m *CompanyModule) growth(user *users.UserRecord, args []string) string {
	if len(args) == 0 {
		return m.growthView(user.UserId)
	}
	if len(args) < 2 {
		return growthUsage
	}
	// The stat is the last word, so a member's full name may have spaces.
	member, stat := strings.Join(args[:len(args)-1], " "), args[len(args)-1]
	args = []string{member, stat}
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	focus := ""
	switch strings.ToLower(args[1]) {
	case "balanced", "none", "clear":
	default:
		i, ok := domain.GrowthStatIndex(args[1])
		if !ok {
			return growthUsage
		}
		focus = domain.GrowthStats[i]
	}
	if usercommands.InBattle(user) {
		return "Not in the middle of a battle. Settle your company's growth once the fighting is done."
	}
	before, ok := m.registry.Get(user.UserId)
	if !ok {
		return "You have no companion like that."
	}
	companion, found := resolveCompanion(before, args[0])
	if !found {
		return "You have no companion like that."
	}
	name := nameOf(companion, "#"+strconv.Itoa(companion.ID))
	if companion.GrowthFocus == focus {
		return fmt.Sprintf("%s already grows that way.", name)
	}
	record, _ := m.registry.Get(user.UserId) // Get copies the companions
	for i := range record.Companions {
		if record.Companions[i].ID == companion.ID {
			record.Companions[i].GrowthFocus = focus
		}
	}
	// In memory only: the company file also carries in-memory gear
	// snapshots, so it is written only at the 22b seams. The focus reaches
	// disk with the next save; a crash before then only loses the focus,
	// and training is re-derived from whatever focus is on disk.
	m.registry.Put(record)
	m.retrain(user.UserId, companion.ID)
	if focus == "" {
		return fmt.Sprintf("%s grows by %s training alone now; their points are dealt again at once.", name, archetypeLabel(companion.Archetype))
	}
	return fmt.Sprintf("%s favours %s now; their points are dealt again at once.", name, focus)
}

// growthView lists each companion's leanings, focus, and trained points.
func (m *CompanyModule) growthView(leaderUserID int) string {
	record, ok := m.registry.Get(leaderUserID)
	if !ok || len(record.Companions) == 0 {
		return "You have no companions to train."
	}
	lines := []string{"Company growth (each level's stat points, dealt by these weights):"}
	for _, c := range record.Companions {
		w := growthWeightsOf(c)
		level := 1
		if c.State != nil && c.State.Level > 0 {
			level = c.State.Level
		}
		if id, ok := m.instance(leaderUserID, c.ID); ok {
			if live, _, _, ok := m.runtime.Progress(id); ok {
				level = live
			}
		}
		focus := "balanced"
		if c.GrowthFocus != "" {
			focus = c.GrowthFocus
		}
		lines = append(lines,
			fmt.Sprintf("  #%d %s, %s, level %d, focus %s", c.ID, nameOf(c, strconv.Itoa(c.MobTemplateID)), archetypeLabel(c.Archetype), level, focus),
			"      weights: "+growthWeightsText(w),
			"      trained: "+growthDealtText(domain.Deal(characters.StatPointsAtLevel(level), w)))
	}
	lines = append(lines, "Set a focus with company growth <member> <stat>, or balanced to clear it. See help growth.")
	return strings.Join(lines, "\n")
}

func growthWeightsText(w domain.GrowthWeights) string {
	var parts []string
	for i, v := range w {
		if v > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", domain.GrowthStats[i], v))
		}
	}
	return strings.Join(parts, ", ")
}

func growthDealtText(dealt [6]int) string {
	var parts []string
	for i, v := range dealt {
		if v > 0 {
			parts = append(parts, fmt.Sprintf("%s +%d", domain.GrowthStats[i], v))
		}
	}
	if len(parts) == 0 {
		return "nothing yet"
	}
	return strings.Join(parts, ", ")
}
