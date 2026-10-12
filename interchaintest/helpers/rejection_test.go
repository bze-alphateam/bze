package helpers_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestTxFee(t *testing.T) {
	res := helpers.TxResult{Events: []helpers.Event{
		{Type: "tx", Attributes: map[string]string{"acc_seq": "bze1x/1"}},
		{Type: "tx", Attributes: map[string]string{"fee": "2500ubze,7ufoo"}},
	}}

	fee, err := helpers.TxFee(res, "ubze")
	require.NoError(t, err)
	require.Equal(t, "2500", fee.String())

	fee, err = helpers.TxFee(res, "ubar")
	require.NoError(t, err)
	require.True(t, fee.IsZero())

	_, err = helpers.TxFee(helpers.TxResult{}, "ubze")
	require.Error(t, err)

	_, err = helpers.TxFee(helpers.TxResult{Events: []helpers.Event{
		{Type: "tx", Attributes: map[string]string{"fee": "not-a-coin!"}},
	}}, "ubze")
	require.Error(t, err)
}

func TestEventsWith(t *testing.T) {
	events := []helpers.Event{
		{Type: "e", Attributes: map[string]string{"denom": "a", "prize": "x"}},
		{Type: "e", Attributes: map[string]string{"denom": "b", "prize": "x"}},
		{Type: "e", Attributes: map[string]string{"denom": "a"}},
	}

	require.Len(t, helpers.EventsWith(events, map[string]string{"denom": "a"}), 2)
	got := helpers.EventsWith(events, map[string]string{"denom": "a", "prize": "x"})
	require.Len(t, got, 1)
	require.Equal(t, "x", got[0].Attributes["prize"])
	require.Empty(t, helpers.EventsWith(events, map[string]string{"denom": "c"}))
	require.Len(t, helpers.EventsWith(events, nil), 3)
	require.Empty(t, helpers.EventsWith(nil, map[string]string{"denom": "a"}))
}

func TestRejectionReason(t *testing.T) {
	msg, err := helpers.RejectionReason(helpers.TxResult{}, &helpers.SimulationError{Output: "denom reward has no stakers"})
	require.NoError(t, err)
	require.Equal(t, "denom reward has no stakers", msg)

	msg, err = helpers.RejectionReason(helpers.TxResult{Code: 5, RawLog: "insufficient funds"}, nil)
	require.NoError(t, err)
	require.Equal(t, "insufficient funds", msg)

	infra := errors.New("node unreachable")
	_, err = helpers.RejectionReason(helpers.TxResult{}, infra)
	require.ErrorIs(t, err, infra)

	_, err = helpers.RejectionReason(helpers.TxResult{TxHash: "AB", Height: 7}, nil)
	require.ErrorContains(t, err, "accepted")
}
