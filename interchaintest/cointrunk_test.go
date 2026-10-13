package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

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
	ctAdminFunds = 100_000_000_000 // 100,000 BZE: proposals
	ctPub1Funds  = 1_000_000_000   // 1,000 BZE: gas, its articles are free
	ctPub2Funds  = 50_000_000_000  // 50,000 BZE: one paid article once inactive
	ctAnonFunds  = 100_000_000_000 // 100,000 BZE: three paid articles at 25,000 BZE and one at 1,000 BZE
	ctFanFunds   = 10_000_000_000  // 10,000 BZE: respect payments
	ctPoorFunds  = 100_000_000     // 100 BZE: gas, never an article

	ctDomain   = "example.com"
	ctInactive = "inactive.org"
	ctOther    = "other.net"

	ctPub1Name    = "Alice News"
	ctPub2Name    = "Bob Blog"
	ctPub2Renamed = "Bob Blog Archive"

	// step 6 lowers the monthly limit of paid articles to ctLimit; step 10
	// raises it to ctNewLimit, makes an anonymous article cost ctNewCost and
	// the respect tax ctNewTax
	ctLimit    = 3
	ctNewLimit = 10
	ctNewCost  = 1_000_000_000 // 1,000 BZE
	ctNewTax   = "0.25"

	ctRespect         = 1_000_000 // tax 20% = 200,000
	ctSmallRespect    = 4         // tax 0.8, truncated to 0
	ctInactiveRespect = 1_000
	ctNewTaxRespect   = 100 // tax 25% = 25

	ctMsgGas = 400_000

	ctEventPrefix = "bze.cointrunk."
)

// TestCointrunk drives x/cointrunk on a real network: accepted domains and
// publishers through gov proposals, free articles from active publishers,
// paid articles from anonymous authors and inactive publishers (the fee to
// the community pool, the monthly limit), the AddArticle refusals, picture
// URLs, publisher respect with its community-pool tax, the params by gov and
// every query. Subtests run in order and share state.
func TestCointrunk(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupCointrunk(t, ctx, net.Chain)

	t.Run("1 accepted domains by gov", s.acceptDomains)
	t.Run("2 publishers by gov", s.savePublishers)
	t.Run("3 free articles from active publishers", s.freeArticles)
	t.Run("4 anonymous article pays", s.anonymousArticle)
	t.Run("5 inactive publisher pays", s.inactivePublisher)
	t.Run("6 monthly limit of paid articles", s.monthlyLimit)
	t.Run("7 AddArticle refusals", s.articleRefusals)
	t.Run("8 picture URL", s.pictureURL)
	t.Run("9 publisher respect", s.publisherRespect)
	t.Run("10 params by gov", s.paramsByGov)
	t.Run("11 queries", s.queries)
}

// cointrunkSuite reuses the fees suite's distribution, balance and refusal
// helpers (community pool at a height, transfers, requireRefused).
type cointrunkSuite struct {
	*feesSuite

	pub1, pub2, anon, fan ibc.Wallet

	params     ctParams
	articles   []ctArticle
	publishers map[string]ctPublisher
	paid       uint64 // paid articles this month
	month      string // block-time YYYYMM of the paid articles
}

func setupCointrunk(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *cointrunkSuite {
	s := &cointrunkSuite{feesSuite: &feesSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}}, publishers: map[string]ctPublisher{}}

	for _, u := range []struct {
		w     *ibc.Wallet
		name  string
		funds int64
	}{
		{&s.admin, "ct-admin", ctAdminFunds},
		{&s.pub1, "ct-pub1", ctPub1Funds},
		{&s.pub2, "ct-pub2", ctPub2Funds},
		{&s.anon, "ct-anon", ctAnonFunds},
		{&s.fan, "ct-fan", ctFanFunds},
		{&s.poor, "ct-poor", ctPoorFunds},
	} {
		*u.w = interchaintest.GetAndFundTestUsers(t, ctx, u.name, sdkmath.NewInt(u.funds), bze)[0]
	}
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	for name, addr := range map[string]*string{
		"fee_collector": &s.feeCollector,
		"distribution":  &s.distr,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	s.tax = s.communityTax(t)

	s.params = s.queryParams(t)
	require.Equal(t, ctParams{
		AnonArticleLimit:       5,
		AnonArticleCost:        sdk.NewInt64Coin(chain.Denom, 25_000_000_000),
		PublisherRespectParams: ctRespectParams{Tax: sdkmath.LegacyMustNewDecFromStr("0.2"), Denom: chain.Denom},
	}, s.params, "default params")
	return s
}

