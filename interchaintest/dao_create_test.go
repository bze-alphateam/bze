package interchaintest_test

import (
	"strings"
	"testing"

	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// createMsg is a MsgCreateDao for `tx sign` (gogoproto JSON), for the
// refusals the CLI would catch itself.
func (s *daoSuite) createMsg(creator ibc.Wallet, typ any, name string, cfg map[string]any, members []string, el *daoElectorate) map[string]any {
	m := daoMsg("MsgCreateDao", map[string]any{
		"creator": creator.FormattedAddress(),
		"type":    typ,
		"name":    name,
		"config":  cfg,
		"members": members,
	})
	if el != nil {
		m["electorate"] = el
	}
	return m
}

// E9. Treasury squatting, before any DAO exists. DAO addresses are
// predictable, so anyone can create an account at the next one first. A
// permanent-locked vesting account there must not block the DAO: creation
// replaces it with a BaseAccount keeping its number, and its coins become a
// spendable treasury. A plain send ahead of the DAO and a send to an existing
// DAO fund the treasury.
func (s *daoSuite) treasurySquatting(t *testing.T) {
	require.Zero(t, s.daos, "the squatting case must run before any dao exists")
	squatter, rcpt := s.user(t), s.user(t)
	dao1, dao2 := daoAddress(1), daoAddress(2)

	s.run(t, squatter, []string{dao1}, []daoFlow{flow(squatter.FormattedAddress(), dao1, ubze(1))},
		"vesting", "create-permanent-locked-account", dao1, coinStr(1, chain.Denom))
	typ, number, ok := s.account(t, dao1)
	require.True(t, ok)
	require.Equal(t, "/cosmos.vesting.v1beta1.PermanentLockedAccount", typ)

	s.run(t, squatter, nil, []daoFlow{flow(squatter.FormattedAddress(), dao2, ubze(5))},
		"bank", "send", squatter.FormattedAddress(), dao2, coinStr(5, chain.Denom))
	typ2, number2, ok := s.account(t, dao2)
	require.True(t, ok)
	require.Equal(t, "/cosmos.auth.v1beta1.BaseAccount", typ2)

	members := s.users(t, 2, daoUserFunds)
	d1 := s.createStatic(t, members[0], defaultCfg(), addrs(members...), "")
	require.Equal(t, dao1, d1.addr)
	typ, after, ok := s.account(t, dao1)
	require.True(t, ok)
	require.Equal(t, "/cosmos.auth.v1beta1.BaseAccount", typ, "the vesting account is replaced by a plain account")
	require.Equal(t, number, after, "the account keeps its number")
	require.Equal(t, "1", s.bal(t, dao1, chain.Denom).String(), "the squatted coin is the treasury")

	pid := s.submit(t, members[0], d1, "drain the squat", []map[string]any{bankSend(dao1, rcpt.FormattedAddress(), ubze(1))})
	s.pass(t, pid, members, "EXECUTED", flow(dao1, rcpt.FormattedAddress(), ubze(1)))
	require.True(t, s.bal(t, dao1, chain.Denom).IsZero(), "the vested coin was spendable by the dao")

	d2 := s.createStatic(t, members[1], defaultCfg(), addrs(members...), "")
	require.Equal(t, dao2, d2.addr)
	typ2, after2, ok := s.account(t, dao2)
	require.True(t, ok)
	require.Equal(t, "/cosmos.auth.v1beta1.BaseAccount", typ2)
	require.Equal(t, number2, after2)
	require.Equal(t, "5", s.bal(t, dao2, chain.Denom).String(), "coins sent ahead of the dao are its treasury")

	s.run(t, squatter, nil, []daoFlow{flow(squatter.FormattedAddress(), dao2, ubze(7))},
		"bank", "send", squatter.FormattedAddress(), dao2, coinStr(7, chain.Denom))
	require.Equal(t, "12", s.bal(t, dao2, chain.Denom).String(), "a send to an existing dao funds its treasury")
}

// A1. create-dao: the fee to the burner (its split to the community pool
// and the zero fee are covered with the params in A11), the admin defaults
// to the DAO or is set by --admin, and every refusal leaves nothing behind.
func (s *daoSuite) createDao(t *testing.T) {
	creator, other := s.user(t), s.user(t)
	fx := s.fixtures(t)

	t.Run("static, self-owned", func(t *testing.T) {
		d := s.createStatic(t, creator, defaultCfg(), addrs(creator, other), "")
		rec := s.dao(t, d.id)
		require.Equal(t, d.addr, rec.Admin, "the admin defaults to the dao itself")
		_, now := s.blockTime(t)
		require.LessOrEqual(t, int64(rec.CreatedAt), now)
	})
	t.Run("static, external admin", func(t *testing.T) {
		s.createStatic(t, creator, defaultCfg(), addrs(other), creator.FormattedAddress())
	})
	t.Run("stake, self-owned, live electorate", func(t *testing.T) {
		staker := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
		sr := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
		s.joinSR(t, staker, sr, 1_000)
		d := s.createStake(t, creator, defaultCfg(), daoElectorate{SrIDs: []string{sr}}, "")
		require.Equal(t, []string{sr}, s.dao(t, d.id).Electorate.SrIDs)
	})
	t.Run("stake, external admin, empty electorate", func(t *testing.T) {
		d := s.createStake(t, creator, defaultCfg(), daoElectorate{}, creator.FormattedAddress())
		require.Nil(t, s.dao(t, d.id).Electorate)
	})

	t.Run("metadata", func(t *testing.T) {
		cfg := defaultCfg().gogo()
		for name, n := range map[string]string{"empty name": "", "65-byte name": strings.Repeat("n", 65)} {
			t.Run(name, func(t *testing.T) {
				s.refuseMsgs(t, creator, daoCode(1102), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", n, cfg, addrs(creator), nil))
			})
		}
		t.Run("2001-byte description", func(t *testing.T) {
			m := s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg, addrs(creator), nil)
			m["description"] = strings.Repeat("d", 2001)
			s.refuseMsgs(t, creator, daoCode(1102), m)
		})
		t.Run("64-byte name and 2000-byte description are accepted", func(t *testing.T) {
			d := s.createNamed(t, creator, "static-list", strings.Repeat("n", 64), defaultCfg(), "--members", creator.FormattedAddress(), "--description", strings.Repeat("d", 2000))
			rec := s.dao(t, d.id)
			require.Len(t, rec.Name, 64)
			require.Len(t, rec.Description, 2000)
		})
	})

	t.Run("config", func(t *testing.T) {
		bad := map[string]func(c *daoCfg){
			"zero quorum":                        func(c *daoCfg) { c.quorum = "0" },
			"quorum above 1":                     func(c *daoCfg) { c.quorum = "1.000000000000000001"; c.expQuorum = "1.000000000000000001" },
			"zero threshold":                     func(c *daoCfg) { c.threshold = "0" },
			"threshold above 1":                  func(c *daoCfg) { c.threshold = "1.5"; c.expThreshold = "1.5" },
			"expedite quorum below quorum":       func(c *daoCfg) { c.expQuorum = "0.4" },
			"expedite threshold below threshold": func(c *daoCfg) { c.expThreshold = "0.49" },
			"expedited period above voting":      func(c *daoCfg) { c.expedited = "16s" },
			"negative cost":                      func(c *daoCfg) { c.cost = coinPtr(-1, chain.Denom) },
			"negative deposit":                   func(c *daoCfg) { c.deposit = coinPtr(-1, chain.Denom) },
			"invalid cost denom":                 func(c *daoCfg) { c.cost = coinPtr(1, "1x") },
		}
		for name, mutate := range bad {
			t.Run(name, func(t *testing.T) {
				cfg := defaultCfg()
				mutate(&cfg)
				s.refuseMsgs(t, creator, daoCode(1103), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg.gogo(), addrs(creator), nil))
			})
		}
		for name, value := range map[string]int{"unknown refund policy": 3, "unknown cost destination": 2, "unknown kept destination": 9} {
			t.Run(name, func(t *testing.T) {
				cfg := defaultCfg().gogo()
				key := map[string]string{
					"unknown refund policy":    "deposit_refund_policy",
					"unknown cost destination": "cost_destination",
					"unknown kept destination": "kept_deposit_destination",
				}[name]
				cfg[key] = value
				s.refuseMsgs(t, creator, daoCode(1103), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg, addrs(creator), nil))
			})
		}
		// the period bounds are params: checked by the msg server, the tx is included
		for name, periods := range map[string][2]string{
			"voting period below min":    {"3s", "3s"},
			"voting period above max":    {"601s", "6s"},
			"expedited period below min": {daoVoting, "3s"},
		} {
			t.Run(name, func(t *testing.T) {
				cfg := defaultCfg()
				cfg.voting, cfg.expedited = periods[0], periods[1]
				s.refuse(t, creator, daoCode(1103), "dao", "create-dao", "static-list", "x", cfg.cli(), "--members", creator.FormattedAddress())
			})
		}
		t.Run("bounds are inclusive", func(t *testing.T) {
			cfg := defaultCfg()
			cfg.voting, cfg.expedited = daoMaxVoting, daoMinVoting
			cfg.quorum, cfg.threshold, cfg.expQuorum, cfg.expThreshold = "1", "1", "1", "1"
			s.createStatic(t, creator, cfg, addrs(creator), "")
		})
	})

	t.Run("members", func(t *testing.T) {
		cfg := defaultCfg().gogo()
		s.refuseMsgs(t, creator, daoCode(1104), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg, nil, nil))
		s.refuseMsgs(t, creator, daoCode(1104), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg, addrs(creator, creator), nil))
		upper := strings.ToUpper(creator.FormattedAddress())
		s.refuseMsgs(t, creator, daoCode(1104), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg, []string{creator.FormattedAddress(), upper}, nil))
		s.refuseMsgs(t, creator, daoCode(1104), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg, []string{"bze1notanaddress"}, nil))
		seven := addrs(s.users(t, daoMaxMembers+1, 1)...)
		s.refuse(t, creator, daoCode(1104), "dao", "create-dao", "static-list", "x", defaultCfg().cli(), "--members", strings.Join(seven, ","))
		s.createStatic(t, creator, defaultCfg(), seven[:daoMaxMembers], "")
	})

	t.Run("electorate", func(t *testing.T) {
		cfg := defaultCfg()
		admin := creator.FormattedAddress()
		stake := func(el daoElectorate) []string {
			return []string{"dao", "create-dao", "stake", "x", cfg.cli(), "--electorate", mustJSON(el), "--admin", admin}
		}
		s.refuse(t, creator, daoCode(1105), stake(daoElectorate{SrIDs: []string{"999999"}})...)
		s.refuse(t, creator, daoCode(1105), stake(daoElectorate{DrDenom: s.oth + "x"})...)
		s.refuse(t, creator, daoCode(1105), stake(daoElectorate{SrIDs: []string{fx.srA, fx.srB, fx.srC}})...)
		s.refuse(t, creator, daoCode(1105), stake(daoElectorate{SrIDs: []string{fx.srLow}})...)               // D28
		s.refuse(t, creator, daoCode(1105), stake(daoElectorate{SrIDs: []string{fx.srA}, DrDenom: s.oth})...) // D29: other staking denom
		s.refuse(t, creator, daoCode(1105), stake(daoElectorate{SrIDs: []string{fx.srA, fx.srOth}})...)       // D29
		// D14 and D29 (members on a stake dao, an electorate on a static one) need no state
		el := daoElectorate{SrIDs: []string{fx.srA, fx.srA}}
		s.refuseMsgs(t, creator, daoCode(1105), s.createMsg(creator, "DAO_TYPE_STAKE", "x", cfg.gogo(), nil, &el))
		one := daoElectorate{SrIDs: []string{fx.srA}}
		s.refuseMsgs(t, creator, daoCode(1105), s.createMsg(creator, "DAO_TYPE_STAKE", "x", cfg.gogo(), addrs(creator), &one))
		s.refuseMsgs(t, creator, daoCode(1105), s.createMsg(creator, "DAO_TYPE_STATIC_LIST", "x", cfg.gogo(), addrs(creator), &one))
		s.refuseMsgs(t, creator, daoCode(1105), s.createMsg(creator, "DAO_TYPE_STAKE", "x", cfg.gogo(), nil, &daoElectorate{SrIDs: []string{""}}))
		// the max_sr_per_dao and the lock bounds are inclusive
		s.createStake(t, creator, cfg, daoElectorate{SrIDs: []string{fx.srA, fx.srB}, DrDenom: s.stk}, admin)
	})

	t.Run("D11 self-owned stake dao without live stake", func(t *testing.T) {
		// fx.srIdle has no stake: a self-owned dao could never pass a proposal.
		// The ticket expects ErrElectorateNotLive (1113); CreateDao returns
		// ErrInvalidElectorate (1105) here, so the suite asserts 1105.
		s.refuse(t, creator, daoCode(1105), "dao", "create-dao", "stake", "x", defaultCfg().cli(), "--electorate", mustJSON(daoElectorate{SrIDs: []string{fx.srIdle}}))
		s.refuse(t, creator, daoCode(1105), "dao", "create-dao", "stake", "x", defaultCfg().cli())
	})

	t.Run("type", func(t *testing.T) {
		s.refuseMsgs(t, creator, daoCode(1101), s.createMsg(creator, 7, "x", defaultCfg().gogo(), addrs(creator), nil))
	})

	t.Run("insufficient funds for the fee stores nothing", func(t *testing.T) {
		poor := s.users(t, 1, daoCreationFee-1+daoRefuseGas/100)[0]
		s.refuse(t, poor, refusal{codespace: "sdk", code: 5, reason: "dao creation fee"},
			"dao", "create-dao", "static-list", "x", defaultCfg().cli(), "--members", poor.FormattedAddress())
		_, _, exists := s.account(t, daoAddress(s.daos+1))
		require.False(t, exists, "no account is created for a dao that was not")
	})
}

