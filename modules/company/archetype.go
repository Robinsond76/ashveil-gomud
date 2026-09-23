package company

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var _ domain.ArchetypeProvider = (*CompanyModule)(nil)

// defaultCompanionArchetypes is used when the module has no plugin config
// (tests): the training dummy is a warrior.
var defaultCompanionArchetypes = map[int]string{58: "warrior"}

// parseCompanionArchetypes reads CompanionArchetypes
// ([{MobTemplateId, Archetype}]), skipping malformed entries.
func parseCompanionArchetypes(raw any) map[int]string {
	out := map[int]string{}
	list, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, entry := range list {
		fields := lowerKeys(entry)
		if fields == nil {
			continue
		}
		id, ok := configInt(fields["mobtemplateid"])
		archetype, _ := fields["archetype"].(string)
		archetype = strings.ToLower(strings.TrimSpace(archetype))
		if !ok || id <= 0 || archetype == "" {
			continue
		}
		out[id] = archetype
	}
	return out
}

func lowerKeys(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[strings.ToLower(k)] = v
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			if name, ok := k.(string); ok {
				out[strings.ToLower(name)] = v
			}
		}
		return out
	}
	return nil
}

func (m *CompanyModule) companionArchetypes() map[int]string {
	if m.plug != nil {
		return parseCompanionArchetypes(m.plug.Config.Get("CompanionArchetypes"))
	}
	return defaultCompanionArchetypes
}

// assignConfiguredArchetype records the configured archetype for a newly
// summoned companion. An archetype the archetype module doesn't know is
// skipped (and logged), never guessed.
func (m *CompanyModule) assignConfiguredArchetype(leaderUserID int, companion domain.Companion) {
	archetype, ok := m.companionArchetypes()[companion.MobTemplateID]
	if !ok {
		return
	}
	if !archetypes.Exists(archetype) {
		mudlog.Warn("company: unknown companion archetype", "template", companion.MobTemplateID, "archetype", archetype)
		return
	}
	if err := m.registry.SetCompanionArchetype(leaderUserID, companion.ID, archetype); err != nil {
		mudlog.Warn("company: set companion archetype", "leader", leaderUserID, "companion", companion.ID, "error", err)
	}
}

// CompanionArchetype implements domain.ArchetypeProvider.
func (m *CompanyModule) CompanionArchetype(leaderUserID, companionID int) (string, bool) {
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "", false
	}
	for _, c := range record.Companions {
		if c.ID == companionID {
			return c.Archetype, c.Archetype != ""
		}
	}
	return "", false
}

// archetypeLabel is a companion's archetype for display.
func archetypeLabel(archetype string) string {
	if archetype == "" {
		return "no archetype"
	}
	if name, ok := archetypes.Name(archetype); ok {
		return name
	}
	return archetype
}

// setArchetype gives a companion without one its archetype (once).
func (m *CompanyModule) setArchetype(leaderUserID int, selector, archetype string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	archetype = strings.ToLower(strings.TrimSpace(archetype))
	if !archetypes.Exists(archetype) {
		return fmt.Sprintf(`There is no archetype called "%s".`, archetype)
	}
	before, ok := m.registry.Get(leaderUserID)
	if !ok {
		return "You have no companion like that."
	}
	companion, found := resolveCompanion(before, selector)
	if !found {
		return "You have no companion like that."
	}
	if err := m.registry.SetCompanionArchetype(leaderUserID, companion.ID, archetype); err != nil {
		if errors.Is(err, domain.ErrArchetypeAlreadySet) {
			return fmt.Sprintf("#%d is already a %s. The choice is permanent.", companion.ID, archetypeLabel(companion.Archetype))
		}
		return err.Error()
	}
	if err := m.save(); err != nil {
		m.registry.Put(before)
		return err.Error()
	}
	return fmt.Sprintf("#%d %s is now a %s.", companion.ID, templateName(companion.MobTemplateID, "companion"), archetypeLabel(archetype))
}