// 1. Domains are accepted only by gov. One proposal adds example.com and
// inactive.org, then turns inactive.org off: two added events and one
// updated event, in message order.
func (s *cointrunkSuite) acceptDomains(t *testing.T) {
	t.Run("signed by a user", func(t *testing.T) {
		s.requireMsgRefused(t, s.fan, s.domainMsg(s.fan.FormattedAddress(), ctDomain, true), "cointrunk", 1001, "invalid authority")
	})
	t.Run("invalid domain", func(t *testing.T) {
		_, err := s.submitProposal("invalid domain", s.domainMsg(s.gov, "not_a_domain", true))
		require.ErrorContains(t, err, "proposal domain is invalid")
	})
	require.Empty(t, s.acceptedDomains(t), "no domain accepted by refused messages")

	h := s.passProposal(t, "accepted domains",
		s.domainMsg(s.gov, ctDomain, true),
		s.domainMsg(s.gov, ctInactive, true),
		s.domainMsg(s.gov, ctInactive, false))

	var added []ctDomainEntry
	for _, e := range s.blockTypedEvents(t, h, "AcceptedDomainAddedEvent") {
		added = append(added, decodeNormalized[ctDomainEntry](t, e.Attributes["accepted_domain"]))
	}
	require.Equal(t, []ctDomainEntry{{ctDomain, true}, {ctInactive, true}}, added, "added events")
	updated := s.blockTypedEvents(t, h, "AcceptedDomainUpdatedEvent")
	require.Len(t, updated, 1, "updated events")
	require.Equal(t, ctDomainEntry{ctInactive, false}, decodeNormalized[ctDomainEntry](t, updated[0].Attributes["accepted_domain"]))

	require.Equal(t, []ctDomainEntry{{ctDomain, true}, {ctInactive, false}}, s.acceptedDomains(t))
}

// 2. Publishers are saved only by gov. A new publisher starts with no
// articles and respect "0", created at the block that executed the proposal.
func (s *cointrunkSuite) savePublishers(t *testing.T) {
	t.Run("signed by a user", func(t *testing.T) {
		s.requireMsgRefused(t, s.fan, s.publisherMsg(s.fan.FormattedAddress(), ctPub1Name, s.pub1.FormattedAddress(), true), "cointrunk", 1001, "invalid authority")
	})
	for _, c := range []struct {
		name, pubName, addr, reason string
	}{
		{"name too short", "ab", s.pub1.FormattedAddress(), "invalid name: expecting at least 3 characters"},
		{"bad address", ctPub1Name, "bze1notanaddress", "invalid publisher address"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.submitProposal(c.name, s.publisherMsg(s.gov, c.pubName, c.addr, true))
			require.ErrorContains(t, err, c.reason)
		})
	}
	require.Empty(t, s.queryPublishers(t), "no publisher saved by refused messages")

	h := s.passProposal(t, "publishers",
		s.publisherMsg(s.gov, ctPub1Name, s.pub1.FormattedAddress(), true),
		s.publisherMsg(s.gov, ctPub2Name, s.pub2.FormattedAddress(), true))
	created := helpers.Int64(s.blockTime(t, h).Unix())
	for _, p := range []struct {
		w    ibc.Wallet
		name string
	}{{s.pub1, ctPub1Name}, {s.pub2, ctPub2Name}} {
		s.publishers[p.w.FormattedAddress()] = ctPublisher{
			Name: p.name, Address: p.w.FormattedAddress(), Active: true, CreatedAt: created, Respect: "0",
		}
	}

	var added []ctPublisher
	for _, e := range s.blockTypedEvents(t, h, "PublisherAddedEvent") {
		added = append(added, decodeNormalized[ctPublisher](t, e.Attributes["publisher"]))
	}
	require.Equal(t, []ctPublisher{s.publishers[s.pub1.FormattedAddress()], s.publishers[s.pub2.FormattedAddress()]}, added, "added events")
	require.Empty(t, s.blockTypedEvents(t, h, "PublisherUpdatedEvent"))
	s.requirePublisher(t, s.pub1)
	s.requirePublisher(t, s.pub2)
}

