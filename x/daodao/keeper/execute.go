package keeper

import (
	"bytes"
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// validateProposalMsgSigners enforces the Epic-5 invariant on every
// element of a proposal's msgs[]:
//
//   - The Any decodes to a registered sdk.Msg in the codec.
//   - The message claims exactly one signer.
//   - That signer equals the DAO's account address.
//
// Multi-signer messages are rejected because the proposal mechanism can
// only attach a single authorizing party (the DAO); a second signer's
// authorization would have to come from elsewhere.
//
// The check is stateless wrt chain data (DAO account address is derived
// from dao_id) but uses the keeper's Codec.GetMsgAnySigners — which
// works uniformly on Anys whether their cached value is populated
// (fresh from a tx) or not (loaded from store, where MustUnmarshal
// doesn't auto-unpack Any.cachedValue).
//
// Called at MsgCreateProposal (so bad bundles never reach VOTING) AND
// at MsgExecuteProposal (defense-in-depth — guards against a chain
// upgrade changing signer semantics for an in-flight proposal).
func (k Keeper) validateProposalMsgSigners(daoID uint64, msgs []*cdctypes.Any) error {
	expected := types.DaoAccountAddress(daoID).Bytes()
	// Pre-compute the canonical type URL for the bundle-msg denylist. We
	// derive it from a freshly-allocated sentinel rather than hardcoding
	// the proto path so a future rename / move stays self-correcting.
	disallowedTypeURL := sdk.MsgTypeURL(&types.MsgExecuteProposal{})

	for i, anyMsg := range msgs {
		if anyMsg == nil {
			return errorsmod.Wrapf(types.ErrInvalidProposalSigners, "msgs[%d] is nil", i)
		}
		// Bundle-msg denylist: MsgExecuteProposal inside a bundle re-enters
		// the dispatcher (outer proposal is still PASSED — status only
		// flips to EXECUTED after dispatch returns). A bundle that
		// targets its own proposal-id, or two proposals that target each
		// other, would recurse until gas / stack exhaustion. There's no
		// legitimate use case for an executable proposal that runs ANOTHER
		// executable proposal — that's what depositing+voting on the
		// second proposal directly is for.
		if anyMsg.TypeUrl == disallowedTypeURL {
			return errorsmod.Wrapf(types.ErrBundleMsgTypeNotAllowed,
				"msgs[%d]: %s is not allowed in proposal bundles (would re-enter the dispatcher)",
				i, anyMsg.TypeUrl)
		}
		signers, _, err := k.cdc.GetMsgAnySigners(anyMsg)
		if err != nil {
			return errorsmod.Wrapf(types.ErrInvalidProposalSigners,
				"msgs[%d] (%s): %s", i, anyMsg.TypeUrl, err.Error())
		}
		if len(signers) != 1 {
			return errorsmod.Wrapf(types.ErrInvalidProposalSigners,
				"msgs[%d] (%s): expected exactly 1 signer, got %d",
				i, anyMsg.TypeUrl, len(signers))
		}
		if !bytes.Equal(signers[0], expected) {
			return errorsmod.Wrapf(types.ErrInvalidProposalSigners,
				"msgs[%d] (%s): signer %s != DAO account %s",
				i, anyMsg.TypeUrl,
				sdk.AccAddress(signers[0]).String(),
				sdk.AccAddress(expected).String())
		}
	}
	return nil
}

// dispatchProposalMsgs runs every entry of `msgs` through the wired
// MsgServiceRouter inside a cached context. Atomicity:
//
//   - All writes happen against ctxCache.
//   - If ANY handler errors, the cache is dropped (write() not called)
//     and an error is returned. The proposal's status stays PASSED so
//     a retry is possible after fixing the underlying precondition
//     (e.g., topping up the DAO's spendable balance for a bank.MsgSend).
//   - On full success, write() commits the cache.
//
// Returns the index of the failed message (-1 on success) and the error,
// so the caller can emit a structured "execution_failed" event with the
// offending index + type URL.
func (k Keeper) dispatchProposalMsgs(ctx context.Context, msgs []*cdctypes.Any) (failedIdx int, err error) {
	if k.msgRouter == nil {
		return -1, fmt.Errorf("daodao msgRouter is not wired; app.go must call SetMsgRouter after baseapp build")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, write := sdkCtx.CacheContext()

	for i, anyMsg := range msgs {
		// Proposals loaded from store have anyMsg.cachedValue == nil
		// (MustUnmarshal doesn't auto-unpack Anys). Prefer the cached
		// value if it's already populated; otherwise unpack via the
		// interface registry.
		var decoded sdk.Msg
		if cached, ok := anyMsg.GetCachedValue().(sdk.Msg); ok && cached != nil {
			decoded = cached
		} else {
			if err := k.cdc.UnpackAny(anyMsg, &decoded); err != nil {
				return i, errorsmod.Wrapf(types.ErrUnknownMsgType,
					"msgs[%d] (%s): %s", i, anyMsg.TypeUrl, err.Error())
			}
		}
		handler := k.msgRouter.Handler(decoded)
		if handler == nil {
			return i, errorsmod.Wrapf(types.ErrUnknownMsgType,
				"msgs[%d]: no handler for type %s", i, anyMsg.TypeUrl)
		}
		if _, err := handler(cacheCtx, decoded); err != nil {
			// Drop the cache; ALL writes from this and prior msgs roll back.
			return i, err
		}
	}
	write()
	return -1, nil
}
