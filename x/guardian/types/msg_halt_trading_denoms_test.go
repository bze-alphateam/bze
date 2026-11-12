package types_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
		err    error // nil: valid
		reason string
	}{
		{"one denom", signer, []string{"factory/bze1abc/utoken"}, nil, ""},
		{"ibc denom", signer, []string{"ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"}, nil, ""},
		{"several", signer, []string{"factory/bze1abc/utoken", "uother"}, nil, ""},
		{"max", signer, denomList(tradebintypes.MaxDenomsPerHaltMsg), nil, ""},
		// the native denom check needs the tradebin params: done by the keeper
		{"native passes the stateless check", signer, []string{"ubze"}, nil, ""},
		{"bad signer", "nope", []string{"utoken"}, sdkerrors.ErrInvalidAddress, "invalid signer address"},
		{"empty signer", "", []string{"utoken"}, sdkerrors.ErrInvalidAddress, "invalid signer address"},
		{"no denoms", signer, nil, types.ErrInvalidDenom, "no denoms provided"},
		{"empty list", signer, []string{}, types.ErrInvalidDenom, "no denoms provided"},
		{"invalid denom", signer, []string{"u token"}, types.ErrInvalidDenom, `invalid denom "u token"`},
		{"empty denom", signer, []string{""}, types.ErrInvalidDenom, `invalid denom ""`},
		{"duplicate", signer, []string{"utoken", "uother", "utoken"}, types.ErrInvalidDenom, "duplicate denom utoken"},
		{"too many", signer, denomList(tradebintypes.MaxDenomsPerHaltMsg + 1), types.ErrInvalidDenom, "at most 1000 denoms per message, got 1001"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := (&types.MsgHaltTradingDenoms{Signer: c.signer, Denoms: c.denoms}).ValidateBasic()
			if c.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, c.err)
			require.ErrorContains(t, err, c.reason)
		})
	}
}