// 3. An active publisher's article is free: no community-pool transfer, the
// author pays only the tx fee, ids count up from 1 and the publisher's
// article count grows.
func (s *cointrunkSuite) freeArticles(t *testing.T) {
	for _, pub := range []ibc.Wallet{s.pub1, s.pub2} {
		before := s.addrBalances(t, pub.FormattedAddress(), 0)
		res := s.addArticle(t, pub, "A free article by "+pub.KeyName(), "https://"+ctDomain+"/free/"+pub.KeyName(), "", false)
		require.Empty(t, s.transfers(t, res.Events, pub.FormattedAddress(), s.distr), "nothing to the community pool")
		require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, pub.FormattedAddress(), 0), "only the tx fee")
	}
	require.EqualValues(t, []helpers.Int64{1, 2}, []helpers.Int64{s.articles[0].ID, s.articles[1].ID})
	s.requireArticles(t)
	s.requirePublisher(t, s.pub1)
	s.requirePublisher(t, s.pub2)
	require.Empty(t, s.anonCounters(t), "no paid article yet")
}

// 4. An author who is not a publisher pays anon_article_cost into the
// community pool and the month's paid-article counter grows. An author who
// cannot pay is refused and the counter stays.
func (s *cointrunkSuite) anonymousArticle(t *testing.T) {
	s.requirePaidArticle(t, s.anon, "An anonymous article")

	t.Run("author cannot pay", func(t *testing.T) {
		msg := s.articleMsg(s.poor.FormattedAddress(), "An article nobody pays for", "https://"+ctDomain+"/poor", "")
		s.requireMsgRefused(t, s.poor, msg, "sdk", 5, "insufficient funds")
		s.requireArticles(t)
		s.requireCounter(t)
	})
}

// 5. A publisher turned inactive by gov keeps its name change, article count,
// respect and creation time, and from then on pays for articles like an
// anonymous author, while its article count still grows.
func (s *cointrunkSuite) inactivePublisher(t *testing.T) {
	addr := s.pub2.FormattedAddress()
	h := s.passProposal(t, "deactivate publisher", s.publisherMsg(s.gov, ctPub2Renamed, addr, false))
	p := s.publishers[addr]
	p.Name, p.Active = ctPub2Renamed, false
	s.publishers[addr] = p

	updated := s.blockTypedEvents(t, h, "PublisherUpdatedEvent")
	require.Len(t, updated, 1)
	require.Equal(t, p, decodeNormalized[ctPublisher](t, updated[0].Attributes["publisher"]))
	require.Empty(t, s.blockTypedEvents(t, h, "PublisherAddedEvent"))
	s.requirePublisher(t, s.pub2)

	s.requirePaidArticle(t, s.pub2, "An article by an inactive publisher")
}

// 6. With anon_article_limit lowered by gov to the paid articles so far + 1,
// one more paid article goes through and the next is refused; active
// publishers are not limited. (A new month resets the counter; a real-time
// chain cannot reach it, see the README's known gaps.)
func (s *cointrunkSuite) monthlyLimit(t *testing.T) {
	require.EqualValues(t, ctLimit-1, s.paid, "paid articles before the limit")
	p := s.params
	p.AnonArticleLimit = ctLimit
	s.setParams(t, p)

	s.requirePaidArticle(t, s.anon, "The last paid article of the month")

	msg := s.articleMsg(s.anon.FormattedAddress(), "One paid article too many", "https://"+ctDomain+"/over", "")
	before := s.addrBalances(t, s.anon.FormattedAddress(), 0)
	res := s.requireMsgRefused(t, s.anon, msg, "sdk", 18, "Paid article limit reached for current period")
	require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.anon.FormattedAddress(), 0), "only the tx fee")
	s.requireCounter(t)

	s.addArticle(t, s.pub1, "Active publishers ignore the limit", "https://"+ctDomain+"/unlimited", "", false)
	s.requireArticles(t)
	s.requireCounter(t)
}

