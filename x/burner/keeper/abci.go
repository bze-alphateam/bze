package keeper

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (k Keeper) WithdrawLuckyRaffleParticipants(ctx sdk.Context, height int64) {
	participants := k.GetAllPrefixedRaffleParticipants(ctx, height)
	if len(participants) == 0 {
		k.Logger().Info("no raffle participants found.")
		return
	}

	for _, participant := range participants {
		logger := k.Logger().With("participant", participant)
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

		potInt, ok := math.NewIntFromString(raffle.GetPot())
		if !ok {
			logger.Error("could not parse pot to sdk int", err)
			k.RemoveRaffleParticipant(ctx, participant)
			continue
		}

		currentPot := sdk.NewCoin(raffle.GetDenom(), potInt)
		if currentPot.IsPositive() && k.IsLucky(ctx, &raffle, creatorAddr.String()) {
			logger.With("address", creatorAddr.String(), "denom", raffle.Denom).Info("user won raffle")
			//user won
			wonCoin := k.getWonCoin(&raffle, currentPot)

			err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.RaffleModuleName, creatorAddr, sdk.NewCoins(wonCoin))
			if err != nil {
				logger.Error("could not send coins from module to account", err)
				k.RemoveRaffleParticipant(ctx, participant)
				continue
			}

			raffle.Winners += 1
			totalWonInt, ok := math.NewIntFromString(raffle.TotalWon)
			if !ok {
				logger.Error("could not parse total won")
			} else {
				totalWonInt = totalWonInt.Add(wonCoin.Amount)
				raffle.TotalWon = totalWonInt.String()
				raffle.Pot = currentPot.Amount.Sub(wonCoin.Amount).String()
			}

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
