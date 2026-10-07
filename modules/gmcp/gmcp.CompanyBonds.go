package gmcp

// Phase 65: the Company.Bonds message, the web client's Bonds tab: how each
// pair of companions feels, with what it does in battle, and each
// companion's view of the others, in words. The company module builds it
// (company.BondsOf); like the other extras it is sent only when it changes.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func bondsExtra() companyExtra {
	return companyExtra{module: "Company.Bonds", buildKeyed: func(user *users.UserRecord, _ []byte) ([]byte, []byte) {
		panel, ok := company.BondsOf(user.UserId)
		if !ok {
			return nil, nil
		}
		body, err := json.Marshal(panel)
		if err != nil {
			return nil, nil
		}
		return body, body
	}}
}
