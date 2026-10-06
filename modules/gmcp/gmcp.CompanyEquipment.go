package gmcp

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// equipmentExtra builds the Gear editor only for a client showing it
// (watching): its previews are the costliest extra, and most players are
// not looking at them (Phase 34 review).
func equipmentExtra(watching func(userID int) (string, bool), member ...func(userID int) string) companyExtra {
	return companyExtra{module: "Company.Equipment", build: func(u *users.UserRecord) []byte {
		// Legacy worlds retain their Char.Inventory Gear view.
		slot, open := watching(u.UserId)
		if !u.Character.CompanyCargo || !open {
			return nil
		}
		who := "me"
		if len(member) > 0 {
			who = member[0](u.UserId)
		}
		data, _ := json.Marshal(company.EquipmentViewMember(u.UserId, who, slot))
		return data
	}}
}
