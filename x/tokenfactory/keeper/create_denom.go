package keeper

import (
	"fmt"
	"strings"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// defaultDisplayExponent is the number of decimals of a new denom's display unit.
const defaultDisplayExponent = 6

// Runs CreateDenom logic after the charge and all denom validation has been handled.
// Made into a second function for genesis initialization.
func (k Keeper) CreateDenomAfterValidation(ctx sdk.Context, creatorAddr string, denom string) (err error) {
	_, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		k.bankKeeper.SetDenomMetaData(ctx, defaultDenomMetadata(denom))
	}

	dAuth := types.DenomAuthority{
		Admin: creatorAddr,
	}
	err = k.SetDenomAuthority(ctx, denom, dAuth)
	if err != nil {
		return err
	}

	k.addDenomFromCreator(ctx, creatorAddr, denom)
	return nil
}

// defaultDenomMetadata builds the bank metadata of a new denom from its subdenom,
// so the stored metadata passes bank's own validation: name = subdenom,
// symbol = UPPER(subdenom) and a display unit UPPER(subdenom) with 6 decimals.
// When UPPER(subdenom) is not a valid coin denom (too short, or not starting
// with a letter) the base stays the only unit and the display.
// The creator changes it afterwards with MsgSetDenomMetadata.
func defaultDenomMetadata(denom string) banktypes.Metadata {
	subdenom := denom
	if parts := strings.SplitN(denom, "/", 3); len(parts) == 3 {
		subdenom = parts[2]
	}

	symbol := strings.ToUpper(subdenom)
	md := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{{Denom: denom, Exponent: 0}},
		Base:       denom,
		Display:    denom,
		Name:       subdenom,
		Symbol:     symbol,
	}

	if sdk.ValidateDenom(symbol) == nil && symbol != denom {
		md.DenomUnits = append(md.DenomUnits, &banktypes.DenomUnit{Denom: symbol, Exponent: defaultDisplayExponent})
		md.Display = symbol
	}

	return md
}

func (k Keeper) validateCreateDenom(ctx sdk.Context, creatorAddr string, subdenom string) (newTokenDenom string, err error) {
	// copied from terra-money tokenfactory: Temporary check until IBC bug is sorted out
	if k.bankKeeper.HasSupply(ctx, subdenom) {
		return "", fmt.Errorf("temporary error until IBC bug is sorted out, can't create subdenoms that are the same as a native denom")
	}

	if strings.Contains(subdenom, "_") {
		return "", types.ErrInvalidSubdenom
	}

	denom, err := types.GetTokenDenom(creatorAddr, subdenom)
	if err != nil {
		return "", err
	}

	_, found := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if found {
		return "", types.ErrDenomExists
	}

	return denom, nil
}

func (k Keeper) chargeForCreateDenom(ctx sdk.Context, creatorAddr string) (err error) {
	params := k.GetParams(ctx)

	// if DenomCreationFee is non-zero, transfer the tokens from the creator
	// account to community pool
	if params.CreateDenomFee.IsPositive() {
		accAddr, err := sdk.AccAddressFromBech32(creatorAddr)
		if err != nil {
			return err
		}

		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, accAddr, sdk.NewCoins(params.CreateDenomFee), types.ModuleName)
		if err != nil {
			return err
		}

		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee)
		if err != nil {
			return err
		}
	}

	return nil
}
