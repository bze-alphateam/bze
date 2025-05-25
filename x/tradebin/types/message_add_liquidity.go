package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAddLiquidity{}

func NewMsgAddLiquidity(creator, poolId string, baseAmount, quoteAmount, minLpTokens math.Int) *MsgAddLiquidity {
	return &MsgAddLiquidity{
		Creator:     creator,
		PoolId:      poolId,
		BaseAmount:  baseAmount,
		QuoteAmount: quoteAmount,
		MinLpTokens: minLpTokens,
	}
}

func (msg *MsgAddLiquidity) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.PoolId) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "pool id cannot be empty")
	}

	if !msg.MinLpTokens.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "min lp tokens must be positive")
	}

	if !msg.BaseAmount.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "base amount must be positive")
	}

	if !msg.QuoteAmount.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "quote amount must be positive")
	}

	return nil
}
