package v820

import (
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// Noble USDC wind-down (BZE-143).
//
// Circle is retiring USDC on the Noble chain: from 2027-01-12 the USDC.n vouchers held
// on BZE are backed by escrow nobody can redeem any more. The v8.2.0 upgrade therefore
// does exactly two things, both driven by the table below:
//
//  1. it stops new USDC.n from arriving, by writing the (channel, base denom) pair into
//     the x/txfeecollector BlockedIbcInbound param. Exits are untouched: outgoing
//     transfers to Noble, their acknowledgements and their timeout refunds all keep
//     working, and the channel is never closed;
//  2. it hands the USDC.n/BZE LP shares owned by the burner black hole to the admin
//     address, so the liquidity behind them can be withdrawn and redeployed as
//     USDC.inj liquidity instead of dying with the voucher. Nothing else the black
//     hole holds is touched.
//
// The values are hardcoded on purpose: governance voting is too slow for the deadline.
// This whole file, and the handler steps that read it, are meant to be deleted in the
// release after v8.2.0.

// WindDown describes the per-chain values of the Noble USDC wind-down. Both parts are
// independent: an empty BlockedInbound leaves the param empty, an empty LpDenom skips
// the LP share transfer.
type WindDown struct {
	// BlockedInbound is written to the x/txfeecollector BlockedIbcInbound param.
	BlockedInbound []txfeecollectortypes.BlockedIbcTransfer
	// LpDenom is the single black-hole denomination handed to AdminAddress. Every other
	// balance of the black hole stays where it is.
	LpDenom string
	// AdminAddress receives the LP shares. It is an ordinary wallet controlled by the
	// team, not a module account.
	AdminAddress string
}

const (
	// nobleUsdcChannel is the BZE side of the BZE <-> Noble transfer channel.
	nobleUsdcChannel = "channel-3"
	// nobleUsdcBaseDenom is the denomination Noble sends over that channel; it becomes
	// ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4 on BZE.
	nobleUsdcBaseDenom = "uusdc"
	// nobleUsdcLpDenom is the LP denomination of the USDC.n/BZE liquidity pool.
	nobleUsdcLpDenom = "ulp_ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4_ubze"
	// adminAddress is the wallet that receives the black hole's LP shares. The same
	// wallet is used on the testnet, which shares the mainnet bech32 prefix.
	adminAddress = "bze1jx4x3kn8mlz2s03zdpf2a38x9gl66llvjqdd55"
)

// The testnet has no Noble channel and no USDC.n pool, so mainnet values would make
// both steps log-and-skip no-ops there and the rehearsal would prove nothing. It runs
// the same code against the assets it does have: its single transfer channel and the
// LP shares its own black hole holds.
const (
	// testnetChannel is the only transfer channel open on bzetestnet-3.
	testnetChannel = "channel-0"
	// testnetBaseDenom is the denomination that arrives over it; it becomes
	// ibc/9DA252F9F9C86132CC282EA431DFB7DE7729501F6DC9A3E0F50EC8C6EE380CC7 on BZE.
	testnetBaseDenom = "ulmn"
	// testnetLpDenom is the LP denomination of that voucher's pool with BZE; the
	// testnet black hole holds shares of it, so the transfer step actually moves coins.
	testnetLpDenom = "ulp_ibc/9DA252F9F9C86132CC282EA431DFB7DE7729501F6DC9A3E0F50EC8C6EE380CC7_ubze"
)

// nobleUsdcWindDown is the mainnet configuration.
var nobleUsdcWindDown = WindDown{
	BlockedInbound: []txfeecollectortypes.BlockedIbcTransfer{
		{ChannelId: nobleUsdcChannel, BaseDenom: nobleUsdcBaseDenom},
	},
	LpDenom:      nobleUsdcLpDenom,
	AdminAddress: adminAddress,
}

// testnetWindDown mirrors it on bzetestnet-3 with assets that exist there, so both
// steps can be observed: an inbound ulmn transfer must come back with an error
// acknowledgement, and the black hole's ulmn/BZE LP shares must land on the admin
// address. Governance can lift the block again with MsgUpdateParams.
var testnetWindDown = WindDown{
	BlockedInbound: []txfeecollectortypes.BlockedIbcTransfer{
		{ChannelId: testnetChannel, BaseDenom: testnetBaseDenom},
	},
	LpDenom:      testnetLpDenom,
	AdminAddress: adminAddress,
}

// windDownByChainID keys the wind-down values by chain id. A chain id that is absent
// (a devnet, a local test chain, a retired testnet) runs neither part of the
// wind-down: bzetestnet-3 is the only testnet still running.
var windDownByChainID = map[string]WindDown{
	"beezee-1":     nobleUsdcWindDown,
	"bzetestnet-3": testnetWindDown,
}

// GetWindDown returns the wind-down configuration for a chain id, and whether one
// exists at all.
func GetWindDown(chainID string) (WindDown, bool) {
	cfg, found := windDownByChainID[chainID]

	return cfg, found
}
