package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestUnquote(t *testing.T) {
	cases := map[string]string{
		`"day"`:           "day",
		`"5"`:             "5",
		`""`:              "",
		`"a \"q\" b"`:     `a "q" b`,
		`"é"`:             "é",
		`day`:             "day",
		`5`:               "5",
		`100ubze`:         "100ubze",
		`{"a":"b"}`:       `{"a":"b"}`,
		`["x"]`:           `["x"]`,
		`"`:               `"`,
		`"unterminated`:   `"unterminated`,
		`"bad \x escape"`: `"bad \x escape"`,
	}
	for in, want := range cases {
		require.Equal(t, want, helpers.Unquote(in), "Unquote(%s)", in)
	}
}

func TestTxEvents(t *testing.T) {
	res := helpers.TxResult{Events: []helpers.Event{
		{Type: "tx", Attributes: map[string]string{"fee": "10ubze"}},
		{Type: "transfer", Attributes: map[string]string{"amount": "1ubze"}},
		{Type: "tx", Attributes: map[string]string{"acc_seq": "bze1x/1"}},
	}}

	got := helpers.TxEvents(res, "tx")
	require.Len(t, got, 2)
	require.Equal(t, "10ubze", got[0].Attributes["fee"])
	require.Equal(t, "bze1x/1", got[1].Attributes["acc_seq"])

	require.Len(t, helpers.TxEvents(res, "transfer"), 1)
	require.Empty(t, helpers.TxEvents(res, "message"))
	require.Empty(t, helpers.TxEvents(helpers.TxResult{}, "tx"))
}

func TestSimulationErrorMessage(t *testing.T) {
	err := &helpers.SimulationError{Output: "insufficient funds"}
	require.Equal(t, "tx simulation failed: insufficient funds", err.Error())
}
