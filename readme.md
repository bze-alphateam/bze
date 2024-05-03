# BeeZee (BZE)
A blockchain governed by the community built on top of Cosmos SDK. BZE moved to Cosmos from 
Zcash codebase in 2022 with the help of the community.

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
[Osmosis Pool](https://frontier.osmosis.zone/pool/856)  
[Dex tracker]( https://dexscreener.com/osmosis/856)  
[LiveCoinWatch](https://www.livecoinwatch.com/price/BZEdge-BZE)  
[CoinGeko](https://www.coingecko.com/en/coins/beezee)  

### Resources:
[Configs & Utils](https://github.com/bze-alphateam/bze-configs)  
⚠️ Use Chain-assets repo to get details needed to run a node.⚠️  
[Cosmos Chain-assets](https://github.com/cosmos/chain-registry/tree/master/beezee)  
[Graphics](https://github.com/bze-alphateam/Official-BZEdge-Graphics)  

### Genesis files:
[Mainnet (beezee-1)](https://github.com/bze-alphateam/bze/blob/main/genesis.json)  
[Testnet (bzetestnet-2)](https://github.com/bze-alphateam/bze/blob/main/genesis-testnet-2.json)

### IBC:  
#### BZE - Osmosis:
**BZE**: channel-0  
**Osmosis**: channel-340 

### Building from source
#### Checkout to the branch/tag you want to build 
`git checkout v6.1.0`

#### Build binaries:
`make build-all`  
This will build binaries for all supported platforms and compress them in ./build directory

#### Build for specific platform:
`make build-linux`
This will build the binary for linux amd64 - check Makefile for more details and platforms

#### Epochs hooks

