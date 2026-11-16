package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

func (e *env) expectBlacklisted(blacklisted bool) {
	e.rl.EXPECT().IsDenomBlacklisted(gomock.Any(), testDenom).Return(blacklisted)
}

// expectSupply answers the bank's supply check of the denom.
func (e *env) expectSupply(denom string, exists bool) {
	e.bank.EXPECT().HasSupply(gomock.Any(), denom).Return(exists)
}

func eventTypes(ctx sdk.Context) []string {
	var out []string
	for _, ev := range ctx.EventManager().Events() {
		out = append(out, ev.Type)
	}
	return out
}

func TestBlockDenom_GuardianAndAuthorityBlock(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.expectSupply(testDenom, true)
			e.expectBlacklisted(false)
			e.rl.EXPECT().AddDenomToBlacklist(gomock.Any(), testDenom).Times(1)

			_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: signer, Denom: testDenom})
			require.NoError(t, err)
			require.Equal(t, []string{"bze.guardian.v1.DenomBlockedEvent"}, eventTypes(e.ctx))
		})
	}
}

func TestBlockDenom_BlocksWithoutTouchingRateLimits(t *testing.T) {
	e := newEnv(t)
	e.expectSupply(testDenom, true)
	e.expectBlacklisted(false)
	e.rl.EXPECT().AddDenomToBlacklist(gomock.Any(), testDenom)
	// GetRateLimit / SetRateLimit have no expectation: a call fails the test

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: guardianAddr, Denom: testDenom})
	require.NoError(t, err)
}

func TestBlockDenom_StrangerIsRefusedAndNothingIsCalled(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: strangerAddr, Denom: testDenom})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Empty(t, eventTypes(e.ctx))
}

func TestBlockDenom_EmptyGuardianLeavesOnlyTheAuthority(t *testing.T) {
	e := newEnv(t)
	require.NoError(t, e.k.SetParams(e.ctx, types.DefaultParams()))

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: guardianAddr, Denom: testDenom})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	e.expectSupply(testDenom, true)
	e.expectBlacklisted(false)
	e.rl.EXPECT().AddDenomToBlacklist(gomock.Any(), testDenom)
	_, err = e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: authority, Denom: testDenom})
	require.NoError(t, err)
}

// An uppercase spelling of the guardian address is the same account, and a
// different string: the comparison is on the decoded bytes.
func TestBlockDenom_SignerIsComparedOnBytes(t *testing.T) {
	e := newEnv(t)
	e.expectSupply(testDenom, true)
	e.expectBlacklisted(false)
	e.rl.EXPECT().AddDenomToBlacklist(gomock.Any(), testDenom)

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: strings.ToUpper(guardianAddr), Denom: testDenom})
	require.NoError(t, err)
}

func TestBlockDenom_AlreadyBlockedIsRefusedWithoutEvent(t *testing.T) {
	e := newEnv(t)
	e.expectSupply(testDenom, true)
	e.expectBlacklisted(true)
	// no AddDenomToBlacklist expectation

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: guardianAddr, Denom: testDenom})
	require.ErrorIs(t, err, types.ErrAlreadyBlocked)
	require.Empty(t, eventTypes(e.ctx))
}

func TestBlockDenom_InvalidMessageIsRefusedBeforeAnythingElse(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: guardianAddr, Denom: "not a denom"})
	require.ErrorIs(t, err, types.ErrInvalidDenom)

	_, err = e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: "nope", Denom: testDenom})
	require.Error(t, err)
}

func TestBlockDenom_WithoutRateLimitKeeperIsRefused(t *testing.T) {
	k, ctx := newBareKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.NewParams(guardianAddr)))

	_, err := keeper.NewMsgServerImpl(&k).BlockDenom(ctx, &types.MsgBlockDenom{Signer: guardianAddr, Denom: testDenom})
	require.ErrorIs(t, err, types.ErrRateLimitKeeperNotSet)
}

func TestUnblockDenom_AuthorityUnblocks(t *testing.T) {
	e := newEnv(t)
	e.expectBlacklisted(true)
	e.rl.EXPECT().RemoveDenomFromBlacklist(gomock.Any(), testDenom).Times(1)

	_, err := e.msgServer.UnblockDenom(e.ctx, &types.MsgUnblockDenom{Authority: authority, Denom: testDenom})
	require.NoError(t, err)
	require.Equal(t, []string{"bze.guardian.v1.DenomUnblockedEvent"}, eventTypes(e.ctx))
}

