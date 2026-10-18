package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) SubmitProposal(goCtx context.Context, msg *types.MsgSubmitProposal) (*types.MsgSubmitProposalResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	proposer := sdk.MustAccAddressFromBech32(msg.Proposer)
	if err := k.checkProposer(ctx, dao, proposer); err != nil {
		return nil, err
	}

	denominator := k.votingDenominator(ctx, dao)
	if !denominator.IsPositive() {
		return nil, errorsmod.Wrapf(types.ErrNoVotingPower, "dao %d has no voters", dao.Id)
	}

	params := k.GetParams(ctx)
	if k.countOpenProposals(ctx, dao.Id, params.MaxOpenProposals) >= params.MaxOpenProposals {
		return nil, errorsmod.Wrapf(types.ErrTooManyProposals, "dao %d already has %d proposals in voting", dao.Id, params.MaxOpenProposals)
	}

	if err := validateProposalContent(msg, params); err != nil {
		return nil, err
	}

	msgs, err := msg.UnpackMsgs()
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidMsgs, err.Error())
	}

	// the executor checks the signers again when the proposal runs; checking
	// them now refuses a proposal that could never execute (D1, D24)
	daoAddr := sdk.MustAccAddressFromBech32(dao.Address)
	for i, m := range msgs {
		if err := k.ensureDaoSigner(m, daoAddr); err != nil {
			return nil, errorsmod.Wrapf(err, "message %d", i)
		}
	}

	period := dao.Config.VotingPeriod
	if msg.Expedited {
		if !dao.Config.AllowExpedite {
			return nil, errorsmod.Wrapf(types.ErrExpediteNotAllowed, "dao %d", dao.Id)
		}

		period = dao.Config.ExpeditedPeriod
	}

	// funds move before any write, so a proposer who can't pay leaves nothing behind
	if err := k.chargeProposalCost(ctx, proposer, daoAddr, dao.Config); err != nil {
		return nil, err
	}

	deposit, err := k.escrowProposalDeposit(ctx, proposer, dao.Config)
	if err != nil {
		return nil, err
	}

	proposalId := k.GetProposalCounter(ctx) + 1
	proposal := types.Proposal{
		Id:                proposalId,
		DaoId:             dao.Id,
		Proposer:          proposer.String(),
		Title:             msg.Title,
		Summary:           msg.Summary,
		Msgs:              msg.Msgs,
		Expedited:         msg.Expedited,
		Status:            types.ProposalStatus_VOTING,
		SubmitHeight:      ctx.BlockHeight(),
		SubmitTime:        ctx.BlockTime().Unix(),
		VotingEnd:         ctx.BlockTime().Add(period).Unix(),
		PinnedConfig:      dao.Config,
		PinnedElectorate:  cloneElectorate(dao.Electorate),
		PinnedDenominator: denominator,
		PinnedDeposit:     deposit,
		Yes:               math.ZeroInt(),
		No:                math.ZeroInt(),
		Abstain:           math.ZeroInt(),
	}

	k.SetProposalCounter(ctx, proposalId)
	k.setNewProposal(ctx, proposal)
	// a source a proposal votes with stays pinned until the proposal leaves voting (I4)
	for _, source := range proposal.PinnedElectorate.Sources() {
		k.pinSource(ctx, source)
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.EventProposalSubmitted{
		Id:          proposal.Id,
		DaoId:       proposal.DaoId,
		Proposer:    proposal.Proposer,
		Expedited:   proposal.Expedited,
		VotingEnd:   proposal.VotingEnd,
		Denominator: denominator.String(),
	}); err != nil {
		return nil, err
	}

	return &types.MsgSubmitProposalResponse{ProposalId: proposalId}, nil
}

// checkProposer allows the DAO account itself (a proposal or the admin acting
// as the DAO), anyone when the DAO has open submission, and otherwise only its
// voters: a member of a static list, or a staker with eligible stake now (D22).
func (k Keeper) checkProposer(ctx sdk.Context, dao types.Dao, proposer sdk.AccAddress) error {
	if proposer.String() == dao.Address || dao.Config.OpenSubmission {
		return nil
	}

	switch dao.Type {
	case types.DaoType_DAO_TYPE_STATIC_LIST:
		if slices.Contains(dao.Members, proposer.String()) {
			return nil
		}
	case types.DaoType_DAO_TYPE_STAKE:
		if weight, _ := k.eligibleWeight(ctx, dao.Electorate, proposer, ctx.BlockHeight()); weight.IsPositive() {
			return nil
		}
	}

	return errorsmod.Wrapf(types.ErrNotProposer, "%s is not a voter of dao %d", proposer, dao.Id)
}

