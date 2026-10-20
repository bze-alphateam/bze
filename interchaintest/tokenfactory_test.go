package interchaintest_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
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

	// the payer creates the gas denom and its ubze pool, then pays a create
	// fee with a tx fee in gas
	tfPayerFunds = 300_000_000_000 // 300,000 BZE: two create fees, a pool fee and the pool's ubze side
	tfGasMint    = 1_000_000_000_000
	// the ubze side is twice MinNativeLiquidityForModuleSwap (1e11), so it
	// stays above it after the create fee is bought out of it
	tfPoolGas   = 200_000_000_000
	tfPoolBze   = 200_000_000_000
	tfPoolFee   = "0.002"
	tfPoolDest  = `{"treasury":"0","burner":"0","providers":"1"}`
	tfPoolTxGas = 500_000
	tfGasTxGas  = 1_000_000
	tfGasTxFee  = 20_000 // gas, twice the minimum at the pool's 1:1 price

	tfCodespace = "tokenfactory"
	tfUndefined = "undefined" // a plain Go error: codespace undefined, code 1
	tfSdk       = "sdk"
	tfMaxSubLen = 44
	tfMaxFont   = 32
)

// TestTokenFactory drives a factory denom through its admin's whole life on a
// real network: create, mint, burn, metadata, on-chain branding, an admin
// handover and back, then the renounce that freezes the denom for everyone,
// and the governance proposal that gives the renounced denom a new admin.
// Before the renounce, while the admin is live, a second account creates a
// denom and a ubze pool for it and pays a create fee through that pool, and
// the module's refusals are checked one by one. Subtests run in order and
// share state.
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
	t.Run("5 create fee to the community pool and default metadata", s.createFeeAndDefaultMetadata)
	t.Run("6 create fee paid with a non-native tx fee denom", s.createFeeNonNative)
	t.Run("7 create-denom refusals", s.createRefusals)
	t.Run("8 mint and burn refusals", s.mintBurnRefusals)
	t.Run("9 change-admin refusals", s.changeAdminRefusals)
	t.Run("10 metadata and branding refusals", s.metadataBrandingRefusals)
	t.Run("11 queries of unknown denoms", s.unknownDenomQueries)
	t.Run("4 renounce freezes the denom", s.renounce)
	t.Run("12 governance assigns an admin to the renounced denom", s.govSetAdmin)
}

type tokenFactorySuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	admin, other, payer ibc.Wallet
	denom               string
	// gas is the payer's denom, gasPool its ubze pool and lpDenom the pool's
	// LP denom; nope is a factory denom nobody created
	gas, gasPool, lpDenom, nope string
	tfModule, tradebin, cp      string
}

func setupTokenFactory(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *tokenFactorySuite {
	s := &tokenFactorySuite{ctx: ctx, bze: bze}
	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "tfadmin", sdkmath.NewInt(tfAdminFunds), bze)[0]
	s.other = interchaintest.GetAndFundTestUsers(t, ctx, "tfother", sdkmath.NewInt(tfUserFunds), bze)[0]
	s.payer = interchaintest.GetAndFundTestUsers(t, ctx, "tfpayer", sdkmath.NewInt(tfPayerFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))
	s.denom = "factory/" + s.admin.FormattedAddress() + "/tok"
	s.nope = "factory/" + s.admin.FormattedAddress() + "/nope"
	s.gas = "factory/" + s.payer.FormattedAddress() + "/gas"
	// pool ids order the denoms: "factory/…" sorts before "ubze"
	s.gasPool = s.gas + "_" + chain.Denom
	for name, addr := range map[string]*string{
		"tokenfactory":      &s.tfModule,
		"tradebin":          &s.tradebin,
		"txfeecollector_cp": &s.cp,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
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

// 12. A governance proposal with MsgSetDenomAdmin gives the renounced denom
// to other: the event carries the empty previous admin, other can mint and
// hand the denom on, the renounced creator stays refused. The same message
// signed by a plain account is refused.
func (s *tokenFactorySuite) govSetAdmin(t *testing.T) {
	admin, other := s.admin.FormattedAddress(), s.other.FormattedAddress()
	gov, err := helpers.GovAuthority(s.ctx, s.bze)
	require.NoError(t, err)

	s.requireRefused(t, "expected gov account", s.admin, tfSetDenomAdminMsg(admin, s.denom, admin))

	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	_, err = helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{tfSetDenomAdminMsg(gov, s.denom, other)},
		Title:    "Give tok to other",
		Summary:  "e2e: MsgSetDenomAdmin on a renounced denom",
	})
	require.NoError(t, err)
	require.Equal(t, other, s.authority(t))

	to, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	var events []helpers.Event
	for h := from; h <= to; h++ {
		got, err := helpers.BlockEvents(s.ctx, s.bze, h, tfEventPrefix+"DenomAdminChangeEvent")
		require.NoError(t, err)
		events = append(events, helpers.EventsWith(got, map[string]string{"denom": s.denom})...)
	}
	require.Len(t, events, 1, "one DenomAdminChangeEvent in the block that executed the proposal")
	require.Empty(t, events[0].Attributes["admin"], "the previous admin of a renounced denom is empty")
	require.Equal(t, other, events[0].Attributes["new_admin"])

	s.requireRefused(t, "unauthorized", s.admin, mintMsg(admin, coinStr(1, s.denom)))
	before := s.supply(t)
	s.mustExec(t, s.other, "tokenfactory", "mint", coinStr(tfOtherMint, s.denom))
	require.Equal(t, before.AddRaw(tfOtherMint).String(), s.supply(t).String())

	res := s.mustExec(t, s.other, "tokenfactory", "change-admin", s.denom, admin)
	s.requireAdminChange(t, res, other, admin)
	require.Equal(t, admin, s.authority(t))
}

