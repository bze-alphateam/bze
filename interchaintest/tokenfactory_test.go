package interchaintest_test

import (
	"context"
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	tfAdminFunds = 30_000_000_000 // 30,000 BZE: one denom
	tfUserFunds  = 1_000_000_000  // 1,000 BZE for gas

	tfMint      = 1_000_000
	tfBurn      = 250_000
	tfSend      = 100_000
	tfOtherMint = 5_000

	tfEventPrefix = "bze.tokenfactory."
)

// TestTokenFactory drives a factory denom through its admin's whole life on a
// real network: create, mint, burn, metadata, on-chain branding, an admin
// handover and back, then the renounce that freezes the denom for everyone.
// Subtests run in order and share state.
func TestTokenFactory(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	t.Parallel()
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupTokenFactory(t, ctx, net.Chain)

	t.Run("1 create, mint, burn and set metadata", s.createAndUse)
	t.Run("2 branding set, clear and refusals", s.branding)
	t.Run("3 change admin and back", s.changeAdmin)
	t.Run("4 renounce freezes the denom", s.renounce)
}

type tokenFactorySuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	admin, other ibc.Wallet
	denom        string
}

func setupTokenFactory(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *tokenFactorySuite {
	s := &tokenFactorySuite{ctx: ctx, bze: bze}
	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "tfadmin", sdkmath.NewInt(tfAdminFunds), bze)[0]
	s.other = interchaintest.GetAndFundTestUsers(t, ctx, "tfother", sdkmath.NewInt(tfUserFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))
	s.denom = "factory/" + s.admin.FormattedAddress() + "/tok"
	return s
}

func (s *tokenFactorySuite) createAndUse(t *testing.T) {
	admin := s.admin.FormattedAddress()
	fee := s.createDenomFee(t)
	ubzeBefore := s.balance(t, admin, chain.Denom)

	res := s.mustExec(t, s.admin, "tokenfactory", "create-denom", "tok")
	require.Equal(t, ubzeBefore.Sub(fee.Amount).Sub(s.txFee(t, res)).String(), s.balance(t, admin, chain.Denom).String(),
		"admin must pay exactly the create fee and the tx fee")
	require.Equal(t, admin, s.authority(t))
	require.Equal(t, "0", s.supply(t).String())

	s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(tfMint, s.denom))
	require.Equal(t, sdkmath.NewInt(tfMint).String(), s.supply(t).String())
	require.Equal(t, sdkmath.NewInt(tfMint).String(), s.balance(t, admin, s.denom).String())

	s.mustExec(t, s.admin, "tokenfactory", "burn", coinStr(tfBurn, s.denom))
	require.Equal(t, sdkmath.NewInt(tfMint-tfBurn).String(), s.supply(t).String())
	require.Equal(t, sdkmath.NewInt(tfMint-tfBurn).String(), s.balance(t, admin, s.denom).String())

	res = s.mustExecMsg(t, s.admin, s.metadataMsg(admin, "Token"))
	s.requireTxEvent(t, res, "DenomMetadataChangeEvent", map[string]string{"denom": s.denom})
	md := s.metadata(t)
	require.Equal(t, "Token", md.Name)
	require.Equal(t, "TOK", md.Symbol)
	require.Equal(t, "TOK", md.Display)
}

func (s *tokenFactorySuite) branding(t *testing.T) {
	admin := s.admin.FormattedAddress()
	want := tfBranding("inter")

	res := s.mustExecMsg(t, s.admin, s.brandingMsg(admin, want))
	s.requireTxEvent(t, res, "DenomBrandingChangeEvent", map[string]string{"denom": s.denom})
	got, found := s.queryBranding(t)
	require.True(t, found, "the branding must be stored")
	require.Equal(t, want, got)

	// another account may not touch it
	s.requireRefused(t, "unauthorized", s.other, s.brandingMsg(s.other.FormattedAddress(), tfBranding("other")))
	// the font is a slug: lowercase letters, digits and dashes
	s.requireRefused(t, "invalid font", s.admin, s.brandingMsg(admin, tfBranding("Inter Bold")))

	// an empty branding clears it
	res = s.mustExecMsg(t, s.admin, s.brandingMsg(admin, nil))
	s.requireTxEvent(t, res, "DenomBrandingChangeEvent", map[string]string{"denom": s.denom})
	_, found = s.queryBranding(t)
	require.False(t, found, "an empty branding must clear the stored one")
	s.requireRefused(t, "unauthorized", s.other, s.brandingMsg(s.other.FormattedAddress(), nil))
}

