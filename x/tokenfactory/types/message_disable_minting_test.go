package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

func TestMsgDisableMinting_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  *types.MsgDisableMinting
		err  error
		// fails is set when the error is not a registered sentinel (sdk.ValidateDenom errors)
		fails bool
	}{
		{
			name: "valid",
			msg:  types.NewMsgDisableMinting(creator, "factory/"+creator+"/token"),
		},
		{
			name: "valid with a denom created by another address",
			msg:  types.NewMsgDisableMinting(creator, "factory/"+sample.AccAddress()+"/token"),
		},
		{
			name: "invalid creator",
			msg:  types.NewMsgDisableMinting("invalid_address", "factory/"+creator+"/token"),
			err:  sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator",
			msg:  types.NewMsgDisableMinting("", "factory/"+creator+"/token"),
			err:  sdkerrors.ErrInvalidAddress,
		},
		{
			name:  "empty denom",
			msg:   types.NewMsgDisableMinting(creator, ""),
			fails: true,
		},
		{
			name: "native denom",
			msg:  types.NewMsgDisableMinting(creator, "ubze"),
			err:  types.ErrInvalidDenom,
		},
		{
			name: "ibc denom",
			msg:  types.NewMsgDisableMinting(creator, "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"),
			err:  types.ErrInvalidDenom,
		},
		{
			name: "factory denom with an invalid creator part",
			msg:  types.NewMsgDisableMinting(creator, "factory/notanaddress/token"),
			err:  types.ErrInvalidDenom,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.fails {
				require.Error(t, err)
				return
			}
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