// 7. AddArticle refusals, all from the active publisher: the domain checks of
// the message server and the URL and title checks of ValidateBasic. None
// stores an article.
func (s *cointrunkSuite) articleRefusals(t *testing.T) {
	long := strings.Repeat("x", 321)
	for _, c := range []struct {
		name, title, url, picture, reason string
	}{
		{"domain not accepted", "A valid title", "https://" + ctOther + "/a", "", "provided url domain (" + ctOther + ") is not an accepted domain"},
		{"domain inactive", "A valid title", "https://" + ctInactive + "/a", "", "provided url domain (" + ctInactive + ") is NOT active"},
		{"picture domain not accepted", "A valid title", "https://" + ctDomain + "/a", "https://" + ctOther + "/p.png", "provided picture domain (" + ctOther + ") is not an accepted domain"},
		{"picture domain inactive", "A valid title", "https://" + ctDomain + "/a", "https://" + ctInactive + "/p.png", "provided picture domain (" + ctInactive + ") is NOT active"},
		{"http url", "A valid title", "http://" + ctDomain + "/a", "", "invalid url scheme: only https accepted"},
		{"http picture", "A valid title", "https://" + ctDomain + "/a", "http://" + ctDomain + "/p.png", "invalid url scheme: only https accepted"},
		{"title too short", "Too short", "https://" + ctDomain + "/a", "", "invalid title: expecting between 10 and 320 characters"},
		{"title too long", long, "https://" + ctDomain + "/a", "", "invalid title: expecting between 10 and 320 characters"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s.requireMsgRefused(t, s.pub1, s.articleMsg(s.pub1.FormattedAddress(), c.title, c.url, c.picture), "sdk", 18, c.reason)
		})
	}
	s.requireArticles(t)
	s.requirePublisher(t, s.pub1)
}

// 8. The picture is optional (every article so far has none) and must sit on
// an accepted, active domain. A www. host counts as its bare domain.
func (s *cointrunkSuite) pictureURL(t *testing.T) {
	s.addArticle(t, s.pub1, "An article with a picture", "https://www."+ctDomain+"/pictured", "https://"+ctDomain+"/img/pictured.png", false)
	s.requireArticles(t)
	s.requirePublisher(t, s.pub1)
}

// 9. Respect paid to a publisher: the amount minus the tax to the publisher,
// the tax (truncated) to the community pool, the gross amount added to the
// publisher's respect. Inactive publishers can be paid too.
func (s *cointrunkSuite) publisherRespect(t *testing.T) {
	t.Run("taxed", func(t *testing.T) { s.requireRespect(t, s.pub1, ctRespect) })
	t.Run("tax truncated to zero", func(t *testing.T) { s.requireRespect(t, s.pub1, ctSmallRespect) })
	t.Run("inactive publisher", func(t *testing.T) { s.requireRespect(t, s.pub2, ctInactiveRespect) })

	for _, c := range []struct {
		name      string
		payer     ibc.Wallet
		publisher string
		amount    string
		code      uint32
		reason    string
	}{
		{"wrong denom", s.fan, s.pub1.FormattedAddress(), "1000uatom", 18, "invalid coin denom. Accepted (ubze) got (uatom)"},
		{"unknown publisher", s.fan, s.anon.FormattedAddress(), coinStr(1000, chain.Denom), 18, fmt.Sprintf("publisher (%s) could not be found", s.anon.FormattedAddress())},
		{"zero amount", s.fan, s.pub1.FormattedAddress(), coinStr(0, chain.Denom), 10, "invalid amount: amount should be positive"},
		// the reward is sent first: the payer covers it, not the tax after it
		{"cannot pay the tax", s.poor, s.pub1.FormattedAddress(), coinStr(ctPoorFunds, chain.Denom), 18, "Could not fund community pool (spendable balance"},
		{"cannot pay the reward", s.poor, s.pub1.FormattedAddress(), coinStr(ctAdminFunds, chain.Denom), 5, "insufficient funds"},
	} {
		t.Run(c.name, func(t *testing.T) {
			pubBefore := s.addrBalances(t, s.pub1.FormattedAddress(), 0)
			s.requireMsgRefused(t, c.payer, s.respectMsg(c.payer.FormattedAddress(), c.publisher, c.amount), "sdk", c.code, c.reason)
			require.Equal(t, pubBefore, s.addrBalances(t, s.pub1.FormattedAddress(), 0), "the whole payment is reverted")
			s.requirePublisher(t, s.pub1)
		})
	}
}

