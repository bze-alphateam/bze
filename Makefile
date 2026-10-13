PACKAGES=$(shell go list ./... | grep -v '/simulation')
PACKAGE_NAME:=github.com/bze-alphateam/bze-v5
GOLANG_CROSS_VERSION  = v1.23
VERSION := $(shell echo $(shell git describe --tags 2>/dev/null ) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
NETWORK ?= mainnet
COVERAGE ?= coverage.txt
BUILDDIR ?= $(CURDIR)/build
LEDGER_ENABLED ?= false

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=bze \
	-X github.com/cosmos/cosmos-sdk/version.AppName=bzed \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-w -s

BUILD_FLAGS := -ldflags '$(ldflags)'
TESTNET_FLAGS ?=

ledger ?= HID
ifeq ($(LEDGER_ENABLED),true)
	BUILD_TAGS := -tags cgo,ledger,!test_ledger_mock,!ledger_mock
	ifeq ($(ledger), ZEMU)
		BUILD_TAGS := $(BUILD_TAGS),ledger_zemu
	else
		BUILD_TAGS := $(BUILD_TAGS),!ledger_zemu
	endif
endif

ifeq ($(NETWORK),testnet)
	BUILD_TAGS := $(BUILD_TAGS),testnet
	TEST_TAGS := "--tags=testnet"
endif

SIMAPP = github.com/bze-alphateam/bze-v5
BINDIR ?= ~/go/bin

OS := $(shell uname)

all: download install

download:
	git submodule update --init --recursive

install: check-version lint check-network go.sum
		go install -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) ./cmd/bzed

build: check-version check-network go.sum
		go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/bzed ./cmd/bzed

build-win64: check-version check-network go.sum
		go build -buildmode=exe -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/win64/bzed.exe ./cmd/bzed

.PHONY: build

build-linux: check-version check-network go.sum
ifeq ($(OS), Linux)
		GOOS=linux GOARCH=amd64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/linux-amd64/bzed-linux-amd64 ./cmd/bzed
else
		LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/linux-amd64/bzed-linux-amd64 ./cmd/bzed
endif

build-linux-arm64: check-version check-network go.sum
ifeq ($(OS), Linux)
		GOOS=linux GOARCH=arm64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/linux-arm64/bzed-linux-arm64 ./cmd/bzed
else
		LEDGER_ENABLED=false GOOS=linux GOARCH=arm64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/linux-arm64/bzed-linux-arm64 ./cmd/bzed
endif

build-mac: check-version check-network go.sum
ifeq ($(OS), Darwin)
		GOOS=darwin GOARCH=amd64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/darwin-amd64/bzed-darwin-amd64 ./cmd/bzed
else
		LEDGER_ENABLED=false GOOS=darwin GOARCH=amd64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/darwin-amd64/bzed-darwin-amd64 ./cmd/bzed
endif

build-mac-arm64: check-version check-network go.sum
ifeq ($(OS), Darwin)
		GOOS=darwin GOARCH=arm64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/darwin-arm64/bzed-darwin-arm64 ./cmd/bzed
else
		LEDGER_ENABLED=false GOOS=darwin GOARCH=arm64 go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -trimpath -o $(BUILDDIR)/darwin-arm64/bzed-darwin-arm64 ./cmd/bzed
endif

build-all: check-version lint all build-win64 build-mac build-mac-arm64 build-linux build-linux-arm64 compress-build

compress-build:
	rm -rf $(BUILDDIR)/compressed
	mkdir $(BUILDDIR)/compressed
	zip -j $(BUILDDIR)/compressed/bze-$(VERSION)-win64.zip $(BUILDDIR)/win64/bzed.exe
	cp $(BUILDDIR)/darwin-arm64/bzed-darwin-arm64 $(BUILDDIR)/darwin-arm64/bzed
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-darwin-arm64.tar.gz -C $(BUILDDIR)/darwin-arm64/ bzed
	rm $(BUILDDIR)/darwin-arm64/bzed
	cp $(BUILDDIR)/darwin-amd64/bzed-darwin-amd64 $(BUILDDIR)/darwin-amd64/bzed
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-darwin-amd64.tar.gz -C $(BUILDDIR)/darwin-amd64/ bzed
	rm $(BUILDDIR)/darwin-amd64/bzed
	cp $(BUILDDIR)/linux-arm64/bzed-linux-arm64 $(BUILDDIR)/linux-arm64/bzed
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-linux-arm64.tar.gz -C $(BUILDDIR)/linux-arm64/ bzed
	rm $(BUILDDIR)/linux-arm64/bzed
	cp $(BUILDDIR)/linux-amd64/bzed-linux-amd64 $(BUILDDIR)/linux-amd64/bzed
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-linux-amd64.tar.gz -C $(BUILDDIR)/linux-amd64/ bzed
	rm $(BUILDDIR)/linux-amd64/bzed

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		GO111MODULE=on go mod verify

