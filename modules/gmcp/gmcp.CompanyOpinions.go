package gmcp

// Phase 64: the Company.Opinions message, the web client's Opinions tab:
// what each companion likes, dislikes and has lately said about the
// leader's choices, in words, with the company's lifetime mercy counts.
// The company module builds it (company.OpinionsOf); like the other extras
// it is sent only when it changes.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func opinionsExtra() companyExtra {
	return companyExtra{module: "Company.Opinions", build: func(user *users.UserRecord) []byte {
		panel, ok := company.OpinionsOf(user.UserId)
		if !ok {
			return nil
		}
		body, err := json.Marshal(panel)
		if err != nil {
			return nil
		}
		return body
	}}
}
