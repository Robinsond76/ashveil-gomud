package gmcp

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func equipmentExtra() companyExtra {
	return companyExtra{module: "Company.Equipment", build: func(u *users.UserRecord) []byte {
		// Legacy worlds retain their Char.Inventory Gear view.
		if !u.Character.CompanyCargo {
			return nil
		}
		data, _ := json.Marshal(company.EquipmentViewOf(u.UserId))
		return data
	}}
}
