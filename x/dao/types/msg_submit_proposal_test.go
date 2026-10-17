package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgSubmitProposal_ValidateBasic(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		proposer string
		daoId    uint64
		title    string
		msgs     []sdk.Msg
		wantErr  error
	}{
		{desc: "valid", proposer: execAdmin.String(), daoId: 1, title: "pay", msgs: []sdk.Msg{treasurySend()}},
		{desc: "no msgs (signalling)", proposer: execAdmin.String(), daoId: 1, title: "signal"},
		{desc: "invalid proposer", proposer: "bze1nope", daoId: 1, title: "pay", wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty proposer", daoId: 1, title: "pay", wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", proposer: execAdmin.String(), title: "pay", wantErr: types.ErrDaoNotFound},
		{desc: "empty title", proposer: execAdmin.String(), daoId: 1, wantErr: types.ErrInvalidProposal},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			msg, err := types.NewMsgSubmitProposal(tc.proposer, tc.daoId, tc.title, "summary", tc.msgs, false)
			require.NoError(t, err)

			err = msg.ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// An Any the codec has not unpacked holds no sdk.Msg.
func TestMsgSubmitProposal_ValidateBasic_UnpackedAnyRejected(t *testing.T) {
	msg := &types.MsgSubmitProposal{
		Proposer: execAdmin.String(),
		DaoId:    1,
		Title:    "pay",
		Msgs:     []*codectypes.Any{{TypeUrl: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidMsgs)
}

// The inner msgs survive a codec round trip, in the msg and in a stored Proposal.
func TestMsgSubmitProposal_UnpacksMsgsAfterDecoding(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	msg, err := types.NewMsgSubmitProposal(execAdmin.String(), 1, "pay", "", []sdk.Msg{treasurySend()}, true)
	require.NoError(t, err)
	packed, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	bz, err := cdc.MarshalJSON(packed)
	require.NoError(t, err)

	var decoded codectypes.Any
	require.NoError(t, cdc.UnmarshalJSON(bz, &decoded))
	var got sdk.Msg
	require.NoError(t, cdc.UnpackAny(&decoded, &got))

	msgs, err := got.(*types.MsgSubmitProposal).UnpackMsgs()
	require.NoError(t, err)
	require.Equal(t, []sdk.Msg{treasurySend()}, msgs)

	proposalBz, err := cdc.Marshal(&types.Proposal{Id: 1, Msgs: msg.Msgs})
	require.NoError(t, err)
	var proposal types.Proposal
	require.NoError(t, cdc.Unmarshal(proposalBz, &proposal))
	require.Equal(t, treasurySend(), proposal.Msgs[0].GetCachedValue())
}