// 10. The params change only through gov, and only to valid values. New
// values apply to the next article and respect payment.
func (s *cointrunkSuite) paramsByGov(t *testing.T) {
	t.Run("signed by a user", func(t *testing.T) {
		s.requireMsgRefused(t, s.fan, s.paramsMsg(s.fan.FormattedAddress(), s.params), "cointrunk", 1001, "invalid authority")
	})
	for _, c := range []struct {
		name   string
		edit   func(m map[string]any)
		reason string
	}{
		{"tax zero", func(m map[string]any) { m["publisher_respect_params"].(map[string]any)["tax"] = "0" }, "tax should be between 0 (exclusive) and 1 (exclusive)"},
		{"tax one", func(m map[string]any) { m["publisher_respect_params"].(map[string]any)["tax"] = "1" }, "tax should be between 0 (exclusive) and 1 (exclusive)"},
		{"negative cost", func(m map[string]any) {
			m["anon_article_cost"] = map[string]string{"denom": chain.Denom, "amount": "-5"}
		}, "invalid anonArticleLimit coin"},
	} {
		t.Run(c.name, func(t *testing.T) {
			msg := s.paramsMsg(s.gov, s.params)
			c.edit(msg["params"].(map[string]any))
			_, err := s.submitProposal("invalid cointrunk params", msg)
			require.ErrorContains(t, err, c.reason)
		})
	}
	require.Equal(t, s.params, s.queryParams(t), "params unchanged")

	s.setParams(t, ctParams{
		AnonArticleLimit:       ctNewLimit,
		AnonArticleCost:        sdk.NewInt64Coin(chain.Denom, ctNewCost),
		PublisherRespectParams: ctRespectParams{Tax: sdkmath.LegacyMustNewDecFromStr(ctNewTax), Denom: chain.Denom},
	})
	t.Run("new cost", func(t *testing.T) { s.requirePaidArticle(t, s.anon, "An anonymous article at the new cost") })
	t.Run("new tax", func(t *testing.T) { s.requireRespect(t, s.pub1, ctNewTaxRespect) })
}

// 11. Every query returns exactly the state the steps built.
func (s *cointrunkSuite) queries(t *testing.T) {
	require.Equal(t, s.params, s.queryParams(t), "params")
	require.Equal(t, []ctDomainEntry{{ctDomain, true}, {ctInactive, false}}, s.acceptedDomains(t), "accepted-domain")
	s.requireArticles(t)
	s.requireCounter(t)

	want := make([]ctPublisher, 0, len(s.publishers))
	for _, p := range s.publishers {
		want = append(want, p)
	}
	// the store keys publishers by address
	sort.Slice(want, func(i, j int) bool { return want[i].Address < want[j].Address })
	require.Equal(t, want, s.queryPublishers(t), "publishers")
	s.requirePublisher(t, s.pub1)
	s.requirePublisher(t, s.pub2)
	require.EqualValues(t, 3, s.publishers[s.pub1.FormattedAddress()].ArticlesCount, "pub1 articles")
	require.Equal(t, strconv.Itoa(ctRespect+ctSmallRespect+ctNewTaxRespect), s.publishers[s.pub1.FormattedAddress()].Respect, "pub1 respect")
}

// requirePaidArticle adds an article that pays anon_article_cost and asserts
// the fee reached the community pool exactly, the author paid it plus the tx
// fee and the month's counter grew.
func (s *cointrunkSuite) requirePaidArticle(t *testing.T, author ibc.Wallet, title string) {
	t.Helper()
	cost := s.params.AnonArticleCost
	before := s.addrBalances(t, author.FormattedAddress(), 0)
	res := s.addArticle(t, author, title, fmt.Sprintf("https://%s/paid/%d", ctDomain, len(s.articles)+1), "", true)

	require.Equal(t, sdk.NewCoins(cost), s.transfers(t, res.Events, author.FormattedAddress(), s.distr), "article fee to the community pool")
	s.requireCommunityPool(t, res.Height, sdk.NewCoins(cost))
	require.Equal(t, before.Sub(sdk.NewCoins(cost).Add(sdk.NewCoin(chain.Denom, s.txFee(t, res)))...), s.addrBalances(t, author.FormattedAddress(), 0), "author pays the cost and the tx fee")

	month := s.blockTime(t, res.Height).UTC().Format("200601")
	if s.month == "" {
		s.month = month
	}
	require.Equal(t, s.month, month, "the suite's paid articles must fall in one month")
	s.paid++
	s.requireCounter(t)
	s.requireArticles(t)
	if _, ok := s.publishers[author.FormattedAddress()]; ok {
		s.requirePublisher(t, author)
	}
}

