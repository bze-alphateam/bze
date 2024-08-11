package keeper

import (
	"context"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) FundBurner(goCtx context.Context, msg *types.MsgFundBurner) (*types.MsgFundBurnerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	amount, err := sdk.ParseCoinsNormalized(msg.Amount)
	if err != nil {
		return nil, err
	}

	creatorAccount, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAccount, types.ModuleName, amount)
	if err != nil {
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.FundBurnerEvent{From: msg.Creator, Amount: amount.String()})
	if err != nil {
		return nil, err
	}

	_ = ctx

	return &types.MsgFundBurnerResponse{}, nil
}

func (k msgServer) StartRaffle(goCtx context.Context, msg *types.MsgStartRaffle) (*types.MsgStartRaffleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.bankKeeper.HasSupply(ctx, msg.Denom) {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "denom %s does not exist", msg.Denom)
	}

	_, alreadyStarted := k.GetRaffle(ctx, msg.Denom)
	if alreadyStarted {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "raffle already running for this coin")
	}

	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	raffle, err := k.raffleFromMsgStartRaffle(ctx, msg)
	if err != nil {
		return nil, err
	}

	//do not check if OK because it is checked in basic validation and in method that converts message to storage struct
	potAmt, _ := sdk.NewIntFromString(raffle.Pot)
	pot := sdk.NewCoin(raffle.Denom, potAmt)
	if !k.checkCreatorRafflePotBalance(ctx, pot, creatorAcc) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "not enough balance")
	}

	if err = k.captureRafflePot(ctx, pot, creatorAcc); err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "could not capture pot: (%s)", err.Error())
	}

	k.SetRaffle(ctx, raffle)
	k.SetRaffleDeleteHook(ctx, types.RaffleDeleteHook{
		Denom: raffle.Denom,
		EndAt: raffle.EndAt,
	})

	return &types.MsgStartRaffleResponse{}, nil
}

func (k Keeper) captureRafflePot(ctx sdk.Context, pot sdk.Coin, creator sdk.AccAddress) error {
	//call it to make sure the account is created
	raffleAcc := k.accKeeper.GetModuleAccount(ctx, types.RaffleModuleName)
	if raffleAcc == nil {

		return fmt.Errorf("could not get module account %s ", types.RaffleModuleName)
	}

	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, creator, types.RaffleModuleName, sdk.NewCoins(pot))
}

func (k Keeper) checkCreatorRafflePotBalance(ctx sdk.Context, pot sdk.Coin, creator sdk.AccAddress) bool {
	balances := k.bankKeeper.SpendableCoins(ctx, creator)

	return pot.Amount.LTE(balances.AmountOf(pot.Denom))
}

func (k Keeper) raffleFromMsgStartRaffle(ctx sdk.Context, msg *types.MsgStartRaffle) (types.Raffle, error) {
	raffle, err := msg.ToStorageRaffle()
	if err != nil {
		return types.Raffle{}, err
	}

	raffle.Winners = 0
	currentEpoch := k.GetRaffleCurrentEpoch(ctx)
	raffle.EndAt = currentEpoch + (raffle.Duration * 24)

	return raffle, nil
}