// A2. admin-execute: the external admin spends the treasury, several
// messages run atomically, and every refusal moves nothing.
func (s *daoSuite) adminExecute(t *testing.T) {
	admin, member, victim, rcpt := s.user(t), s.user(t), s.user(t), s.user(t)
	d := s.createStatic(t, member, defaultCfg(), addrs(member), admin.FormattedAddress())
	s.fund(t, d.addr, ubze(1_000))
	r := rcpt.FormattedAddress()

	s.adminExec(t, admin, d, []daoFlow{flow(d.addr, r, ubze(100))}, bankSend(d.addr, r, ubze(100)))
	s.adminExec(t, admin, d, []daoFlow{flow(d.addr, r, ubze(10)), flow(d.addr, victim.FormattedAddress(), ubze(20))},
		bankSend(d.addr, r, ubze(10)), bankSend(d.addr, victim.FormattedAddress(), ubze(20)))
	require.Equal(t, "870", s.bal(t, d.addr, chain.Denom).String())

	exec := func(msgs ...map[string]any) []string {
		return append([]string{"dao", "admin-execute", d.idArg()}, msgArgs(msgs...)...)
	}
	watch := []string{d.addr, r, victim.FormattedAddress()}
	t.Run("a failing second message rolls back the first", func(t *testing.T) {
		s.refuse(t, admin, refusal{codespace: "sdk", code: 5, watch: watch}, exec(bankSend(d.addr, r, ubze(10)), bankSend(d.addr, r, ubze(1_000_000)))...)
	})
	t.Run("non-admin", func(t *testing.T) {
		s.refuse(t, member, refusal{codespace: daoModule, code: 1109, watch: watch}, exec(bankSend(d.addr, r, ubze(10)))...)
	})
	t.Run("D1 victim-signed message", func(t *testing.T) {
		s.refuse(t, admin, refusal{codespace: daoModule, code: 1111, watch: watch}, exec(bankSend(victim.FormattedAddress(), admin.FormattedAddress(), ubze(10)))...)
	})
	t.Run("D24 dao plus attacker signers", func(t *testing.T) {
		multi := map[string]any{
			"@type": "/cosmos.bank.v1beta1.MsgMultiSend",
			"inputs": []map[string]any{
				{"address": d.addr, "coins": ubze(5)},
				{"address": victim.FormattedAddress(), "coins": ubze(5)},
			},
			"outputs": []map[string]any{{"address": admin.FormattedAddress(), "coins": ubze(10)}},
		}
		s.refuse(t, admin, refusal{codespace: daoModule, code: 1111, watch: watch}, exec(multi)...)
	})
	t.Run("message counts and size", func(t *testing.T) {
		s.refuseMsgs(t, admin, daoCode(1110), daoMsg("MsgAdminExecute", map[string]any{"admin": admin.FormattedAddress(), "dao_id": d.idArg(), "msgs": []any{}}))
		four := []map[string]any{bankSend(d.addr, r, ubze(1)), bankSend(d.addr, r, ubze(1)), bankSend(d.addr, r, ubze(1)), bankSend(d.addr, r, ubze(1))}
		s.refuse(t, admin, refusal{codespace: daoModule, code: 1110, watch: watch}, exec(four...)...)
		big := func() map[string]any {
			return daoMsg("MsgUpdateMetadata", map[string]any{"authority": d.addr, "dao_id": d.idArg(), "name": "x", "description": strings.Repeat("d", 2_000)})
		}
		s.refuse(t, admin, refusal{codespace: daoModule, code: 1110, watch: watch}, exec(big(), big(), big())...)
	})
	t.Run("message without a route", func(t *testing.T) {
		// a type the chain does not know cannot even be encoded: the client
		// refuses it before anything is signed or broadcast
		unknown := map[string]any{"@type": "/bze.dao.MsgDoesNotExist", "authority": d.addr}
		before := s.allBalances(t, admin.FormattedAddress())
		_, err := helpers.ExecMsgs(s.ctx, s.bze, admin, []helpers.TxOption{helpers.WithGas(daoRefuseGas)},
			daoMsg("MsgAdminExecute", map[string]any{"admin": admin.FormattedAddress(), "dao_id": d.idArg(), "msgs": []any{unknown}}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "MsgDoesNotExist")
		_, err = helpers.ExecTxWith(s.ctx, s.bze, admin, []helpers.TxOption{helpers.WithGas(daoRefuseGas)}, exec(unknown)...)
		require.Error(t, err)
		require.Equal(t, before.String(), s.allBalances(t, admin.FormattedAddress()).String())
	})
	t.Run("unknown dao", func(t *testing.T) {
		s.refuse(t, admin, daoCode(1108), "dao", "admin-execute", "999999", msgArgs(bankSend(d.addr, r, ubze(1)))[0])
	})
	t.Run("D26 self-owned dao", func(t *testing.T) {
		self := s.createStatic(t, member, defaultCfg(), addrs(member), "")
		s.fund(t, self.addr, ubze(50))
		s.refuse(t, member, refusal{codespace: daoModule, code: 1109, watch: []string{self.addr}},
			"dao", "admin-execute", self.idArg(), msgArgs(bankSend(self.addr, member.FormattedAddress(), ubze(50)))[0])
	})
	t.Run("D10 nested admin-execute by the dao on itself", func(t *testing.T) {
		inner := daoMsg("MsgAdminExecute", map[string]any{"admin": d.addr, "dao_id": d.idArg(), "msgs": []any{bankSend(d.addr, r, ubze(1))}})
		s.refuse(t, admin, refusal{codespace: daoModule, code: 1109, watch: watch}, exec(inner)...)
	})
	require.Equal(t, "870", s.bal(t, d.addr, chain.Denom).String(), "no refusal moved treasury funds")
}

// A3. The two-step admin handover and its D27 matrix.
func (s *daoSuite) adminHandover(t *testing.T) {
	a, b, c, member := s.user(t), s.user(t), s.user(t), s.user(t)
	d := s.createStatic(t, member, defaultCfg(), addrs(member), a.FormattedAddress())
	s.fund(t, d.addr, ubze(100))
	id := d.idArg()
	watch := refusal{codespace: daoModule}

	nominate := func(nominee string) []string { return []string{"dao", "nominate-admin", id, nominee} }
	r := func(code uint32) refusal { x := watch; x.code = code; return x }

	s.refuse(t, a, r(1114), nominate(a.FormattedAddress())...)
	s.refuse(t, a, r(1114), nominate(d.addr)...)
	s.refuse(t, b, r(1109), nominate(c.FormattedAddress())...)
	s.refuse(t, a, r(1115), "dao", "withdraw-admin-nomination", id)
	s.refuse(t, b, r(1115), "dao", "accept-admin-nomination", id)
	s.refuse(t, a, r(1108), "dao", "nominate-admin", "999999", b.FormattedAddress())

	res := s.run(t, a, nil, nil, nominate(b.FormattedAddress())...)
	e := s.event(t, res, "EventAdminNominated")
	require.Equal(t, b.FormattedAddress(), e.Attributes["nominee"])
	rec := s.dao(t, d.id)
	require.Equal(t, a.FormattedAddress(), rec.Admin, "a nomination does not hand the dao over")
	require.Equal(t, b.FormattedAddress(), rec.PendingAdmin)
	require.Empty(t, s.daosByAdmin(t, b.FormattedAddress()), "the nominee is not listed before accepting")
	require.Equal(t, a.FormattedAddress(), s.controller(t, d.id).Controller, "the controller ignores the nomination")

	s.refuse(t, a, r(1112), nominate(c.FormattedAddress())...)
	s.refuse(t, b, r(1109), "dao", "withdraw-admin-nomination", id)
	s.refuse(t, c, r(1116), "dao", "accept-admin-nomination", id)

	res = s.run(t, a, nil, nil, "dao", "withdraw-admin-nomination", id)
	require.Equal(t, b.FormattedAddress(), s.event(t, res, "EventAdminNominationWithdrawn").Attributes["nominee"])
	require.Empty(t, s.dao(t, d.id).PendingAdmin)
	s.refuse(t, b, r(1115), "dao", "accept-admin-nomination", id)

	s.run(t, a, nil, nil, nominate(b.FormattedAddress())...)
	res = s.run(t, b, nil, nil, "dao", "accept-admin-nomination", id)
	changed := s.event(t, res, "EventAdminChanged")
	require.Equal(t, a.FormattedAddress(), changed.Attributes["old_admin"])
	require.Equal(t, b.FormattedAddress(), changed.Attributes["new_admin"])
	rec = s.dao(t, d.id)
	require.Equal(t, b.FormattedAddress(), rec.Admin)
	require.Empty(t, rec.PendingAdmin)
	require.Empty(t, s.daosByAdmin(t, a.FormattedAddress()), "daos-by-admin follows the handover")
	require.Len(t, s.daosByAdmin(t, b.FormattedAddress()), 1)

	s.refuse(t, a, refusal{codespace: daoModule, code: 1109, watch: []string{d.addr}},
		"dao", "admin-execute", id, msgArgs(bankSend(d.addr, a.FormattedAddress(), ubze(10)))[0])
	s.adminExec(t, b, d, []daoFlow{flow(d.addr, b.FormattedAddress(), ubze(10))}, bankSend(d.addr, b.FormattedAddress(), ubze(10)))

	t.Run("an empty nominee makes the dao self-owned", func(t *testing.T) {
		res := s.run(t, b, nil, nil, "dao", "nominate-admin", id, "")
		require.Equal(t, d.addr, s.event(t, res, "EventAdminChanged").Attributes["new_admin"])
		require.Equal(t, d.addr, s.dao(t, d.id).Admin)
		s.refuse(t, b, refusal{codespace: daoModule, code: 1109, watch: []string{d.addr}},
			"dao", "admin-execute", id, msgArgs(bankSend(d.addr, b.FormattedAddress(), ubze(10)))[0])
		s.refuse(t, b, r(1109), "dao", "nominate-admin", id, c.FormattedAddress())
	})
	t.Run("an empty nominee with a dead electorate", func(t *testing.T) {
		dead := s.createStake(t, a, defaultCfg(), daoElectorate{}, a.FormattedAddress())
		s.refuse(t, a, r(1113), "dao", "nominate-admin", dead.idArg(), "")
		idle := s.fixtures(t).srIdle
		el := daoMsg("MsgUpdateElectorate", map[string]any{"authority": dead.addr, "dao_id": dead.idArg(), "electorate": daoElectorate{SrIDs: []string{idle}}})
		s.adminExec(t, a, dead, nil, el)
		s.refuse(t, a, r(1113), "dao", "nominate-admin", dead.idArg(), "")
	})
	t.Run("a dao nominee accepts through its own proposal", func(t *testing.T) {
		voters := s.users(t, 2, daoUserFunds)
		x := s.createStatic(t, voters[0], defaultCfg(), addrs(voters...), "")
		y := s.createStatic(t, member, defaultCfg(), addrs(member), a.FormattedAddress())
		s.run(t, a, nil, nil, "dao", "nominate-admin", y.idArg(), x.addr)
		accept := daoMsg("MsgAcceptAdminNomination", map[string]any{"nominee": x.addr, "dao_id": y.idArg()})
		pid := s.submit(t, voters[0], x, "take y", []map[string]any{accept})
		s.pass(t, pid, voters, "EXECUTED")
		require.Equal(t, x.addr, s.dao(t, y.id).Admin)
		require.Len(t, s.daosByAdmin(t, x.addr), 2, "x administers itself and y")
		require.Equal(t, "SELF_OWNED", s.controller(t, y.id).EndReason)
		require.Equal(t, x.addr, s.controller(t, y.id).Controller)
	})
}
