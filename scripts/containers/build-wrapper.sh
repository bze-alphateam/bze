#!/usr/bin/env sh
set -x

export PATH=$PATH:/beezee/build/bzed
BINARY=/beezee/build/bzed
ID=${ID:-0}
LOG=${LOG:-bzed.log}

if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found."
	exit 1
fi

export BZEDHOME="/beezee/data/node${ID}/bzed"

if [ -d "$(dirname "${BZEDHOME}"/"${LOG}")" ]; then
  "${BINARY}" --home "${BZEDHOME}" "$@" | tee "${BZEDHOME}/${LOG}"
else
  "${BINARY}" --home "${BZEDHOME}" "$@"
fi