// The guardian can stop a denom but never reopen it, with or without the
// denom being blocked; nothing reaches the ratelimit keeper.
func TestUnblockDenom_GuardianAndStrangerAreRefusedAndNothingIsCalled(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "stranger": strangerAddr} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)

			_, err := e.msgServer.UnblockDenom(e.ctx, &types.MsgUnblockDenom{Authority: signer, Denom: testDenom})
			require.ErrorIs(t, err, types.ErrUnauthorized)
			require.Empty(t, eventTypes(e.ctx))
		})
	}
}

func TestUnblockDenom_NotBlockedIsRefusedWithoutEvent(t *testing.T) {
	e := newEnv(t)
	e.expectBlacklisted(false)
	// no RemoveDenomFromBlacklist expectation

	_, err := e.msgServer.UnblockDenom(e.ctx, &types.MsgUnblockDenom{Authority: authority, Denom: testDenom})
	require.ErrorIs(t, err, types.ErrNotBlocked)
	require.Empty(t, eventTypes(e.ctx))
}

func TestUnblockDenom_InvalidMessageIsRefused(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.UnblockDenom(e.ctx, &types.MsgUnblockDenom{Authority: authority, Denom: ""})
	require.ErrorIs(t, err, types.ErrInvalidDenom)
}

func TestUnblockDenom_WithoutRateLimitKeeperIsRefused(t *testing.T) {
	k, ctx := newBareKeeper(t)

	_, err := keeper.NewMsgServerImpl(&k).UnblockDenom(ctx, &types.MsgUnblockDenom{Authority: authority, Denom: testDenom})
	require.ErrorIs(t, err, types.ErrRateLimitKeeperNotSet)
}

func TestBlockedDenoms_PassesTheBlacklistThrough(t *testing.T) {
	e := newEnv(t)
	e.rl.EXPECT().GetAllBlacklistedDenoms(gomock.Any()).Return([]string{"ubze", "ibc/AA"})

	res, err := e.k.BlockedDenoms(e.ctx, &types.QueryBlockedDenomsRequest{})
	require.NoError(t, err)
	require.Equal(t, []string{"ubze", "ibc/AA"}, res.Denoms)

	_, err = e.k.BlockedDenoms(e.ctx, nil)
	require.Error(t, err)
}

// Only a denom with a supply on chain can be blocked. The blacklist is matched
// on the exact denom string, so a lowercase hash, a base name or a trace path
// of an IBC denom, or a denom that does not exist, would block nothing: each
// is refused before the blacklist is read or written, and no event is emitted.
func TestBlockDenom_DenomWithoutSupplyIsRefused(t *testing.T) {
	for name, denom := range map[string]string{
		"lowercase ibc hash": "ibc/6490a7eab61059bfc1cddeb05917dd70bdf3a611654162a1a47db930d40d8af4",
		"base name":          "uusdc",
		"trace path":         "transfer/channel-3/uusdc",
		"unknown denom":      "factory/bze1unknown/nothing",
	} {
		t.Run(name, func(t *testing.T) {
			for _, signer := range []string{guardianAddr, authority} {
				e := newEnv(t)
				e.expectSupply(denom, false)
				// no IsDenomBlacklisted / AddDenomToBlacklist expectation

				_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: signer, Denom: denom})
				require.ErrorIs(t, err, types.ErrDenomNotFound)
				require.Empty(t, eventTypes(e.ctx))
			}
		})
	}
}

// An existing IBC denom, spelled as the chain stores it, is blocked.
func TestBlockDenom_ExistingIBCDenomIsBlocked(t *testing.T) {
	const denom = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"
	e := newEnv(t)
	e.expectSupply(denom, true)
	e.rl.EXPECT().IsDenomBlacklisted(gomock.Any(), denom).Return(false)
	e.rl.EXPECT().AddDenomToBlacklist(gomock.Any(), denom).Times(1)

	_, err := e.msgServer.BlockDenom(e.ctx, &types.MsgBlockDenom{Signer: guardianAddr, Denom: denom})
	require.NoError(t, err)
	require.Equal(t, []string{"bze.guardian.v1.DenomBlockedEvent"}, eventTypes(e.ctx))
}

// Governance can lift any entry of the blacklist, whether the denom has a
// supply or not: the unblock never asks the bank.
func TestUnblockDenom_DoesNotCheckTheSupply(t *testing.T) {
	const denom = "uusdc"
	e := newEnv(t)
	// no HasSupply expectation
	e.rl.EXPECT().IsDenomBlacklisted(gomock.Any(), denom).Return(true)
	e.rl.EXPECT().RemoveDenomFromBlacklist(gomock.Any(), denom).Times(1)

	_, err := e.msgServer.UnblockDenom(e.ctx, &types.MsgUnblockDenom{Authority: authority, Denom: denom})
	require.NoError(t, err)
}
