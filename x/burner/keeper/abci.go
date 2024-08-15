package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (k Keeper) WithdrawLuckyRaffleParticipants(ctx sdk.Context, height int64) {
	participants := k.GetAllPrefixedRaffleParticipants(ctx, ctx.BlockHeight())
	if len(participants) == 0 {
		k.Logger(ctx).Info("no raffle participants found.")
		return
	}

	//get raffle module account
	raffleAcc := k.accKeeper.GetModuleAccount(ctx, types.RaffleModuleName)
	if raffleAcc == nil {
		k.Logger(ctx).Error("could not find raffle module account")

		return
	}
	for _, participant := range participants {
		logger := k.Logger(ctx).With("participant", participant)
		raffle, found := k.GetRaffle(ctx, participant.Denom)
		if !found {
			logger.Error("could not find raffle for this participant")
			k.RemoveRaffleParticipant(ctx, participant)
			continue
		}

		creatorAddr, err := sdk.AccAddressFromBech32(participant.Participant)
		if err != nil {
			logger.Error("could not parse creator address", err)
			k.RemoveRaffleParticipant(ctx, participant)
			continue
		}

		//get raffle module balance for the raffle denom before capturing the ticket price
		currentPot := k.bankKeeper.GetBalance(ctx, raffleAcc.GetAddress(), raffle.Denom)
		if !currentPot.IsPositive() {
			logger.Error("current pot is not positive")
			k.RemoveRaffleParticipant(ctx, participant)
			continue
		}

		//get ticket price to enter the raffle
		ticketPriceInt, ok := sdk.NewIntFromString(raffle.TicketPrice)
		if !ok {
			//should never happen
			logger.Error("could not parse ticket price")
		}

		if k.IsLucky(ctx, &raffle, creatorAddr.String()) {
			logger.With("address", creatorAddr.String(), "denom", raffle.Denom).Info("user won raffle")
			//user won
			winRatio := sdk.MustNewDecFromStr(raffle.Ratio)
			wonAmount := currentPot.Amount.Sub(ticketPriceInt).ToDec().Mul(winRatio).TruncateInt()
			if !wonAmount.IsPositive() {
				wonAmount = currentPot.Amount.Sub(ticketPriceInt)
			}
			wonCoin := sdk.NewCoin(currentPot.Denom, wonAmount)

			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.RaffleModuleName, creatorAddr, sdk.NewCoins(wonCoin))
			if err != nil {
				logger.Error("could not send coins from module to account", err)
				k.RemoveRaffleParticipant(ctx, participant)
				continue
			}

			raffle.Winners += 1
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
				k.Logger(ctx).Error("failed to emit raffle winner event", err.Error())
			}

		} else {
			logger.With("address", creatorAddr.String(), "denom", raffle.Denom).Info("user lost raffle")
			err = ctx.EventManager().EmitTypedEvent(&types.RaffleLostEvent{
				Denom:       raffle.Denom,
				Participant: participant.Participant,
			})
			if err != nil {
				//just log it, don't hinder the response for this error
				k.Logger(ctx).Error("failed to emit raffle lost event", err.Error())
			}
		}

		k.RemoveRaffleParticipant(ctx, participant)
	}
}
