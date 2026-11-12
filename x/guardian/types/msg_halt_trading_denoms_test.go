package types_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

func denomList(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("factory/bze1abc/u%04d", i)
	}
	return out
}

func TestMsgHaltTradingDenoms_ValidateBasic(t *testing.T) {
	signer := sdk.AccAddress("signer______________").String()

	for _, c := range []struct {
		name   string
		signer string
		denoms []string
		valid  bool
	}{
		{"one denom", signer, []string{"factory/bze1abc/utoken"}, true},
		{"ibc denom", signer, []string{"ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"}, true},
		{"several", signer, []string{"factory/bze1abc/utoken", "uother"}, true},
		{"max", signer, denomList(tradebintypes.MaxDenomsPerHaltMsg), true},
		// the native denom check needs the tradebin params: done by the keeper
		{"native passes the stateless check", signer, []string{"ubze"}, true},
		{"bad signer", "nope", []string{"utoken"}, false},
		{"empty signer", "", []string{"utoken"}, false},
		{"no denoms", signer, nil, false},
		{"empty list", signer, []string{}, false},
		{"invalid denom", signer, []string{"u token"}, false},
		{"empty denom", signer, []string{""}, false},
		{"duplicate", signer, []string{"utoken", "uother", "utoken"}, false},
		{"too many", signer, denomList(tradebintypes.MaxDenomsPerHaltMsg + 1), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := (&types.MsgHaltTradingDenoms{Signer: c.signer, Denoms: c.denoms}).ValidateBasic()
			if c.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestMsgHaltTradingDenoms_DenomErrorsAreInvalidDenom(t *testing.T) {
	signer := sdk.AccAddress("signer______________").String()
	for _, denoms := range [][]string{nil, {"u token"}, {"utoken", "utoken"}, denomList(tradebintypes.MaxDenomsPerHaltMsg + 1)} {
		err := (&types.MsgHaltTradingDenoms{Signer: signer, Denoms: denoms}).ValidateBasic()
		require.ErrorIs(t, err, types.ErrInvalidDenom)
	}
}
