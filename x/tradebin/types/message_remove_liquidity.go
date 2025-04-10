package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRemoveLiquidity = "remove_liquidity"

var _ sdk.Msg = &MsgRemoveLiquidity{}

func NewMsgRemoveLiquidity(creator, poolId string, lpTokens, minBase, minQuote uint64) *MsgRemoveLiquidity {
	return &MsgRemoveLiquidity{
		Creator:  creator,
		PoolId:   poolId,
		LpTokens: lpTokens,
		MinBase:  minBase,
		MinQuote: minQuote,
	}
}

func (msg *MsgRemoveLiquidity) Route() string {
	return RouterKey
}

func (msg *MsgRemoveLiquidity) Type() string {
	return TypeMsgRemoveLiquidity
}

func (msg *MsgRemoveLiquidity) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRemoveLiquidity) GetCreatorAcc() sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil
	}

	return creator
}

func (msg *MsgRemoveLiquidity) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRemoveLiquidity) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.PoolId) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "pool id cannot be empty")
	}

	if msg.LpTokens <= 0 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid lpTokens %d", msg.LpTokens)
	}

	if msg.MinBase <= 0 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid minBase %d", msg.MinBase)
	}

	if msg.MinQuote <= 0 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid minQuote %d", msg.MinQuote)
	}

	return nil
}