// 5. The payer creates the gas denom paying the create fee in ubze: the fee
// goes from the payer to the tokenfactory module and on to the community-pool
// collector, and the new denom gets the default bank metadata built from the
// subdenom: name "gas", symbol and 6-decimal display unit "GAS".
func (s *tokenFactorySuite) createFeeAndDefaultMetadata(t *testing.T) {
	payer := s.payer.FormattedAddress()
	fee := s.createDenomFee(t)
	before := s.allBalances(t, payer)

	res := s.mustExec(t, s.payer, "tokenfactory", "create-denom", "gas")
	require.Equal(t, before.Sub(fee.Add(sdk.NewCoin(chain.Denom, s.txFee(t, res)))), s.allBalances(t, payer),
		"payer must pay exactly the create fee and the tx fee")
	events := helpers.TxEvents(res, "transfer")
	require.Equal(t, fee.Amount.String(), s.transferred(t, events, payer, s.tfModule, chain.Denom).String(), "the create fee to the tokenfactory module")
	require.Equal(t, fee.Amount.String(), s.transferred(t, events, s.tfModule, s.cp, chain.Denom).String(), "the create fee to the txfeecollector_cp module")

	admin, err := s.authorityOf(s.gas)
	require.NoError(t, err)
	require.Equal(t, payer, admin)
	require.Equal(t, "0", s.supplyOf(t, s.gas).String())
	md, err := s.metadataOf(s.gas)
	require.NoError(t, err)
	require.Equal(t, tfMetadata{
		Base:       s.gas,
		DenomUnits: []tfDenomUnit{{Denom: s.gas}, {Denom: "GAS", Exponent: 6}},
		Display:    "GAS",
		Name:       "gas",
		Symbol:     "GAS",
	}, md, "default metadata of a new denom")
}

