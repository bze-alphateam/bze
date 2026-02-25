package v2types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgCreateLiquidityPool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.Base) == 0 || len(msg.Quote) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing assets")
	}

	if msg.Base == msg.Quote {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "base and quote cannot be the same")
	}

	if !msg.Fee.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "fee must be positive")
	}

	if len(msg.FeeDest) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing fee destination")
	}

	if !msg.InitialBase.IsPositive() || !msg.InitialQuote.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "missing initial liquidity")
	}

	if msg.Stable {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "stable pools are not supported yet")
	}

	return nil
}
