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

	var path []string
	for {
		path = append(path, dao.Address)
		res := &types.QueryDaoControllerResponse{Path: path}

		if dao.Admin == dao.Address {
			res.Controller, res.EndReason = dao.Address, types.ControllerEndReason_SELF_OWNED
			return res, nil
		}

		adminId, isDao := k.GetDaoIdByAddress(goCtx, sdk.MustAccAddressFromBech32(dao.Admin))
		if !isDao {
			res.Controller, res.EndReason = dao.Admin, types.ControllerEndReason_ACCOUNT
			return res, nil
		}

		if slices.Contains(path, dao.Admin) {
			res.Controller, res.EndReason = dao.Admin, types.ControllerEndReason_CYCLE
			return res, nil
		}

		if len(path) == maxControllerDepth {
			res.Controller, res.EndReason = dao.Address, types.ControllerEndReason_DEPTH_CAP
			return res, nil
		}

		admin := dao.Admin
		dao, found = k.GetDao(goCtx, adminId)
		if !found {
			return nil, status.Errorf(codes.Internal, "dao %d of %s not found", adminId, admin)
		}
	}
}
