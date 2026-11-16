package tradebin_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// TestUpdatePools_ThroughRouter drives MsgUpdatePools through the real msg service router the way a
// passed proposal is executed: the gov authority changes a pool's fee and split, any other signer is
// refused.
func TestUpdatePools_ThroughRouter(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)

	dest := types.FeeDestination{
		Treasury:  math.LegacyMustNewDecFromStr("0.3"),
		Burner:    math.LegacyMustNewDecFromStr("0.3"),
		Providers: math.LegacyMustNewDecFromStr("0.4"),
	}
	f.k.SetLiquidityPool(f.ctx, types.LiquidityPool{
		Id:           "stake_ubze",
		Base:         "stake",
		Quote:        "ubze",
		LpDenom:      "ulp_stake_ubze",
		Fee:          math.LegacyMustNewDecFromStr("0.003"),
		FeeDest:      &dest,
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(1000),
	})

	update := types.PoolUpdate{
		PoolId: "stake_ubze",
		Fee:    math.LegacyMustNewDecFromStr("0.02"),
		FeeDest: types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyMustNewDecFromStr("0.5"),
			Providers: math.LegacyMustNewDecFromStr("0.5"),
		},
	}

	stranger := types.NewMsgUpdatePools(sdk.AccAddress("not-the-gov-account_").String(), []types.PoolUpdate{update})
	handler := msr.Handler(stranger)
	require.NotNil(t, handler, "MsgUpdatePools must be routable after RegisterServices")
	_, err := handler(f.ctx, stranger)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
	pool, _ := f.k.GetLiquidityPool(f.ctx, "stake_ubze")
	require.True(t, pool.Fee.Equal(math.LegacyMustNewDecFromStr("0.003")), "a rejected proposal changes nothing")

	msg := types.NewMsgUpdatePools(govAuthority(), []types.PoolUpdate{update})
	_, err = msr.Handler(msg)(f.ctx, msg)
	require.NoError(t, err)

	pool, _ = f.k.GetLiquidityPool(f.ctx, "stake_ubze")
	require.True(t, pool.Fee.Equal(update.Fee))
	require.True(t, pool.FeeDest.Burner.Equal(update.FeeDest.Burner))
	require.True(t, pool.FeeDest.Treasury.IsZero())
}