// 6. With a tx fee in gas, the create fee is bought with gas: the chain takes
// from the payer the gas that buys the fee out of the gas/ubze pool (the
// input rounded up), swaps it and sends the whole swap output to the
// community-pool collector, so the collector gets at least the fee. The payer
// spends no ubze. The swap output itself is not computed: it must be at least
// the fee and at most 1% above it.
func (s *tokenFactorySuite) createFeeNonNative(t *testing.T) {
	payer := s.payer.FormattedAddress()
	s.mustExec(t, s.payer, "tokenfactory", "mint", coinStr(tfGasMint, s.gas))
	s.mustExecWith(t, s.payer, []helpers.TxOption{helpers.WithGas(tfPoolTxGas)},
		"tradebin", "create-liquidity-pool", s.gas, chain.Denom, tfPoolFee, tfPoolDest, "false",
		strconv.Itoa(tfPoolGas), strconv.Itoa(tfPoolBze))
	pool := s.pool(t)
	require.Equal(t, s.gas, pool.Base)
	require.Equal(t, sdkmath.NewInt(tfPoolBze).String(), pool.reserveQuote().String())
	require.True(t, pool.reserveQuote().GTE(s.minNativeLiquidity(t)), "the pool must be deep enough to pay fees in gas")
	s.lpDenom = pool.LpDenom

	fee := s.createDenomFee(t)
	before := s.allBalances(t, payer)
	res := s.mustExecWith(t, s.payer, []helpers.TxOption{helpers.WithFees(coinStr(tfGasTxFee, s.gas)), helpers.WithGas(tfGasTxGas)},
		"tokenfactory", "create-denom", "tkn")
	paid, err := helpers.TxFee(res, s.gas)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(tfGasTxFee).String(), paid.String(), "tx fee in gas")
	require.True(t, s.txFee(t, res).IsZero(), "no ubze tx fee")

	events := helpers.TxEvents(res, "transfer")
	gasIn := s.transferred(t, events, payer, s.tradebin, s.gas)
	require.True(t, gasIn.IsPositive(), "the payer must pay the create fee in gas to tradebin")
	require.Equal(t, before.Sub(sdk.NewCoin(s.gas, gasIn.AddRaw(tfGasTxFee))), s.allBalances(t, payer),
		"the payer pays the tx fee and exactly the gas tradebin took, and no ubze")
	require.True(t, s.transferred(t, events, payer, s.tfModule, chain.Denom).IsZero(), "no ubze from the payer")

	swapped := s.transferred(t, events, s.tradebin, s.tfModule, chain.Denom)
	toCp := s.transferred(t, events, s.tfModule, s.cp, chain.Denom)
	require.Equal(t, swapped.String(), toCp.String(), "the whole swap output must reach the txfeecollector_cp module")
	require.True(t, toCp.GTE(fee.Amount), "the collector got %s, less than the create fee %s", toCp, fee.Amount)
	require.True(t, toCp.LTE(fee.Amount.Add(fee.Amount.QuoRaw(100))), "the collector got %s, more than 1%% above the create fee %s", toCp, fee.Amount)
	t.Logf("create fee %s bought with %s%s: %subze to the collector", fee, gasIn, s.gas, toCp)

	tkn := "factory/" + payer + "/tkn"
	admin, err := s.authorityOf(tkn)
	require.NoError(t, err)
	require.Equal(t, payer, admin)
}

// tfRefusal is how the chain refuses a message: in CheckTx (ValidateBasic,
// never included, nothing paid) or in DeliverTx (included, the tx fee paid).
type tfRefusal struct {
	checkTx   bool
	codespace string
	code      uint32
	reason    string
}

// 7. Refused denom creations charge no create fee and create nothing. The
// chain checks the subdenom for "_" and "/" (ValidateBasic), its length, that
// the subdenom itself has no supply, and the full denom's characters; it does
// not compare the full denom with supplied denoms (only with denoms that have
// bank metadata, the duplicate case).
func (s *tokenFactorySuite) createRefusals(t *testing.T) {
	admin := s.admin.FormattedAddress()
	for _, c := range []struct {
		name, subdenom string
		want           tfRefusal
	}{
		{"duplicate subdenom", "tok", tfRefusal{false, tfCodespace, 1101, "attempting to create a denom that already exists (has bank metadata)"}},
		{"empty subdenom", "", tfRefusal{true, tfSdk, 18, "subdenom must not be empty"}},
		{"underscore", "a_b", tfRefusal{true, tfSdk, 18, "subdenom should not contain _"}},
		{"slash", "a/b", tfRefusal{true, tfSdk, 18, "subdenom should not contain /"}},
		{"45 characters", strings.Repeat("a", tfMaxSubLen+1), tfRefusal{false, tfCodespace, 1106, "subdenom too long, max length is 44 bytes"}},
		{"subdenom with supply", chain.Denom, tfRefusal{false, tfUndefined, 1, "can't create subdenoms that are the same as a native denom"}},
		{"invalid denom character", "tok!", tfRefusal{false, tfUndefined, 1, "invalid denom: factory/" + admin + "/tok!"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			msg := map[string]any{"@type": "/bze.tokenfactory.MsgCreateDenom", "creator": admin, "subdenom": c.subdenom}
			s.requireMsgRefused(t, s.admin, msg, c.want)
			if c.subdenom == "tok!" {
				// an invalid denom can not be stored: the bank query refuses it
				// before looking for metadata
				denom := "factory/" + admin + "/" + c.subdenom
				_, err := s.authorityOf(denom)
				require.ErrorContains(t, err, "denom authority not found for denom "+denom)
				_, err = s.metadataOf(denom)
				require.ErrorContains(t, err, "invalid denom: "+denom)
				_, err = helpers.Query[json.RawMessage](s.ctx, s.bze, "bank", "total-supply-of", denom)
				require.ErrorContains(t, err, "invalid denom: "+denom)
			} else if c.subdenom != "tok" && c.subdenom != "" {
				s.requireNoDenom(t, "factory/"+admin+"/"+c.subdenom)
			}
		})
	}
}

