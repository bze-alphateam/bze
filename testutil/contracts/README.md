# CosmWasm Local Testing Guide

Instructions for deploying and interacting with smart contracts on a local BZE node.

## Official Contract Repositories

Download pre-built `.wasm` binaries from these repos:

| Repository | Description | Download |
|------------|-------------|----------|
| [cw-plus](https://github.com/CosmWasm/cw-plus) | Production-quality contracts: CW20 (tokens), CW721 (NFTs), CW1 (proxy), CW3 (multisig), CW4 (group) | [Releases](https://github.com/CosmWasm/cw-plus/releases) |
| [cw-nfts](https://github.com/CosmWasm/cw-nfts) | NFT standard contracts (CW721 base, metadata, on-chain) | [Releases](https://github.com/CosmWasm/cw-nfts/releases) |
| [cw-template](https://github.com/CosmWasm/cw-template) | Minimal counter contract, good starting point | Build with `cargo generate` |
| [cosmwasm-examples](https://github.com/CosmWasm/cosmwasm-examples) | Example contracts: nameservice, escrow, erc20, voting | Build from source |
| [cw-storage-plus](https://github.com/CosmWasm/cw-storage-plus) | Storage abstractions (not a contract, but essential library) | — |

For this guide, download:
- **counter.wasm** — build from [cw-template](https://github.com/CosmWasm/cw-template) or [cosmwasm-examples/contracts/counter](https://github.com/CosmWasm/cosmwasm-examples)
- **cw20_base.wasm** — from [cw-plus releases](https://github.com/CosmWasm/cw-plus/releases) (artifact: `cw20_base.wasm`)

## Prerequisites

- A running local node with a funded account
- A key available in the keyring (examples below use `mykey`)

```bash
# Verify your key exists and has funds
bzed keys list
bzed q bank balances $(bzed keys show mykey -a)
```

## Counter Contract

A minimal contract that stores an integer and supports increment, reset, and query.

### Store the contract

```bash
bzed tx wasm store counter.wasm \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 10000000ubze \
  -y
```

### Get the code ID

```bash
bzed q wasm list-code
```

### Instantiate

```bash
bzed tx wasm instantiate <code-id> '{"count": 0}' \
  --label "counter-test" \
  --no-admin \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 5000000ubze \
  -y
```

### Get the contract address

```bash
bzed q wasm list-contract-by-code <code-id>
```

### Execute: Increment

```bash
bzed tx wasm execute <contract-addr> '{"increment": {}}' \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 1000000ubze \
  -y
```

### Execute: Reset

```bash
bzed tx wasm execute <contract-addr> '{"reset": {"count": 0}}' \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 1000000ubze \
  -y
```

### Query: Get Count

```bash
bzed q wasm contract-state smart <contract-addr> '{"get_count": {}}'
```

## CW20 Base Contract (Fungible Token)

The standard CW20 token contract — supports minting, transferring, burning, and allowances.

### Store the contract

```bash
bzed tx wasm store cw20_base.wasm \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 10000000ubze \
  -y
```

### Instantiate (with initial balances and minter)

Replace `<your-bze-address>` with the output of `bzed keys show mykey -a`.

```bash
bzed tx wasm instantiate <code-id> '{
  "name": "Test Token",
  "symbol": "TEST",
  "decimals": 6,
  "initial_balances": [
    {"address": "<your-bze-address>", "amount": "1000000000"}
  ],
  "mint": {
    "minter": "<your-bze-address>"
  }
}' \
  --label "cw20-test" \
  --no-admin \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 5000000ubze \
  -y
```

### Execute: Transfer

```bash
bzed tx wasm execute <contract-addr> '{
  "transfer": {
    "recipient": "<recipient-bze-address>",
    "amount": "1000000"
  }
}' \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 1000000ubze \
  -y
```

### Execute: Mint (requires minter)

```bash
bzed tx wasm execute <contract-addr> '{
  "mint": {
    "recipient": "<recipient-bze-address>",
    "amount": "500000"
  }
}' \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 1000000ubze \
  -y
```

### Execute: Burn

```bash
bzed tx wasm execute <contract-addr> '{
  "burn": {
    "amount": "100000"
  }
}' \
  --from mykey \
  --gas auto --gas-adjustment 1.3 \
  --fees 1000000ubze \
  -y
```

### Query: Balance

```bash
bzed q wasm contract-state smart <contract-addr> '{"balance": {"address": "<bze-address>"}}'
```

### Query: Token Info

```bash
bzed q wasm contract-state smart <contract-addr> '{"token_info": {}}'
```

### Query: All Accounts

```bash
bzed q wasm contract-state smart <contract-addr> '{"all_accounts": {}}'
```

## Notes

- **CW deploy fee**: This chain charges an additional fee on `MsgStoreCode` (on top of gas). Query the current fee with:
  ```bash
  bzed q txfeecollector params
  ```
- **Gas estimation**: Always use `--gas auto --gas-adjustment 1.3` for wasm transactions, as gas usage varies by contract size and complexity.
- **Contract admin**: Use `--no-admin` for test contracts. To allow future migrations, pass `--admin <address>` instead.
- **Chain capabilities**: This chain supports CosmWasm up to v2.2 with `iterator`, `staking`, and `stargate` capabilities.
