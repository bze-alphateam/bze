package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

// The event and the response carry the paid amounts as uint64: an amount past
// uint64 is refused in ValidateBasic.
func TestMsgPayPublisherRespect_AmountUint64Bound(t *testing.T) {
	msg := func(amount string) *types.MsgPayPublisherRespect {
		return types.NewMsgPayPublisherRespect(sample.AccAddress(), sample.AccAddress(), amount)
	}

	require.NoError(t, msg("18446744073709551615ubze").ValidateBasic(), "max uint64 is accepted")

	err := msg("18446744073709551616ubze").ValidateBasic()
	require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
	require.ErrorContains(t, err, "amount too large")
}
