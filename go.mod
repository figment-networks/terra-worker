module github.com/figment-networks/terra-worker

go 1.16

// replace github.com/figment-networks/indexing-engine => /Users/pacmessica/.go/src/github.com/figment-networks/indexing-engine

require (
	github.com/bearcherian/rollzap v1.0.2
	github.com/cosmos/cosmos-sdk v0.43.0-rc0
	github.com/davecgh/go-spew v1.1.1
	github.com/figment-networks/indexer-manager v0.4.1
	github.com/figment-networks/indexing-engine v0.4.4
	github.com/gogo/protobuf v1.3.3
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rollbar/rollbar-go v1.2.0
	github.com/tendermint/tendermint v0.34.11
	github.com/terra-money/core v0.5.0-rc0
	go.uber.org/zap v1.17.0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.38.0
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace github.com/cosmos/ledger-cosmos-go => github.com/terra-project/ledger-terra-go v0.11.1-terra

replace github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4

replace github.com/CosmWasm/go-cosmwasm => github.com/terra-project/go-cosmwasm v0.10.4
