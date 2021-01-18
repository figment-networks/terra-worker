module github.com/figment-networks/terra-worker

go 1.15

require (
	github.com/bearcherian/rollzap v1.0.2
	github.com/cosmos/cosmos-sdk v0.39.1
	github.com/figment-networks/indexer-manager v0.0.8
	github.com/figment-networks/indexing-engine v0.1.14
	github.com/golang/mock v1.4.4
	github.com/google/uuid v1.1.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rollbar/rollbar-go v1.2.0
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.33.8
	github.com/terra-project/core v0.4.0
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	google.golang.org/grpc v1.33.1
)

replace github.com/CosmWasm/go-cosmwasm => github.com/terra-project/go-cosmwasm v0.10.1-terra
