package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRemoveLiquidity{}

func NewMsgRemoveLiquidity(creator, poolId string, lpTokens, minBase, minQuote math.Int) *MsgRemoveLiquidity {
	return &MsgRemoveLiquidity{
		Creator:  creator,
		PoolId:   poolId,
		LpTokens: lpTokens,
		MinBase:  minBase,
		MinQuote: minQuote,
	}
}

func (msg *MsgRemoveLiquidity) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.PoolId) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "pool id cannot be empty")
	}

	if !msg.LpTokens.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid lpTokens %s", msg.LpTokens.String())
	}

	if !msg.MinBase.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid minBase %s", msg.MinBase.String())
	}

	if !msg.MinQuote.IsPositive() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid minQuote %s", msg.MinQuote.String())
	}

	return nil
}