// 8. Mint checks the denom has bank metadata, then that it is a factory
// denom; burn checks only the denom's admin, then the balance.
func (s *tokenFactorySuite) mintBurnRefusals(t *testing.T) {
	admin, other := s.admin.FormattedAddress(), s.other.FormattedAddress()
	_, err := s.metadataOf(chain.Denom)
	require.ErrorContains(t, err, "client metadata for denom "+chain.Denom, "ubze has no bank metadata on the e2e network")
	balance := sdk.NewCoin(s.denom, s.balance(t, admin, s.denom))
	above := balance.AddAmount(sdkmath.OneInt())
	lpSupply := s.supplyOf(t, s.lpDenom)

	for _, c := range []struct {
		name string
		user ibc.Wallet
		msg  map[string]any
		want tfRefusal
	}{
		{"mint a denom without metadata", s.admin, mintMsg(admin, coinStr(1, chain.Denom)), tfRefusal{false, tfCodespace, 1108, "denom: " + chain.Denom + ": denom does not exist"}},
		{"mint an LP denom", s.admin, mintMsg(admin, coinStr(1, s.lpDenom)), tfRefusal{false, tfCodespace, 1103, "not enough parts of denom " + s.lpDenom + ": invalid denom"}},
		{"mint an unknown factory denom", s.admin, mintMsg(admin, coinStr(1, s.nope)), tfRefusal{false, tfCodespace, 1108, "denom: " + s.nope + ": denom does not exist"}},
		{"burn ubze", s.admin, burnMsg(admin, coinStr(1, chain.Denom)), tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + chain.Denom}},
		{"burn an LP denom", s.admin, burnMsg(admin, coinStr(1, s.lpDenom)), tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + s.lpDenom}},
		{"burn an unknown factory denom", s.admin, burnMsg(admin, coinStr(1, s.nope)), tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + s.nope}},
		{"burn above the balance", s.admin, burnMsg(admin, above.String()), tfRefusal{false, tfSdk, 5, "spendable balance " + balance.String() + " is smaller than " + above.String()}},
		{"burn by a non-admin", s.other, burnMsg(other, coinStr(1, s.denom)), tfRefusal{false, tfCodespace, 1102, "unauthorized account"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s.requireMsgRefused(t, c.user, c.msg, c.want)
		})
	}
	require.Equal(t, lpSupply.String(), s.supplyOf(t, s.lpDenom).String(), "LP supply")
	s.requireNoDenom(t, s.nope)
}

// 9. ChangeAdmin refuses the creator itself and invalid addresses in
// ValidateBasic, an unknown denom and a non-admin (also renouncing) in
// DeliverTx.
func (s *tokenFactorySuite) changeAdminRefusals(t *testing.T) {
	admin, other := s.admin.FormattedAddress(), s.other.FormattedAddress()
	for _, c := range []struct {
		name string
		user ibc.Wallet
		msg  map[string]any
		want tfRefusal
	}{
		{"to self", s.admin, s.changeAdminMsg(admin, admin), tfRefusal{true, tfSdk, 7, "new admin must be different from creator"}},
		{"to an invalid address", s.admin, s.changeAdminMsg(admin, "bze1invalid"), tfRefusal{true, tfSdk, 7, "invalid new admin address"}},
		{"renounce by a non-admin", s.other, s.changeAdminMsg(other, ""), tfRefusal{false, tfCodespace, 1102, "unauthorized account"}},
		{"on an unknown denom", s.admin, tfChangeAdminMsg(admin, s.nope, other), tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + s.nope}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s.requireMsgRefused(t, c.user, c.msg, c.want)
		})
	}
	s.requireNoDenom(t, s.nope)
}

