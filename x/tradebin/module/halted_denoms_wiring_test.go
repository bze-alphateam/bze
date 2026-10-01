package tradebin_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	tradebin "github.com/bze-alphateam/bze/x/tradebin/module"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const haltedDenomForWiring = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"

func govAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// TestHaltedDenoms_UpdateParamsThroughRouter_SetsAndClears drives MsgUpdateParams through the real
// msg service router the way a passed governance proposal is executed: the gov authority can list a
// denom, the keeper answers "halted" for it, and the same message without the denom clears it.
func TestHaltedDenoms_UpdateParamsThroughRouter_SetsAndClears(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)
	f.bank.EXPECT().HasSupply(gomock.Any(), v2types.DefaultNativeDenom).Return(true).AnyTimes()

	require.Empty(t, f.k.GetParams(f.ctx).HaltedDenoms, "the halt list ships empty")
	require.False(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))

	// halt
	params := f.k.GetParams(f.ctx)
	params.HaltedDenoms = []string{haltedDenomForWiring}
	msg := &types.MsgUpdateParams{Authority: govAuthority(), Params: params}
	handler := msr.Handler(msg)
	require.NotNil(t, handler, "MsgUpdateParams must be routable after RegisterServices")
	_, err := handler(f.ctx, msg)
	require.NoError(t, err)

	stored := f.k.GetParams(f.ctx)
	require.Equal(t, []string{haltedDenomForWiring}, stored.HaltedDenoms)
	require.Equal(t, params.MinNativeLiquidityForModuleSwap, stored.MinNativeLiquidityForModuleSwap, "the other params travel with the message")
	require.True(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))

	// un-halt: the same proposal without the denom
	params.HaltedDenoms = nil
	msg = &types.MsgUpdateParams{Authority: govAuthority(), Params: params}
	_, err = msr.Handler(msg)(f.ctx, msg)
	require.NoError(t, err)

	require.Empty(t, f.k.GetParams(f.ctx).HaltedDenoms)
	require.False(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))
}

// A signer other than the gov module account cannot halt a denom.
func TestHaltedDenoms_UpdateParamsThroughRouter_NonAuthorityRejected(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)

	params := f.k.GetParams(f.ctx)
	params.HaltedDenoms = []string{haltedDenomForWiring}
	msg := &types.MsgUpdateParams{Authority: sdk.AccAddress("not-the-gov-account_").String(), Params: params}

	_, err := msr.Handler(msg)(f.ctx, msg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrInvalidSigner)

	require.Empty(t, f.k.GetParams(f.ctx).HaltedDenoms, "a rejected proposal changes nothing")
	require.False(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))
}

// Params.Validate runs inside the MsgUpdateParams handler: a proposal listing the native denom is
// rejected by the router path, not only by a direct Validate call.
func TestHaltedDenoms_UpdateParamsThroughRouter_NativeDenomRejected(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)

	params := f.k.GetParams(f.ctx)
	params.HaltedDenoms = []string{params.NativeDenom}
	msg := &types.MsgUpdateParams{Authority: govAuthority(), Params: params}

	_, err := msr.Handler(msg)(f.ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be halted")

	require.Empty(t, f.k.GetParams(f.ctx).HaltedDenoms)
}

// The stored list survives a genesis export/import and validates on the way.
func TestHaltedDenoms_GenesisRoundTrip(t *testing.T) {
	f := newWiringFixture(t)

	params := v2types.DefaultParams()
	params.HaltedDenoms = []string{haltedDenomForWiring}
	genesis := types.GenesisState{Params: params}
	require.NoError(t, genesis.Validate())

	tradebin.InitGenesis(f.ctx, f.k, genesis)
	require.True(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))

	exported := tradebin.ExportGenesis(f.ctx, f.k)
	require.NotNil(t, exported)
	require.NoError(t, exported.Validate())
	require.Equal(t, []string{haltedDenomForWiring}, exported.Params.HaltedDenoms)

	// genesis validation rejects a list the params would reject
	bad := types.GenesisState{Params: v2types.DefaultParams()}
	bad.Params.HaltedDenoms = []string{v2types.DefaultNativeDenom}
	require.Error(t, bad.Validate())
}
