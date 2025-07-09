#!/usr/bin/env bash

set -e

BZE_CMD="./build/bzed-old"
BZE_HOME_DIR="$HOME/.bzelocalnet"
CHAIN_ID="testing"

# Check if home directory already exists
if [ -d "$BZE_HOME_DIR" ]; then
  echo "Directory $BZE_HOME_DIR already exists."
  read -p "Do you want to delete it and continue? (y/N): " confirm
  case "$confirm" in
    [yY][eE][sS]|[yY])
      echo "Deleting $BZE_HOME_DIR..."
      rm -rf "$BZE_HOME_DIR"
      ;;
    *)
      echo "Aborting."
      exit 1
      ;;
  esac
fi

$BZE_CMD init --chain-id=$CHAIN_ID testing --home="$BZE_HOME_DIR"
$BZE_CMD keys add validator --keyring-backend=os --home="$BZE_HOME_DIR"
VALIDATOR_ADDR=$($BZE_CMD keys show validator -a --keyring-backend=os --home="$BZE_HOME_DIR")

$BZE_CMD add-genesis-account "$VALIDATOR_ADDR" 1000000000ubze,1000000000valtoken --home="$BZE_HOME_DIR"
sed -i -e "s/stake/ubze/g" "$BZE_HOME_DIR/config/genesis.json"
$BZE_CMD gentx validator 500000000ubze --commission-rate="0.0" --keyring-backend=os --home="$BZE_HOME_DIR" --chain-id=$CHAIN_ID
$BZE_CMD collect-gentxs --home="$BZE_HOME_DIR"

# Set initial height
jq '.initial_height = "711800"' "$BZE_HOME_DIR/config/genesis.json" > "$BZE_HOME_DIR/config/tmp_genesis.json" && mv "$BZE_HOME_DIR/config/tmp_genesis.json" "$BZE_HOME_DIR/config/genesis.json"
# Set min_deposit[0].denom = "valtoken"
jq '.app_state.gov.deposit_params.min_deposit[0].denom = "valtoken"' "$BZE_HOME_DIR/config/genesis.json" > "$BZE_HOME_DIR/config/tmp_genesis.json" && mv "$BZE_HOME_DIR/config/tmp_genesis.json" "$BZE_HOME_DIR/config/genesis.json"
# Set min_deposit[0].amount = "100"
jq '.app_state.gov.deposit_params.min_deposit[0].amount = "100"' "$BZE_HOME_DIR/config/genesis.json" > "$BZE_HOME_DIR/config/tmp_genesis.json" && mv "$BZE_HOME_DIR/config/tmp_genesis.json" "$BZE_HOME_DIR/config/genesis.json"
# Set voting period
jq '.app_state.gov.voting_params.voting_period = "120s"' "$BZE_HOME_DIR/config/genesis.json" > "$BZE_HOME_DIR/config/tmp_genesis.json" && mv "$BZE_HOME_DIR/config/tmp_genesis.json" "$BZE_HOME_DIR/config/genesis.json"

jq '.app_state.cointrunk.publisher_list[0] = {
  name: "CosmosBG",
  address: "bze1edwj9fhuugzggcv5magm9j4vnur4hzsf26s2ws",
  active: true,
  articles_count: 90,
  created_at: "1744098925",
  respect: "40500000000"
}' "$BZE_HOME_DIR/config/genesis.json" > "$BZE_HOME_DIR/config/tmp_genesis.json" && mv "$BZE_HOME_DIR/config/tmp_genesis.json" "$BZE_HOME_DIR/config/genesis.json"

$BZE_CMD start --home="$BZE_HOME_DIR"
