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
	// ProposalKeyPrefix holds the Proposal records by id.
	ProposalKeyPrefix = "prop/v/"
	// OpenProposalKeyPrefix marks the proposals in voting: {dao_id}/{proposal_id}.
	OpenProposalKeyPrefix = "prop/d/"
	// DaoProposalKeyPrefix indexes every proposal of a DAO: {dao_id}/{proposal_id}.
	// An entry is deleted only when its proposal is cleaned up.
	DaoProposalKeyPrefix = "prop/i/"
	// JoinKeyPrefix holds the height an address joined a pinned source:
	// {source}/{address} -> int64.
	JoinKeyPrefix = "join/"
	// BallotKeyPrefix holds the ballots: {proposal_id}/{voter} -> Ballot.
	BallotKeyPrefix = "ballot/"
	// VoteIndexKeyPrefix marks every counted stake share of a ballot:
	// {source}/{address}/{proposal_id}. The Exit hook finds an address's open
	// ballots on a source through it.
	VoteIndexKeyPrefix = "vidx/"

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

// ProposalCounterKey is the key of the proposal id counter inside CounterKeyPrefix.
func ProposalCounterKey() []byte {
	return []byte{2}
}

// ProposalKey is the key of a Proposal inside ProposalKeyPrefix.
func ProposalKey(proposalId uint64) []byte {
	return sdk.Uint64ToBigEndian(proposalId)
}

// DaoProposalsPrefix is the part of OpenProposalKeyPrefix and DaoProposalKeyPrefix
// keys that holds one DAO's proposals.
func DaoProposalsPrefix(daoId uint64) []byte {
	return sdk.Uint64ToBigEndian(daoId)
}

// DaoProposalKey is the key of a proposal inside OpenProposalKeyPrefix and
// DaoProposalKeyPrefix: {dao_id}/{proposal_id}.
func DaoProposalKey(daoId, proposalId uint64) []byte {
	return append(DaoProposalsPrefix(daoId), sdk.Uint64ToBigEndian(proposalId)...)
}

// JoinKey is the key of a join record inside JoinKeyPrefix. Both parts are
// length-prefixed: sources and addresses have variable lengths.
func JoinKey(source []byte, addr sdk.AccAddress) []byte {
	return append(address.MustLengthPrefix(source), address.MustLengthPrefix(addr.Bytes())...)
}

// ProposalBallotsPrefix is the part of BallotKeyPrefix keys that holds one
// proposal's ballots.
func ProposalBallotsPrefix(proposalId uint64) []byte {
	return sdk.Uint64ToBigEndian(proposalId)
}

// BallotKey is the key of a ballot inside BallotKeyPrefix: {proposal_id}/{voter}.
func BallotKey(proposalId uint64, voter sdk.AccAddress) []byte {
	return append(ProposalBallotsPrefix(proposalId), address.MustLengthPrefix(voter.Bytes())...)
}

// VoteIndexPrefix is the part of VoteIndexKeyPrefix keys that holds one
// address's ballots counting a source: {source}/{address}/.
func VoteIndexPrefix(source []byte, addr sdk.AccAddress) []byte {
	return JoinKey(source, addr)
}

// VoteIndexKey is the key of a vote-index entry inside VoteIndexKeyPrefix:
// {source}/{address}/{proposal_id}.
func VoteIndexKey(source []byte, addr sdk.AccAddress, proposalId uint64) []byte {
	return append(VoteIndexPrefix(source, addr), sdk.Uint64ToBigEndian(proposalId)...)
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
