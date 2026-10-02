package company

// EquipmentView is an owned leader's read-only equipment proposal catalogue.
// The game-loop provider uses the same proposal and capacity guard as commands.
type EquipmentStats struct {
	EdgeBonus          int            `json:"edge_bonus"`
	EdgeStrikes        int            `json:"edge_strikes"`
	OffhandEdgeBonus   int            `json:"offhand_edge_bonus"`
	OffhandEdgeStrikes int            `json:"offhand_edge_strikes"`
	Hands              int            `json:"hands"`
	Reach              bool           `json:"reach"`
	Shield             bool           `json:"shield"`
	HealthMax          int            `json:"health_max"`
	ManaMax            int            `json:"mana_max"`
	Damage             string         `json:"damage"`
	OffhandDamage      string         `json:"offhand_damage"`
	Defense            int            `json:"defense"`
	WornG              int            `json:"worn_g"`
	Burden             string         `json:"burden"`
	DodgePct           int            `json:"dodge_pct"`
	Stats              map[string]int `json:"stats"`
	PackCapacityG      int            `json:"pack_capacity_g"`
	CapacityG          int            `json:"capacity_g"`
	CargoG             int            `json:"cargo_g"`
}
type EquipmentChoice struct {
	Returned []string        `json:"returned,omitempty"`
	Ref      string          `json:"ref"`
	Label    string          `json:"label"`
	Allowed  bool            `json:"allowed"`
	Reason   string          `json:"reason,omitempty"`
	Command  string          `json:"command,omitempty"`
	After    *EquipmentStats `json:"after,omitempty"`
}
type EquipmentSlot struct {
	Slot     string            `json:"slot"`
	Label    string            `json:"label"`
	Equipped *EquipmentChoice  `json:"equipped,omitempty"`
	Remove   *EquipmentChoice  `json:"remove,omitempty"`
	Choices  []EquipmentChoice `json:"choices"`
}
type EquipmentView struct {
	Available bool            `json:"available"`
	Reason    string          `json:"reason,omitempty"`
	Current   EquipmentStats  `json:"current"`
	Slots     []EquipmentSlot `json:"slots"`
}
type EquipmentViewProvider interface{ EquipmentView(int) EquipmentView }

func EquipmentViewOf(id int) EquipmentView {
	formationProviderMu.RLock()
	p := formationProvider
	formationProviderMu.RUnlock()
	if ep, ok := p.(EquipmentViewProvider); ok {
		return ep.EquipmentView(id)
	}
	return EquipmentView{Reason: "Equipment service unavailable.", Slots: []EquipmentSlot{}}
}
