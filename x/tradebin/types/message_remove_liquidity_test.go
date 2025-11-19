package types

import (
	"cosmossdk.io/math"
	math2 "math"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgRemoveLiquidity_ValidateBasic(t *testing.T) {
	validAddr := sample.AccAddress()

	tests := []struct {
		name   string
		msg    MsgRemoveLiquidity
		err    error
		errMsg string
	}{
		{
			name: "valid message",
			msg: MsgRemoveLiquidity{
				Creator:  validAddr,
				PoolId:   "pool1",
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(50),
				MinQuote: math.NewInt(100),
			},
			err: nil,
		},
		{
			name: "invalid address",
			msg: MsgRemoveLiquidity{
				Creator:  "invalid_address",
				PoolId:   "pool1",
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(50),
				MinQuote: math.NewInt(100),
			},
			err:    sdkerrors.ErrInvalidAddress,
			errMsg: "invalid creator address",
		},
		{
			name: "empty pool id",
			msg: MsgRemoveLiquidity{
				Creator:  validAddr,
				PoolId:   "",
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(50),
				MinQuote: math.NewInt(100),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "pool id cannot be empty",
		},
		{
			name: "zero lp tokens",
			msg: MsgRemoveLiquidity{
				Creator:  validAddr,
				PoolId:   "pool1",
				LpTokens: math.ZeroInt(),
				MinBase:  math.NewInt(50),
				MinQuote: math.NewInt(100),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "invalid lpTokens 0",
		},
		{
			name: "zero min base",
			msg: MsgRemoveLiquidity{
				Creator:  validAddr,
				PoolId:   "pool1",
				LpTokens: math.NewInt(100),
				MinBase:  math.ZeroInt(),
				MinQuote: math.NewInt(100),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "invalid minBase 0",
		},
		{
			name: "zero min quote",
			msg: MsgRemoveLiquidity{
				Creator:  validAddr,
				PoolId:   "pool1",
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(50),
				MinQuote: math.ZeroInt(),
			},
			err:    sdkerrors.ErrInvalidRequest,
			errMsg: "invalid minQuote 0",
		},
		{
			name: "multiple issues",
			msg: MsgRemoveLiquidity{
				Creator:  "invalid_address",
				PoolId:   "",
				LpTokens: math.ZeroInt(),
				MinBase:  math.ZeroInt(),
				MinQuote: math.ZeroInt(),
			},
			err:    sdkerrors.ErrInvalidAddress, // First error encountered
			errMsg: "invalid creator address",
		},
		{
			name: "max uint64 values",
			msg: MsgRemoveLiquidity{
				Creator:  validAddr,
				PoolId:   "pool1",
				LpTokens: math.NewIntFromUint64(math2.MaxUint64),
				MinBase:  math.NewIntFromUint64(math2.MaxUint64),
				MinQuote: math.NewIntFromUint64(math2.MaxUint64),
			},
			err: nil, // Should be valid as these are positive numbers
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