test: check-network
	@go test $(TEST_TAGS) -v -mod=readonly $(PACKAGES) -coverprofile=$(COVERAGE) -covermode=atomic
.PHONY: test

# look into .golangci.yml for enabling / disabling linters
lint:
	@#echo "--> Running linter"
	@#golangci-lint run
	@#go mod verify

# a trick to make all the lint commands execute, return error when at least one fails.
# golangci-lint is run in standalone job in ci
lint-ci:
	@echo "--> Running linter for CI"
	@nix run -f ./. lint-env -c lint-ci

# Add check to make sure we are using the proper Go version before proceeding with anything
check-version:
	@if ! go version | grep -q "go1.26"; then \
		echo "\033[0;31mERROR:\033[0m Go version 1.26 is required for compiling bzed. It looks like you are using" "$(shell go version) \nThere are potential consensus-breaking changes that can occur when running binaries compiled with different versions of Go. Please download Go version 1.26 and retry. Thank you!"; \
		exit 1; \
	fi

test-sim-nondeterminism: check-network
	@echo "Running non-determinism test..."
	@go test $(TEST_TAGS) -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

test-sim-custom-genesis-fast: check-network
	@echo "Running custom genesis simulation..."
	@echo "By default, ${HOME}/.bze/config/genesis.json will be used."
	@go test $(TEST_TAGS) -mod=readonly $(SIMAPP) -run TestFullAppSimulation -Genesis=${HOME}/.bze/config/genesis.json \
		-Enabled=true -NumBlocks=100 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v -timeout 24h

test-sim-import-export:
	@echo "Running Chain import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) 25 5 TestAppImportExport

test-sim-after-import:
	@echo "Running application simulation-after-import. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) 50 5 TestAppSimulationAfterImport

###############################################################################
###                                Localnet                                 ###
###############################################################################
start-localnet-ci: build
	rm -rf ~/.bzed-liveness
	./build/bzed init liveness --staking-bond-denom ubze --chain-id liveness --home ~/.bzed-liveness
	./build/bzed config chain-id liveness --home ~/.bzed-liveness
	./build/bzed config keyring-backend test --home ~/.bzed-liveness
	./build/bzed keys add val --home ~/.bzed-liveness
	./build/bzed genesis add-genesis-account val 1000000000000ubze --home ~/.bzed-liveness --keyring-backend test
	./build/bzed keys add user --home ~/.bzed-liveness
	./build/bzed genesis add-genesis-account user 1000000000ubze --home ~/.bzed-liveness --keyring-backend test
	./build/bzed genesis gentx val 1000000000ubze --home ~/.bzed-liveness --chain-id liveness
	./build/bzed genesis collect-gentxs --home ~/.bzed-liveness
	sed -i.bak 's#^minimum-gas-prices = .*#minimum-gas-prices = "0.001ubze,0.0001stake"#g' ~/.bzed-liveness/config/app.toml
	./build/bzed start --home ~/.bzed-liveness
.PHONY: start-localnet-ci

build-docker-bzednode:
	$(MAKE) -C check-networks/local

# Run a 4-node testnet locally
localnet-start: build-linux build-docker-testbzednode localnet-stop
	@if ! [ -f $(BUILDDIR)/node0/.testbze/config/genesis.json ]; \
	then docker run --rm -v $(BUILDDIR):/testbzed:Z bze/testbzednode testnet --v 4 -o . --starting-ip-address 192.168.10.2 $(TESTNET_FLAGS); \
	fi
	BUILDDIR=$(BUILDDIR) docker-compose up -d

# Stop testnet
localnet-stop:
	docker-compose down
	docker check-network prune -f

# local build pystarport
build-pystarport:
	pip install ./pystarport

# Run a local testnet by pystarport
localnet-pystartport: build-pystarport
	pystarport serve

clean:
	rm -rf $(BUILDDIR)/

clean-docker-compose: localnet-stop
	rm -rf $(BUILDDIR)/node* $(BUILDDIR)/gentxs

create-systemd:
	./networks/create-service.sh

make-proto:
	./makeproto.sh
###############################################################################
###                                Nix                                      ###
###############################################################################
# nix installation: https://nixos.org/download.html
nix-integration-test: check-network make-proto
	nix run -f ./default.nix run-integration-tests -c run-integration-tests

nix-integration-test-upgrade: check-network
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v -m upgrade"

nix-integration-test-ledger: check-network
	nix run -f ./default.nix run-integration-tests-zemu -c run-integration-tests "pytest -v -m ledger"

nix-integration-test-slow: check-network
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v -m slow"

nix-integration-test-ibc: check-network
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v -m ibc"

nix-integration-test-byzantine: check-network
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v -m byzantine"

nix-integration-test-gov: check-network
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v -m gov"

nix-integration-test-grpc: check-network make-proto
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v -m grpc"

