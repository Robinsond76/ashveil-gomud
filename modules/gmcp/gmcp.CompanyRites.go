package gmcp

// Phase 74: the Company.Rites message, the web client's Rites tab: who the
// company has lost and not yet mourned, and whether the leader stands at a
// camp or an inn where the rites can be held. The company module builds it
// (company.RitesOf); like the other extras it is sent only when it changes.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func ritesExtra() companyExtra {
	return companyExtra{module: "Company.Rites", buildKeyed: func(user *users.UserRecord, _ []byte) ([]byte, []byte) {
		panel, ok := company.RitesOf(user.UserId)
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