// 10. Metadata is validated by the bank rules in DeliverTx (the message's
// ValidateBasic checks only the creator) and needs the denom's admin, which a
// non-factory denom never has. Branding is validated in ValidateBasic: a
// complete package, a font slug and #RRGGBB colours.
func (s *tokenFactorySuite) metadataBrandingRefusals(t *testing.T) {
	admin, other := s.admin.FormattedAddress(), s.other.FormattedAddress()
	base := tfDenomUnit{Denom: s.denom}
	tok := tfDenomUnit{Denom: "TOK", Exponent: 6}
	for _, c := range []struct {
		name string
		user ibc.Wallet
		msg  map[string]any
		want tfRefusal
	}{
		{"blank name", s.admin, tfMetadataMsg(admin, s.denom, "TOK", "", base, tok), tfRefusal{false, tfUndefined, 1, "name field cannot be blank"}},
		{"display not a unit", s.admin, tfMetadataMsg(admin, s.denom, "XYZ", "Token", base, tok), tfRefusal{false, tfUndefined, 1, "metadata must contain a denomination unit with display denom 'XYZ'"}},
		{"base not the first unit", s.admin, tfMetadataMsg(admin, s.denom, "TOK", "Token", tok, base), tfRefusal{false, tfUndefined, 1, "metadata's first denomination unit must be the one with base denom '" + s.denom + "'"}},
		{"unsorted exponents", s.admin, tfMetadataMsg(admin, s.denom, "TOK", "Token", base, tok, tfDenomUnit{Denom: "mTOK", Exponent: 3}), tfRefusal{false, tfUndefined, 1, "denom units should be sorted asc by exponent"}},
		{"someone else's denom", s.other, s.metadataMsg(other, "Hijack"), tfRefusal{false, tfCodespace, 1102, "unauthorized account"}},
		{"unknown denom", s.admin, tfMetadataMsg(admin, s.nope, "TOK", "Nope", tfDenomUnit{Denom: s.nope}, tok), tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + s.nope}},
		{"ubze", s.admin, tfMetadataMsg(admin, chain.Denom, "BZE", "BZE", tfDenomUnit{Denom: chain.Denom}, tfDenomUnit{Denom: "BZE", Exponent: 6}), tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + chain.Denom}},
	} {
		t.Run("metadata "+c.name, func(t *testing.T) {
			s.requireMsgRefused(t, c.user, c.msg, c.want)
		})
	}
	_, err := s.metadataOf(chain.Denom)
	require.ErrorContains(t, err, "client metadata for denom "+chain.Denom, "ubze must still have no metadata")

	// a stored branding, so the refusals must leave it as it is
	s.mustExecMsg(t, s.admin, s.brandingMsg(admin, tfBranding("inter")))
	light := map[string]any{"background": "#FFFFFF", "text": "#111111", "primary": "#FF8800", "secondary": "#0088FF"}
	dark := map[string]any{"background": "#000000", "text": "#EEEEEE", "primary": "#FFAA33", "secondary": "#33AAFF"}
	with := func(p map[string]any, key, value string) map[string]any {
		out := map[string]any{key: value}
		for k, v := range p {
			if k != key {
				out[k] = v
			}
		}
		return out
	}
	full := map[string]any{"font": "inter", "light": light, "dark": dark}
	longFont := strings.Repeat("a", tfMaxFont+1)
	for _, c := range []struct {
		name     string
		denom    string
		branding map[string]any
		want     tfRefusal
	}{
		{"colour name", s.denom, map[string]any{"font": "inter", "light": with(light, "primary", "blue"), "dark": dark}, tfRefusal{true, tfCodespace, 1111, "invalid light primary color: blue"}},
		{"short hex colour", s.denom, map[string]any{"font": "inter", "light": light, "dark": with(dark, "text", "#12345")}, tfRefusal{true, tfCodespace, 1111, "invalid dark text color: #12345"}},
		{"colour without #", s.denom, map[string]any{"font": "inter", "light": with(light, "background", "FFFFFF"), "dark": dark}, tfRefusal{true, tfCodespace, 1111, "invalid light background color: FFFFFF"}},
		{"missing dark palette", s.denom, map[string]any{"font": "inter", "light": light}, tfRefusal{true, tfCodespace, 1111, "dark palette is missing"}},
		{"font only", s.denom, map[string]any{"font": "inter"}, tfRefusal{true, tfCodespace, 1111, "light palette is missing"}},
		{"empty font", s.denom, map[string]any{"light": light, "dark": dark}, tfRefusal{true, tfCodespace, 1111, "invalid font: : invalid branding"}},
		{"font too long", s.denom, map[string]any{"font": longFont, "light": light, "dark": dark}, tfRefusal{true, tfCodespace, 1111, "invalid font: " + longFont}},
		{"empty denom", "", full, tfRefusal{true, tfCodespace, 1103, "denom cannot be empty"}},
		{"unknown denom", s.nope, full, tfRefusal{false, tfUndefined, 1, "denom authority not found for denom " + s.nope}},
	} {
		t.Run("branding "+c.name, func(t *testing.T) {
			msg := map[string]any{"@type": "/bze.tokenfactory.MsgSetDenomBranding", "creator": admin, "denom": c.denom, "branding": c.branding}
			s.requireMsgRefused(t, s.admin, msg, c.want)
		})
	}
	s.requireNoDenom(t, s.nope)
}

