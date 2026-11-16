package keeper

import (
	"strconv"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) WithdrawLuckyRaffleParticipants(ctx sdk.Context, height int64) {
	participants := k.GetAllPrefixedRaffleParticipants(ctx, height)
	if len(participants) == 0 {
		k.Logger().Info("no raffle participants found.")
		return
	}

	for _, participant := range participants {
		// each ticket is resolved in its own cached context: one that fails (an error or a panic) doesn't void the
		// other draws at this height. A failed ticket is removed like any other, without a draw.
		err := bzeutils.ApplyFuncIfNoError(ctx, func(cached sdk.Context) error {
			k.resolveRaffleTicket(cached, participant)

			return nil
		})
		if err != nil {
			k.Logger().Error("raffle ticket resolution failed, removing it", "participant", participant, "error", err)
			k.RemoveRaffleParticipant(ctx, participant)
		}
	}
}

// resolveRaffleTicket draws one raffle ticket: a winner is paid its share of the pot, a loser's ticket price is
// added to the pot. The ticket is removed either way.
func (k Keeper) resolveRaffleTicket(ctx sdk.Context, participant types.RaffleParticipant) {
	logger := k.Logger().With("participant", participant)
	raffle, found := k.GetRaffle(ctx, participant.Denom)
	if !found {
		logger.Error("could not find raffle for this participant")
		k.RemoveRaffleParticipant(ctx, participant)
		return
	}

	creatorAddr, err := sdk.AccAddressFromBech32(participant.Participant)
	if err != nil {
		logger.Error("could not parse creator address", err)
		k.RemoveRaffleParticipant(ctx, participant)
		return
	}

	potInt, ok := math.NewIntFromString(raffle.GetPot())
	if !ok {
		logger.Error("could not parse pot to sdk int", err)
		k.RemoveRaffleParticipant(ctx, participant)
		return
	}

	currentPot := sdk.NewCoin(raffle.GetDenom(), potInt)
	if currentPot.IsPositive() && k.IsLucky(ctx, &raffle, creatorAddr.String()) {
		logger.With("address", creatorAddr.String(), "denom", raffle.Denom).Info("user won raffle")
		//user won
		wonCoin := k.getWonCoin(&raffle, currentPot)

		// the pot and the total won are updated together with the payout: when the total won can't be read the
		// winner isn't paid, the ticket is removed without a draw
		totalWonInt, ok := math.NewIntFromString(raffle.TotalWon)
		if !ok {
			logger.Error("could not parse total won, removing the ticket without a payout")
			k.RemoveRaffleParticipant(ctx, participant)
			return
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.RaffleModuleName, creatorAddr, sdk.NewCoins(wonCoin))
		if err != nil {
			logger.Error("could not send coins from module to account", err)
			k.RemoveRaffleParticipant(ctx, participant)
			return
		}

		raffle.Winners += 1
		// the total won is only displayed: when it can't grow any more it is left as it is
		if newTotalWon, err := totalWonInt.SafeAdd(wonCoin.Amount); err != nil {
			logger.Error("total won overflow, leaving it unchanged", "error", err)
		} else {
			raffle.TotalWon = newTotalWon.String()
		}
		raffle.Pot = currentPot.Amount.Sub(wonCoin.Amount).String()

		k.SetRaffle(ctx, raffle)
		k.SetRaffleWinner(ctx, types.RaffleWinner{
			Index:  strconv.Itoa(int(raffle.Winners) % 100), //keep only 100 winners
			Denom:  raffle.Denom,
			Amount: wonCoin.Amount.String(),
			Winner: creatorAddr.String(),
		})

		err = ctx.EventManager().EmitTypedEvent(&types.RaffleWinnerEvent{
			Denom:  raffle.Denom,
			Winner: creatorAddr.String(),
			Amount: wonCoin.Amount.String(),
		})
		if err != nil {
			//just log it, don't hinder the response for this error
			k.Logger().Error("failed to emit raffle winner event", err.Error())
		}

	} else {
		logger.With("address", creatorAddr.String(), "denom", raffle.Denom).Info("user lost raffle")

		//add ticket price to pot
		ticketPrice, ok := math.NewIntFromString(raffle.TicketPrice)
		if !ok {
			logger.Error("could not parse ticket price to sdk int")
		} else {
			raffle.Pot = currentPot.Amount.Add(ticketPrice).String()
			k.SetRaffle(ctx, raffle)
		}

		err = ctx.EventManager().EmitTypedEvent(&types.RaffleLostEvent{
			Denom:       raffle.Denom,
			Participant: participant.Participant,
		})
		if err != nil {
			//just log it, don't hinder the response for this error
			k.Logger().Error("failed to emit raffle lost event", err.Error())
		}
	}

	k.RemoveRaffleParticipant(ctx, participant)
}

func (k Keeper) getWonCoin(raffle *types.Raffle, pot sdk.Coin) sdk.Coin {
	winRatio := math.LegacyMustNewDecFromStr(raffle.Ratio)
	potAmount := math.LegacyNewDecFromInt(pot.Amount)

	prize := potAmount.Mul(winRatio).TruncateInt()
	if !prize.IsPositive() {
		prize = math.ZeroInt()
	}

	return sdk.NewCoin(pot.Denom, prize)
}
