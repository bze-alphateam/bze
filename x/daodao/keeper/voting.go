package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// votingBackend is the internal abstraction over a DAO's voting-power
// source. Two implementations live side by side: staticVotingBackend and
// rewardStakedVotingBackend (Epic 2).
//
// Backends operate on a fully-loaded `types.Dao` (not just an id) so they
// can read backend-specific fields (`reward_id`, etc.) without re-reading
// the DAO record on every call. The keeper resolves the backend with
// `backendFor(dao)` and then calls into it.
//
// Method context is `context.Context` to match the keeper's modern API.
// Implementations that need `sdk.Context` (e.g. for the rewards keeper)
// unwrap internally.
type votingBackend interface {
	// Power returns the current voting power of `addr` within the DAO.
	// Returns 0, nil for addresses that are not members.
	Power(ctx context.Context, dao types.Dao, addr sdk.AccAddress) (uint64, error)

	// TotalPower returns the current total voting power across the DAO.
	TotalPower(ctx context.Context, dao types.Dao) (uint64, error)

	// SnapshotAll captures every member's current power into the
	// SnapshotPowerKey rows for (dao.id, snapshotID), and writes the
	// total to SnapshotTotalKey.
	//
	// Called by Epic 3's MsgCreateProposal. Defined here so Epic 2 sets
	// up the abstraction completely; concrete backends may stub it for
	// now if they need cross-module work that arrives in Epic 3 (see
	// rewardStakedVotingBackend.SnapshotAll).
	SnapshotAll(ctx context.Context, dao types.Dao, snapshotID uint64) error
}

// backendFor returns the votingBackend that should service a given DAO.
// Unknown / unset backends are a hard error — every DAO must have a valid
// backend after MsgCreateDao runs.
func (k Keeper) backendFor(dao types.Dao) (votingBackend, error) {
	switch dao.VotingBackend {
	case types.VotingBackendType_VOTING_BACKEND_STATIC:
		return staticVotingBackend{k: k}, nil
	case types.VotingBackendType_VOTING_BACKEND_REWARD_STAKED:
		return rewardStakedVotingBackend{k: k}, nil
	default:
		return nil, errorsmod.Wrapf(types.ErrMissingVotingConfig,
			"DAO %d has unknown voting_backend %v", dao.Id, dao.VotingBackend)
	}
}