// requireRespect pays amount of respect from the fan to the publisher and
// asserts the split, the event, the respect and the community pool.
func (s *cointrunkSuite) requireRespect(t *testing.T, pub ibc.Wallet, amount int64) {
	t.Helper()
	tax := s.params.PublisherRespectParams.Tax.MulInt64(amount).TruncateInt()
	reward := sdkmath.NewInt(amount).Sub(tax)
	fanBefore := s.addrBalances(t, s.fan.FormattedAddress(), 0)
	pubBefore := s.addrBalances(t, pub.FormattedAddress(), 0)

	res := s.mustExec(t, s.fan, "cointrunk", "pay-publisher-respect", pub.FormattedAddress(), coinStr(amount, chain.Denom))

	require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, reward)), s.transfers(t, res.Events, s.fan.FormattedAddress(), pub.FormattedAddress()), "reward to the publisher")
	funded := sdk.NewCoins(sdk.NewCoin(chain.Denom, tax))
	require.Equal(t, funded, s.transfers(t, res.Events, s.fan.FormattedAddress(), s.distr), "tax to the community pool")
	s.requireCommunityPool(t, res.Height, funded)
	require.Equal(t, fanBefore.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res).AddRaw(amount))), s.addrBalances(t, s.fan.FormattedAddress(), 0), "fan pays amount and tx fee")
	require.Equal(t, pubBefore.Add(sdk.NewCoin(chain.Denom, reward)), s.addrBalances(t, pub.FormattedAddress(), 0), "publisher gets the reward")

	s.requireTxEvent(t, res, "PublisherRespectPaidEvent", map[string]string{
		"publisher":            pub.FormattedAddress(),
		"respect_paid":         strconv.FormatInt(amount, 10),
		"publisher_reward":     reward.String(),
		"community_pool_funds": tax.String(),
	})

	p := s.publishers[pub.FormattedAddress()]
	p.Respect = parseInt(p.Respect).AddRaw(amount).String()
	s.publishers[pub.FormattedAddress()] = p
	s.requirePublisher(t, pub)
}

// requireCommunityPool asserts the community pool grew at height by exactly
// the community tax of the BeginBlock fee allocation plus funded.
func (s *cointrunkSuite) requireCommunityPool(t *testing.T, height int64, funded sdk.Coins) {
	t.Helper()
	all := sdk.NewDecCoinsFromCoins(s.transfers(t, s.blockEvents(t, height), s.feeCollector, s.distr)...)
	taxed := all.Sub(all.MulDecTruncate(sdkmath.LegacyOneDec().Sub(s.tax)))
	requireDecDelta(t, "community pool", s.decCoinsAt(t, height-1, "community-pool"), s.decCoinsAt(t, height, "community-pool"),
		taxed.Add(sdk.NewDecCoinsFromCoins(funded...)...))
}

// addArticle adds an article through the CLI, asserts its event and records
// it (and the publisher's article count) as expected state.
func (s *cointrunkSuite) addArticle(t *testing.T, author ibc.Wallet, title, url, picture string, paid bool) helpers.TxResult {
	t.Helper()
	args := []string{"cointrunk", "add-article", title, url}
	if picture != "" {
		args = append(args, "--picture", picture)
	}
	res := s.mustExec(t, author, args...)

	id := helpers.Int64(len(s.articles) + 1)
	s.requireTxEvent(t, res, "ArticleAddedEvent", map[string]string{
		"article_id": strconv.FormatInt(int64(id), 10),
		"publisher":  author.FormattedAddress(),
		"paid":       strconv.FormatBool(paid),
	})
	s.articles = append(s.articles, ctArticle{
		ID: id, Title: title, URL: url, Picture: picture, Publisher: author.FormattedAddress(), Paid: paid,
		CreatedAt: helpers.Int64(s.blockTime(t, res.Height).Unix()),
	})
	if p, ok := s.publishers[author.FormattedAddress()]; ok {
		p.ArticlesCount++
		s.publishers[author.FormattedAddress()] = p
	}
	return res
}

// requireMsgRefused signs msg with user (fixed gas, so the tx is broadcast)
// and asserts the refusal. It returns the result for fee checks.
func (s *cointrunkSuite) requireMsgRefused(t *testing.T, user ibc.Wallet, msg any, codespace string, code uint32, reason string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(ctMsgGas)}, msg)
	s.requireRefused(t, res, err, codespace, code, reason)
	return res
}

func (s *cointrunkSuite) requireTxEvent(t *testing.T, res helpers.TxResult, name string, attrs map[string]string) {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, ctEventPrefix+name), attrs)
	require.Len(t, events, 1, "want one %s with %v in tx %s, got %+v", name, attrs, res.TxHash, helpers.TxEvents(res, ctEventPrefix+name))
}

func (s *cointrunkSuite) blockTypedEvents(t *testing.T, height int64, name string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, ctEventPrefix+name)
	require.NoError(t, err)
	return events
}

func (s *cointrunkSuite) submitProposal(title string, msgs ...any) (uint64, error) {
	return helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: msgs,
		Title:    title,
		Summary:  "e2e: " + title,
	})
}

