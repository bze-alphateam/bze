package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgAddLiquidity_ValidateBasic(t *testing.T) {
	validAddr := sample.AccAddress()

	tests := []struct {
		name   string
		msg    MsgAddLiquidity
		err    error
		errMsg string
	}{
		{
			name: "valid message",
			msg: MsgAddLiquidity{
				Creator:     validAddr,
				PoolId:      "pool1",
				BaseAmount:  sdk.NewInt(100),
				QuoteAmount: sdk.NewInt(200),
				MinLpTokens: sdk.NewInt(10),
			},
			err: nil,
		},
		{
			name: "invalid address",
			msg: MsgAddLiquidity{
				Creator:     "invalid_address",
				PoolId:      "pool1",
				BaseAmount:  sdk.NewInt(100),
				QuoteAmount: sdk.NewInt(200),
				MinLpTokens: sdk.NewInt(10),
			},
			err:    sdkerrors.ErrInvalidAddress,
			errMsg: "invalid creator address",
		},
		{
			name: "empty pool id",
			msg: MsgAddLiquidity{
				Creator:     validAddr,
				PoolId:      "",
				BaseAmount:  sdk.NewInt(100),
				QuoteAmount: sdk.NewInt(200),
				MinLpTokens: sdk.NewInt(10),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "pool id cannot be empty",
		},
		{
			name: "zero min lp tokens",
			msg: MsgAddLiquidity{
				Creator:     validAddr,
				PoolId:      "pool1",
				BaseAmount:  sdk.NewInt(100),
				QuoteAmount: sdk.NewInt(200),
				MinLpTokens: sdk.ZeroInt(),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "min lp tokens must be positive",
		},
		{
			name: "zero base amount",
			msg: MsgAddLiquidity{
				Creator:     validAddr,
				PoolId:      "pool1",
				BaseAmount:  sdk.ZeroInt(),
				QuoteAmount: sdk.NewInt(200),
				MinLpTokens: sdk.NewInt(10),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "base amount must be positive",
		},
		{
			name: "zero quote amount",
			msg: MsgAddLiquidity{
				Creator:     validAddr,
				PoolId:      "pool1",
				BaseAmount:  sdk.NewInt(100),
				QuoteAmount: sdk.ZeroInt(),
				MinLpTokens: sdk.NewInt(10),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "quote amount must be positive",
		},
		{
			name: "multiple issues",
			msg: MsgAddLiquidity{
				Creator:     "invalid_address",
				PoolId:      "",
				BaseAmount:  sdk.ZeroInt(),
				QuoteAmount: sdk.ZeroInt(),
				MinLpTokens: sdk.ZeroInt(),
			},
			err:    sdkerrors.ErrInvalidAddress, // First error encountered
			errMsg: "invalid creator address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()

			if tt.err != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}
