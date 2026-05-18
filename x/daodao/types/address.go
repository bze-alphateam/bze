package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

// DaoAccountAddress returns the deterministic on-chain address used as the
// `BaseAccount` for the DAO with the given id. Two properties hold:
//
//  1. The address is fully determined by `id` and the module name; no
//     randomness, no chain state inputs.
//  2. Two DAOs cannot collide because uvarint(id) is injective.
//
// The address is registered as a `BaseAccount` in x/auth at MsgCreateDao
// time. It is NOT a module account (no mint/burn permissions) — it behaves
// like any normal user account from bank's perspective.
func DaoAccountAddress(id uint64) sdk.AccAddress {
	return address.Module(ModuleName, []byte(fmt.Sprintf("dao/%d", id)))
}

// DepositEscrowAddress returns the deterministic on-chain address that
// holds deposits for the DAO's proposals. One escrow per DAO, shared
// across all that DAO's proposals — Proposal.deposit_collected and the
// per-(proposal, depositor) DepositRecord rows together track which
// share of the escrow belongs to which depositor / proposal.
//
// Like DaoAccountAddress, this is derived via address.Module so the
// address is unique and reproducible. We DON'T register a BaseAccount
// for it — bank.SendCoins / SendCoinsFromAccountToModule work fine on
// addresses without an auth account, and creating BaseAccounts for
// purely-bank-controlled escrow addresses just bloats x/auth.
func DepositEscrowAddress(daoID uint64) sdk.AccAddress {
	return address.Module(ModuleName, []byte(fmt.Sprintf("deposit/%d", daoID)))
}
