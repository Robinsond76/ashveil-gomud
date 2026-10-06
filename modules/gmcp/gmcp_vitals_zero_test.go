package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVitalsAndWorthSendZeroValues: a player at 0 SP or with 0 gold must
// still receive those fields. The web client replaces each namespace
// wholesale, so an omitted field became undefined and the Vitals bar read
// "undefined / 50" while the Worth panel showed a dash instead of 0.
func TestVitalsAndWorthSendZeroValues(t *testing.T) {
	vitals, err := json.Marshal(GMCPCharModule_Payload_Vitals{HpMax: 50, SpMax: 20})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"hp":0,"hp_max":50,"sp":0,"sp_max":20}`, string(vitals))

	worth, err := json.Marshal(GMCPCharModule_Payload_Worth{TNL: 100})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"gold_carry":0,"gold_bank":0,"tnl":100,"xp":0}`, string(worth))
}
