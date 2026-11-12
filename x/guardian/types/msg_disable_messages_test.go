package types_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestMsgDisableMessages_ValidateBasic(t *testing.T) {
	signer := sdk.AccAddress("signer______________").String()

	many := func(n int) []string {
		urls := make([]string, n)
		for i := range urls {
			urls[i] = fmt.Sprintf("/bze.tradebin.Msg%d", i)
		}
		return urls
	}

	for _, tc := range []struct {
		name   string
		signer string
		urls   []string
		err    error
	}{
		{"one", signer, []string{"/bze.tradebin.MsgMultiSwap"}, nil},
		{"several", signer, []string{"/bze.tradebin.MsgMultiSwap", "/cosmos.bank.v1beta1.MsgSend"}, nil},
		{"the maximum", signer, many(types.MaxDisableMessages), nil},
		{"over the maximum", signer, many(types.MaxDisableMessages + 1), types.ErrInvalidMsgTypeURLs},
		{"none", signer, nil, types.ErrInvalidMsgTypeURLs},
		{"empty url", signer, []string{""}, types.ErrInvalidMsgTypeURLs},
		{"slash only", signer, []string{"/"}, types.ErrInvalidMsgTypeURLs},
		{"no leading slash", signer, []string{"bze.tradebin.MsgMultiSwap"}, types.ErrInvalidMsgTypeURLs},
		{"duplicate", signer, []string{"/bze.tradebin.MsgMultiSwap", "/bze.tradebin.MsgMultiSwap"}, types.ErrInvalidMsgTypeURLs},
		{"protected after a valid one", signer, []string{"/bze.tradebin.MsgMultiSwap", "/cosmos.gov.v1.MsgVote"}, types.ErrProtectedMsgType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (&types.MsgDisableMessages{Signer: tc.signer, MsgTypeUrls: tc.urls}).ValidateBasic()
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.err)
		})
	}

	require.Error(t, (&types.MsgDisableMessages{Signer: "bad", MsgTypeUrls: []string{"/bze.tradebin.MsgMultiSwap"}}).ValidateBasic())
}

func TestIsProtectedMsgTypeURL(t *testing.T) {
	for _, url := range []string{
		"/cosmos.gov.v1.MsgSubmitProposal",
		"/cosmos.gov.v1.MsgVote",
		"/cosmos.gov.v1.MsgExecLegacyContent",
		"/cosmos.gov.v1beta1.MsgVote",
		"/cosmos.circuit.v1.MsgResetCircuitBreaker",
		"/cosmos.circuit.v1.MsgTripCircuitBreaker",
		"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
		"/cosmos.upgrade.v1beta1.MsgCancelUpgrade",
		"/bze.guardian.v1.MsgDisableMessages",
		"/bze.guardian.v1.MsgUpdateParams",
		"/bze.dao.MsgSubmitProposal",
		"/bze.dao.MsgVote",
		"/bze.dao.MsgSettleProposal",
	} {
		require.True(t, types.IsProtectedMsgTypeURL(url), url)
		err := (&types.MsgDisableMessages{Signer: sdk.AccAddress("signer______________").String(), MsgTypeUrls: []string{url}}).ValidateBasic()
		require.ErrorIs(t, err, types.ErrProtectedMsgType, url)
	}

	for _, url := range []string{
		"/bze.tradebin.MsgMultiSwap",
		"/bze.dao.MsgCreateDao",
		"/bze.dao.MsgCancelProposal",
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.authz.v1beta1.MsgExec",
		"/cosmos.governance.MsgX", // only the exact package prefix is protected
		"/bze.guardianx.MsgX",
	} {
		require.False(t, types.IsProtectedMsgTypeURL(url), url)
	}
}
