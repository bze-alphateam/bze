package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// CreateDao implements the MsgCreateDao RPC.
//
// Order of operations:
//  1. Stateless validation (ValidateBasic).
//  2. Parse creator + decide admin (defaults to creator).
//  3. Validate parent_dao_id (exists + no cycle).
//  4. Pay creation fee (route per Params).
//  5. Allocate id, register BaseAccount at deterministic address.
//  6. Persist DAO + secondary indices.
//  7. Emit event.
func (k msgServer) CreateDao(goCtx context.Context, msg *types.MsgCreateDao) (*types.MsgCreateDaoResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, err.Error())
	}

	// Resolve admin. Empty `admin` field defaults to creator.
	adminStr := msg.Admin
	if adminStr == "" {
		adminStr = msg.Creator
	}

	// Validate parent linkage. Cycle check uses the not-yet-allocated id; the
	// counter is only consumed below, so peekNextDaoID is the right input.
	if msg.ParentDaoId != 0 {
		if _, found := k.GetDao(ctx, msg.ParentDaoId); !found {
			return nil, errorsmod.Wrapf(types.ErrParentNotFound, "parent_dao_id=%d", msg.ParentDaoId)
		}
		nextID := k.peekNextDaoID(ctx)
		if k.hasParentCycle(ctx, nextID, msg.ParentDaoId) {
			return nil, errorsmod.Wrapf(types.ErrParentCycle, "parent_dao_id=%d", msg.ParentDaoId)
		}
	}

	// Fee routing. Skip entirely when fee amount is zero. We pay the fee
	// BEFORE allocating the id so a failure leaves no orphan state.
	params := k.GetParams(ctx)
	if !params.DaoCreationFee.IsZero() {
		if err := k.payCreationFee(ctx, creator, params); err != nil {
			return nil, err
		}
	}

	// Allocate id, derive deterministic address.
	id := k.consumeNextDaoID(ctx)
	addr := types.DaoAccountAddress(id)

	// DAO addresses are deterministic and someone could have pre-funded the
	// next DAO address (via bank send), which auto-creates a BaseAccount in
	// x/auth. If the account already exists at the derived address we reuse
	// it — the DAO claims it as its treasury, including any pre-existing
	// balance. Overwriting the account would clobber its account_number and
	// potentially break account-history tracking; doing nothing is correct
	// because no one holds the private key for an address.Module-derived
	// address anyway.
	if !k.accountKeeper.HasAccount(ctx, addr) {
		acc := k.accountKeeper.NewAccountWithAddress(ctx, addr)
		k.accountKeeper.SetAccount(ctx, acc)
	}

	// Resolve voting backend. ValidateBasic already enforced Static-only and
	// validated the member list; we do the state write here.
	staticCfg := msg.GetStatic()
	if staticCfg == nil {
		// Should be impossible given ValidateBasic, but defensive.
		return nil, errorsmod.Wrap(types.ErrMissingVotingConfig, "static voting config is required")
	}
	if _, err := k.applyStaticMembersInit(ctx, id, staticCfg.Members); err != nil {
		return nil, err
	}

	// Persist DAO record + secondary indices.
	dao := types.Dao{
		Id:             id,
		Metadata:       msg.Metadata,
		Creator:        msg.Creator,
		AccountAddress: addr.String(),
		Admin:          adminStr,
		ParentDaoId:    msg.ParentDaoId,
		CreatedAtBlock: ctx.BlockHeight(),
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     msg.Governance,
		Deposit:        msg.Deposit,
	}
	// Epic 3: chain-state-dependent governance checks (Param ceiling on
	// voting_period; flash-vote lock rule for REWARD_STAKED, which is
	// unreachable here because MsgCreateDao rejects REWARD_STAKED — but the
	// hook is shaped so MsgUpdateVotingBackend can reuse it in Epic 5).
	if err := k.validateGovernanceAgainstChainState(ctx, dao, msg.Governance); err != nil {
		return nil, err
	}
	// Epic 4: deposit_period bounded by Params.max_deposit_period; the
	// stateless caps already ran in ValidateBasic.
	if err := types.ValidateDepositConfigAgainstParams(msg.Deposit, params.MaxDepositPeriod); err != nil {
		return nil, err
	}
	k.SetDao(ctx, dao)
	if err := k.SetDaoIndices(ctx, dao); err != nil {
		// Indices failing implies bech32 in `dao` is malformed; we already
		// validated all inputs, so this is a programmer error.
		return nil, fmt.Errorf("set dao indices: %w", err)
	}

	// Emit event.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateDao,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", id)),
		sdk.NewAttribute(types.AttributeKeyAccountAddress, addr.String()),
		sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
		sdk.NewAttribute(types.AttributeKeyAdmin, adminStr),
		sdk.NewAttribute(types.AttributeKeyParentDaoID, fmt.Sprintf("%d", msg.ParentDaoId)),
		sdk.NewAttribute(types.AttributeKeyVotingBackend, dao.VotingBackend.String()),
	))

	return &types.MsgCreateDaoResponse{
		DaoId:          id,
		AccountAddress: addr.String(),
	}, nil
}
