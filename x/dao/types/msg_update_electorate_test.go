package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgUpdateElectorate_ValidateBasic(t *testing.T) {
	authority := sdk.AccAddress("dao_authority_______").String()
	electorate := func(srIds []string, drDenom string) *types.StakeElectorate {
		return &types.StakeElectorate{SrIds: srIds, DrDenom: drDenom}
	}

	for _, tc := range []struct {
		desc    string
		msg     *types.MsgUpdateElectorate
		wantErr error
		errText string
	}{
		{desc: "staking rewards and a denom reward", msg: types.NewMsgUpdateElectorate(authority, 1, electorate([]string{"1", "2"}, "ubze"))},
		{desc: "denom reward only", msg: types.NewMsgUpdateElectorate(authority, 1, electorate(nil, "ubze"))},
		{desc: "empty electorate", msg: types.NewMsgUpdateElectorate(authority, 1, electorate(nil, ""))},
		{desc: "nil electorate", msg: types.NewMsgUpdateElectorate(authority, 1, nil)},
		{desc: "bad authority", msg: types.NewMsgUpdateElectorate("bad", 1, electorate([]string{"1"}, "")), wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", msg: types.NewMsgUpdateElectorate(authority, 0, electorate([]string{"1"}, "")), wantErr: types.ErrDaoNotFound},
		{desc: "D14: duplicate staking reward", msg: types.NewMsgUpdateElectorate(authority, 1, electorate([]string{"1", "2", "1"}, "")), wantErr: types.ErrInvalidElectorate, errText: "duplicate staking reward 1"},
		{desc: "empty staking reward id", msg: types.NewMsgUpdateElectorate(authority, 1, electorate([]string{""}, "")), wantErr: types.ErrInvalidElectorate, errText: "empty staking reward id"},
		{desc: "invalid denom reward denom", msg: types.NewMsgUpdateElectorate(authority, 1, electorate(nil, "1x")), wantErr: types.ErrInvalidElectorate, errText: "invalid denom reward denom"},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			require.ErrorContains(t, err, tc.errText)
		})
	}
}
