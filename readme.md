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

### Building from source
#### Checkout to the branch/tag you want to build 
`git checkout v8.1.0`

#### Build binaries:
`make build-all`  
This will build binaries for all supported platforms and compress them in ./build directory

#### Build for specific platform:
`make build-linux`
This will build the binary for linux amd64 - check Makefile for more details and platforms

