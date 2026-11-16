#!/usr/bin/env bash

set -eo pipefail

mkdir -p ./tmp-swagger-gen

cd proto
# Get the path of the cosmos-sdk repo from go/pkg/mod
cosmos_sdk_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-sdk) || { echo "Error: Failed to find github.com/cosmos/cosmos-sdk"; exit 1; }
#alliance_dir=$(go list -f '{{ .Dir }}' -m github.com/terra-money/alliance) || { echo "Error: Failed to find github.com/terra-money/alliance"; exit 1; }
# IBC protos at the exact versions pinned in go.mod
ibc_go_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/ibc-go/v10) || { echo "Error: Failed to find github.com/cosmos/ibc-go/v10"; exit 1; }
ratelimit_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/ibc-apps/modules/rate-limiting/v10) || { echo "Error: Failed to find github.com/cosmos/ibc-apps/modules/rate-limiting/v10"; exit 1; }
# Only the IBC query services the node registers on its REST gateway (see app/ibc.go).
# Not listed: ibc/core/client/v2 (gRPC only, ibc-go registers no gateway route for it)
# and ibc/lightclients/wasm (the 08-wasm light client is not wired).
ibc_proto_dirs=(
  "${ibc_go_dir}/proto/ibc/core/client/v1"
  "${ibc_go_dir}/proto/ibc/core/connection/v1"
  "${ibc_go_dir}/proto/ibc/core/channel/v1"
  "${ibc_go_dir}/proto/ibc/core/channel/v2"
  "${ibc_go_dir}/proto/ibc/applications/transfer/v1"
  "${ibc_go_dir}/proto/ibc/applications/interchain_accounts/controller/v1"
  "${ibc_go_dir}/proto/ibc/applications/interchain_accounts/host/v1"
  "${ratelimit_dir}/proto/ratelimit/v1"
)
if [ -d "${cosmos_sdk_dir}/proto" ]; then
  proto_dirs=$( (find ./bze "${cosmos_sdk_dir}/proto" -name '*.proto' -print0 | xargs -0 -n1 dirname; printf '%s\n' "${ibc_proto_dirs[@]}") | sort | uniq)
  for dir in $proto_dirs; do
    # generate swagger files (filter query files)
    query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))

    if [[ ! -z "$query_file" ]]; then
      buf generate --template buf.gen.swagger.yaml $query_file
      echo $query_file
    fi
  done
fi

cd ..
if [ -d "./client/docs" ]; then
  cd ./client/docs
  yarn install
  yarn combine
  yarn convert
  yarn build
  cd ../../
fi

# clean swagger files
echo "removing temp swagger files"
rm -rf ./tmp-swagger-gen

echo "generating statik"
statik -src=client/docs/static -dest=client/docs -f -m