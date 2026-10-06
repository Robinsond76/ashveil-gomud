package modconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntAcceptsEveryYAMLNumberShape(t *testing.T) {
	cases := []struct {
		name string
		raw  any
		want int
		ok   bool
	}{
		{"int", 7, 7, true},
		{"int64", int64(8), 8, true},
		{"uint64", uint64(9), 9, true},
		{"integral float", 10.0, 10, true},
		{"negative float", -3.0, -3, true},
		{"padded string", " 11 ", 11, true},
		{"fractional float is rejected", 2.5, 0, false},
		{"huge float is rejected", 1e12, 0, false},
		{"huge uint is rejected", uint64(1 << 40), 0, false},
		{"blank string", "  ", 0, false},
		{"word", "ten", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Int(tc.raw)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
	assert.Equal(t, 5, IntOr(nil, 5))
	assert.Equal(t, 5, IntOr(2.5, 5))
	assert.Equal(t, 3, IntOr("3", 5))
}

func TestFloatAcceptsNumbersAndNumericStrings(t *testing.T) {
	for raw, want := range map[any]float64{1.5: 1.5, 2: 2, int64(3): 3, uint64(4): 4, " 0.25 ": 0.25} {
		got, ok := Float(raw)
		assert.True(t, ok, "%v", raw)
		assert.Equal(t, want, got)
	}
	for _, raw := range []any{nil, "", "x", true, []any{1}} {
		_, ok := Float(raw)
		assert.False(t, ok, "%v", raw)
	}
	assert.Equal(t, 0.5, FloatOr("junk", 0.5))
	assert.Equal(t, 0.75, FloatOr("0.75", 0.5))
}

func TestStringAndStrings(t *testing.T) {
	assert.Equal(t, "bazaar", String("bazaar"))
	assert.Equal(t, " keep ", String(" keep "), "String does not trim")
	assert.Equal(t, "", String(12))
	assert.Equal(t, "", String(nil))

	assert.Equal(t, []string{"a", "b"}, Strings([]string{" a ", "", "b"}))
	assert.Equal(t, []string{"a", "b"}, Strings([]any{"a", 3, " b ", "  "}))
	assert.Nil(t, Strings("a"))
	assert.Nil(t, Strings(nil))
}

func TestMapLowercasesKeysFromEitherDecoderShape(t *testing.T) {
	assert.Equal(t, map[string]any{"itemid": 1, "price": 2}, Map(map[string]any{"ItemId": 1, "PRICE": 2}))
	assert.Equal(t, map[string]any{"zone": "x"}, Map(map[any]any{"Zone": "x", 7: "dropped"}))
	assert.Nil(t, Map([]any{"not a map"}))
	assert.Nil(t, Map(nil))
}

func TestDurationAcceptsGoStringsAndWholeSeconds(t *testing.T) {
	for raw, want := range map[any]time.Duration{"90s": 90 * time.Second, " 1h30m ": 90 * time.Minute, 45: 45 * time.Second, int64(2): 2 * time.Second, 3.0: 3 * time.Second, "7": 7 * time.Second, "0": 0} {
		got, ok := Duration(raw)
		assert.True(t, ok, "%v", raw)
		assert.Equal(t, want, got, "%v", raw)
	}
	for _, raw := range []any{nil, "", "soon", "-5s", -1, 2.5, true} {
		_, ok := Duration(raw)
		assert.False(t, ok, "%v", raw)
	}
}

func TestBoolAcceptsBoolsAndParseBoolStrings(t *testing.T) {
	for _, raw := range []any{true, "true", " 1 ", "T"} {
		assert.True(t, Bool(raw), "%v", raw)
	}
	for _, raw := range []any{false, "false", "0", "yes", 1, nil} {
		assert.False(t, Bool(raw), "%v", raw)
	}
}
