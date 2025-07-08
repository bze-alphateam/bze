#!/usr/bin/env bash

set -e

BZE_CMD="./bzed-old"
BZE_HOME_DIR="$HOME/.bzelocalnet"
CHAIN_ID="testing"

VALIDATOR_ADDR=$($BZE_CMD keys show validator -a --keyring-backend=os --home="$BZE_HOME_DIR")

echo "$VALIDATOR_ADDR"

$BZE_CMD tx gov submit-proposal software-upgrade "v8.0.0" --title="Upgrade network to v8.0.0" --description="if passed the chain upgrades to v8"  --upgrade-height 711830 --upgrade-info "the info we all want" --from "$VALIDATOR_ADDR" --fees 400000ubze --chain-id="$CHAIN_ID" --gas auto --home="$BZE_HOME_DIR" --deposit 200valtoken --broadcast-mode block
$BZE_CMD tx gov vote 1 yes --from "$VALIDATOR_ADDR" --fees 2000ubze --chain-id $CHAIN_ID --home="$BZE_HOME_DIR" --broadcast-mode block
$BZE_CMD q gov proposals --home="$BZE_HOME_DIR"
