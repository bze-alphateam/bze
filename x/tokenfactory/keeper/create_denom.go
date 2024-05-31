package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Runs CreateDenom logic after the charge and all denom validation has been handled.
// Made into a second function for genesis initialization.
func (k Keeper) CreateDenomAfterValidation(ctx sdk.Context, creatorAddr string, denom string) (err error) {
	_, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		denomMetaData := banktypes.Metadata{
			DenomUnits: []*banktypes.DenomUnit{{
				Denom:    denom,
				Exponent: 0,
			}},
			Base: denom,
		}

		k.bankKeeper.SetDenomMetaData(ctx, denomMetaData)
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

func (k Keeper) validateCreateDenom(ctx sdk.Context, creatorAddr string, subdenom string) (newTokenDenom string, err error) {
	// copied from terra-money tokenfactory: Temporary check until IBC bug is sorted out
	if k.bankKeeper.HasSupply(ctx, subdenom) {
		return "", fmt.Errorf("temporary error until IBC bug is sorted out, can't create subdenoms that are the same as a native denom")
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
	if params.CreateDenomFee != "" {
		accAddr, err := sdk.AccAddressFromBech32(creatorAddr)
		if err != nil {
			return err
		}
		normCoins, err := sdk.ParseCoinsNormalized(params.CreateDenomFee)
		if err != nil {
			return err
		}

		//don't try to fund the community pool if the tax is 0
		if normCoins.IsZero() {
			return nil
		}

		if err := k.distrKeeper.FundCommunityPool(ctx, normCoins, accAddr); err != nil {
			return err
		}
	}

	return nil
}
