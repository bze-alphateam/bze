package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName defines the module name
	ModuleName = "dao"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// CounterKeyPrefix holds the id counters (never reused).
	CounterKeyPrefix = "dao/c/"
	// DaoKeyPrefix holds the Dao records by id.
	DaoKeyPrefix = "dao/v/"
	// DaoAddressKeyPrefix maps a DAO account address to its dao id.
	DaoAddressKeyPrefix = "dao/a/"
	// DaoAdminKeyPrefix indexes DAOs by admin: {admin}/{dao_id}.
	DaoAdminKeyPrefix = "dao/m/"
	// PinKeyPrefix counts the DAOs (and later the open proposals) that use an
	// electorate source: {source} -> uint64.
	PinKeyPrefix = "dao/p/"

	// sourceStakingReward and sourceDenomReward prefix the electorate source keys.
	sourceStakingReward = "sr/"
	sourceDenomReward   = "dr/"
)

var (
	ParamsKey = []byte("p_dao")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// DaoCounterKey is the key of the dao id counter inside CounterKeyPrefix.
func DaoCounterKey() []byte {
	return []byte{1}
}

// DaoKey is the key of a Dao inside DaoKeyPrefix.
func DaoKey(daoId uint64) []byte {
	return sdk.Uint64ToBigEndian(daoId)
}

// DaoAddressKey is the key of a DAO address inside DaoAddressKeyPrefix.
func DaoAddressKey(daoAddr sdk.AccAddress) []byte {
	return daoAddr.Bytes()
}

// DaoAdminKey is the key of an admin index entry inside DaoAdminKeyPrefix.
func DaoAdminKey(admin sdk.AccAddress, daoId uint64) []byte {
	return append(address.MustLengthPrefix(admin.Bytes()), sdk.Uint64ToBigEndian(daoId)...)
}

// DaoAddress derives the account address of a DAO from its id.
func DaoAddress(daoId uint64) sdk.AccAddress {
	return address.Module(ModuleName, sdk.Uint64ToBigEndian(daoId))
}

// StakingRewardSource is the source key of a staking reward: sr/{reward_id}.
func StakingRewardSource(rewardId string) []byte {
	return sourceKey(sourceStakingReward, rewardId)
}

// DenomRewardSource is the source key of a Denom Reward: dr/{staking_denom}.
func DenomRewardSource(stakingDenom string) []byte {
	return sourceKey(sourceDenomReward, stakingDenom)
}

// sourceKey builds every electorate source key, so all stores that refer to a
// source (pins, join records, proposal snapshots) agree on its form.
func sourceKey(kind, id string) []byte {
	return []byte(kind + id)
}
