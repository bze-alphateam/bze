package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestMsgPauseAutomation_ValidateBasic(t *testing.T) {
	require.NoError(t, (&types.MsgPauseAutomation{Signer: sdk.AccAddress("signer______________").String()}).ValidateBasic())
	require.Error(t, (&types.MsgPauseAutomation{Signer: "nope"}).ValidateBasic())
	require.Error(t, (&types.MsgPauseAutomation{}).ValidateBasic())
}

func TestMsgResumeAutomation_ValidateBasic(t *testing.T) {
	require.NoError(t, (&types.MsgResumeAutomation{Authority: sdk.AccAddress("authority___________").String()}).ValidateBasic())
	require.Error(t, (&types.MsgResumeAutomation{Authority: "nope"}).ValidateBasic())
	require.Error(t, (&types.MsgResumeAutomation{}).ValidateBasic())
}
