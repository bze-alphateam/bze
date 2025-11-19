package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateLiquidityPool{}

func NewMsgCreateLiquidityPool(creator, base, quote, fee, feeDest string, stable bool, initialBase, initialQuote math.Int) *MsgCreateLiquidityPool {
	return &MsgCreateLiquidityPool{
		Creator:      creator,
		Base:         base,
		Quote:        quote,
		Fee:          fee,
		FeeDest:      feeDest,
		Stable:       stable,
		InitialBase:  initialBase,
		InitialQuote: initialQuote,
	}
}

func (msg *MsgCreateLiquidityPool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.Base) == 0 && len(msg.Quote) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "missing assets")
	}

	if len(msg.Fee) == 0 || len(msg.FeeDest) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "missing fee parameters")
	}

	if !msg.InitialBase.IsPositive() || !msg.InitialQuote.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "missing initial liquidity")
	}

	return nil
}

func (msg *MsgCreateLiquidityPool) ParseFeeDestination() (FeeDestination, error) {
	mdByte := []byte(msg.FeeDest)
	mdStruct := FeeDestination{}
	err := json.Unmarshal(mdByte, &mdStruct)
	if err != nil {
		return mdStruct, err
	}

	return mdStruct, nil
}
