package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

func (suite *IntegrationTestSuite) TestGetParams() {
	params := types.DefaultParams()
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.Require().EqualValues(params, suite.k.GetParams(suite.ctx))
}

// ---------- Pure types-level validation tests (no keeper) ----------
//
// Validation logic for Params lives entirely in types/params.go and has
// no keeper dependency, so we exercise it via plain Test* functions
// instead of dragging in the suite.

func TestDefaultParamsValidate(t *testing.T) {
	require.NoError(t, types.DefaultParams().Validate())
}

func TestParamsValidate_RejectsBadDestination(t *testing.T) {
	p := types.DefaultParams()
	p.DaoCreationFeeDestination = "not_a_real_destination"
	require.Error(t, p.Validate())
}

func TestParamsValidate_RejectsUnderFloorVotingPeriod(t *testing.T) {
	p := types.DefaultParams()
	p.MaxVotingPeriod = time.Minute // below 1h floor
	require.Error(t, p.Validate())
}

func TestParamsValidate_RejectsUnderFloorDepositPeriod(t *testing.T) {
	p := types.DefaultParams()
	p.MaxDepositPeriod = time.Hour // below 1d floor
	require.Error(t, p.Validate())
}

func TestParamsValidate_RejectsZeroMaxMsgs(t *testing.T) {
	p := types.DefaultParams()
	p.MaxMsgsPerProposal = 0
	require.Error(t, p.Validate())
}

func TestParamsValidate_NonZeroFeeAccepted(t *testing.T) {
	p := types.DefaultParams()
	p.DaoCreationFee = sdk.NewCoin("ubze", math.NewInt(1_000_000))
	require.NoError(t, p.Validate())
}
