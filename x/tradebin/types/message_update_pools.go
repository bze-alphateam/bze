package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MaxUpdatesPerUpdatePoolsMsg bounds the pools one MsgUpdatePools changes.
const MaxUpdatesPerUpdatePoolsMsg = 100

var _ sdk.Msg = &MsgUpdatePools{}

func NewMsgUpdatePools(authority string, updates []PoolUpdate) *MsgUpdatePools {
	return &MsgUpdatePools{Authority: authority, Updates: updates}
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUpdatePools) ValidateBasic() error {
	return m.Validate()
}

// Validate validates the MsgUpdatePools request: a bech32 authority and 1 to MaxUpdatesPerUpdatePoolsMsg
// updates with distinct, non-empty pool ids, each fee and fee split passing the pool creation rules.
// Whether the pools exist needs the store and is checked by the msg server.
func (m *MsgUpdatePools) Validate() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if len(m.Updates) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no pool updates provided")
	}

	if len(m.Updates) > MaxUpdatesPerUpdatePoolsMsg {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "too many pool updates: %d (max %d)", len(m.Updates), MaxUpdatesPerUpdatePoolsMsg)
	}

	seen := make(map[string]struct{}, len(m.Updates))
	for _, u := range m.Updates {
		if _, ok := seen[u.PoolId]; ok {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate pool id %s", u.PoolId)
		}
		seen[u.PoolId] = struct{}{}

		if err := u.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate checks one pool update: a non-empty pool id and a fee and fee split that pass the pool
// creation rules.
func (u PoolUpdate) Validate() error {
	if u.PoolId == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty pool id")
	}

	// an omitted Dec decodes as nil: refuse it here, the shared validators expect set values
	if u.Fee.IsNil() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "pool %s: missing fee", u.PoolId)
	}

	if u.FeeDest.Treasury.IsNil() || u.FeeDest.Burner.IsNil() || u.FeeDest.Providers.IsNil() {
		return errorsmod.Wrapf(ErrInvalidFeeDestination, "pool %s: missing fee destination share", u.PoolId)
	}

	if err := ValidatePoolFee(u.Fee); err != nil {
		return errorsmod.Wrapf(err, "pool %s", u.PoolId)
	}

	if err := ValidatePoolFeeDestination(u.FeeDest); err != nil {
		return errorsmod.Wrapf(err, "pool %s", u.PoolId)
	}

	return nil
}
