package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestIbcDenom(t *testing.T) {
	// ATOM on Osmosis, a widely published voucher denom
	require.Equal(t, "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2", helpers.IbcDenom("channel-0", "uatom"))
	// a returning multi-hop path hashes as a whole
	require.NotEqual(t, helpers.IbcDenom("channel-0", "uatom"), helpers.IbcDenom("channel-0", "transfer/channel-1/uatom"))
}

func TestEscrowAddress(t *testing.T) {
	a, err := helpers.EscrowAddress("cosmos", "channel-0")
	require.NoError(t, err)
	b, err := helpers.EscrowAddress("cosmos", "channel-1")
	require.NoError(t, err)
	require.NotEqual(t, a, b, "one escrow per channel")
	require.Regexp(t, `^cosmos1[02-9ac-hj-np-z]{38}$`, a)
}