// passProposal passes msgs as one proposal and returns the height of the
// block whose gov EndBlock executed it.
func (s *cointrunkSuite) passProposal(t *testing.T, title string, msgs ...any) int64 {
	t.Helper()
	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	id, err := s.submitProposal(title, msgs...)
	require.NoError(t, err)

	to, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	attrs := map[string]string{"proposal_id": strconv.FormatUint(id, 10), "proposal_result": "proposal_passed"}
	for h := from; h <= to; h++ {
		events, err := helpers.BlockEvents(s.ctx, s.bze, h, "active_proposal")
		require.NoError(t, err)
		if len(helpers.EventsWith(events, attrs)) > 0 {
			return h
		}
	}
	require.FailNow(t, "no active_proposal event", "proposal %d between heights %d and %d", id, from, to)
	return 0
}

func (s *cointrunkSuite) setParams(t *testing.T, p ctParams) {
	t.Helper()
	s.passProposal(t, "cointrunk params", s.paramsMsg(s.gov, p))
	s.params = p
	require.Equal(t, p, s.queryParams(t), "params")
}

func (s *cointrunkSuite) domainMsg(authority, domain string, active bool) map[string]any {
	return map[string]any{"@type": "/bze.cointrunk.MsgAcceptDomain", "authority": authority, "domain": domain, "active": active}
}

func (s *cointrunkSuite) publisherMsg(authority, name, addr string, active bool) map[string]any {
	return map[string]any{"@type": "/bze.cointrunk.MsgSavePublisher", "authority": authority, "name": name, "address": addr, "active": active}
}

func (s *cointrunkSuite) articleMsg(publisher, title, url, picture string) map[string]any {
	return map[string]any{"@type": "/bze.cointrunk.MsgAddArticle", "publisher": publisher, "title": title, "url": url, "picture": picture}
}

func (s *cointrunkSuite) respectMsg(creator, publisher, amount string) map[string]any {
	return map[string]any{"@type": "/bze.cointrunk.MsgPayPublisherRespect", "creator": creator, "address": publisher, "amount": amount}
}

// paramsMsg is a cointrunk MsgUpdateParams with the params by proto field name.
func (s *cointrunkSuite) paramsMsg(authority string, p ctParams) map[string]any {
	return map[string]any{
		"@type":     "/bze.cointrunk.MsgUpdateParams",
		"authority": authority,
		"params": map[string]any{
			"anon_article_limit": strconv.FormatUint(uint64(p.AnonArticleLimit), 10),
			"anon_article_cost":  map[string]string{"denom": p.AnonArticleCost.Denom, "amount": p.AnonArticleCost.Amount.String()},
			"publisher_respect_params": map[string]any{
				"tax":   p.PublisherRespectParams.Tax.String(),
				"denom": p.PublisherRespectParams.Denom,
			},
		},
	}
}

// ctParams are the cointrunk params; decoded from the query, whose key
// spelling depends on the encoder (see decodeNormalized).
type ctParams struct {
	AnonArticleLimit       helpers.Int64
	AnonArticleCost        sdk.Coin
	PublisherRespectParams ctRespectParams
}

type ctRespectParams struct {
	Tax   sdkmath.LegacyDec
	Denom string
}

