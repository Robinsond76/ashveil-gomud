package archetypes

// Capability is a currently configured field/camp utility, not a new skill.
type Capability struct {
	Mode        string `json:"mode,omitempty"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Group       string `json:"group"`
	Description string `json:"description"`
	Skill       string `json:"skill"`
	Rank        int    `json:"rank"`
	Enabled     bool   `json:"enabled"`
	Reason      string `json:"reason,omitempty"`
}

type CapabilityProvider interface{ PlayerCapabilities(userID int) []Capability }

func PlayerCapabilities(userID int) []Capability {
	if cp, ok := current().(CapabilityProvider); ok {
		return cp.PlayerCapabilities(userID)
	}
	return []Capability{}
}
