package types

import (
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
				Input:     "100ubze",
				MinOutput: "200uusdc",
			},
			err: nil,
		},
		{
			name: "invalid address",
			msg: MsgMultiSwap{
				Creator:   "invalid_address",
				Routes:    []string{"pool1", "pool2"},
				Input:     "100ubze",
				MinOutput: "200uusdc",
			},
			err:    sdkerrors.ErrInvalidAddress,
			errMsg: "invalid creator address",
		},
		{
			name: "empty routes",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{},
				Input:     "100ubze",
				MinOutput: "200uusdc",
			},
			err:    ErrInvalidRoutes,
			errMsg: "routes length must be between 0 and",
		},
		{
			name: "too many routes",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2", "pool3", "pool4", "pool5", "pool6"},
				Input:     "100ubze",
				MinOutput: "200uusdc",
			},
			err:    ErrInvalidRoutes,
			errMsg: "routes length must be between 0 and",
		},
		{
			name: "empty input",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "",
				MinOutput: "200uusdc",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "invalid input amount",
		},
		{
			name: "empty min output",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "100ubze",
				MinOutput: "",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "invalid minimum output",
		},
		{
			name: "invalid input coin format",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "invalid_coin",
				MinOutput: "200uusdc",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "invalid input",
		},
		{
			name: "invalid min output coin format",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "100ubze",
				MinOutput: "invalid_coin",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "invalid minimum output",
		},
		{
			name: "negative input amount",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "-100ubze",
				MinOutput: "200uusdc",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "invalid input",
		},
		{
			name: "negative min output amount",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "100ubze",
				MinOutput: "-200uusdc",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "invalid minimum output",
		},
		{
			name: "zero input amount",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "0ubze",
				MinOutput: "200uusdc",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "input is not positive",
		},
		{
			name: "zero min output amount",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2"},
				Input:     "100ubze",
				MinOutput: "0uusdc",
			},
			err:    ErrInvalidOrderAmount,
			errMsg: "minimum output is not positive",
		},
		{
			name: "valid single route",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1"},
				Input:     "100ubze",
				MinOutput: "200uusdc",
			},
			err: nil,
		},
		{
			name: "valid max routes",
			msg: MsgMultiSwap{
				Creator:   validAddr,
				Routes:    []string{"pool1", "pool2", "pool3", "pool4", "pool5"},
				Input:     "100ubze",
				MinOutput: "200uusdc",
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
