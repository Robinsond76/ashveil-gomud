package gmcp

// Phase 70: the Company.Errands message, the web client's Errands tab: who
// is away on an errand and when they are due, who is free to send, and what
// can be sent from the room the leader stands in. The company module builds
// it (company.ErrandsOf); like the other extras it is sent only when it
// changes, and the client runs each countdown from the due time.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func errandsExtra() companyExtra {
	return companyExtra{module: "Company.Errands", buildKeyed: func(user *users.UserRecord, _ []byte) ([]byte, []byte) {
		panel, ok := company.ErrandsOf(user.UserId)
		if !ok {
			return nil, nil
		}
		body, err := json.Marshal(panel)
		if err != nil {
			return nil, nil
		}
		// The key leaves out the clock and each countdown, which tick every
		// second: a panel resends when an errand starts, ends or comes due,
		// or the leader moves, not when time passes.
		keyed := panel
		keyed.Now = 0
		keyed.Rows = make([]company.ErrandRow, len(panel.Rows))
		for i, row := range panel.Rows {
			row.Remaining = 0
			keyed.Rows[i] = row
		}
		key, err := json.Marshal(keyed)
		if err != nil {
			return nil, nil
		}
		return body, key
	}}
}
