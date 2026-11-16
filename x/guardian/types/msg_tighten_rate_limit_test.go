package types_test

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

func validTighten() *types.MsgTightenRateLimit {
	return &types.MsgTightenRateLimit{
		Signer:         sdk.AccAddress("signer______________").String(),
		Denom:          "ubze",
		ChannelId:      "channel-0",
		MaxPercentSend: math.NewInt(2),
		MaxPercentRecv: math.NewInt(0),
	}
}

func TestMsgTightenRateLimit_ValidateBasic(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.MsgTightenRateLimit)
		valid  bool
	}{
		{"valid", func(*types.MsgTightenRateLimit) {}, true},
		{"0/0", func(m *types.MsgTightenRateLimit) {
			m.MaxPercentSend, m.MaxPercentRecv = math.ZeroInt(), math.ZeroInt()
		}, true},
		{"100/100", func(m *types.MsgTightenRateLimit) {
			m.MaxPercentSend, m.MaxPercentRecv = math.NewInt(100), math.NewInt(100)
		}, true},
		{"bad bech32", func(m *types.MsgTightenRateLimit) { m.Signer = "nope" }, false},
		{"empty signer", func(m *types.MsgTightenRateLimit) { m.Signer = "" }, false},
		{"empty denom", func(m *types.MsgTightenRateLimit) { m.Denom = "" }, false},
		{"bad channel", func(m *types.MsgTightenRateLimit) { m.ChannelId = "chan-0" }, false},
		{"empty channel", func(m *types.MsgTightenRateLimit) { m.ChannelId = "" }, false},
		{"send 101", func(m *types.MsgTightenRateLimit) { m.MaxPercentSend = math.NewInt(101) }, false},
		{"recv 101", func(m *types.MsgTightenRateLimit) { m.MaxPercentRecv = math.NewInt(101) }, false},
		{"send negative", func(m *types.MsgTightenRateLimit) { m.MaxPercentSend = math.NewInt(-1) }, false},
		{"recv negative", func(m *types.MsgTightenRateLimit) { m.MaxPercentRecv = math.NewInt(-1) }, false},
		{"send unset", func(m *types.MsgTightenRateLimit) { m.MaxPercentSend = math.Int{} }, false},
		{"recv unset", func(m *types.MsgTightenRateLimit) { m.MaxPercentRecv = math.Int{} }, false},
		{"huge percent", func(m *types.MsgTightenRateLimit) {
			m.MaxPercentSend, _ = math.NewIntFromString("1" + strings.Repeat("0", 40))
		}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := validTighten()
			tc.mutate(msg)

			err := msg.ValidateBasic()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	addr := sdk.AccAddress("authority____________").String()

	require.NoError(t, (&types.MsgUpdateParams{Authority: addr, Params: types.DefaultParams()}).ValidateBasic())
	require.NoError(t, (&types.MsgUpdateParams{Authority: addr, Params: types.NewParams(addr)}).ValidateBasic())
	require.Error(t, (&types.MsgUpdateParams{Authority: "nope", Params: types.DefaultParams()}).ValidateBasic())
	require.Error(t, (&types.MsgUpdateParams{Authority: addr, Params: types.NewParams("nope")}).ValidateBasic())
}
