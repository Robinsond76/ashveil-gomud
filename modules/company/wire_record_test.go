package company

import (
	"reflect"
	"strings"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/company"
)

// Phase 74 review: Phase 65's bonds were saved but never read back because
// the decoder's wireRecord missed the field. Every saved Record field must
// have a twin on wireRecord, so a new one cannot be dropped on load again.
func TestWireRecordReadsEverySavedRecordField(t *testing.T) {
	wire := map[string]bool{}
	wt := reflect.TypeOf(wireRecord{})
	for i := 0; i < wt.NumField(); i++ {
		wire[strings.Split(wt.Field(i).Tag.Get("yaml"), ",")[0]] = true
	}
	rt := reflect.TypeOf(domain.Record{})
	for i := 0; i < rt.NumField(); i++ {
		name := strings.Split(rt.Field(i).Tag.Get("yaml"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		if !wire[name] {
			t.Errorf("Record.%s (yaml %q) is saved but wireRecord never reads it back", rt.Field(i).Name, name)
		}
	}
}
