package helpers_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestInt64UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int64
	}{
		{"string", `"42"`, 42},
		{"number", `42`, 42},
		{"negative string", `"-7"`, -7},
		{"empty string", `""`, 0},
		{"null", `null`, 0},
		{"max int64", `"9223372036854775807"`, 9223372036854775807},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v helpers.Int64
			require.NoError(t, json.Unmarshal([]byte(tc.in), &v))
			require.Equal(t, tc.want, int64(v))
		})
	}

	for _, bad := range []string{`"abc"`, `"1.5"`, `1.5`, `true`, `"99999999999999999999"`} {
		t.Run("invalid "+bad, func(t *testing.T) {
			var v helpers.Int64
			require.Error(t, json.Unmarshal([]byte(bad), &v))
		})
	}

	t.Run("in a struct", func(t *testing.T) {
		var s struct {
			Height helpers.Int64 `json:"height"`
			Code   helpers.Int64 `json:"code"`
		}
		require.NoError(t, json.Unmarshal([]byte(`{"height":"120","code":5}`), &s))
		require.Equal(t, int64(120), int64(s.Height))
		require.Equal(t, int64(5), int64(s.Code))
	})
}
