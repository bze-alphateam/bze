PACKAGES=$(shell go list ./... | grep -v '/simulation')
PACKAGE_NAME:=github.com/bze-alphateam/bze-v5
GOLANG_CROSS_VERSION  = v1.19.3
VERSION := $(shell echo $(shell git describe --tags 2>/dev/null ) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
NETWORK ?= mainnet
COVERAGE ?= coverage.txt
BUILDDIR ?= $(CURDIR)/build
LEDGER_ENABLED ?= true

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=bze \
	-X github.com/cosmos/cosmos-sdk/version.AppName=bzed \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'
TESTNET_FLAGS ?=

check_version:
ifneq ($(GO_MINOR_VERSION),19)
		@echo ""
		@echo -e "\033[0;31mERROR:\033[0m Go version 1.19 is required for building bzed. It looks like you are using:"
		@echo ""
		@echo "$(shell go version)"
		@echo "" 
		@echo "There are potential consensus-breaking changes that can occur when running binaries compiled with different versions of Go. Please download Go version 1.19 and retry."
		@echo ""
		@echo "Thank you!"
		exit 1
endif

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

install: check_version lint check-network go.sum
		go install -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) ./cmd/bzed

build: check_version check-network go.sum
		go build -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -o $(BUILDDIR)/bzed ./cmd/bzed

build-win64: check_version check-network go.sum
		go build -buildmode=exe -mod=readonly $(BUILD_FLAGS) $(BUILD_TAGS) -o $(BUILDDIR)/win64/bzed.exe ./cmd/bzed

.PHONY: build

build-linux: check_version check-network go.sum
ifeq ($(OS), Linux)
		GOOS=linux GOARCH=amd64 $(MAKE) build
else
		LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build
endif

build-linux-arm64: check_version check-network go.sum
ifeq ($(OS), Linux)
		GOOS=linux GOARCH=arm64 $(MAKE) build
else
		LEDGER_ENABLED=false GOOS=linux GOARCH=arm64 $(MAKE) build
endif

build-mac: check_version check-network go.sum
ifeq ($(OS), Darwin)
		GOOS=darwin GOARCH=amd64 $(MAKE) build
else
		LEDGER_ENABLED=false GOOS=darwin GOARCH=amd64 $(MAKE) build
endif

build-mac-arm64: check_version check-network go.sum
ifeq ($(OS), Darwin)
		LEDGER_ENABLED=false GOOS=darwin GOARCH=arm64 $(MAKE) build
else
		LEDGER_ENABLED=false GOOS=darwin GOARCH=arm64 $(MAKE) build
endif

build-all: lint all build-win64 build-mac build-mac-arm64 build-linux build-linux-arm64 compress-build

compress-build:
	rm -rf $(BUILDDIR)/compressed
	mkdir $(BUILDDIR)/compressed
	zip -j $(BUILDDIR)/compressed/bze-$(VERSION)-win64.zip $(BUILDDIR)/win64/bzed.exe
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-darwin-arm64.tar.gz -C $(BUILDDIR)/darwin-arm64/ .
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-darwin-amd64.tar.gz -C $(BUILDDIR)/darwin-amd64/ .
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-linux-arm64.tar.gz -C $(BUILDDIR)/linux-arm64/ .
	tar -czvf $(BUILDDIR)/compressed/bze-$(VERSION)-linux-amd64.tar.gz -C $(BUILDDIR)/linux-amd64/ .

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		GO111MODULE=on go mod verify

test: check-network
	@go test $(TEST_TAGS) -v -mod=readonly $(PACKAGES) -coverprofile=$(COVERAGE) -covermode=atomic
.PHONY: test

# look into .golangci.yml for enabling / disabling linters
lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify

# a trick to make all the lint commands execute, return error when at least one fails.
# golangci-lint is run in standalone job in ci
lint-ci:
	@echo "--> Running linter for CI"
	@nix run -f ./. lint-env -c lint-ci

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