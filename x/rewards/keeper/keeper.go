package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

const (
	epochIdentifierDay  = "day"
	epochIdentifierHour = "hour"

	distributionEpoch = epochIdentifierDay
	expirationEpoch   = epochIdentifierHour
)

type (
	Keeper struct {
		cdc           codec.BinaryCodec
		storeService  store.KVStoreService
		logger        log.Logger
		bankKeeper    types.BankKeeper
		distrKeeper   types.DistrKeeper
		epochKeeper   types.EpochKeeper
		tradeKeeper   types.TradingKeeper
		accountKeeper types.AccountKeeper

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
	epochKeeper types.EpochKeeper,
	tradeKeeper types.TradingKeeper,
	accountKeeper types.AccountKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		logger:        logger,
		bankKeeper:    bankKeeper,
		distrKeeper:   distrKeeper,
		epochKeeper:   epochKeeper,
		tradeKeeper:   tradeKeeper,
		accountKeeper: accountKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) smallZeroFillId(id uint64) string {
	return fmt.Sprintf("%012d", id)
}

func (k Keeper) getAmountToCapture(denom, amount string, multiplier int64) (sdk.Coins, error) {
	amtInt, ok := math.NewIntFromString(amount)
	if !ok {
		return nil, fmt.Errorf("could not convert povided amount to int: %s", amount)
	}

	toCapture := sdk.NewCoin(denom, amtInt)
	toCapture.Amount = toCapture.Amount.MulRaw(multiplier)
	if !toCapture.IsValid() || !toCapture.IsPositive() {
		//should never happen
		return nil, fmt.Errorf("calculated amount to capture is not positive")
	}

	return sdk.NewCoins(toCapture), nil
}

func (k Keeper) getPrefixedStore(ctx sdk.Context, p []byte) prefix.Store {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	return prefix.NewStore(storeAdapter, p)
}

func (k Keeper) getNewTradingRewardExpireAt(ctx sdk.Context) uint32 {
	cnt := uint32(k.epochKeeper.GetEpochCountByIdentifier(ctx, expirationEpoch))

	return cnt + expirationPeriodInHours
}
