package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgMultiSwap_ValidateBasic(t *testing.T) {
	validAddr := sample.AccAddress()

	tests := []struct {
		name   string
		msg    MsgMultiSwap
		err    error
		errMsg string
	}{
		{
			name: "valid message",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     sdk.NewInt64Coin("ubze", 100),  //"100ubze",
				MinOutput: sdk.NewInt64Coin("uusdc", 200), //"200uusdc",
			},
			err: nil,
		},
		{
			name: "invalid address",
			msg: MsgMultiSwap{
				Creator:   "invalid_address",
				Routes:    []string{"pool1", "pool2"},
				Input:     sdk.NewInt64Coin("ubze", 100),  //"100ubze",
				MinOutput: sdk.NewInt64Coin("uusdc", 200), //"200uusdc",
			},
			err:    sdkerrors.ErrInvalidAddress,
			errMsg: "invalid creator address",
		},
		{
			name: "empty routes",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{},
				Input:     sdk.NewInt64Coin("ubze", 100),
				MinOutput: sdk.NewInt64Coin("uusdc", 200),
			},
			err:    ErrInvalidRoutes,
			errMsg: "routes length must be between 0 and",
		},
		{
			name: "too many routes",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2", "pool3", "pool4", "pool5", "pool6"},
				Input:     sdk.NewInt64Coin("ubze", 100),
				MinOutput: sdk.NewInt64Coin("uusdc", 200),
			},
			err:    ErrInvalidRoutes,
			errMsg: "routes length must be between 0 and",
		},
		{
			name: "empty input",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     sdk.NewInt64Coin("ubze", 0),
				MinOutput: sdk.NewInt64Coin("uusdc", 200),
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "not positive",
		},
		{
			name: "empty min output",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     sdk.NewInt64Coin("ubze", 100),
				MinOutput: sdk.NewInt64Coin("uusdc", 0),
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "minimum output",
		},
		{
			name: "valid single route",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1"},
				Input:     sdk.NewInt64Coin("ubze", 100),
				MinOutput: sdk.NewInt64Coin("uusdc", 200),
			},
			err: nil,
		},
		{
			name: "valid max routes",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2", "pool3", "pool4", "pool5"},
				Input:     sdk.NewInt64Coin("ubze", 100),
				MinOutput: sdk.NewInt64Coin("uusdc", 200),
			},
			err: nil,
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
