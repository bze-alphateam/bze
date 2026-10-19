package keeper_test

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const fakeSendEvent = "fake_bank_send"

// fakeBank is an in-memory x/bank Msg service, so the executor's messages run
// through a real router without a bank keeper. MsgSend is served; MsgMultiSend
// always panics, with panicWith when it is set.
type fakeBank struct {
	banktypes.UnimplementedMsgServer

	balances  map[string]sdk.Coins
	panicWith any
}

func newFakeBank() *fakeBank {
	return &fakeBank{balances: map[string]sdk.Coins{}}
}

func (b *fakeBank) fund(addr sdk.AccAddress, coins sdk.Coins) {
	b.balances[addr.String()] = b.balances[addr.String()].Add(coins...)
}

func (b *fakeBank) balance(addr sdk.AccAddress) sdk.Coins {
	return b.balances[addr.String()]
}

func (b *fakeBank) Send(goCtx context.Context, msg *banktypes.MsgSend) (*banktypes.MsgSendResponse, error) {
	from, negative := b.balances[msg.FromAddress].SafeSub(msg.Amount...)
	if negative {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, "%s has %s", msg.FromAddress, b.balances[msg.FromAddress])
	}

	b.balances[msg.FromAddress] = from
	b.balances[msg.ToAddress] = b.balances[msg.ToAddress].Add(msg.Amount...)
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvent(sdk.NewEvent(fakeSendEvent))

	return &banktypes.MsgSendResponse{}, nil
}

func (b *fakeBank) MultiSend(context.Context, *banktypes.MsgMultiSend) (*banktypes.MsgMultiSendResponse, error) {
	if b.panicWith != nil {
		panic(b.panicWith)
	}

	panic("multi send exploded")
}
