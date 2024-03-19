package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	epochIdentifierDay = "day"

	distributionEpoch = epochIdentifierDay
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper    types.BankKeeper
		distrKeeper   types.DistrKeeper
		tradingKeeper types.TradingKeeper
		epochKeeper   types.EpochKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
	tradingKeeper types.TradingKeeper,
	epochKeeper types.EpochKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{

		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		paramstore:    ps,
		bankKeeper:    bankKeeper,
		distrKeeper:   distrKeeper,
		tradingKeeper: tradingKeeper,
		epochKeeper:   epochKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) smallZeroFillId(id uint64) string {
	return fmt.Sprintf("%012d", id)
}

func (k Keeper) getAmountToCapture(feeParam, denom, amount string, multiplier int64) (sdk.Coins, error) {
	amtInt, ok := sdk.NewIntFromString(amount)
	if !ok {
		return nil, fmt.Errorf("could not convert povided amount to int: %s", amount)
	}

	toCapture := sdk.NewCoin(denom, amtInt)
	toCapture.Amount = toCapture.Amount.MulRaw(multiplier)
	if !toCapture.IsPositive() {
		//should never happen
		return nil, fmt.Errorf("calculated amount to capture is not positive")
	}

	result := sdk.NewCoins(toCapture)
	if feeParam == "" {
		return result, nil
	}

	fee, err := sdk.ParseCoinNormalized(feeParam)
	if err != nil {
		return nil, fmt.Errorf("could not parse fee param")
	}

	if !fee.IsPositive() {
		return result, nil
	}

	result = result.Add(fee)
	//just avoid any accidental panic
	if !result.IsValid() {
		return nil, fmt.Errorf("invalid amount to capture")
	}

	return result, nil
}
