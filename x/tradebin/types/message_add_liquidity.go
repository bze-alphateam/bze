package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddLiquidity = "add_liquidity"

var _ sdk.Msg = &MsgAddLiquidity{}

func NewMsgAddLiquidity(creator, poolId string, baseAmount, quoteAmount, minLpTokens sdk.Int) *MsgAddLiquidity {
	return &MsgAddLiquidity{
		Creator:     creator,
		PoolId:      poolId,
		BaseAmount:  baseAmount,
		QuoteAmount: quoteAmount,
		MinLpTokens: minLpTokens,
	}
}

func (msg *MsgAddLiquidity) Route() string {
	return RouterKey
}

func (msg *MsgAddLiquidity) Type() string {
	return TypeMsgAddLiquidity
}

func (msg *MsgAddLiquidity) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddLiquidity) GetCreatorAcc() sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil
	}

	return creator
}

func (msg *MsgAddLiquidity) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddLiquidity) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.PoolId) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "pool id cannot be empty")
	}

	if !msg.MinLpTokens.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "min lp tokens must be positive")
	}

	if !msg.BaseAmount.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "base amount must be positive")
	}

	if !msg.QuoteAmount.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "quote amount must be positive")
	}

	return nil
}
