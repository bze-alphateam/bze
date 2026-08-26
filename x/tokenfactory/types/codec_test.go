package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// TestRegisterInterfaces_ResolvesEveryMsg guards against a tokenfactory Msg being omitted
// from RegisterInterfaces — specifically the v8.2.0 MsgSetDenomBranding, which shares a
// RegisterImplementations call with MsgSetDenomMetadata and would be un-routable if dropped.
func TestRegisterInterfaces_ResolvesEveryMsg(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msgs := []sdk.Msg{
		&types.MsgCreateDenom{},
		&types.MsgMint{},
		&types.MsgBurn{},
		&types.MsgChangeAdmin{},
		&types.MsgSetDenomMetadata{},
		&types.MsgUpdateParams{},
		// v8.2.0: on-chain denom branding
		&types.MsgSetDenomBranding{},
	}

	for _, msg := range msgs {
		url := sdk.MsgTypeURL(msg)
		resolved, err := registry.Resolve(url)
		require.NoErrorf(t, err, "message %s not registered in the interface registry", url)
		require.NotNil(t, resolved, "resolved nil for %s", url)
	}
}
