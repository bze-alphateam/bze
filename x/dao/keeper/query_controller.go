package keeper

import (
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// maxControllerDepth caps how many DAOs the admin chain walk visits.
const maxControllerDepth = 16

// DaoController follows the admin chain from a DAO to whoever controls it in
// the end. Pending nominations are ignored: only the current admin counts.
func (k Keeper) DaoController(goCtx context.Context, req *types.QueryDaoControllerRequest) (*types.QueryDaoControllerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	dao, found := k.GetDao(goCtx, req.DaoId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "dao %d not found", req.DaoId)
	}

	chain, controller, endReason, err := k.adminChain(goCtx, dao)
	if err != nil {
		return nil, err
	}

	path := make([]string, len(chain))
	for i, link := range chain {
		path[i] = link.Address
	}

	return &types.QueryDaoControllerResponse{Controller: controller, EndReason: endReason, Path: path}, nil
}

// adminChain follows the admin chain from dao, which comes first in the
// returned DAOs, through every admin that is a DAO. It stops at an account, a
// self-owned DAO, a DAO already visited, or maxControllerDepth DAOs, and says
// who controls the chain and why the walk ended.
func (k Keeper) adminChain(ctx context.Context, dao types.Dao) (chain []types.Dao, controller string, endReason types.ControllerEndReason, err error) {
	var path []string
	for {
		chain = append(chain, dao)
		path = append(path, dao.Address)

		if dao.Admin == dao.Address {
			return chain, dao.Address, types.ControllerEndReason_SELF_OWNED, nil
		}

		adminId, isDao := k.GetDaoIdByAddress(ctx, sdk.MustAccAddressFromBech32(dao.Admin))
		if !isDao {
			return chain, dao.Admin, types.ControllerEndReason_ACCOUNT, nil
		}

		if slices.Contains(path, dao.Admin) {
			return chain, dao.Admin, types.ControllerEndReason_CYCLE, nil
		}

		if len(path) == maxControllerDepth {
			return chain, dao.Address, types.ControllerEndReason_DEPTH_CAP, nil
		}

		admin := dao.Admin
		var found bool
		dao, found = k.GetDao(ctx, adminId)
		if !found {
			return nil, "", 0, status.Errorf(codes.Internal, "dao %d of %s not found", adminId, admin)
		}
	}
}
