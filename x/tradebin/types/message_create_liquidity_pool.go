package types

import (
	"cosmossdk.io/errors"
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCreateLiquidityPool = "create_liquidity_pool"

var _ sdk.Msg = &MsgCreateLiquidityPool{}

func NewMsgCreateLiquidityPool(creator string, base string, quote string, fee string, feeDest string, stable bool, initialBase string, initialQuote string) *MsgCreateLiquidityPool {
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

func (msg *MsgCreateLiquidityPool) Route() string {
	return RouterKey
}

func (msg *MsgCreateLiquidityPool) Type() string {
	return TypeMsgCreateLiquidityPool
}

func (msg *MsgCreateLiquidityPool) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreateLiquidityPool) GetCreatorAcc() sdk.AccAddress {
	signers := msg.GetSigners()
	if len(signers) == 0 {
		return nil
	}

	return signers[0]
}

func (msg *MsgCreateLiquidityPool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateLiquidityPool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.Base) == 0 && len(msg.Quote) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing assets")
	}

	if len(msg.Fee) == 0 || len(msg.FeeDest) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing fee parameters")
	}

	if len(msg.InitialBase) == 0 || len(msg.InitialQuote) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing initial liquidity")
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