func (s *tokenFactorySuite) changeAdmin(t *testing.T) {
	admin, other := s.admin.FormattedAddress(), s.other.FormattedAddress()
	s.requireRefused(t, "unauthorized", s.other, s.changeAdminMsg(other, admin))

	res := s.mustExec(t, s.admin, "tokenfactory", "change-admin", s.denom, other)
	s.requireAdminChange(t, res, admin, other)
	require.Equal(t, other, s.authority(t))

	s.requireRefused(t, "unauthorized", s.admin, mintMsg(admin, coinStr(1, s.denom)))
	s.mustExec(t, s.other, "tokenfactory", "mint", coinStr(tfOtherMint, s.denom))
	require.Equal(t, sdkmath.NewInt(tfOtherMint).String(), s.balance(t, other, s.denom).String())
	require.Equal(t, sdkmath.NewInt(tfMint-tfBurn+tfOtherMint).String(), s.supply(t).String())

	res = s.mustExec(t, s.other, "tokenfactory", "change-admin", s.denom, admin)
	s.requireAdminChange(t, res, other, admin)
	require.Equal(t, admin, s.authority(t))
	s.requireRefused(t, "unauthorized", s.other, mintMsg(other, coinStr(1, s.denom)))
}

func (s *tokenFactorySuite) renounce(t *testing.T) {
	admin, other := s.admin.FormattedAddress(), s.other.FormattedAddress()
	// brand it first, so the renounce must leave a branding nobody can change
	s.mustExecMsg(t, s.admin, s.brandingMsg(admin, tfBranding("inter")))

	res := s.mustExecMsg(t, s.admin, s.changeAdminMsg(admin, ""))
	s.requireAdminChange(t, res, admin, "")
	require.Empty(t, s.authority(t), "a renounced denom has no admin")

	for _, pair := range [][2]ibc.Wallet{{s.admin, s.other}, {s.other, s.admin}} {
		user, addr, peer := pair[0], pair[0].FormattedAddress(), pair[1].FormattedAddress()
		for name, msg := range map[string]any{
			"mint":           mintMsg(addr, coinStr(1, s.denom)),
			"burn":           burnMsg(addr, coinStr(1, s.denom)),
			"set metadata":   s.metadataMsg(addr, "Renamed"),
			"set branding":   s.brandingMsg(addr, tfBranding("other")),
			"clear branding": s.brandingMsg(addr, nil),
			"change admin":   s.changeAdminMsg(addr, peer),
		} {
			t.Run(user.KeyName()+" "+name, func(t *testing.T) {
				s.requireRefused(t, "unauthorized", user, msg)
			})
		}
	}
	require.Equal(t, "Token", s.metadata(t).Name)
	got, found := s.queryBranding(t)
	require.True(t, found)
	require.Equal(t, tfBranding("inter"), got)

	// the coin itself still moves
	before := s.balance(t, other, s.denom)
	s.mustExec(t, s.admin, "bank", "send", admin, other, coinStr(tfSend, s.denom))
	require.Equal(t, before.AddRaw(tfSend).String(), s.balance(t, other, s.denom).String())
	require.Equal(t, sdkmath.NewInt(tfMint-tfBurn+tfOtherMint).String(), s.supply(t).String())
}

func (s *tokenFactorySuite) requireAdminChange(t *testing.T, res helpers.TxResult, from, to string) {
	t.Helper()
	ev := s.requireTxEvent(t, res, "DenomAdminChangeEvent", map[string]string{"denom": s.denom, "admin": from})
	require.Equal(t, to, ev.Attributes["new_admin"])
}

// tfBranding is a complete branding package with the given font; nil fields
// are never used, an empty branding is passed as nil.
func tfBranding(font string) *brandingInfo {
	return &brandingInfo{
		Font:  font,
		Light: brandingColors{Background: "#FFFFFF", Text: "#111111", Primary: "#FF8800", Secondary: "#0088FF"},
		Dark:  brandingColors{Background: "#000000", Text: "#EEEEEE", Primary: "#FFAA33", Secondary: "#33AAFF"},
	}
}