func (s *cointrunkSuite) queryParams(t *testing.T) ctParams {
	t.Helper()
	res := s.query(t, "cointrunk", "params")
	var raw struct {
		Params struct {
			AnonArticleLimit helpers.Int64 `json:"anonarticlelimit"`
			AnonArticleCost  struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"anonarticlecost"`
			PublisherRespectParams struct {
				Tax   string `json:"tax"`
				Denom string `json:"denom"`
			} `json:"publisherrespectparams"`
		} `json:"params"`
	}
	require.NoError(t, json.Unmarshal(res, &raw), "%s", res)
	p := raw.Params
	return ctParams{
		AnonArticleLimit:       p.AnonArticleLimit,
		AnonArticleCost:        sdk.NewCoin(p.AnonArticleCost.Denom, parseInt(p.AnonArticleCost.Amount)),
		PublisherRespectParams: ctRespectParams{Tax: legacyDec(t, p.PublisherRespectParams.Tax), Denom: p.PublisherRespectParams.Denom},
	}
}

type ctDomainEntry struct {
	Domain string `json:"domain"`
	Active bool   `json:"active"`
}

func (s *cointrunkSuite) acceptedDomains(t *testing.T) []ctDomainEntry {
	t.Helper()
	var res struct {
		Domains []ctDomainEntry `json:"accepteddomain"`
	}
	raw := s.query(t, "cointrunk", "accepted-domain")
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	return res.Domains
}

type ctArticle struct {
	ID        helpers.Int64 `json:"id"`
	Title     string        `json:"title"`
	URL       string        `json:"url"`
	Picture   string        `json:"picture"`
	Publisher string        `json:"publisher"`
	Paid      bool          `json:"paid"`
	CreatedAt helpers.Int64 `json:"createdat"`
}

// requireArticles asserts all-articles returns exactly the articles added so
// far, in id order.
func (s *cointrunkSuite) requireArticles(t *testing.T) {
	t.Helper()
	var res struct {
		Articles []ctArticle `json:"article"`
	}
	raw := s.query(t, "cointrunk", "all-articles")
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	require.Equal(t, s.articles, res.Articles, "all-articles")
}

type ctPublisher struct {
	Name          string        `json:"name"`
	Address       string        `json:"address"`
	Active        bool          `json:"active"`
	ArticlesCount helpers.Int64 `json:"articlescount"`
	CreatedAt     helpers.Int64 `json:"createdat"`
	Respect       string        `json:"respect"`
}

func (s *cointrunkSuite) queryPublishers(t *testing.T) []ctPublisher {
	t.Helper()
	var res struct {
		Publishers []ctPublisher `json:"publisher"`
	}
	raw := s.query(t, "cointrunk", "publishers")
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	return res.Publishers
}

// requirePublisher asserts the publisher query returns the expected state.
func (s *cointrunkSuite) requirePublisher(t *testing.T, pub ibc.Wallet) {
	t.Helper()
	var res struct {
		Publisher ctPublisher `json:"publisher"`
	}
	raw := s.query(t, "cointrunk", "publisher", pub.FormattedAddress())
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	require.Equal(t, s.publishers[pub.FormattedAddress()], res.Publisher, "publisher %s", pub.KeyName())
}

type ctCounter struct {
	Key     string        `json:"key"`
	Counter helpers.Int64 `json:"counter"`
}

func (s *cointrunkSuite) anonCounters(t *testing.T) []ctCounter {
	t.Helper()
	var res struct {
		Counters []ctCounter `json:"anonarticlescounters"`
	}
	raw := s.query(t, "cointrunk", "all-anon-articles-counters")
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	return res.Counters
}

// requireCounter asserts the one monthly counter: its key is the block-time
// month followed by the counter's own key ("counter/") inside the month's
// prefix store.
func (s *cointrunkSuite) requireCounter(t *testing.T) {
	t.Helper()
	require.Equal(t, []ctCounter{{Key: s.month + "counter/", Counter: helpers.Int64(s.paid)}}, s.anonCounters(t), "anon articles counters")
}

// blockTime reads the header time of the block at height.
func (s *cointrunkSuite) blockTime(t *testing.T, height int64) time.Time {
	t.Helper()
	raw := s.query(t, "block", "--type=height", strconv.FormatInt(height, 10))
	var res struct {
		Header struct {
			Time time.Time `json:"time"`
		} `json:"header"`
		Block struct {
			Header struct {
				Time time.Time `json:"time"`
			} `json:"header"`
		} `json:"block"`
	}
	require.NoError(t, json.Unmarshal(raw, &res), "%s", raw)
	if !res.Header.Time.IsZero() {
		return res.Header.Time
	}
	require.False(t, res.Block.Header.Time.IsZero(), "no header time in %s", raw)
	return res.Block.Header.Time
}

// query runs a query and returns its JSON with every object key normalized
// (normalizeKey: lower case, no underscores), since the CLI prints proto
// names or JSON names depending on the encoder.
func (s *cointrunkSuite) query(t *testing.T, args ...string) []byte {
	t.Helper()
	res, err := helpers.Query[any](s.ctx, s.bze, args...)
	require.NoError(t, err)
	out, err := json.Marshal(normalizeKeys(res))
	require.NoError(t, err)
	return out
}

// decodeNormalized decodes a JSON object (a typed event attribute) with its
// keys normalized like query does.
func decodeNormalized[T any](t *testing.T, raw string) T {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v), "%s", raw)
	b, err := json.Marshal(normalizeKeys(v))
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(b, &out), "%s", raw)
	return out
}

func normalizeKeys(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[normalizeKey(k)] = normalizeKeys(val)
		}
		return out
	case []any:
		for i := range x {
			x[i] = normalizeKeys(x[i])
		}
		return x
	default:
		return v
	}
}
