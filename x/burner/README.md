# Burner Module for Cosmos SDK by BeeZee (BZE)

## Overview
The Burner module, developed by BeeZee (BZE) for Cosmos SDK ecosystem, enabling users to effectively manage the token supply through coin burning mechanisms. This module introduces governance proposals for burning coins, along with the ability to fund and query the burner operations.

## Features
- **Fund Burner**: Allows users to send coins to the Burner module for burning.
- **Burn Coins via Governance Proposals**: Supports burning all coins in the module through a `BurnCoinsProposal`.
- **Query Burned Coins**: Enables querying details of burned coins, including the block height at which the burn occurred.
- **Event Logging**: Records key events like coins being burned and funds being added to the burner.

## Protobuf Definitions

### Messages
- **Params**: Represents the parameters for the Burner module.
- **CoinsBurnedEvent**: Logs when coins are burned. Fields:
    - `burned`: Amount of coins burned.
- **FundBurnerEvent**: Logs when the burner is funded. Fields:
    - `from`: Address of the funder.
    - `amount`: Amount funded.
- **BurnedCoins**: Represents a record of burned coins. Fields:
    - `burned`: Amount of coins burned.
    - `height`: Blockchain height at which the burn occurred.
- **BurnCoinsProposal**: Represents a proposal to burn coins. Fields:
    - `title`: Title of the proposal.
    - `description`: Description of the proposal.

### Services
- **Msg Service**: Handles transactions like funding the Burner module.
- **Query Service**: Offers RPC methods for querying module parameters and the record of burned coins.
## CLI Commands

### Fund Burner
Use this command to fund the Burner module with the specified amount.
```bash
bzed tx burner fund-burner <amount> --from <your-key>
```

### Query Burned Coins
Use this command to retrieve a list of all burned coins.
```bash
bzed query burner all-burned-coins
```


#### HTTP API Endpoints Section
## HTTP API Endpoints
### Query All Burned Coins
- **GET** `/bze/burner/v1/all_burned_coins`
  - Fetches details about all coins burned by the module.

### Query Module Parameters
- **GET** `/bze/burner/v1/params`
  - Retrieves the current parameters of the Burner module.

## Event Logging
- **CoinsBurnedEvent**: Dispatched when the burning of coins occurs.
- **FundBurnerEvent**: Dispatched when funding Burner module.

## Contributing
Contributions to this module are encouraged. Feel free to submit issues and PR.
