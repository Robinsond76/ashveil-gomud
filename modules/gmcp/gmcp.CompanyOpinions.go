package gmcp

// Phase 64: the Company.Opinions message, the web client's Opinions tab:
// what each companion likes, dislikes and has lately said about the
// leader's choices, in words, with the company's lifetime mercy counts.
// The company module builds it (company.OpinionsOf); like the other extras
// it is sent only when it changes, ignoring the notes' ages.

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func opinionsExtra() companyExtra {
	return companyExtra{module: "Company.Opinions", buildKeyed: func(user *users.UserRecord, _ []byte) ([]byte, []byte) {
		panel, ok := company.OpinionsOf(user.UserId)
		if !ok {
			return nil, nil
		}
		body, err := json.Marshal(panel)
		if err != nil {
			return nil, nil
		}
		// The key leaves out each note's "ago", which ticks every minute:
		// a panel resends when what a companion thinks changes, not when
		// the clock does (Phase 64 review). The ages refresh with it.
		keyed := panel
		keyed.Members = make([]company.OpinionRow, len(panel.Members))
		for i, row := range panel.Members {
			row.Notes = append([]company.OpinionNoteView(nil), row.Notes...)
			for j := range row.Notes {
				row.Notes[j].Ago = ""
			}
			keyed.Members[i] = row
		}
		key, err := json.Marshal(keyed)
		if err != nil {
			return nil, nil
		}
		return body, key
	}}
}