type brandingColors struct {
	Background string `json:"background"`
	Text       string `json:"text"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
}

type brandingInfo struct {
	Font  string         `json:"font"`
	Light brandingColors `json:"light"`
	Dark  brandingColors `json:"dark"`
}

func (s *tokenFactorySuite) brandingMsg(creator string, b *brandingInfo) map[string]any {
	msg := map[string]any{"@type": "/bze.tokenfactory.MsgSetDenomBranding", "creator": creator, "denom": s.denom}
	if b != nil {
		msg["branding"] = b
	}
	return msg
}

func (s *tokenFactorySuite) changeAdminMsg(creator, newAdmin string) map[string]any {
	return map[string]any{"@type": "/bze.tokenfactory.MsgChangeAdmin", "creator": creator, "denom": s.denom, "newAdmin": newAdmin}
}

func (s *tokenFactorySuite) metadataMsg(creator, name string) map[string]any {
	return map[string]any{
		"@type":   "/bze.tokenfactory.MsgSetDenomMetadata",
		"creator": creator,
		"metadata": map[string]any{
			"description": name + " for the e2e suite",
			"denom_units": []map[string]any{
				{"denom": s.denom, "exponent": 0},
				{"denom": "TOK", "exponent": 6},
			},
			"base":    s.denom,
			"display": "TOK",
			"name":    name,
			"symbol":  "TOK",
		},
	}
}

func mintMsg(creator, coins string) map[string]any {
	return map[string]any{"@type": "/bze.tokenfactory.MsgMint", "creator": creator, "coins": coins}
}

func burnMsg(creator, coins string) map[string]any {
	return map[string]any{"@type": "/bze.tokenfactory.MsgBurn", "creator": creator, "coins": coins}
}

func (s *tokenFactorySuite) mustExec(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTx(s.ctx, s.bze, user, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

func (s *tokenFactorySuite) mustExecMsg(t *testing.T, user ibc.Wallet, msg any) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, nil, msg)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", msg, res.RawLog)
	return res
}

// requireRefused asserts the message is rejected with reason and that the
// denom's admin, supply, metadata and branding did not change.
func (s *tokenFactorySuite) requireRefused(t *testing.T, reason string, user ibc.Wallet, msg any) {
	t.Helper()
	before := s.denomState(t)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, nil, msg)
	why, err := helpers.RejectionReason(res, err)
	require.NoError(t, err, "msg %v", msg)
	require.NotZero(t, res.Code)
	require.Contains(t, why, reason)
	require.Equal(t, before, s.denomState(t), "a refused tx must not change the denom")
}

type tfDenomState struct {
	Admin, Supply string
	Metadata      metadataInfo
	Branding      *brandingInfo
}

func (s *tokenFactorySuite) denomState(t *testing.T) tfDenomState {
	t.Helper()
	st := tfDenomState{Admin: s.authority(t), Supply: s.supply(t).String(), Metadata: s.metadata(t)}
	if b, found := s.queryBranding(t); found {
		st.Branding = b
	}
	return st
}

func (s *tokenFactorySuite) requireTxEvent(t *testing.T, res helpers.TxResult, name string, attrs map[string]string) helpers.Event {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, tfEventPrefix+name), attrs)
	require.Len(t, events, 1, "want one %s with %v in tx %s, got %+v", name, attrs, res.TxHash, helpers.TxEvents(res, tfEventPrefix+name))
	return events[0]
}

func (s *tokenFactorySuite) authority(t *testing.T) string {
	t.Helper()
	res, err := helpers.Query[struct {
		DenomAuthority struct {
			Admin string `json:"admin"`
		} `json:"denomAuthority"`
	}](s.ctx, s.bze, "tokenfactory", "denom-authority", s.denom)
	require.NoError(t, err)
	return res.DenomAuthority.Admin
}

// queryBranding returns the stored branding; the query answers NotFound when
// there is none.
func (s *tokenFactorySuite) queryBranding(t *testing.T) (*brandingInfo, bool) {
	t.Helper()
	all, err := helpers.Query[struct {
		DenomBrandings []struct {
			Denom    string        `json:"denom"`
			Branding *brandingInfo `json:"branding"`
		} `json:"denom_brandings"`
	}](s.ctx, s.bze, "tokenfactory", "all-denom-branding")
	require.NoError(t, err)
	var listed *brandingInfo
	for _, r := range all.DenomBrandings {
		if r.Denom == s.denom {
			listed = r.Branding
		}
	}

	one, err := helpers.Query[struct {
		Branding *brandingInfo `json:"branding"`
	}](s.ctx, s.bze, "tokenfactory", "denom-branding", s.denom)
	if err != nil {
		require.Contains(t, err.Error(), "not found")
		require.Nil(t, listed, "the list has a branding the single query does not")
		return nil, false
	}
	require.NotNil(t, one.Branding)
	require.Equal(t, listed, one.Branding, "the list and the single query disagree")
	return one.Branding, true
}

type metadataInfo struct {
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Display string `json:"display"`
	Base    string `json:"base"`
}

func (s *tokenFactorySuite) metadata(t *testing.T) metadataInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		Metadata metadataInfo `json:"metadata"`
	}](s.ctx, s.bze, "bank", "denom-metadata", s.denom)
	require.NoError(t, err)
	require.Equal(t, s.denom, res.Metadata.Base)
	return res.Metadata
}

func (s *tokenFactorySuite) supply(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Amount struct {
			Amount string `json:"amount"`
		} `json:"amount"`
	}](s.ctx, s.bze, "bank", "total-supply-of", s.denom)
	require.NoError(t, err)
	return parseInt(res.Amount.Amount)
}

func (s *tokenFactorySuite) balance(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

func (s *tokenFactorySuite) txFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	fee, err := helpers.TxFee(res, chain.Denom)
	require.NoError(t, err)
	return fee
}

// createDenomFee reads the create_denom_fee param, in either JSON spelling.
func (s *tokenFactorySuite) createDenomFee(t *testing.T) sdk.Coin {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "tokenfactory", "params")
	require.NoError(t, err)
	raw, ok := res.Params["createDenomFee"]
	if !ok {
		raw, ok = res.Params["create_denom_fee"]
	}
	require.True(t, ok, "param create_denom_fee missing from %v", res.Params)
	var fee sdk.Coin
	require.NoError(t, json.Unmarshal(raw, &fee))
	require.Equal(t, chain.Denom, fee.Denom, "create_denom_fee is not in %s", chain.Denom)
	return fee
}