nix-integration-test-all: check-network make-proto
	nix run -f ./default.nix run-integration-tests -c run-integration-tests "pytest -v"


nix-build-%: check-network check-os
	@if [ -e ~/.nix/remote-build-env ]; then \
		. ~/.nix/remote-build-env; \
	fi && \
	nix-build -o $* -A $* docker.nix;
	docker load < $*;

chaindImage: nix-build-chaindImage
pystarportImage: nix-build-pystarportImage

check-network:
ifeq ($(NETWORK),mainnet)
else ifeq ($(NETWORK),testnet)
else
	@echo "Unrecognized network: ${NETWORK}"
endif
	@echo "building network: ${NETWORK}"

check-os:
ifeq ($(OS), Darwin)
ifneq ("$(wildcard ~/.nix/remote-build-env))","")
	@echo "installed nix-remote-builder before" && \
	docker run --restart always --name nix-docker -d -p 3022:22 lnl7/nix:ssh 2> /dev/null || echo "nix-docker is already running"
else
	@echo install nix-remote-builder
	git clone https://github.com/holidaycheck/nix-remote-builder.git
	cd nix-remote-builder && ./init.sh
	rm -rf nix-remote-builder
endif
endif

###############################################################################
###                              Release                                    ###
###############################################################################
.PHONY: release-dry-run
release-dry-run:
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-v ${GOPATH}/pkg:/go/pkg \
		-w /go/src/$(PACKAGE_NAME) \
		troian/golang-cross:${GOLANG_CROSS_VERSION} \
		--rm-dist --skip-validate --skip-publish

.PHONY: release
release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		troian/golang-cross:${GOLANG_CROSS_VERSION} \
		release --rm-dist --skip-validate

###############################################################################
###                              Documentation                              ###
###############################################################################

# generate api swagger document
document:
	make all -f MakefileDoc

# generate protobuf files
# ./proto -> ./x
proto-all:
	make proto-all -f MakefileDoc

###############################################################################
###                                  E2E                                    ###
###############################################################################
# The e2e suite (interchaintest/) runs bzed in Docker. Images are never built
# implicitly: build them once with e2e-image / e2e-image-ref, then run suites.
E2E_TIMEOUT ?= 60m

.PHONY: e2e-image e2e-image-ref e2e e2e-smoke e2e-denomrewards e2e-amm e2e-tokenfactory e2e-halt e2e-ibc e2e-orderbook e2e-tradingrewards e2e-burner e2e-fees e2e-cointrunk e2e-upgrade e2e-unit e2e-lint

# Builds the working tree as bze:local.
e2e-image:
	./scripts/e2e/build-image.sh local

# Builds a git ref as bze:<ref>, e.g. make e2e-image-ref REF=v8.2.0
e2e-image-ref:
	@if [ -z "$(REF)" ]; then echo "usage: make e2e-image-ref REF=<git-ref>"; exit 1; fi
	./scripts/e2e/build-image.sh $(REF)

# Runs every e2e suite but the upgrade suite, which needs the previous
# release's image and a plan name (make e2e-upgrade).
e2e:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -skip TestChainUpgrade ./...

# Runs the harness smoke test.
e2e-smoke:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestSmoke ./...

# Runs the denom rewards lifecycle suite.
e2e-denomrewards:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestDenomRewards ./...

# Runs the AMM and fees suite.
e2e-amm:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestAmm ./...

# Runs the staking rewards and tokenfactory suites (in parallel).
e2e-tokenfactory:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run 'TestStakingRewards|TestTokenFactory' ./...

# Runs the halted denoms suite (halt and unhalt through gov proposals).
e2e-halt:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestHaltedDenoms ./...

# Runs the order-book engine suite.
e2e-orderbook:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestOrderBook ./...

# Runs the trading rewards suite (waits a week of hour epochs, ~20 min).
e2e-tradingrewards:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestTradingRewards ./...

# Runs the burner suite (bze <-> simd through Hermes; waits three week ticks, ~15 min).
e2e-burner:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestBurner ./...

# Runs the blocked inbound IBC suite (bze <-> simd through Hermes).
e2e-ibc:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestBlockedInboundIbc ./...

# Runs the fee destinations and minimum gas price suite.
e2e-fees:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestFees ./...

e2e-cointrunk:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestCointrunk ./...

# Runs the chain upgrade suite: E2E_UPGRADE_FROM -> E2E_UPGRADE_TO through a gov
# plan named E2E_UPGRADE_NAME (see interchaintest/README.md). Run before every tag.
e2e-upgrade:
	cd interchaintest && go test -v -timeout $(E2E_TIMEOUT) -run TestChainUpgrade ./...

# Runs the harness unit tests only (no Docker).
e2e-unit:
	cd interchaintest && go test -short ./...

# Lints the e2e module.
e2e-lint:
	cd interchaintest && golangci-lint run ./...
