package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestMsgPauseAutomation_ValidateBasic(t *testing.T) {
	require.NoError(t, (&types.MsgPauseAutomation{Signer: sdk.AccAddress("signer______________").String()}).ValidateBasic())
	require.ErrorContains(t, (&types.MsgPauseAutomation{Signer: "nope"}).ValidateBasic(), "invalid signer address")
	require.ErrorContains(t, (&types.MsgPauseAutomation{}).ValidateBasic(), "invalid signer address")
}

func TestMsgResumeAutomation_ValidateBasic(t *testing.T) {
	require.NoError(t, (&types.MsgResumeAutomation{Authority: sdk.AccAddress("authority___________").String()}).ValidateBasic())
	require.ErrorContains(t, (&types.MsgResumeAutomation{Authority: "nope"}).ValidateBasic(), "invalid authority address")
	require.ErrorContains(t, (&types.MsgResumeAutomation{}).ValidateBasic(), "invalid authority address")
}
