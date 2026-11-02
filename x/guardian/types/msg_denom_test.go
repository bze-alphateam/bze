package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

var denomCases = []struct {
	name  string
	denom string
	valid bool
}{
	{"native", "ubze", true},
	{"ibc hash", "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", true},
	{"factory", "factory/bze1abc/utoken", true},
	{"empty", "", false},
	{"space", "u bze", false},
	{"too short", "ab", false},
	{"too long", strings.Repeat("a", 129), false},
	{"leading digit", "1ubze", false},
}

func TestMsgBlockDenom_ValidateBasic(t *testing.T) {
	signer := sdk.AccAddress("signer______________").String()

	require.Error(t, (&types.MsgBlockDenom{Signer: "nope", Denom: "ubze"}).ValidateBasic())
	require.Error(t, (&types.MsgBlockDenom{Signer: "", Denom: "ubze"}).ValidateBasic())

	for _, c := range denomCases {
		t.Run(c.name, func(t *testing.T) {
			err := (&types.MsgBlockDenom{Signer: signer, Denom: c.denom}).ValidateBasic()
			if c.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidDenom)
		})
	}
}

func TestMsgUnblockDenom_ValidateBasic(t *testing.T) {
	authority := sdk.AccAddress("authority___________").String()

	require.Error(t, (&types.MsgUnblockDenom{Authority: "nope", Denom: "ubze"}).ValidateBasic())
	require.Error(t, (&types.MsgUnblockDenom{Authority: "", Denom: "ubze"}).ValidateBasic())

	for _, c := range denomCases {
		t.Run(c.name, func(t *testing.T) {
			err := (&types.MsgUnblockDenom{Authority: authority, Denom: c.denom}).ValidateBasic()
			if c.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidDenom)
		})
	}
}
