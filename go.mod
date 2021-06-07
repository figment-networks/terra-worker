module github.com/figment-networks/terra-worker

go 1.16

require (
	github.com/bearcherian/rollzap v1.0.2
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/figment-networks/indexer-manager v0.3.8
	github.com/figment-networks/indexing-engine v0.2.1
	github.com/golang/mock v1.5.0
	github.com/google/uuid v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rollbar/rollbar-go v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.33.9
	github.com/terra-project/core v0.4.5
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/grpc v1.36.0
)

replace github.com/cosmos/ledger-cosmos-go => github.com/terra-project/ledger-terra-go v0.11.1-terra

replace github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4

replace github.com/CosmWasm/go-cosmwasm => github.com/terra-project/go-cosmwasm v0.10.4
