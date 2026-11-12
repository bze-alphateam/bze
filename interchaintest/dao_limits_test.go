package interchaintest_test

import (
	"net/http"
	"testing"

	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// daoMaxExecutionDepth is x/dao's MaxDaoExecutionDepth: nested DAO
// executions, counting the outermost proposal or admin-execute.
const daoMaxExecutionDepth = 5

// daoDepthGas is the fixed gas limit of the refused six-level execution.
const daoDepthGas = 5_000_000

// E15. Execution limits. On the admin path a DAO's admin cannot sign, as the
// DAO, an authz grant, a fee allowance or an authz exec: the whole
// admin-execute is refused before any of its messages runs. No DAO execution
// may settle a proposal: an admin-execute carrying one is refused, and a
// proposal settling another one ends EXECUTION_FAILED with the fixed reason
// "message not allowed" while the other proposal stays open and settles
// normally. DAO executions nest at most five levels deep. The proposals
// query refuses a status that is not a proposal status.
func (s *daoSuite) executionLimits(t *testing.T) {
	admin, member, g, rcpt := s.user(t), s.user(t), s.user(t), s.user(t)
	d := s.createStatic(t, member, defaultCfg(), addrs(member), admin.FormattedAddress())
	s.fund(t, d.addr, ubze(1_000))
	grantee, r := g.FormattedAddress(), rcpt.FormattedAddress()
	exec := func(ref daoRef, msgs ...map[string]any) []string {
		return append([]string{"dao", "admin-execute", ref.idArg()}, msgArgs(msgs...)...)
	}
	notAllowed := refusal{codespace: daoModule, code: 1133, watch: []string{d.addr, grantee, r}}

	grant := map[string]any{
		"@type": "/cosmos.authz.v1beta1.MsgGrant", "granter": d.addr, "grantee": grantee,
		"grant": map[string]any{"authorization": map[string]any{"@type": "/cosmos.bank.v1beta1.SendAuthorization", "spend_limit": ubze(100)}},
	}
	allowance := map[string]any{
		"@type": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", "granter": d.addr, "grantee": grantee,
		"allowance": map[string]any{"@type": "/cosmos.feegrant.v1beta1.BasicAllowance", "spend_limit": ubze(50_000)},
	}
	t.Run("admin path: authz grant refused", func(t *testing.T) {
		s.refuse(t, admin, notAllowed, exec(d, grant)...)
	})
	t.Run("admin path: fee allowance refused", func(t *testing.T) {
		s.refuse(t, admin, notAllowed, exec(d, allowance)...)
	})
	t.Run("admin path: authz exec refused", func(t *testing.T) {
		s.refuse(t, admin, notAllowed, exec(d, authzExec(d.addr, bankSend(d.addr, r, ubze(1))))...)
	})
	t.Run("admin path: a grant after a valid send refuses both", func(t *testing.T) {
		s.refuse(t, admin, notAllowed, exec(d, bankSend(d.addr, r, ubze(10)), grant)...)
	})

	target := s.submit(t, member, d, "spend", []map[string]any{bankSend(d.addr, r, ubze(5))})
	s.vote(t, member, target, "yes")
	t.Run("admin path: settle refused", func(t *testing.T) {
		s.refuse(t, admin, notAllowed, exec(d, settleMsg(d.addr, target))...)
	})
	t.Run("a proposal cannot settle another one", func(t *testing.T) {
		pid := s.submit(t, member, d, "settle the other", []map[string]any{settleMsg(d.addr, target)})
		res := s.pass(t, pid, []ibc.Wallet{member}, "EXECUTION_FAILED")
		executed := s.event(t, res, "EventProposalExecuted")
		require.Equal(t, "false", executed.Attributes["success"])
		require.Equal(t, "message not allowed", executed.Attributes["error"])
		require.Equal(t, "VOTING", s.prop(t, target).status(), "the other proposal is untouched")
		s.settleFirst(t, member, target, "EXECUTED", flow(d.addr, r, ubze(5)))
	})

	t.Run("nesting depth", func(t *testing.T) {
		// chain[0] is administered by admin, chain[i] by chain[i-1]
		chain := []daoRef{s.createStatic(t, member, defaultCfg(), addrs(member), admin.FormattedAddress())}
		for i := 1; i <= daoMaxExecutionDepth; i++ {
			chain = append(chain, s.createStatic(t, member, defaultCfg(), addrs(member), chain[i-1].addr))
		}
		for _, ref := range chain {
			s.fund(t, ref.addr, ubze(10))
		}
		// nested(n) is an admin-execute on chain[0] reaching chain[n-1]: n
		// DAO executions, the innermost a send by chain[n-1]
		nested := func(n int) []map[string]any {
			inner := bankSend(chain[n-1].addr, r, ubze(1))
			for i := n - 1; i >= 1; i-- {
				inner = adminExecMsg(chain[i-1].addr, chain[i].id, inner)
			}
			return []map[string]any{inner}
		}
		watch := []string{r}
		for _, ref := range chain {
			watch = append(watch, ref.addr)
		}
		// the deepest allowed nesting runs: only the innermost send moves
		// coins, and every DAO on the way reports its own admin execution,
		// innermost first (each is emitted once its message has run)
		res := s.run(t, admin, watch, []daoFlow{flow(chain[daoMaxExecutionDepth-1].addr, r, ubze(1))},
			append([]string{"dao", "admin-execute", chain[0].idArg()}, msgArgs(nested(daoMaxExecutionDepth)...)...)...)
		executed := daoEvents(res, "EventAdminExecuted")
		require.Len(t, executed, daoMaxExecutionDepth, "one EventAdminExecuted per nested execution in tx %s: %+v", res.TxHash, executed)
		for k, e := range executed {
			i := daoMaxExecutionDepth - 1 - k
			wantAdmin := admin.FormattedAddress()
			if i > 0 {
				wantAdmin = chain[i-1].addr
			}
			require.Equal(t, chain[i].idArg(), e.Attributes["id"], "event %d: dao", k)
			require.Equal(t, wantAdmin, e.Attributes["admin"], "event %d: admin of dao %d", k, chain[i].id)
			require.Equal(t, "1", e.Attributes["msg_count"], "event %d: dao %d executed one message", k, chain[i].id)
		}
		// a generous fixed gas limit: six nested executions must reach the
		// depth check, not run out of gas
		args := exec(chain[0], nested(daoMaxExecutionDepth+1)...)
		s.refuseSend(t, admin, refusal{codespace: daoModule, code: 1134, watch: watch}, args, func() (helpers.TxResult, error) {
			return helpers.ExecTxWith(s.ctx, s.bze, admin, []helpers.TxOption{helpers.WithGas(daoDepthGas)}, args...)
		})
	})

	t.Run("proposals query refuses an unknown status", func(t *testing.T) {
		base := "/bze/dao/proposals/" + d.idArg()
		require.Equal(t, http.StatusOK, s.restGet(t, base+"?statuses=VOTING", nil))
		require.Equal(t, http.StatusBadRequest, s.restGet(t, base+"?statuses=99", nil))
		require.Equal(t, http.StatusBadRequest, s.restGet(t, base+"?statuses=VOTING&statuses=99", nil))
	})
}