// votingDenominator is the DAO's total voting power, pinned by a new
// proposal: the member count of a static list, or the stake of the electorate.
func (k Keeper) votingDenominator(ctx sdk.Context, dao types.Dao) math.Int {
	if dao.Type == types.DaoType_DAO_TYPE_STATIC_LIST {
		return math.NewInt(int64(len(dao.Members)))
	}

	if dao.Electorate.IsEmpty() {
		return math.ZeroInt()
	}

	return k.electorateStake(ctx, dao.Electorate)
}

// validateProposalContent checks the title, summary and messages against the params.
func validateProposalContent(msg *types.MsgSubmitProposal, params types.Params) error {
	if uint64(len(msg.Title)) > params.MaxTitleLen {
		return errorsmod.Wrapf(types.ErrInvalidProposal, "title must have at most %d bytes, got %d", params.MaxTitleLen, len(msg.Title))
	}

	if uint64(len(msg.Summary)) > params.MaxSummaryLen {
		return errorsmod.Wrapf(types.ErrInvalidProposal, "summary must have at most %d bytes, got %d", params.MaxSummaryLen, len(msg.Summary))
	}

	if uint64(len(msg.Msgs)) > params.MaxMsgsPerProposal {
		return errorsmod.Wrapf(types.ErrInvalidMsgs, "at most %d messages, got %d", params.MaxMsgsPerProposal, len(msg.Msgs))
	}

	size := 0
	for _, m := range msg.Msgs {
		size += m.Size()
	}
	if uint64(size) > params.MaxProposalBytes {
		return errorsmod.Wrapf(types.ErrInvalidMsgs, "messages take %d bytes, at most %d allowed", size, params.MaxProposalBytes)
	}

	return nil
}

// chargeProposalCost sends the proposal cost to the burner or to the DAO's
// treasury. An absent or zero cost is not charged, nor is a cost the DAO would
// pay to itself.
func (k Keeper) chargeProposalCost(ctx sdk.Context, proposer, daoAddr sdk.AccAddress, config types.DaoConfig) error {
	cost := config.ProposalCost
	if cost == nil || !cost.IsPositive() {
		return nil
	}

	var err error
	switch config.CostDestination {
	case types.FundsDestination_TREASURY:
		if proposer.Equals(daoAddr) {
			return nil
		}

		err = k.bankKeeper.SendCoins(ctx, proposer, daoAddr, sdk.NewCoins(*cost))
	default:
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, txfeecollectortypes.BurnerFeeCollector, sdk.NewCoins(*cost))
	}

	if err != nil {
		return errorsmod.Wrap(err, "failed to pay the proposal cost")
	}

	return nil
}

// escrowProposalDeposit locks the proposal deposit in the module account and
// returns it for the proposal to pin; nil when the DAO asks for none.
func (k Keeper) escrowProposalDeposit(ctx sdk.Context, proposer sdk.AccAddress, config types.DaoConfig) (*sdk.Coin, error) {
	deposit := config.ProposalDeposit
	if deposit == nil || !deposit.IsPositive() {
		return nil, nil
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, types.ModuleName, sdk.NewCoins(*deposit)); err != nil {
		return nil, errorsmod.Wrap(err, "failed to escrow the proposal deposit")
	}

	pinned := *deposit

	return &pinned, nil
}

// cloneElectorate copies an electorate, so a pinned snapshot shares no slice
// with the DAO record. An empty electorate is pinned as nil.
func cloneElectorate(electorate *types.StakeElectorate) *types.StakeElectorate {
	if electorate.IsEmpty() {
		return nil
	}

	return &types.StakeElectorate{
		SrIds:   slices.Clone(electorate.SrIds),
		DrDenom: electorate.DrDenom,
	}
}
