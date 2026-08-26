package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/burner/types"
)

// TestRegisterInterfaces_ResolvesEveryMsg pins that every burner Msg resolves by type URL.
// MsgMoveIbcLockedCoins gained an amino.name annotation for v8.2.0 (BZE-35) so it can be
// signed with amino JSON; that signing path relies on the message being registered here (the
// annotation lives in the proto and is resolved through the interface registry, as the burner
// module registers no legacy amino codec).
func TestRegisterInterfaces_ResolvesEveryMsg(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msgs := []sdk.Msg{
		&types.MsgUpdateParams{},
		&types.MsgFundBurner{},
		&types.MsgStartRaffle{},
		&types.MsgJoinRaffle{},
		&types.MsgMoveIbcLockedCoins{},
	}

	for _, msg := range msgs {
		url := sdk.MsgTypeURL(msg)
		resolved, err := registry.Resolve(url)
		require.NoErrorf(t, err, "message %s not registered in the interface registry", url)
		require.NotNil(t, resolved, "resolved nil for %s", url)
	}
}
