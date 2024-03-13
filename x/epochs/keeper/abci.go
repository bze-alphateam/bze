package keeper

import (
	"github.com/bze-alphateam/bze/x/epochs/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
	"time"
)

// BeginBlocker of epochs module.
func (k Keeper) BeginBlocker(ctx sdk.Context) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	k.IterateEpochInfo(ctx, func(index int64, epochInfo types.EpochInfo) (stop bool) {
		logger := k.Logger(ctx).With("epoch_identifier", epochInfo.Identifier).
			With("current_epoch", epochInfo.CurrentEpoch)

		// If block time < initial epoch start time, return
		if ctx.BlockTime().Before(epochInfo.StartTime) {
			return
		}

		// if epoch counting hasn't started, signal we need to start.
		shouldInitialize := !epochInfo.EpochCountingStarted

		epochEndTime := epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration)
		shouldStart := (ctx.BlockTime().After(epochEndTime)) || shouldInitialize

		if !shouldStart {
			return false
		}

		epochInfo.CurrentEpochStartHeight = ctx.BlockHeight()
		if shouldInitialize {
			epochInfo.EpochCountingStarted = true
			epochInfo.CurrentEpoch = 1
			epochInfo.CurrentEpochStartTime = epochInfo.StartTime
			logger.Info("initialized new epoch")
		} else {
			k.emitEpochEndEvent(ctx, epochInfo)
			k.AfterEpochEnd(ctx, epochInfo.Identifier, epochInfo.CurrentEpoch)
			epochInfo.CurrentEpoch += 1
			epochInfo.CurrentEpochStartTime = epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration)
			logger.Info("starting new epoch")
		}

		// emit new epoch start event, set epoch info, and run BeforeEpochStart hook
		k.emitEpochStartEvent(ctx, epochInfo)

		k.setEpochInfo(ctx, epochInfo)
		k.BeforeEpochStart(ctx, epochInfo.Identifier, epochInfo.CurrentEpoch)

		return false
	})
}

func (k Keeper) emitEpochEndEvent(ctx sdk.Context, epoch types.EpochInfo) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.EpochEndEvent{
			Identifier: epoch.Identifier,
			Epoch:      strconv.FormatInt(epoch.CurrentEpoch, 10),
		},
	)
	if err != nil {
		ctx.Logger().With("epoch", epoch).Error(err.Error())
	}
}

func (k Keeper) emitEpochStartEvent(ctx sdk.Context, epoch types.EpochInfo) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.EpochStartEvent{
			Identifier: epoch.Identifier,
			Epoch:      strconv.FormatInt(epoch.CurrentEpoch, 10),
		},
	)
	if err != nil {
		ctx.Logger().With("epoch", epoch).Error(err.Error())
	}
}
