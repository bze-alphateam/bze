package tradebin_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	tradebin "github.com/bze-alphateam/bze/x/tradebin/module"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const haltedDenomForWiring = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"

func govAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// TestHaltedDenoms_HaltAndUnhaltThroughRouter drives the two governance messages through the real msg
// service router the way a passed proposal is executed: the gov authority halts a denom, the keeper
// answers "halted" for it, and MsgUnhaltDenoms clears it again.
func TestHaltedDenoms_HaltAndUnhaltThroughRouter(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)

	require.Empty(t, f.k.GetAllHaltedDenoms(f.ctx), "nothing is halted at genesis")
	require.False(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))

	halt := types.NewMsgHaltDenoms(govAuthority(), []string{haltedDenomForWiring})
	handler := msr.Handler(halt)
	require.NotNil(t, handler, "MsgHaltDenoms must be routable after RegisterServices")
	_, err := handler(f.ctx, halt)
	require.NoError(t, err)

	require.True(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))
	require.Equal(t, []string{haltedDenomForWiring}, f.k.GetAllHaltedDenoms(f.ctx))

	unhalt := types.NewMsgUnhaltDenoms(govAuthority(), []string{haltedDenomForWiring})
	handler = msr.Handler(unhalt)
	require.NotNil(t, handler, "MsgUnhaltDenoms must be routable after RegisterServices")
	_, err = handler(f.ctx, unhalt)
	require.NoError(t, err)

	require.False(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))
	require.Empty(t, f.k.GetAllHaltedDenoms(f.ctx))
}

// A signer other than the gov module account can neither halt nor un-halt.
func TestHaltedDenoms_NonAuthorityRejectedThroughRouter(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)
	stranger := sdk.AccAddress("not-the-gov-account_").String()

	halt := types.NewMsgHaltDenoms(stranger, []string{haltedDenomForWiring})
	_, err := msr.Handler(halt)(f.ctx, halt)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
	require.False(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring), "a rejected proposal changes nothing")

	f.k.SetHaltedDenom(f.ctx, haltedDenomForWiring)
	unhalt := types.NewMsgUnhaltDenoms(stranger, []string{haltedDenomForWiring})
	_, err = msr.Handler(unhalt)(f.ctx, unhalt)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
	require.True(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))
}

// The native denom is refused by the router path too, and the whole message is rejected: the other
// denom it carried is not halted either.
func TestHaltedDenoms_NativeDenomRejectedThroughRouter(t *testing.T) {
	f := newWiringFixture(t)
	msr := f.registerServices(t)

	msg := types.NewMsgHaltDenoms(govAuthority(), []string{haltedDenomForWiring, f.k.GetParams(f.ctx).NativeDenom})
	_, err := msr.Handler(msg)(f.ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be halted")

	require.Empty(t, f.k.GetAllHaltedDenoms(f.ctx))
}

// The halted denoms survive a genesis export/import and validate on the way.
func TestHaltedDenoms_GenesisRoundTrip(t *testing.T) {
	f := newWiringFixture(t)

	genesis := types.GenesisState{Params: v2types.DefaultParams(), HaltedDenoms: []string{"uusdc", haltedDenomForWiring}}
	require.NoError(t, genesis.Validate())

	tradebin.InitGenesis(f.ctx, f.k, genesis)
	require.True(t, f.k.IsDenomHalted(f.ctx, haltedDenomForWiring))
	require.True(t, f.k.IsDenomHalted(f.ctx, "uusdc"))

	exported := tradebin.ExportGenesis(f.ctx, f.k)
	require.NotNil(t, exported)
	require.NoError(t, exported.Validate())
	require.Equal(t, []string{haltedDenomForWiring, "uusdc"}, exported.HaltedDenoms, "exported in store order")

	// genesis validation rejects what the messages would reject
	bad := types.GenesisState{Params: v2types.DefaultParams(), HaltedDenoms: []string{v2types.DefaultNativeDenom}}
	require.Error(t, bad.Validate())
}
