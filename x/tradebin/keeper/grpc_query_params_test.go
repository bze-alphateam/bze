package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"go.uber.org/mock/gomock"
	"testing"

	testkeeper "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)

	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockBank)

	mockAccount := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAccount)

	k, ctx := testkeeper.TradebinKeeper(t, mockBank, mockDistr, mockAccount)

	wctx := sdk.WrapSDKContext(ctx)
	params := types.DefaultParams()
	k.SetParams(ctx, params)

	response, err := k.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