// 11. Queries of a denom nobody created: the authority query fails with the
// keeper's error, the branding query answers NotFound, the branding list
// leaves it out and bank has no metadata nor supply for it.
func (s *tokenFactorySuite) unknownDenomQueries(t *testing.T) {
	s.requireNoDenom(t, s.nope)
	_, err := helpers.Query[struct {
		Branding *brandingInfo `json:"branding"`
	}](s.ctx, s.bze, "tokenfactory", "denom-branding", s.nope)
	require.ErrorContains(t, err, "not found")
	all, err := helpers.Query[struct {
		DenomBrandings []struct {
			Denom string `json:"denom"`
		} `json:"denom_brandings"`
	}](s.ctx, s.bze, "tokenfactory", "all-denom-branding")
	require.NoError(t, err)
	var listed []string
	for _, r := range all.DenomBrandings {
		listed = append(listed, r.Denom)
	}
	require.Equal(t, []string{s.denom}, listed, "only the admin's denom is branded")
}

// requireNoDenom asserts denom has no admin, no bank metadata and no supply.
func (s *tokenFactorySuite) requireNoDenom(t *testing.T, denom string) {
	t.Helper()
	_, err := s.authorityOf(denom)
	require.ErrorContains(t, err, "denom authority not found for denom "+denom)
	_, err = s.metadataOf(denom)
	require.ErrorContains(t, err, "client metadata for denom "+denom)
	require.True(t, s.supplyOf(t, denom).IsZero(), "%s has supply", denom)
}

// requireMsgRefused asserts the message is refused as want says, that the
// admin's denom did not change and that the user paid at most the tx fee.
func (s *tokenFactorySuite) requireMsgRefused(t *testing.T, user ibc.Wallet, msg any, want tfRefusal) {
	t.Helper()
	addr := user.FormattedAddress()
	before, balances := s.denomState(t), s.allBalances(t, addr)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, nil, msg)
	why, err := helpers.RejectionReason(res, err)
	require.NoError(t, err, "msg %v", msg)
	require.Contains(t, why, want.reason)
	require.Equal(t, want.codespace, res.Codespace, why)
	require.EqualValues(t, want.code, res.Code, why)
	if want.checkTx {
		require.Zero(t, res.Height, "a tx refused in CheckTx must not be included")
		require.Equal(t, balances, s.allBalances(t, addr), "a tx refused in CheckTx must cost nothing")
	} else {
		require.Positive(t, res.Height, "a tx refused in DeliverTx is included")
		require.Equal(t, balances.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.allBalances(t, addr), "a refused tx must cost only its fee")
	}
	require.Equal(t, before, s.denomState(t), "a refused tx must not change the denom")
}

