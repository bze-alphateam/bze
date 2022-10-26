module github.com/bze-alphateam/bze

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.45.9
	github.com/cosmos/ibc-go v1.2.2
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.5.0
	github.com/stretchr/testify v1.8.0
	github.com/tendermint/spm v0.1.8
	github.com/tendermint/tendermint v0.34.21
	github.com/tendermint/tm-db v0.6.6
	golang.org/x/text v0.3.8 // indirect
	google.golang.org/genproto v0.0.0-20221014213838-99cd37c6964a
	google.golang.org/grpc v1.50.1
)

replace (
	github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
