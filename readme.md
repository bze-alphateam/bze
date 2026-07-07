# BeeZee (BZE)
Powered by $BZE Coin, BeeZee Network offers users access to decentralized services through a variety of applications built on a fast and cost-efficient blockchain.

## BZE Ecosystem
- [getbze.com](https://getbze.com/) — Official website
- [dex.getbze.com](https://dex.getbze.com/) — BZE DEX for trading and liquidity pools
- [burner.getbze.com](https://burner.getbze.com/) — The Fire Furnace: permanently burn coins and LP tokens
- [factory.getbze.com](https://factory.getbze.com/) — Token Factory (under construction)
- [cointrunk.io](https://cointrunk.io/) — CoinTrunk, a Web3 tools company building practical open-source blockchain solutions in partnership with BeeZee

## Modules

### TradeBin
Orderbook trading and AMM liquidity pools. Create markets, place buy/sell orders, provide liquidity, and swap across multiple pools in a single transaction.
[User Guide](x/tradebin/docs/README.md) | [Parameters](x/tradebin/docs/params.md) | [Technical](x/tradebin/docs/technical.md)

### Burner
Permanently remove tokens from circulation. Coins are classified and handled by type: native coins are burned, LP tokens are permanently locked, and IBC tokens are converted to liquidity. Also hosts on-chain raffles.
[User Guide](x/burner/docs/README.md) | [Technical](x/burner/docs/technical.md)

### TokenFactory
Create and manage custom denominations. Any account can create a `factory/<creator>/<subdenom>` token and control minting, burning, and metadata as its admin.
[User Guide](x/tokenfactory/docs/README.md)

### Rewards
Run staking and trading incentive campaigns. Creators fund prize pools distributed to stakers or top traders on specific markets.
[User Guide](x/rewards/docs/README.md) | [Parameters](x/rewards/docs/params.md) | [Technical](x/rewards/docs/technical.md)

### CoinTrunk
On-chain article curation and publisher tipping. Publishers submit articles under approved domains, and anyone can tip them with a community tax applied.
[User Guide](x/cointrunk/docs/README.md)

### TxFeeCollector
Standardizes transaction fees by converting non-native denominations to native BZE and distributing them across validators, the burner, and the community pool.
[User Guide](x/txfeecollector/docs/README.md) | [Parameters](x/txfeecollector/docs/params.md)

## Official accounts and links:
[Official website](https://getbze.com/)
[Official Twitter account](https://twitter.com/BZEdgeCoin)
[Official Medium](https://medium.com/@bzedge)

### Wallets:
[Vidulum App](https://vidulum.app/)  
[Keplr Browser Extension](https://chains.keplr.app/)  

### Explorers:
https://exporer.getbze.com/beezee  
https://ping.pub/beezee  

### Trading:
[BZE DEX](https://app.osmosis.zone/pool/856)  
[Osmosis Pool](https://app.osmosis.zone/pool/856)  
[Dex Tools](https://www.dextools.io/app/osmosis/pair-explorer/856)  
[LiveCoinWatch](https://www.livecoinwatch.com/price/BZEdge-BZE)  
[CoinGeko](https://www.coingecko.com/en/coins/beezee)  

### Resources:
[Configs & Utils](https://github.com/bze-alphateam/bze-configs)  
⚠️ Use Chain-assets repo to get details needed to run a node.⚠️  
[Cosmos Chain-assets](https://github.com/cosmos/chain-registry/tree/master/beezee)  
[Graphics](https://github.com/bze-alphateam/Official-BZEdge-Graphics)  

### Genesis files:
[Mainnet (beezee-1)](https://github.com/bze-alphateam/bze/blob/main/genesis.json)  
[Testnet (bzetestnet-e)](https://github.com/bze-alphateam/bze/blob/main/genesis-testnet-3.json)

### IBC:  
#### BZE - Osmosis:
**BZE**: channel-0  
**Osmosis**: channel-340 

### CosmWasm
BZE supports [CosmWasm](https://cosmwasm.com/) smart contracts (wasmd v0.54.6, wasmvm v2.2.6).

**Capabilities**: `iterator`, `staking`, `stargate`, `cosmwasm_1_1` through `cosmwasm_2_2`

Contracts can interact with all BZE modules (TradeBin, TokenFactory, Rewards, Burner, CoinTrunk) via Stargate messages (`CosmosMsg::Any`). Governance-only messages (`MsgUpdateParams`, etc.) are protected by authority checks and cannot be called by contracts.

An additional CW deploy fee is charged when uploading contract code. Query current fees with:
```
bzed q txfeecollector params
```

### Building from source

#### Prerequisites

- **Go 1.25** (required, enforced at build time)
- **GCC or compatible C compiler** (required for wasmvm CGO linking)
  - macOS: `xcode-select --install` (provides clang)
  - Ubuntu/Debian: `sudo apt install build-essential`
  - Alpine: `apk add gcc musl-dev`
- **Git**

#### Checkout to the branch/tag you want to build
`git checkout v8.1.0`

#### Build for your current platform (native):
```
make build
```
Produces `./build/bzed`. CGO is enabled automatically for wasmvm support.

#### Install to GOPATH:
```
make install
```

#### Build via Docker (cross-platform, no C compiler needed):
```
make build-docker
```
Produces a statically linked Linux amd64 binary using the project Dockerfile. To build for arm64:
```
make build-docker DOCKER_PLATFORM=linux/arm64
```

#### Build all platforms (native cross-compilation):
```
make build-all
```
Builds binaries for all supported platforms and compresses them in `./build/compressed`.

**Note**: Cross-compiling from a different OS (e.g. Linux from macOS) requires a C cross-compiler because wasmvm uses CGO. For macOS to Linux:
```
brew install filosottile/musl-cross/musl-cross
```
If you don't have a cross-compiler, use `make build-docker` instead.

