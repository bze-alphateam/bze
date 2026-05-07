package app

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	txfeekeeper "github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	txfeetypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// feeChargingWasmMsgServer wraps wasmd's MsgServer to charge fees on every
// CosmWasm code-upload and instantiation path. The wrapper covers:
//
//	Deploy (CwDeployFee):
//	  - MsgStoreCode
//	  - MsgStoreAndInstantiateContract  (also pays instantiate fee)
//	  - MsgStoreAndMigrateContract
//
//	Instantiate (CwInstantiateFee):
//	  - MsgInstantiateContract
//	  - MsgInstantiateContract2
//	  - MsgStoreAndInstantiateContract  (also pays deploy fee)
//
// Every dispatch route — direct user txs, authz exec, contract-issued
// submessages (Stargate `Any`), ICA host packet execution, and gov
// proposals — funnels through the wasm MsgServer registered on the
// MsgServiceRouter, so wrapping that server is the only universal capture
// point. An ante decorator only catches direct user txs.
//
// Fee semantics:
//   - paid by the msg actor (Sender for StoreCode/InstantiateContract*,
//     Authority for the StoreAnd* combined msgs)
//   - charged in native denom; no spot-price conversion
//   - skipped when the actor is the wasm gov authority (gov proposals)
//   - destination follows txfeecollector params (stakers / burner /
//     community pool); both fees use the same destination
type feeChargingWasmMsgServer struct {
	wasmtypes.MsgServer // embedded — non-upload methods inherit unchanged

	txfeeKeeper *txfeekeeper.Keeper
	bankKeeper  txfeetypes.BankKeeper

	// govAuthority is exempt: gov-dispatched msgs have the gov module
	// account as signer, which has no user-facing balance and shouldn't
	// be charged.
	govAuthority string
}

func NewFeeChargingWasmMsgServer(
	inner wasmtypes.MsgServer,
	txfeeKeeper *txfeekeeper.Keeper,
	bankKeeper txfeetypes.BankKeeper,
	govAuthority string,
) wasmtypes.MsgServer {
	return &feeChargingWasmMsgServer{
		MsgServer:    inner,
		txfeeKeeper:  txfeeKeeper,
		bankKeeper:   bankKeeper,
		govAuthority: govAuthority,
	}
}

// --- Code upload paths ---

func (s *feeChargingWasmMsgServer) StoreCode(ctx context.Context, msg *wasmtypes.MsgStoreCode) (*wasmtypes.MsgStoreCodeResponse, error) {
	if err := s.chargeDeployFee(ctx, msg.Sender); err != nil {
		return nil, err
	}
	return s.MsgServer.StoreCode(ctx, msg)
}

func (s *feeChargingWasmMsgServer) StoreAndInstantiateContract(ctx context.Context, msg *wasmtypes.MsgStoreAndInstantiateContract) (*wasmtypes.MsgStoreAndInstantiateContractResponse, error) {
	// Combined msg → pay BOTH fees.
	if err := s.chargeDeployFee(ctx, msg.Authority); err != nil {
		return nil, err
	}
	if err := s.chargeInstantiateFee(ctx, msg.Authority); err != nil {
		return nil, err
	}
	return s.MsgServer.StoreAndInstantiateContract(ctx, msg)
}

func (s *feeChargingWasmMsgServer) StoreAndMigrateContract(ctx context.Context, msg *wasmtypes.MsgStoreAndMigrateContract) (*wasmtypes.MsgStoreAndMigrateContractResponse, error) {
	if err := s.chargeDeployFee(ctx, msg.Authority); err != nil {
		return nil, err
	}
	return s.MsgServer.StoreAndMigrateContract(ctx, msg)
}

// --- Instantiation paths ---

func (s *feeChargingWasmMsgServer) InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	if err := s.chargeInstantiateFee(ctx, msg.Sender); err != nil {
		return nil, err
	}
	return s.MsgServer.InstantiateContract(ctx, msg)
}

func (s *feeChargingWasmMsgServer) InstantiateContract2(ctx context.Context, msg *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error) {
	if err := s.chargeInstantiateFee(ctx, msg.Sender); err != nil {
		return nil, err
	}
	return s.MsgServer.InstantiateContract2(ctx, msg)
}

// --- Charging helpers ---

func (s *feeChargingWasmMsgServer) chargeDeployFee(ctx context.Context, actorBech32 string) error {
	params := s.txfeeKeeper.GetParams(ctx)
	return s.chargeFee(ctx, actorBech32, params.CwDeployFee, params.CwDeployFeeDestination, "cw_deploy_fee")
}

func (s *feeChargingWasmMsgServer) chargeInstantiateFee(ctx context.Context, actorBech32 string) error {
	params := s.txfeeKeeper.GetParams(ctx)
	return s.chargeFee(ctx, actorBech32, params.CwInstantiateFee, params.CwDeployFeeDestination, "cw_instantiate_fee")
}

// chargeFee deducts `fee` from `actorBech32` to the module account resolved
// from `dest`. No-op when the fee is zero or the actor is the gov authority.
// `eventType` is the Cosmos event type emitted on success ("cw_deploy_fee" /
// "cw_instantiate_fee") so off-chain indexers can distinguish them.
func (s *feeChargingWasmMsgServer) chargeFee(ctx context.Context, actorBech32 string, fee sdk.Coins, dest, eventType string) error {
	if actorBech32 == s.govAuthority {
		return nil
	}
	if fee.IsZero() {
		return nil
	}

	actor, err := sdk.AccAddressFromBech32(actorBech32)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid actor for %s: %s", eventType, err)
	}

	destModule, err := feeDestinationToModule(dest)
	if err != nil {
		return err
	}

	if err := s.bankKeeper.SendCoinsFromAccountToModule(ctx, actor, destModule, fee); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, "failed to charge %s: %s", eventType, err)
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		eventType,
		sdk.NewAttribute("payer", actor.String()),
		sdk.NewAttribute("fee", fee.String()),
		sdk.NewAttribute("destination", dest),
	))

	return nil
}

// feeDestinationToModule maps a CwDeployFeeDestination param value to the
// corresponding module account name.
func feeDestinationToModule(dest string) (string, error) {
	switch dest {
	case txfeetypes.FeeDestBurner:
		return txfeetypes.BurnerFeeCollector, nil
	case txfeetypes.FeeDestCommunityPool:
		return txfeetypes.CpFeeCollector, nil
	case txfeetypes.FeeDestStakers:
		return txfeetypes.ModuleName, nil
	default:
		return "", errorsmod.Wrap(sdkerrors.ErrLogic, fmt.Sprintf("unknown cw fee destination: %s", dest))
	}
}