func tfSetDenomAdminMsg(authority, denom, newAdmin string) map[string]any {
	return map[string]any{"@type": "/bze.tokenfactory.MsgSetDenomAdmin", "authority": authority, "denom": denom, "new_admin": newAdmin}
}

func tfChangeAdminMsg(creator, denom, newAdmin string) map[string]any {
	return map[string]any{"@type": "/bze.tokenfactory.MsgChangeAdmin", "creator": creator, "denom": denom, "newAdmin": newAdmin}
}

func tfMetadataMsg(creator, base, display, name string, units ...tfDenomUnit) map[string]any {
	du := make([]map[string]any, 0, len(units))
	for _, u := range units {
		du = append(du, map[string]any{"denom": u.Denom, "exponent": u.Exponent})
	}
	return map[string]any{
		"@type":   "/bze.tokenfactory.MsgSetDenomMetadata",
		"creator": creator,
		"metadata": map[string]any{
			"description": name + " for the e2e suite",
			"denom_units": du,
			"base":        base,
			"display":     display,
			"name":        name,
			"symbol":      "TOK",
		},
	}
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
	return s.supplyOf(t, s.denom)
}

func (s *tokenFactorySuite) supplyOf(t *testing.T, denom string) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Amount struct {
			Amount string `json:"amount"`
		} `json:"amount"`
	}](s.ctx, s.bze, "bank", "total-supply-of", denom)
	require.NoError(t, err)
	return parseInt(res.Amount.Amount)
}

func (s *tokenFactorySuite) authorityOf(denom string) (string, error) {
	res, err := helpers.Query[struct {
		DenomAuthority struct {
			Admin string `json:"admin"`
		} `json:"denomAuthority"`
	}](s.ctx, s.bze, "tokenfactory", "denom-authority", denom)
	return res.DenomAuthority.Admin, err
}

// tfMetadata is the whole bank metadata of a denom; the query omits empty
// fields and zero exponents.
type tfMetadata struct {
	Description string        `json:"description"`
	DenomUnits  []tfDenomUnit `json:"denom_units"`
	Base        string        `json:"base"`
	Display     string        `json:"display"`
	Name        string        `json:"name"`
	Symbol      string        `json:"symbol"`
	URI         string        `json:"uri"`
	URIHash     string        `json:"uri_hash"`
}

type tfDenomUnit struct {
	Denom    string   `json:"denom"`
	Exponent uint32   `json:"exponent"`
	Aliases  []string `json:"aliases"`
}

func (s *tokenFactorySuite) metadataOf(denom string) (tfMetadata, error) {
	res, err := helpers.Query[struct {
		Metadata tfMetadata `json:"metadata"`
	}](s.ctx, s.bze, "bank", "denom-metadata", denom)
	return res.Metadata, err
}

// allBalances returns every balance of addr.
func (s *tokenFactorySuite) allBalances(t *testing.T, addr string) sdk.Coins {
	t.Helper()
	res, err := helpers.Query[struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}](s.ctx, s.bze, "bank", "balances", addr)
	require.NoError(t, err)
	out := sdk.NewCoins()
	for _, b := range res.Balances {
		out = out.Add(sdk.NewCoin(b.Denom, parseInt(b.Amount)))
	}
	return out
}

func (s *tokenFactorySuite) transferred(t *testing.T, events []helpers.Event, from, to, denom string) sdkmath.Int {
	t.Helper()
	got, err := helpers.Transferred(events, from, to, denom)
	require.NoError(t, err)
	return got
}

func (s *tokenFactorySuite) mustExecWith(t *testing.T, user ibc.Wallet, opts []helpers.TxOption, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, opts, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

func (s *tokenFactorySuite) pool(t *testing.T) poolInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		Pool poolInfo `json:"pool"`
	}](s.ctx, s.bze, "tradebin", "liquidity-pool", s.gasPool)
	require.NoError(t, err)
	require.Equal(t, s.gasPool, res.Pool.ID, "pool %s not found", s.gasPool)
	return res.Pool
}

func (s *tokenFactorySuite) minNativeLiquidity(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			MinNativeLiquidityForModuleSwap string `json:"minNativeLiquidityForModuleSwap"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	return parseInt(res.Params.MinNativeLiquidityForModuleSwap)
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
